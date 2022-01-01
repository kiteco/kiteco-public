import * as actions from '../actions/starred'

//proper data structure for storing starred?
//Best way of sorting? => we can use adding and being clicked as proxies to go to the top
const defaultState = {
  starredPaths: [],
  starredMap: {}
}

const addStarred = (state, action) => {
  const starredItem = {
    name: action.name,
    path: action.path,
    pageType: action.pageType
  }
  const newState = {
    ...state,
    starredPaths: [action.path, ...state.starredPaths],
    starredMap: {
      ...state.starredMap,
      [action.path]: starredItem
    }
  }
  return newState
}

const removeStarred = (state, action) => {
  const newState = {
    ...state,
    starredPaths: state.starredPaths.filter(path => path !== action.path),
    starredMap : {
      ...state.starredMap,
      [action.path]: undefined
    }
  }
  return newState
}

const starredClicked = (state, action) => {
  const index = state.starredPaths.findIndex(path => path === action.path)
  return {
    ...state,
    starredPaths: [
      state.starredPaths[index],
      ...state.starredPaths.slice(0, index),
      ...state.starredPaths.slice(index + 1)
    ]
  }
}

const clearStarred = (state, action) => {
  return {
    ...defaultState
  }
}

const starred = (state = defaultState, action) => {
  switch(action.type) {
    case actions.ADD_STARRED:
      return addStarred(state, action)
    case actions.REMOVE_STARRED:
      return removeStarred(state, action)
    case actions.STARRED_CLICKED:
      return starredClicked(state, action)
    case actions.CLEAR_STARRED:
      return clearStarred(state, action)
    default:
      return state
  }
}

export default starred
