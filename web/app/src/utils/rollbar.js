const handleErrorBoundaryCatch = (err) => {
  const payload = {
    type: 'webapp-error-boundary-catch',
    backend: process.env.REACT_APP_BACKEND,
    ...err
  }
  window.Rollbar.error(payload)
}

const handleException = (err) => {
  const payload = {
    type: 'webapp-unhandled-exception',
    backend: process.env.REACT_APP_BACKEND,
    ...err
  }
  window.Rollbar.error(payload)
}

export default {
  handleErrorBoundaryCatch,
  handleException
}