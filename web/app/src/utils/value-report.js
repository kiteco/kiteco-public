import {compact, flatten, head} from './functional'
import {detailLang, detailGet} from './language-details'

const token = (token, tokenType) => ({token, tokenType})

const signatureKwargParameters = (signature, withType) => {
  const kwarg = detailGet(signature, 'kwarg')
  return kwarg ? token(`**${kwarg.name}`, 'argument') : null;
}

const signatureVarargParameters = (signature) => {
  const vararg = detailGet(signature, 'vararg')
  return vararg ? token(`*${vararg.name}`, 'argument') : null
}

const gatherSignatureParameters = (data, withType) => {
  const lang = detailLang(data);
  const baseArgs = (data.parameters || []).map(a => token(a.name, 'argument'));
  switch (lang) {
    case 'python':
      return [
        baseArgs,
        signatureVarargParameters(data),
        signatureKwargParameters(data),
      ]
    default:
      return [
        baseArgs,
      ];
  }
};

const structureSignature = (name, data) => {
  return {
    signature: flatten([
      token(`${name}(`, 'text'),
      compact(flatten(gatherSignatureParameters(data))).reduce((m, o, i, a) => {
        return i < a.length - 1
          ? m.concat([o, {token: ',', tokenType: 'text'}])
          : m.concat(o)
      }, []),
      token(')', 'text'),
    ])
  };
};

const patternKwargParameter = (parameter, withType) => {
  const type = head(parameter.types);
  return token(
    parameter.types.length === 1
      ? compact([parameter.name, type.name]).join('=')
      : parameter.name
    , 'argument');
};

const patternKwargParameters = (signature, withType) => {
  const kwargs = detailGet(signature, 'kwargs')
  return kwargs && kwargs.length
    ? kwargs.map(p => patternKwargParameter(p))
    : null;}

const gatherPatternParameters = (data, withType) => {
  const lang = detailLang(data);
  const baseArgs = (data.args || []).map(a => token(a.name, 'argument'));
  switch (lang) {
    case 'python':
      return [
        baseArgs,
        patternKwargParameters(data),
      ]
    default:
      return [
        baseArgs,
      ];
  }
};

const structurePattern = (name, data) => ({
  signature: flatten([
    token(`${name}(`, 'text'),
    compact(flatten(gatherPatternParameters(data))).reduce((m, o, i, a) => {
      return i < a.length - 1
        ? m.concat([o, {token: ',', tokenType: 'text'}])
        : m.concat(o)
    }, []),
    token(')', 'text'),
  ])
});

export const normalizeSymbolId = (id) => {
  const firstIndex = id.indexOf(';')
  //hueristic for detecting global vs. local symbol
  //2 subsequent ';' will indicate no user_id
  if(
    (firstIndex !== id.length - 1 || firstIndex !== -1) 
    && id[firstIndex + 1] === ';'
  ) {
    //normalize
    return id.replace(/(python;*)(.*)/g, (match, subMatch1, subMatch2) => {
      return `${subMatch1.substring(0, subMatch1.indexOf(';') + 1)}${subMatch2.replace(/;/g, '.')}`
    })
  } else {
    //just use full id
    return id
  }
}

export const normalizeValueReport = data => {
  data = {
    ...data,
    value: data.symbol.value[0],
  }
  //for client routing compatibility with new id structure
  data.value.normalizedId = normalizeSymbolId(data.value.id)
  if (data.value.kind === 'function') {
    if (data.value.details.function) {
      if (data.value.details.function.signatures.length) {
        data.value.details.function = {
          ...data.value.details.function,
          structured_patterns: data.value.details.function.signatures.map(s => structurePattern(data.value.repr, s))
        }
      }
      data.value.details.function.structured_signature = structureSignature(data.value.repr, data.value.details.function);
    }
  } else if (data.value.kind === 'type' && data.value.details.type.language_details.python && data.value.details.type.language_details.python.constructor) {
    data.value.details.function = {
      structured_signature: structureSignature(data.value.repr, data.value.details.type.language_details.python.constructor)
    };

    if (data.value.details.type.language_details.python.constructor.signatures.length) {
      data.value.details.function = {
        ...data.value.details.function,
        ...data.value.details.type.language_details.python.constructor,
        structured_patterns: data.value.details.type.language_details.python.constructor.signatures.map(s => structurePattern(data.value.repr, s))
      }
    }
  }

  //for client routing compatibility with new id structure
  if(data.value.ancestors) {
    data.value.ancestors = data.value.ancestors.map(ancestor => ({
      ...ancestor,
      normalizedId: normalizeSymbolId(ancestor.id)
    }))
  }
  if(data.value.kind === 'module' || data.value.kind === 'type') {
    const members = data.value.details[data.value.kind].members
    if(members) {
      data.value.details[data.value.kind].members = members.map(member => ({
        ...member,
        normalizedId: normalizeSymbolId(member.id)
      }))
    }
  }

  return data;
}

export const normalizeMembersReport = data => {
  //for client routing compatibility with new id structure
  data.members = data.members.map(member => ({
    ...member,
    normalizedId: normalizeSymbolId(member.id)
  }))
  return data
}