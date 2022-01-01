import * as actions from '../actions/polling'

const defaultState = {
  isPolling: false,
  restartError: false,
  attemptRestart: false,
  restartSuccess: false,
  pollingSuccess: false,
  noSupport: false,
}

const reportPolling = (state, action) => ({
  ...state,
  isPolling: action.isPolling,
  restartError: false,
  restartSuccess: false,
  pollingSuccess: false,
  noSupport: false,
})

const attemptRestart = (state, action) => ({
  ...state,
  isPolling: false,
  restartError: false,
  attemptRestart: true,
  restartSuccess: false,
  pollingSuccess: false,
  noSupport: false,
})

const pollingSuccessful = (state, action) => ({
  ...state,
  isPolling: false,
  restartError: false,
  restartSuccess: false,
  pollingSuccess: true,
  noSupport: false,
})

const restartSuccessful = (state, action) => ({
  ...state,
  isPolling: false,
  restartError: false,
  attemptRestart: false,
  restartSuccess: true,
  pollingSuccess: false,
  noSupport: false,
})

const restartError = (state, action) => ({
  ...state,
  isPolling: false,
  restartError: true,
  attemptRestart: false,
  restartSuccess: false,
  pollingSuccess: false,
  noSupport: false,
})

const noSupport = (state, action) => ({
  ...state,
  isPolling: false,
  restartError: false,
  attemptRestart: false,
  restartSuccess: false,
  pollingSuccess: false,
  noSupport: true,
})

const polling = (state = defaultState, action) => {
  switch(action.type) {
    case actions.REPORT_POLLING:
      return reportPolling(state, action)
    case actions.REPORT_ATTEMPT_RESTART:
      return attemptRestart(state, action)
    case actions.REPORT_POLLING_SUCCESSFUL:
      return pollingSuccessful(state, action)
    case actions.REPORT_RESTART_SUCCESSFUL:
      return restartSuccessful(state, action)
    case actions.REPORT_RESTART_ERRORED:
      return restartError(state, action)
    case actions.REPORT_NO_SUPPORT:
      return noSupport(state, action)
    default:
      return state
  }
}

export default polling