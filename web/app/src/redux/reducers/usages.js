import * as actions from '../actions/usages'

const defaultState = {
  status: "loading",
  data: {},
  language: null,
  identifier: null,
  error: null,
}

const loadUsages = (state, action) => {
  return {
    ...state,
    status: "loading",
    language: action.language,
    identifier: action.identifier
  }
}

const loadUsagesFailed = (state, action) => {
  return {
    ...state,
    status: "failed",
    error: action.error
  }
}

const registerUsages = (state, action) => {
  return {
    ...state,
    status: "success",
    data: action.data
  }
}

const usages = (state = defaultState, action) => {
  switch(action.type) {
    case actions.LOAD_USAGES:
      return loadUsages(state, action)
    case actions.LOAD_USAGES_FAILED:
      return loadUsagesFailed(state, action)
    case actions.REGISTER_USAGES:
      return registerUsages(state, action)
    default:
      return state
  }
}

export default usages
