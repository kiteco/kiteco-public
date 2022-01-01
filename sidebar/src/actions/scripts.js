export const LOAD_SCRIPT = 'load script'
export const loadScript = name => dispatch => {
  return dispatch({
    type: LOAD_SCRIPT,
    name,
  })
}
