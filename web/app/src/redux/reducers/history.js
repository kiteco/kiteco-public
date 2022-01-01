import * as actions from '../actions/history'
import { PATH_ID_INDEX, USER_ID_INDEX } from '../../utils/route-parsing'
import { LOCATION_CHANGE } from 'connected-react-router'

const MAX_HISTORY_LENGTH = 25

const defaultState = {
  previousPages: [],
  howToHistory: []
}

const addPageToHistory = (state, action) => {
  if(state.previousPages[0] && state.previousPages[0].path === action.path){
    //avoid subsequent duplicates
    return state
  }
  const newState = {
    ...state,
    previousPages: [{
      pageType: action.pageType,
      name: action.name,
      path: action.path,
    }, ...state.previousPages].slice(0, MAX_HISTORY_LENGTH)
  }
  return newState
}

const setPageNameForPath = (state, action) => {
  //assume will be most recent of that path
  const index = state.previousPages.findIndex(page => {
    return page.id === action.id
  })
  if(index !== -1) {
    //preserve object ref
    const page = state.previousPages[index]
    page.name = action.name
    page.packageName = action.packageName
    const previousPages = [
      ...state.previousPages.slice(0, index),
      state.previousPages[index],
      ...state.previousPages.slice(index + 1)
    ]
    const newState = {
      ...state,
      previousPages
    }
    return newState
  } else {
    return state
  }
}

const getIdFromPath = (path) => {
  const tokens = path.split("/").slice(1)
  //assumes /python/examples/1111111 structure
  return `${tokens[0]}-example-${tokens[2]}`
}

const addToHowtoHistory = (state, action) => {
  if(state.previousPages[0] && state.previousPages[0].path === action.path){
    //avoid subsequent duplicates
    return state
  }
  const page = {
    pageType: action.pageType,
    name: action.name,
    path: action.path,
    id: getIdFromPath(action.path)
  }
  const newState = {
    ...state,
    previousPages: [page, ...state.previousPages].slice(0, MAX_HISTORY_LENGTH),
    howToHistory: [page, ...state.howToHistory]
  }
  return newState
}

const popHowtoHistory = (state, action) => {
  const newState = {
    ...state,
    howToHistory: state.howToHistory.slice(1)
  }
  return newState
}

const resetHowtoHistory = (state, action) => {
  const newState = {
    ...state,
    howToHistory: []
  }
  return newState
}

const initializeHowtoHistory = (state, action) => {
  if(state.previousPages.length > 0) {
    const newState = {
      ...state,
      howToHistory: [state.previousPages[0]]
    }
    return newState
  }
  return state
}

const clearHistory = (state, action) => {
  //keep howToHistory so breadcrumbs stay good
  return {
    ...state,
    previousPages: []
  }
}

const getTypeFromPath = (path) => {
  if(/^\/(python|js)\/examples\//.test(path)) {
    return 'howto'
  }
  return 'identifier'
}

const getNameFromPath = (path) => {
  let name = ""
  if(/^\/(python|js)\/docs\//.test(path)) {
    let dottedPath = path.substr(path.lastIndexOf('/') + 1)
    if(dottedPath.includes(';')) {
      // looking for raw ids to convert
      const dottedPathArr = dottedPath.split(';')
      let dottedPathStartIndex = dottedPathArr[1] === "" //userId presence hueristic
        ? USER_ID_INDEX //local
        : PATH_ID_INDEX //global
      dottedPath = dottedPathArr.slice(dottedPathStartIndex).filter(path => path !== '').join('.')
    }
    const tokens = dottedPath.split('.').reverse()
    if(tokens.length > 1) {
      name = `${tokens[1]}.${tokens[0]}`
    }
    if(tokens.length === 1) {
      name = tokens[0]
    }
  }
  return name
}

const shouldAddToHistory = (path) => {
  return /^\/(python|js)\/examples\//.test(path) || /^\/(python|js)\/docs\//.test(path)
}

const handleLocationChange = (state, action) => {
  if(action.payload.state && (action.payload.state.navAction === 'POP' || action.payload.state.navAction === 'REPLACE') ) {
    if(state.howToHistory.length !== 0) {
      return resetHowtoHistory(state, action)
    }
  }
  //last check is for the case of a page refresh, when there may be stale state on the location action
  else if(!action.payload.state || !action.payload.state.howToBreadcrumb || state.previousPages.length === 0) {
    const type = getTypeFromPath(action.payload.pathname)
    if(type !== 'howto' && state.howToHistory.length !== 0) {
      return resetHowtoHistory(state, action)
    }
    if(type === 'howto' && state.howToHistory.length === 0 && state.previousPages.length > 0) {
      state = initializeHowtoHistory(state, action)
    }
    if(shouldAddToHistory(action.payload.pathname)) {
      const method = type === 'howto' ? addToHowtoHistory : addPageToHistory
      return method(
        state,
        {
          pageType: type,
          name: getNameFromPath(action.payload.pathname),
          path: action.payload.pathname
        }
      )
    }
  }
  return state
}

const history = (state = defaultState, action) => {
  switch(action.type) {
    case actions.ADD_PAGE_TO_HISTORY:
      return addPageToHistory(state, action)
    case actions.SET_PAGE_NAME_FOR_PATH:
      //will this also set in howToHistory, if they share the same refs?
      return setPageNameForPath(state, action)
    case actions.ADD_TO_HOWTO_HISTORY:
      return addToHowtoHistory(state, action)
    case actions.POP_HOWTO_HISTORY:
      return popHowtoHistory(state, action)
    case actions.RESET_HOWTO_HISTORY:
      return resetHowtoHistory(state, action)
    case actions.INITIALIZE_HOWTO_HISTORY:
      return initializeHowtoHistory(state, action)
    case actions.CLEAR_HISTORY:
      return clearHistory(state, action)
    case LOCATION_CHANGE:
      return handleLocationChange(state, action)
    default:
      return state
  }
}

export default history
