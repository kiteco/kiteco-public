import * as actions from '../actions/logs'

const defaultState = {
  logsUrl: null,
  logsGeneratedAt: null,
  profilesUrl: null,
  profilesCapturedAt: null,
}

const uploadLogs = (state, action) => {
  return {
    ...state,
    logsUrl: action.data.url,
    logsGeneratedAt: Date.now(),
  }
}

const capture = (state, action) => {
  return {
    ...state,
    profilesUrl: action.data.url,
    profilesCapturedAt: Date.now(),
  }
}

const logs = (state = defaultState, action) => {
  if (action.type === actions.UPLOAD_LOGS) {
    return uploadLogs(state, action)
  }
  if (action.type === actions.CAPTURE) {
    return capture(state, action)
  }
  return state
}

export default logs
