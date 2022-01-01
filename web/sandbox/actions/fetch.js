import * as request from '../utils/fetch'

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
export const REPORT_NETWORK_FAIL= 'network failure'
export const catchNetworkError = (dispatch, params) => ({ error, middleware }) => {
  if(params.reportNetworkFailure) {
    return dispatch({
      type: REPORT_NETWORK_FAIL,
      error,
      middleware
    })
  }
  return { type: REPORT_NETWORK_FAIL, error, middleware }
}


export const REPORT_NETWORK_CONNECTED= 'network connected'
export const reportNetworkSuccess = dispatch => params => {
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
  ...rest,
}) => {
  let data, error
  const success = response && response.ok
  if (type !== REPORT_NETWORK_FAIL && success) {
    data = success && (json || text)
  } else {
    if (json) {
      error = json.message || json.fail_reason
      data = json
    } else if (text) {
      error = text
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
const wrapMethod = method => params => dispatch => {
  // report first failure
  return method(params)
    .catch(catchNetworkError(dispatch, params))
    .then(interpretErrorSuccess())
}

export const GET = wrapMethod(request.GET)
export const POST = wrapMethod(request.POST)
export const PATCH = wrapMethod(request.PATCH)
export const PUT = wrapMethod(request.PUT)
export const DELETE = wrapMethod(request.DELETE)

export const batchFetch = queries => dispatch => {
  return Promise.all(queries.map(dispatch)).then(results => {
    if (results.every(r => r.success)) {
      return {
        success: true,
        data: results.map(r => r.data),
        responses: results.map(r => r.response)
      }
    } else {
      return {
        success: false,
        data: results.map(r => r.data),
        responses: results.map(r => r.response)
      }
    }
  })
}

export const batchFetchSafe = queries => dispatch => {
  return Promise.all(queries.map(dispatch)).then(results => {
    if (results.some(r => r.success)) {
      return {
        success: true,
        data: results.map(r => r.data),
        responses: results.map(r => r.response)
      }
    } else {
      return {
        success: false,
        data: results.map(r => r.data),
        responses: results.map(r => r.response)
      }
    }
  })
}
