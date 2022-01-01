import { localhostProxy } from '../utils/urls'

const { ipcRenderer, shell } = window.require("electron")

export const handleSetupTransition = ({
  getSetupCompleted,
  setSetupNotCompleted,
  getHaveShownWelcome,
  setHaveShownWelcome,
  forceCheckOnline,
  metricsId
}) => {
  getSetupCompleted().then(resp => {
    if (resp && (!resp.data || resp.data === 'false' || resp.data === 'notset' )) {
      setSetupNotCompleted()
    } else if (resp && resp.data) {
      forceCheckOnline().then(({ success, isOnline }) => {
        if(success && isOnline) {
          getHaveShownWelcome().then(haveShown => {
            if(!haveShown) {
              shell.openExternal(localhostProxy(`/clientapi/desktoplogin?d=/welcome${metricsId ? `?id=${metricsId}` : ''}`))
              setHaveShownWelcome()
            }
          })
        }
      })
    }
  })
}

export const setElectronWindowSettings = ({ getWindowMode, getProxyMode, getProxyURL }) => {
  getWindowMode().then(({success, data}) => {
    if (success) {
      ipcRenderer.send('set-window-mode', data)
    }
  })

  getProxyMode().then(({success, data}) => {
    if(success) {
      ipcRenderer.send('set-proxy-mode', data)
    }
  })

  getProxyURL().then(({success, data}) => {
    if(success) {
      ipcRenderer.send('set-proxy-url', data)
    }
  })
}
