/**
 * show simply shows an error through the
 * Error container found in `src/containers/Error`
 *
 * timeout:   optional parameter; if set, will hide the error
 *            after a specified number of milliseconds
 */
export const SHOW = "show error"
export const HIDE = "hide error"
export const show = timeout => (dispatch, getState) => {
  const { id } = getState().errors
  const newId = id + 1
  dispatch({ type: SHOW, id: newId })
  if (timeout) {
    const p = new Promise(resolve => { setTimeout(resolve, timeout) })
    return p.then(() => {
      const { id } = getState().errors
      if (newId === id) {
        dispatch({ type: HIDE })
      }
    })
  }
}

/**
 * This wraps an action so that the global error
 * will be shown and then later hidden
 */
export const showError = error => dispatch => {
  const result = dispatch(error)
  dispatch(show(5000))
  return result
}
