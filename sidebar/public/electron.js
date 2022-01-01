const path = require('path')
const { app, dialog } = require('electron')
const { isLocalhostDev } = require('./utils/electron-is-dev')
const { uncaughtExceptionHandler, unhandledRejectionHandler } = require('./utils/error-handling')
const { kiteProcess } = require('./utils/kite-process')
const { logger, logFilePath } = require('./utils/logger')
const { Domains } = require('./utils/domains')
const { openURL, openSidebar } = require('./start/sidebar')
const { openNotification } = require('./start/notification')


function handleArgs(args) {
  args = args.slice(1) // program name

  // Dev (non-packaged) builds on include a bunch of extra arguments.
  // Prod builds may get extra arguments from chromium for protocol-handling.
  // On Windows 10, this ends up looks something like:
  // <path to Kite.exe>,--allow-file-access-from-files,--original-process-start-time=<time>,kite://docs/python;;;;;json
  //
  // Furthermore, --key arguments will be split up from other (positional) arguments,
  // so we must always use `--key=value` syntax, and not `--key value` syntax.
  // See https://github.com/electron/electron/issues/20322
  //
  // So we simply search the argument list for arguments that we recognize and handle those.

  const notifArgKey = '--notification='
  const notifArgs = args.filter(a => a.startsWith(notifArgKey))
  if (notifArgs.length > 0) {
    const notifID = notifArgs[0].substr(notifArgKey.length)
    if (notifID) {
      openNotification(notifID)
      return
    }
  }


  var urls = args.filter(s => s.startsWith('kite://'))
  if (urls.length > 0) {
    openURL(urls[0])
    return
  }

  startKite()
}

async function startKite() {
  try {
    if (isLocalhostDev) {
      await openSidebar()
      return
    }

    if ((await kiteProcess.isKiteRunning()).running) {
      await openSidebar()
      return
    }
    await kiteProcess.startKite()
    await openSidebar()
  } catch (err) {
    logger.error("unhandled error ", err)
    var message = `Error trying to run Kite Copilot. We have to exit`
    var detail = `Please open an issue at https://github.com/kiteco/issue-tracker with the contents of ${logFilePath} for assistance.`
    if (err.type && err.type === 'tasklist_error') {
      // Display fix for spawn tasklist ENOENT errors on Windows.
      message = `Error trying to run Kite Copilot. Please ensure %SystemRoot%\system32 is in your PATH.`
      detail = `Please try following these instructions: https://${Domains.Help}/article/124-running-the-copilot-gives-me-a-path-error`
    }
    dialog.showMessageBox({
      type: 'error',
      buttons: ['OK'],
      title: 'Error',
      message,
      detail,
    }, () => {
      app.exit(1)
    })
  }
}


function main() {
  // Note: on Windows in a development env this always returns true
  const gotLock = app.requestSingleInstanceLock()
  if (!gotLock) {
    app.quit()
    return
  }

  if (app.getGPUFeatureStatus().gpu_compositing.includes("disabled")) {
    app.disableHardwareAcceleration()
  }
  if (isLocalhostDev) {
    const { default: installExtension, REACT_DEVELOPER_TOOLS, REDUX_DEVTOOLS } = require('electron-devtools-installer')
    installExtension(REACT_DEVELOPER_TOOLS)
      .then((name) => console.log(`Added Extension:  ${name}`))
      .catch((err) => console.log('An error occurred: ', err))

    installExtension(REDUX_DEVTOOLS)
      .then((name) => console.log(`Added Extension:  ${name}`))
      .catch((err) => console.log('An error occurred: ', err))
  }

  process.on('uncaughtException', uncaughtExceptionHandler)
  process.on('unhandledRejection', unhandledRejectionHandler)

  app.whenReady().then(() => handleArgs(process.argv))
  app.on('activate', () => app.whenReady().then(startKite))
  app.on('second-instance', (e, argv, workingDirectory) => {
    app.whenReady().then(() => handleArgs(argv))
  })

  if (process.env.REACT_APP_ENV === 'development' && process.platform === 'win32') {
    // We are running a non-packaged version of the app on windows:
    // SO link: /questions/45570589/electron-protocol-handler-not-working-on-windows
    app.setAsDefaultProtocolClient('kite', process.execPath, [path.resolve(process.argv[1])])
  } else {
    app.setAsDefaultProtocolClient('kite')
  }
  // macOS protocol handler
  app.on('open-url', function(e, url) {
    e.preventDefault()
    app.whenReady().then(() => openURL(url))
  })

  app.on('window-all-closed', () => {
    app.quit()
  })

  app.on('will-quit', () => {
    process.removeListener('unhandledRejection', unhandledRejectionHandler)
    process.removeListener('uncaughtException', uncaughtExceptionHandler)
  })
}


main()
