import { queryIdToRoute, isLocalId } from "./route-parsing";
import { headOrDefault } from "./functional";

const normalizeAncestors = value => {
  const ancestors = [];
  let path = "/python/docs/";
  if (value.ancestors) {
    value.ancestors.forEach((ancestor, i) => {
      //ignore if ends with ';' - this is an hueristic
      //to identify files and directories that get added
      //as local symbol top level ancestors
      if (!ancestor.id.endsWith(";")) {
        //need local symbol handling here
        if (isLocalId(ancestor.id)) {
          path = `${path}${ancestor.id}`;
        } else {
          if (i === 0) {
            path = `${path}${ancestor.name}`;
          } else {
            path = `${path}.${ancestor.name}`;
          }
        }
        ancestors.push({
          ...ancestor,
          path
        });
      }
    });
  }
  return ancestors;
};

const calcPopularity = (total, i) => {
  const value = (total - i) / total;
  if (value < 0.1) return 1;
  if (value < 0.3) return 2;
  if (value < 0.6) return 3;
  if (value < 0.8) return 4;
  return 5;
};

const normalizeMembers = (members, language) => {
  const newMembers = [];
  if (members) {
    members.forEach((member, i) => {
      const { id, kind } = member.value[0];
      newMembers.push({
        name: member.name,
        link: `/${language}/docs${queryIdToRoute(id)}`,
        type: kind,
        popularity: calcPopularity(members.length, i)
      });
    });
  }
  return newMembers;
};

const tightenFilename = name => {
  return name.substr(name.lastIndexOf("/") + 1);
};

export const normalizeLocalCodeExamples = (
  data,
  fromUsagesEndpoint = false
) => {
  const usages = fromUsagesEndpoint ? data.usages : data.report.usages;
  const newUsages = [];
  if (usages) {
    usages.forEach(usage => {
      newUsages.push({
        ...usage,
        filename: tightenFilename(usage.filename)
      });
    });
  }
  return {
    usages: newUsages,
    total: fromUsagesEndpoint ? data.total : data.report.total_usages
  };
};

const getDescription = data => {
  if (data.report.description_html) return data.report.description_html;
  return "";
};

const getDocString = data => {
  if (data.report.description_text) return data.report.description_text;
  if (data.symbol.value[0]) return data.symbol.value[0].synopsis;
  return data.symbol.synopsis;
};

const normalizeModule = data => {
  const value = data.symbol.value[0];
  const moduleObj = {};
  moduleObj.type = value.kind;
  moduleObj.name = data.symbol.name;
  moduleObj.ancestors = normalizeAncestors(value);
  moduleObj.members = normalizeMembers(
    value.details.module.members,
    data.language
  );
  moduleObj.description_html = getDescription(data);
  moduleObj.documentation_str = getDocString(data);
  moduleObj.exampleIds = data.report.examples;
  moduleObj.localCodeExamples = normalizeLocalCodeExamples(data);
  moduleObj.totalLocalCodeUsages = data.report.total_usages;
  return moduleObj;
};

const normalizeParameters = (params, language_details) => {
  const parameters = [];
  if (params) {
    params.forEach(p => {
      let default_value;
      if (p.language_details.python.default_value) {
        default_value = {
          type: p.language_details.python.default_value[0].type,
          repr: p.language_details.python.default_value[0].repr
        };
      }
      parameters.push({
        name: p.name,
        default_value
      });
    });
  }
  if (language_details.vararg && language_details.vararg.name) {
    parameters.push({ name: `*${language_details.vararg.name}` });
  }
  if (language_details.kwarg && language_details.kwarg.name) {
    parameters.push({ name: `**${language_details.kwarg.name}` });
  }
  return parameters;
};

const createPatternArgs = signature => {
  const args = [];
  if (signature.args) {
    signature.args.forEach((arg, i) => {
      args.push({
        token: arg.name
      });
      if (i < signature.args.length - 1) {
        args.push({
          token: ",",
          tokenType: "punctuation"
        });
      }
    });
  }
  if (signature.language_details.python.kwargs !== null) {
    if (signature.args && signature.args.length > 0) {
      args.push({
        token: ",",
        tokenType: "punctuation"
      });
    }
    signature.language_details.python.kwargs.forEach((kw, i) => {
      args.push({
        token: kw.name
      });
      if (kw.types && kw.types.length === 1 && kw.types[0].name) {
        args.push(
          {
            token: "=",
            tokenType: "punctuation"
          },
          {
            token: headOrDefault(kw.types[0].examples, kw.types[0].name),
            tokenType: "keyword"
          }
        );
      }
      if (i < signature.language_details.python.kwargs.length - 1) {
        args.push({
          token: ",",
          tokenType: "punctuation"
        });
      }
    });
  }
  return args;
};

const normalizePatterns = (signatures, name) => {
  //TODO: fill in according to StructuredCodeBlock
  const patterns = [];
  if (signatures) {
    signatures.forEach(sig => {
      patterns.push([
        { token: name },
        { token: "(", tokenType: "punctuation" },
        ...createPatternArgs(sig),
        { token: ")", tokenType: "punctuation" }
      ]);
    });
  }
  return patterns;
};

const normalizeKwargs = details => {
  const kwargs = [];
  if (details && details.kwarg_parameters) {
    details.kwarg_parameters.forEach(kw => {
      kwargs.push({
        name: kw.name,
        types: kw.inferred_value
      });
    });
  }
  return kwargs;
};

const normalizeFunction = data => {
  const value = data.symbol.value[0];
  const functionObj = {};
  functionObj.type = value.kind;
  functionObj.name = data.symbol.name;
  functionObj.ancestors = normalizeAncestors(value);
  functionObj.description_html = getDescription(data);
  functionObj.documentation_str = getDocString(data);
  functionObj.exampleIds = data.report.examples;
  functionObj.parameters = value.details.function
    ? normalizeParameters(
        value.details.function.parameters,
        value.details.function.language_details.python
      )
    : [];
  functionObj.returnValues = value.details.function
    ? value.details.function.return_value
    : [];
  functionObj.patterns = value.details.function
    ? normalizePatterns(value.details.function.signatures, data.symbol.name)
    : [];
  functionObj.kwargs = value.details.function
    ? normalizeKwargs(value.details.function.language_details.python)
    : [];
  functionObj.localCodeExamples = normalizeLocalCodeExamples(data);
  functionObj.totalLocalCodeUsages = data.report.total_usages;
  return functionObj;
};

const normalizeInstanceTypes = (instance, language) => {
  return instance.type
    ? instance.type.map(type => ({
        name: type.repr.substr(type.repr.lastIndexOf(".") + 1),
        path: `/${language}/docs${queryIdToRoute(type.id)}`
      }))
    : [];
};

const normalizeInstance = data => {
  const value = data.symbol.value[0];
  const instanceObj = {};
  instanceObj.type = value.kind;
  instanceObj.name = data.symbol.name;
  instanceObj.ancestors = normalizeAncestors(value);
  instanceObj.exampleIds = data.report.examples;
  instanceObj.totalLocalCodeUsages = data.report.total_usages;
  instanceObj.localCodeExamples = normalizeLocalCodeExamples(data);
  instanceObj.description_html = getDescription(data);
  instanceObj.documentation_str = getDocString(data);
  instanceObj.types = normalizeInstanceTypes(
    value.details.instance,
    data.language
  );
  return instanceObj;
};

const normalizeType = data => {
  const value = data.symbol.value[0];
  const constructor = value.details.type.language_details.python.constructor;
  const typeObj = {};
  typeObj.type = value.kind;
  typeObj.name = data.symbol.name;
  typeObj.description_html = getDescription(data);
  typeObj.documentation_str = getDocString(data);
  typeObj.ancestors = normalizeAncestors(value);
  typeObj.members = normalizeMembers(value.details.type.members, data.language);
  typeObj.exampleIds = data.report.examples;
  typeObj.localCodeExamples = normalizeLocalCodeExamples(data);
  typeObj.totalLocalCodeUsages = data.report.total_usages;
  if (constructor) {
    typeObj.patterns = normalizePatterns(
      constructor.signatures,
      data.symbol.name
    );
    typeObj.kwargs = normalizeKwargs(constructor.language_details.python);
    typeObj.parameters = normalizeParameters(
      constructor.parameters,
      constructor.language_details.python
    );
    typeObj.returnValues = constructor.return_value;
  }
  return typeObj;
};

const normalizeGeneric = data => {
  const value = data.symbol.value[0];
  const genericObj = {};
  genericObj.type = value.kind;
  genericObj.name = data.symbol.name;
  genericObj.description_html = getDescription(data);
  genericObj.documentation_str = getDocString(data);
  genericObj.ancestors = normalizeAncestors(value);
  genericObj.exampleIds = data.report.examples;
  genericObj.localCodeExamples = normalizeLocalCodeExamples(data);
  genericObj.totalLocalCodeUsages = data.report.total_usages;
  return genericObj;
};

const attachCanonicalLink = (data, obj) => {
  if (data.link_canonical) {
    obj.canonical_link = data.link_canonical;
  }
};

export const normalizeSymbolReport = data => {
  let obj;
  switch (data.symbol.value[0].kind) {
    case "module":
      obj = normalizeModule(data);
      break;
    case "function":
      obj = normalizeFunction(data);
      break;
    case "instance":
      obj = normalizeInstance(data);
      break;
    case "type":
      obj = normalizeType(data);
      break;
    default:
      obj = normalizeGeneric(data);
  }
  attachCanonicalLink(data, obj);
  if (data.answers_links) {
    obj.answers_links = data.answers_links;
  }
  return obj;
};

export const integrateMembers = (data, members, language) => {
  return {
    ...data,
    members: normalizeMembers(members, language)
  };
};
