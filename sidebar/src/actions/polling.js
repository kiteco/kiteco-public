/**
 * For seeing what kited's status is
 */
import { kitedStatusPath } from '../utils/urls'
import * as request from '../utils/fetch'
import { catchKitedNotReachable, catchKitedNotHealthy } from './fetch'

/**
 * Duplicated so as to not dispatch to reportKitedReachable action
 * as in fetch.js
 */
const wrapMethod = method => params => dispatch =>
  method(params)
    .then(response => response)
    .catch(error => {
      dispatch(catchKitedNotReachable(error))
      dispatch(reportPolling(false))
      throw error
    })
    .then(res => {
      //check below could be expanded eventually to detect different
      //non-healthy responses when needed level of granularity demands it
      if(!request.isOK(res)) {
        return dispatch(catchKitedNotHealthy(res))
      }
      return res
    })

const GET = wrapMethod(request.GET)

export const getKitedStatus = () => dispatch => {
  return dispatch(GET({ url: kitedStatusPath() }))
    .then((response) => {
      return response
    })
}

export const REPORT_POLLING = 'report polling'
export const reportPolling = (isPolling) => dispatch => {
  return dispatch({
    type: REPORT_POLLING,
    isPolling,
  })
}

//Returning promises below helps with timing UX messages and state transitions related
//to polling and restarting
export const REPORT_ATTEMPT_RESTART = 'report attempt restart'
export const reportAttemptRestart = () => dispatch => {
  return new Promise((resolve) => {
    dispatch({
      type: REPORT_ATTEMPT_RESTART,
    })
    resolve()
  })
}

export const REPORT_RESTART_SUCCESSFUL = 'report restart successful'
export const reportRestartSuccessful = () => dispatch => {
  return new Promise((resolve) => {
    dispatch({
      type: REPORT_RESTART_SUCCESSFUL,
    })
    resolve()
  })
}

export const REPORT_RESTART_ERRORED = 'report restart errored'
export const reportRestartErrored = () => dispatch => {
  return dispatch({
    type: REPORT_RESTART_ERRORED,
  })
}

export const REPORT_NO_SUPPORT = 'report no support'
export const reportNoSupport = () => dispatch => {
  return dispatch({
    type: REPORT_NO_SUPPORT,
  })
}

export const REPORT_POLLING_SUCCESSFUL = 'report polling successful'
export const reportPollingSuccessful = () => dispatch => {
  return new Promise((resolve) => {
    dispatch({
      type: REPORT_POLLING_SUCCESSFUL,
    })
    resolve()
  })
}