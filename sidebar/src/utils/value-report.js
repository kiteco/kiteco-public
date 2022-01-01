import {compact, flatten, head} from './functional'
import {detailLang, detailGet} from './language-details'

const token = (token, tokenType) => ({token, tokenType})

const patternKwargParameter = (parameter, withType) => {
  let type, value
  if(parameter.types) {
    type = head(parameter.types);
  }
  if (type && type.examples) {
    value = head(type.examples)
  } else if (type) {
    value = type.name
  }

  return token(
    type && parameter.types.length === 1
      ? compact([parameter.name, value]).join('=')
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

const setDefaultValue = (p) => {
  const param = { ...p }
  if(param.language_details.python.default_value) {
    param.default_value = param.language_details.python.default_value[0].repr
  }
  return param
}

const mapInferredValuesToKwargs = (kwargParameters) => {
  kwargParameters && kwargParameters.forEach(kw => {
    if(kw.inferred_value && kw.inferred_value.length) {
      kw.inferred_value_objs = []
      kw.inferred_value.forEach((val, i) => {
        kw.inferred_value_objs.push({ token: val.repr, type: 'val' })
        if(i < kw.inferred_value.length - 1) {
          kw.inferred_value_objs.push({ token: ' ❘ ', type: 'punc' })
        }
      })
    }
  })
}

export const normalizeValueReportFromSymbol = data => {
  // data = JSON.parse(data);
  data = {
    ...data,
    value: data.symbol.value[0],
  }

  if (data.value.kind === 'function') {
    if (data.value.details.function) {
      if (data.value.details.function.signatures.length) {
          data.value.details.function.structured_patterns = data.value.details.function.signatures.map(s => structurePattern(data.value.repr, s))
      }
      if(!data.value.details.function.parameters) {
        data.value.details.function.parameters = []
      }
      data.value.details.function = {
        ...data.value.details.function,
        vararg: data.value.details.function.language_details.python.vararg,
        kwarg: data.value.details.function.language_details.python.kwarg,
        signature_parameters: data.value.details.function.parameters
          .filter(p => !p.language_details.python.keyword_only)
          .map(p => setDefaultValue(p)),
        keyword_only_parameters: data.value.details.function.parameters
          .filter(p => p.language_details.python.keyword_only)
          .map(p => setDefaultValue(p)),
      }
      mapInferredValuesToKwargs(data.value.details.function.language_details.python.kwarg_parameters)
    }
  } else if (data.value.kind === 'type' && data.value.details.type.language_details.python && data.value.details.type.language_details.python.constructor) {
    if (data.value.details.type.language_details.python.constructor.signatures.length) {
      data.value.details.function = {
        ...data.value.details.function,
        ...data.value.details.type.language_details.python.constructor,
        structured_patterns: data.value.details.type.language_details.python.constructor.signatures.map(s => structurePattern(data.value.repr, s)),
        vararg: data.value.details.type.language_details.python.constructor.language_details.python.vararg,
        kwarg: data.value.details.type.language_details.python.constructor.language_details.python.kwarg,
        signature_parameters: data.value.details.type.language_details.python.constructor.parameters 
          ? data.value.details.type.language_details.python.constructor.parameters
              .filter(p => !p.language_details.python.keyword_only)
              .map(p => setDefaultValue(p)) 
          : [],
        keyword_only_parameters: data.value.details.type.language_details.python.constructor.parameters
          ? data.value.details.type.language_details.python.constructor.parameters
              .filter(p => p.language_details.python.keyword_only)
              .map(p => setDefaultValue(p)) 
          : [],
      }
      mapInferredValuesToKwargs(data.value.details.function.language_details.python.kwarg_parameters)
    }
  }

  return data;
}
