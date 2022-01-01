export const addErrorHandling = (store, ipcRenderer) => {
  window.addEventListener('error', function(errorEvent) {
    //Note: this will catch a superset of the errors caught by the ErrorBoundary - so there will be
    // a bit of duplication in terms of reporting (ErrorBoundary catching is useful because of provided Component stack traces)
    const payload = {
      message: errorEvent.message,
      colno: errorEvent.colno,
      lineno: errorEvent.lineno,
      filename: errorEvent.filename,
      stack: errorEvent.error.stack
    }
    ipcRenderer.send('renderer-exception', payload)
    //In the case that we get an error that did not originate from the ErrorBoundary
    //we want to make sure that the same ErrorOverlay gets displayed
    store.dispatch({
      type: 'APP_EXCEPTION'
    })
  })
}