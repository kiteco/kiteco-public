import * as actions from '../actions/search'

const defaultState = {
  autosearchEnabled: true,
  id: "",
}

const enableAutosearch = (state, action) => {
  return {
    ...state,
    autosearchEnabled: true,
  }
}

const disableAutosearch = (state, action) => {
  return {
    ...state,
    autosearchEnabled: false
  }
}

const autosearchEvent = (state, action) => ({
  ...state,
  id: action.data,
})

const search = (state = defaultState, action) => {
  switch (action.type) {
    case actions.ENABLE_AUTOSEARCH:
      return enableAutosearch(state, action)
    case actions.DISABLE_AUTOSEARCH:
      return disableAutosearch(state, action)
    case actions.AUTOSEARCH_EVENT:
      return autosearchEvent(state, action)
    default:
      return state;
  }
}

export default search
