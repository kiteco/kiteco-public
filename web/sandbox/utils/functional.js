
export const head = a => a[0]

export const compact = a => a.filter(v => v != null && v !== '')

export const flatten = a =>
  a.reduce((m, v) => m.concat(Array.isArray(v) ? flatten(v) : v), [])

export const arrayEquals = (a, b, c = (v => v)) =>
  a.length === b.length && a.every((v, i) => c(v) === c(b[i]))
