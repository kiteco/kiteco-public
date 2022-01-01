'use strict'
const { dialog, app } = require("electron")
const { sendRollbarError } = require("./rollbar")
const {logger, logFilePath} = require('./logger')

const uncaughtExceptionHandler = function(e) {
  const payload = {
    type: 'electron-main-error',
    name: e.name,
    message: e.message,
    stack: e.stack
  }
  logger.error('electron-main-error ', e)
  sendRollbarError(payload)
  logger.end()
  //asynchronous error dialog, then exit app - seems a little heavy handed, but
  //want to play it safe with uncaught exceptions
  logger.on('finish', info => {
    if(app.isReady()) {
      dialog.showMessageBox({
        type: 'error',
        buttons: ['OK'],
        title: 'Error',
        message: `Error trying to run Kite Copilot. We have to exit.\nOpen an issue at https://github.com/kiteco/issue-tracker with the contents of ${logFilePath} for assistance`,
        detail: `Error details below:\n${e.name}: ${e.message}`
      }, () => {
        app.exit(1)
      })
    } else {
      // showMessageBox is only available if app.isReady()
      dialog.showErrorBox(
        'Error trying to run Kite Copilot. We have to exit.\nOpen an issue at https://github.com/kiteco/issue-tracker with the contents of ${logFilePath} for assistance',
        `Error details below:\n${e.name}: ${e.message}`
        )
        app.exit(1)
    }
  })
}

const unhandledRejectionHandler = function(reason, p) {
  const payload = {
    type: 'electron-main-unhandled-rejection',
    reason: reason
  }
  logger.error('electron-main-rejection ', reason)
  sendRollbarError(payload)
}

const handleRendererException = function(e, errorEvent) {
  const payload = {
    type: 'react-global-error',
    message: errorEvent.message,
    filename: errorEvent.filename,
    lineno: errorEvent.lineno,
    colno: errorEvent.colno,
    stack: errorEvent.stack
  }
  sendRollbarError(payload)
}

const handleRendererRejection = function(e, rejectionEvent) {
  const payload = {
    type: 'react-unhandled-rejection',
    reason: rejectionEvent.reason
  }
  sendRollbarError(payload)
}

const handleErrorBoundaryCatch = function(e, err) {
  const payload = {
    type: 'react-error-boundary-catch',
    name: err.name,
    message: err.message,
    info: err.info
  }
  sendRollbarError(payload)
}

module.exports = {
  uncaughtExceptionHandler,
  unhandledRejectionHandler,
  handleRendererException,
  handleRendererRejection,
  handleErrorBoundaryCatch,
}
