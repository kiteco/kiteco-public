
const MAX_DESCRIPTION_LENGTH = 140
const MAX_LENGTH = 400

const signature = ({ doc, name }) => {
  if (!doc.patterns) {
    return ''
  }
  if (doc.patterns.length === 0) {
    let parameters = doc.parameters
    if (!parameters) {
      return ''
    }
    if (parameters.length > 5) {
      parameters = [ ...parameters.slice(0, 5), { name: "…" } ]
    }
    return `${name}(${parameters.map(p => p.name).join(", ")})`
  }
  const first = doc.patterns[0]
  return first.reduce((all, f) => (all + f.token), "")
}

const description = ({ doc }) => {
  let content;
  if(doc.description_html) {
    const parser = new DOMParser().parseFromString(doc.description_html, "text/html")
    content = parser.body.textContent || ""
  } else {
    content = doc.documentation_str || ""
  }
  
  return content.length > MAX_DESCRIPTION_LENGTH
    ? content.substring(0, MAX_DESCRIPTION_LENGTH - 1) + "…"
    : content
}

const numFunctionCalls = ({ doc }) => {
  if (!doc.patterns || doc.patterns.length < 2) {
    return ""
  }
  return `${doc.patterns.length} common ways to call this function`
}

const examples = ({ doc }) => {
  if (!doc.exampleIds || doc.exampleIds.length < 1) {
    return ""
  }
  return `examples: ${doc.exampleIds.map(e => e.title).join(", ")}`
}

const members = ({ doc }) => {
  if (!doc.members || doc.members.length === 0) {
    return ""
  }
  return `${doc.members.length} member${(doc.members.length > 1) ? "s": ""}`
}

const topMembers = ({ doc }) => {
  return `top members: ${doc.members.slice(0, 5).map(m => m.name).join(', ')}`
}

const fullyQualifiedName = ({ doc, name }) => {
  return [ ...doc.ancestors.map(a => a.name), name ].join(".")
}

export const func = ({ doc, name }) => {
  const components = [
    signature({ doc, name }),
    description({ doc }),
    numFunctionCalls({ doc }),
    examples({ doc }),
  ].filter(c => c)
  const all = components.join(" - ")
  return all.length > MAX_LENGTH
    ? all.slice(0, MAX_LENGTH - 1) + "…"
    : all
}

export const type = ({ doc, name }) => {
  const descrip = description({ doc })
  const components = ( descrip
    ? [
        name,
        members({ doc }),
        descrip,
      ]
    : [
        signature({ doc, name }),
        members({ doc }),
      ]
  ).filter(c => c)
  const all = components.join(" - ")
  return all.length > MAX_LENGTH
    ? all.slice(0, MAX_LENGTH - 1) + "…"
    : all
}

export const moduleMeta = ({ doc, name }) => {
  const descrip = description({ doc })
  const components = ( descrip
    ? [
        fullyQualifiedName({ doc, name }),
        descrip,
      ]
    : [
        fullyQualifiedName({ doc, name }),
        topMembers({ doc }),
      ]
  ).filter(c => c)
  const all = components.join(" - ")
  return all.length > MAX_LENGTH
    ? all.slice(0, MAX_LENGTH - 1) + "…"
    : all
}

export const fallback = ({ doc, name }) => {
  const components = [
    name,
    description({ doc }),
  ].filter(c => c)
  const all = components.join(" - ")
  return all.length > MAX_LENGTH
    ? all.slice(0, MAX_LENGTH - 1) + "…"
    : all
}
