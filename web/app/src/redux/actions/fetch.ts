import * as request from '../../utils/fetch'
import { DEVELOPMENT } from '../../utils/development'

/**
 * helper methods to detect when kited is and is not reachable
 *
 * Intended usage:
 *
 * fetch(url)
 *   .then(resolveBody)
 *   .then(
 *     reportNetworkSuccess(dispatch),
 *     catchNetworkError(dispatch)
 *   )
 *
 * Note: this returns the dispatched action, so
 * promise handlers should watch out for this action
 * parameters as a signal for a network error
 */
export const REPORT_NETWORK_FAIL = 'network failure'
export const catchNetworkError = (dispatch: any, params: any) => ({ error, middleware }: any) => {
  if (DEVELOPMENT) {
    console.error('network error', error)
  }
  if (DEVELOPMENT || params.reportNetworkFailure) {
    return dispatch({
      type: REPORT_NETWORK_FAIL,
      error,
      middleware
    })
  }
  return { type: REPORT_NETWORK_FAIL, error, middleware }
}


export const REPORT_NETWORK_CONNECTED = 'network connected'
export const reportNetworkSuccess = (dispatch: any) => (params: any) => {
  dispatch({
    type: REPORT_NETWORK_CONNECTED,
  })
  return params
}

export const interpretErrorSuccess = () => ({
  response,
  json,
  text,
  type,
  middleware,
  ...rest
}: any) => {
  let data, error
  const success = response && response.ok
  if (type !== REPORT_NETWORK_FAIL && success) {
    data = success && (json || text)
  } else {
    if (json) {
      error = json.message || json.fail_reason
      if (DEVELOPMENT) {
        console.error('interpretErrorSuccess json error: ', json, 'at', response.url)
      }
      data = json
    } else if (text) {
      error = text
      if (DEVELOPMENT) {
        console.error('interpretErrorSuccess text error: ', error, 'at', response.url)
      }
    }
  }
  return {
    success,
    data,
    error,
    response,
    middleware,
  }
}

/* ==={ METHODS }=== */
/*
 * The below methods simply wrap methods from
 * ../utils/fetch.js so that they can be dispatched and connected
 * to the redux store. In addition, these methods will
 * display a global notification when they fail.
 *
 */
const wrapMethod = (method: any) => (params: any) => (dispatch: any, getState: any) => {
  const { notifications } = getState()
  // switch behavior if we know that a request failed before
  let promise = null
  if (notifications.offline) {
    // if we think the user is currently offline,
    // report success if it succeeds
    promise = method(params)
      .then(
        reportNetworkSuccess(dispatch),
        () => ({ type: REPORT_NETWORK_FAIL }),
      )
      .then(interpretErrorSuccess())
  } else {
    // if we think the user is online,
    // report first failure
    promise = method(params)
      .catch(catchNetworkError(dispatch, params))
      .then(interpretErrorSuccess())
  }
  return promise
}

export const GET = wrapMethod(request.GET)
export const POST: any = wrapMethod(request.POST)
export const PATCH = wrapMethod(request.PATCH)
export const PUT = wrapMethod(request.PUT)
export const DELETE = wrapMethod(request.DELETE)

export const batchFetch = (queries: any) => (dispatch: any) => {
  return Promise.all(queries.map(dispatch)).then(results => {
    if (results.every((r: any) => r.success)) {
      return {
        success: true,
        data: results.map((r: any) => r.data),
        responses: results.map((r: any) => r.response)
      }
    } else {
      return {
        success: false,
        data: results.map((r: any) => r.data),
        responses: results.map((r: any) => r.response)
      }
    }
  })
}

export const batchFetchSafe = (queries: any) => (dispatch: any) => {
  return Promise.all(queries.map(dispatch)).then(results => {
    if (results.some((r: any) => r.success)) {
      return {
        success: true,
        data: results.map((r: any) => r.data),
        responses: results.map((r: any) => r.response)
      }
    } else {
      return {
        success: false,
        data: results.map((r: any) => r.data),
        responses: results.map((r: any) => r.response)
      }
    }
  })
}
