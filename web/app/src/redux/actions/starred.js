export const ADD_STARRED = 'add starred'
export const addStarred = (name, path, pageType) => ({
  type: ADD_STARRED,
  name,
  path,
  pageType
})

export const REMOVE_STARRED = 'remove starred'
export const removeStarred = (path) => ({
  type: REMOVE_STARRED,
  path
})

export const STARRED_CLICKED = 'starred clicked'
export const starredClicked = (path) => ({
  type: STARRED_CLICKED,
  path
})

export const CLEAR_STARRED = 'clear starred'
export const clearStarred = () => ({
  type: CLEAR_STARRED
})