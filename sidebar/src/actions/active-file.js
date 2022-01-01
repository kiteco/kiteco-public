export const ACTIVE_FILE_EVENT = "active-file event"
export const activeFileEvent = ({ filename }) => dispatch => {
  return dispatch({
    type: ACTIVE_FILE_EVENT,
    data: filename,
  })
}
