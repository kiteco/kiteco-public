export const HIDE_NOTIFICATION = 'hide notification'
export const hideNotification = id => ({
  type: HIDE_NOTIFICATION,
  id,
})

/**
 * Record an notification to be shown as a toast to the user
 */
export const REPORT_NOTIFICATION = 'report notification'
export const notify = ({
  message,
  timeout,
  kind,
  error,
}) => dispatch => dispatch({
  type: REPORT_NOTIFICATION,
  message,
  error,
  kind,
  timeout,
})
