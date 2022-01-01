const emptyFolder = {
  files: [],
  folders: {},
  name: "",
}

const syncedToFile = ({ hash: hashed_content }, name) => ({
  hashed_content,
  name,
})

/* syncedToFolder turns a folder node into a folder
 * with the following properties:
 *
 * folders    []folder
 * files      []files
 * name       string: name of the folder
 */
export const syncedToFolder = ({ children }, name) => {
  const folders = {}
  Object.keys(children)
    .filter(f => children[f].children)
    .forEach(f => folders[f] = syncedToFolder(children[f], f))
  const files = Object.keys(children)
    .filter(f => !children[f].children)
    .map(f => syncedToFile(children[f], f))
  return {
    folders,
    files,
    name,
  }
}

/* unravel takes in a path and a root folder
 * and returns the folder at the given path
 *
 * example:
 *    path: /a/b
 *    root: { folders: { a: { folders: { b: { name: "b", folders, files }}}}}
 *    separator: "/"
 *
 *    return: { name: "/a/b", folders, files }
 * NOTE: this currently ignores empty folder names
 */
export const unravel = (path, root, separator) => {
  const folders = path.split(separator)
  const folder = folders
    .reduce(
      (parent, f) => (
        (f && parent)
        ? parent.folders[f]
        : parent // ignore empty folder names
      ),
      root
    ) || { ...emptyFolder }
  return {
    ...folder,
    name: path,
  }
}
