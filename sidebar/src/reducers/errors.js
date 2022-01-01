import * as fetch from '../actions/fetch'
import * as actions from '../actions/errors'

const APP_EXCEPTION = 'APP_EXCEPTION'

const defaultState = {
  errors: [],
  last: null,
  id: 0,
  message: "",
  show: false,
  //'online' will be false in the case that the WHATWG fetch() promise
  //is rejected - see https://github.github.io/fetch for details
  online: true,
  responsive: true,
  pollCount: 0,
  appException: false,
}

const show = (state, action) => {
  return {
    ...state,
    show: true,
    id: action.id,
  }
}

const hide = (state, action) => {
  return {
    ...state,
    show: false,
  }
}

const disconnected = (state, action) => {
  return {
    ...state,
    errors: [...state.errors, action],
    last: action,
    show: true,
    message: "Unable to connect to Kite",
    online: false,
    responsive: false,
    pollCount: state.pollCount + 1
  }
}

const connected = (state, action) => {
  return {
    ...state,
    show: false,
    online: true,
    responsive: true,
    pollCount: 0,
  }
}

const unresponsive = (state, action) => {
  return {
    ...state,
    errors: [...state.errors, {
      error: action.response,
      type: action.type
    }],
    last: action,
    show: true,
    message: "Kite is unresponsive",
    online: true,
    responsive: false,
    pollCount: state.pollCount + 1
  }
}

const appException = (state, action) => {
  return {
    ...state,
    appException: true,
  }
}

const errors = (state = defaultState, action) => {
  switch (action.type) {
    case fetch.REPORT_KITED_UNREACHABLE:
      return disconnected(state, action)
    case fetch.REPORT_KITED_REACHABLE:
      return connected(state, action)
    case fetch.REPORT_KITED_UNHEALTHY:
      return unresponsive(state, action)
    case actions.SHOW:
      return show(state, action)
    case actions.HIDE:
      return hide(state, action)
    case APP_EXCEPTION:
      return appException(state, action)
    default:
      return state
  }
}

export default errors
