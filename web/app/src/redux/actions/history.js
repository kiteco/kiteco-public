export const ADD_PAGE_TO_HISTORY = 'add page to history'
export const addPageToHistory = (type, name, path) => ({
  type: ADD_PAGE_TO_HISTORY,
  pageType: type,
  name,
  path
})

export const ADD_TO_HOWTO_HISTORY = 'add to howto history'
export const addToHowtoHistory = (type, name, path) => ({
  type: ADD_TO_HOWTO_HISTORY,
  pageType: type,
  name,
  path
})

export const POP_HOWTO_HISTORY = 'pop howto history'
export const popHowtoHistory = () => ({
  type: POP_HOWTO_HISTORY
})

export const INITIALIZE_HOWTO_HISTORY = 'initialize howto history'
export const initializeHowtoHistory = () => ({
  type: INITIALIZE_HOWTO_HISTORY
})

export const RESET_HOWTO_HISTORY = 'reset howto history'
export const resetHowtoHistory = () => ({
  type: RESET_HOWTO_HISTORY
})

export const SET_PAGE_NAME_FOR_PATH = 'set page name for path'
export const setPageNameForPath = ({name, path, packageName, id}) => ({
  type: SET_PAGE_NAME_FOR_PATH,
  path,
  name,
  packageName,
  id,
})

export const CLEAR_HISTORY = 'clear history'
export const clearHistory = () => ({
  type: CLEAR_HISTORY
})