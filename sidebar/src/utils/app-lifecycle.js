const { ipcRenderer } = window.require("electron")

export const refreshSidebar = () => {
  window.location.reload()
}

const REFRESH_DELAY = 3000
const POLL_DELAY = 3000

export const reloadSidebar = (delay = REFRESH_DELAY) => {
  setTimeout(() => {
    ipcRenderer.send('restart-kite')
  }, delay)
}

//Roughly, if the sidebar has connection-to-kited-issues AND
//is not in the process of trying to fix it
export const kiteNotWorking = (errors, polling) => {
  return errors && !errors.responsive && !polling.isPolling &&
    !polling.pollingSuccess && !polling.restartSuccess &&
    !polling.restartError && !polling.attemptRestart && !polling.noSupport
}

export const addElectronAppEventListeners = ({
  reportRestartSuccessful,
  reportRestartErrored,
  reportNoSupport,
  push,
  addRoute,
  notify,
  forceCheckOnline,
}) => {
  ipcRenderer.removeAllListeners('restart-kite-success')
  ipcRenderer.on('restart-kite-success', (event, arg) => {
    reportRestartSuccessful().then(() => {
      setTimeout(() => {
        refreshSidebar()
      }, REFRESH_DELAY)
    })
  })

  ipcRenderer.removeAllListeners('no-restart-support')
  ipcRenderer.on('no-restart-support', (event, arg) => {
    reportNoSupport()
  })

  ipcRenderer.removeAllListeners('restart-kite-error')
  ipcRenderer.on('restart-kite-error', (event, arg) => {
    reportRestartErrored()
  })

  ipcRenderer.removeAllListeners('transitionTo')
  ipcRenderer.on('transitionTo', (evt, route) => {
    // do kite://feedback handling here
    if (route && route.includes('kite://')) {
      const matches = /feedback/.exec(route)
      if (matches) {
        switch (matches[0]) {
          case 'feedback':
            forceCheckOnline().then(({ success, isOnline }) => {
              if (success && isOnline) {
                addRoute(route)
                let newRoute = route.replace('kite:/', '')
                push(newRoute)
              } else {
                notify({
                  id: 'offline',
                  component: 'offline',
                  payload: {
                    copy: 'Thanks for wanting to provide feedback!'
                  }
                })
              }
            })
            push('/home')
            break
          default:
            break
        }
      } else {
        addRoute(route)
        let newRoute = route.replace('kite:/', '')
        push(newRoute)
      }
    }
  })
}

export const handleDisconnectedCase = ({
  reportPolling,
  getKitedStatus,
  reportPollingSuccessful,
}) => {
  reportPolling(true)
  setTimeout(() => {
    getKitedStatus().then(res => {
      if (res && res.response && res.response.status === 200) {
        reportPollingSuccessful().then(() => {
          setTimeout(() => {
            refreshSidebar()
          }, REFRESH_DELAY)
        })
      } else {
        reportPolling(false)
      }
    })
  }, POLL_DELAY)
}

export const handleUnresponsiveCase = ({
  reportPolling,
  getKitedStatus,
  reportPollingSuccessful,
}) => {
  reportPolling(true)
  setTimeout(() => {
    getKitedStatus().then(res => {
      if (res && res.response && res.response.status === 200) {
        reportPollingSuccessful().then(() => {
          setTimeout(() => {
            refreshSidebar()
          }, REFRESH_DELAY)
        })
      } else {
        reportPolling(false)
      }
    })
  }, POLL_DELAY)
}
