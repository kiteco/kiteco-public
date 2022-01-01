/* EDITOR */

export const sandboxCompletionsPath = (id) => {
  return `/api/websandbox/completions${id ? `?id=${id}` : ''}`
}

export const signaturedCompletionsPath = (id) => {
  return `/api/websandbox/signatured-completions${id ? `?id=${id}` : ''}`
}