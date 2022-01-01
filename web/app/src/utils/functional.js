
export const head = a => a[0]

export const headOrDefault = (a, d) => a && a[0] ? a[0] : d

export const compact = a => a.filter(v => v != null && v !== '')

export const flatten = a =>
  a.reduce((m, v) => m.concat(Array.isArray(v) ? flatten(v) : v), [])

export const arrayEquals = (a, b, c = (v => v)) => (
  a === b || (a && b &&
  a.length === b.length && a.every((v, i) => c(v) === c(b[i]))))

/**
 * get object from array by id
 * @param id - object id
 * @param array - array of objects
 * @returns {*|Object|{}}
 */
export const getObjectById = (id, array) => array.find(item => item.id === id);

export const filterPostsByPermalink = (arr, permalink) => {
 return arr.filter(post => post.permalink === permalink).pop() || {}
}

/**
 * get array of objects
 * @param array - array of objects
 * @param prop - select property
 * @returns {Array|*[]}
 */
export const getArrayOfObjects = (array, prop) => array.filter(obj => obj.filterCategory && obj.filterCategory.includes(prop.toLowerCase()) && obj.show)
