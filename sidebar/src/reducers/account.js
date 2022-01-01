import * as actions from '../actions/account'

const defaultState = {
  status: "untried",
  user: {},
  details: {},
  plan: {},
  metricsId: "",
}

const loadUser = (state, action) => {
  return {
    ...state,
    status: "logged-in",
    user: action.data,
  }
}

const failedAccountFetch = (state, action) => {
  return {
    ...state,
    status: "logged-out",
  }
}

const identifyMetricsId = (state, action) => {
  return {
    ...state,
    metricsId: action.data,
  }
}

const account = (state = defaultState, action) => {
  switch (action.type) {
    case actions.LOAD_USER:
    case actions.LOG_IN:
    case actions.CREATE_ACCOUNT:
      return loadUser(state, action)
    case actions.FAILED_ACCOUNT_FETCH:
      return failedAccountFetch(state, action)
    case actions.LOG_IN_FAILED:
      return { ...defaultState, status: "logged-out" }
    case actions.LOG_OUT:
      return { ...defaultState, status: "logged-out" }
    case actions.IDENTIFY_METRICS_ID:
      return identifyMetricsId(state, action)
    default:
      return state
  }
}

export default account
