import {
  GET,
  POST,
  DELETE,
} from './fetch'

import {
  emailRequiredPath,
  iconVisibilityPath,
  setupCompletionPath,
  serverPath,
  autosearchDefaultPath,
  windowModePath,
  autoInstallPluginsPath,
  themeDefaultPath,
  hasSeenKiteLocalPath,
  setupStagePath,
  completionsDisabledPath,
  metricsDisabledPath,
  haveShownWelcomePath,
  proxyModePath,
  proxyURLPath,
  kiteServerURLPath,
  kiteServerStatusPath,
  autostartDisabledPath,
  showCompletionsCTAPath,
  rcDisabledCompletionsCTAPath,
  showChooseEnginePath,
  selectedEnginePath,
  maxFileSizePath,
} from '../utils/urls'
const { ipcRenderer } = window.require("electron")

export const GET_COMPLETIONS_DISABLED = 'get completions disabled'
export const getCompletionsDisabled = () => dispatch =>
  dispatch(GET({ url: completionsDisabledPath() }))
    .then(({ success, data }) => {
      if (success) {
        return dispatch({
          type: GET_COMPLETIONS_DISABLED,
          success,
          data: data === 'true',
        })
      } else {
        dispatch({
          type: GET_COMPLETIONS_DISABLED,
          success,
          data: false,
        })
      }
    })

export const SET_COMPLETIONS_DISABLED = 'set completions disabled'
export const setCompletionsDisabled = disabled => dispatch =>
  dispatch(POST({
    url: completionsDisabledPath(),
    options: {
      body: disabled,
    },
  }))
    .then(({ success }) => success && dispatch(getCompletionsDisabled()))

export const GET_METRICS_DISABLED = 'get metrics disabled'
export const getMetricsDisabled = () => dispatch =>
  dispatch(GET({ url: metricsDisabledPath() }))
    .then(({ success, data }) => {
      if (success) {
        return dispatch({
          type: GET_METRICS_DISABLED,
          success,
          data: data === 'true',
        })
      } else {
        dispatch({
          type: GET_METRICS_DISABLED,
          success,
          data: false,
        })
      }
    })

export const SET_METRICS_DISABLED = 'set metrics disabled'
export const setMetricsDisabled = disabled => dispatch =>
  dispatch(POST({
    url: metricsDisabledPath(),
    options: {
      body: disabled,
    },
  }))
    .then(({ success }) => success && dispatch(getMetricsDisabled()))

export const getLastSetupStage = () => dispatch =>
  dispatch(GET({ url: setupStagePath() }))
    .then(({ success, data }) => {
      if (success) {
        return data
      } else {
        return ''
      }
    })

export const setCurrentSetupStage = stage => dispatch =>
  dispatch(POST({
    url: setupStagePath(),
    options: {
      body: stage,
    },
  }))

export const GET_DEFAULT_THEME = 'get default theme'
export const getDefaultTheme = () => dispatch =>
  dispatch(GET({ url: themeDefaultPath() }))
    .then(({ success, data, response }) => {
      if (success) {
        return dispatch({
          type: GET_DEFAULT_THEME,
          success,
          data: typeof data === 'string' ? data : 'dark',
        })
      } else if (response) {
        return dispatch({
          type: GET_DEFAULT_THEME,
          success,
          data: 'dark',
        })
      }
    })

export const SET_DEFAULT_THEME = 'set default theme'
export const setDefaultTheme = (theme) => dispatch => () => {
  return dispatch(POST({
    url: themeDefaultPath(),
    options: {
      body: theme,
    },
  }))
    .then(({ success }) => {
      success && dispatch(getDefaultTheme())
    })
}

export const GET_AUTOSEARCH_DEFAULT = "get autosearch default"
export const getAutosearchDefault = () => dispatch =>
  dispatch(GET({ url: autosearchDefaultPath() }))
    .then(({ success, data, response }) => {
      if (success) {
        return dispatch({
          type: GET_AUTOSEARCH_DEFAULT,
          success,
          data: data === "true",
        })
      } else if (response) {
        return dispatch({
          type: GET_AUTOSEARCH_DEFAULT,
          success,
          data: false,
        })
      }
    }
    )

export const setAutosearchDefaultOn = () => dispatch =>
  dispatch(POST({
    url: autosearchDefaultPath(),
    options: {
      body: "true",
    },
  }))
    .then(({ success }) =>
      success &&
      dispatch(getAutosearchDefault())
    )

export const setAutosearchDefaultOff = () => dispatch =>
  dispatch(POST({
    url: autosearchDefaultPath(),
    options: {
      body: "false",
    },
  }))
    .then(({ success }) =>
      success &&
      dispatch(getAutosearchDefault())
    )

export const GET_WINDOW_MODE = "get window mode"
export const getWindowMode = () => dispatch => {
  return dispatch(GET({ url: windowModePath() })).then(({ success, data }) => {
    return success && dispatch({
      type: GET_WINDOW_MODE,
      success,
      data,
    })
  })
}

export const setWindowMode = mode => dispatch => {
  return dispatch(POST({
    url: windowModePath(),
    options: {
      body: mode,
    },
  }))
    .then(({ success }) => {
      if (success) {
        ipcRenderer.send('set-window-mode', mode)
        return dispatch(getWindowMode())
      }
    })
}

// This is necessary for setting the window mode in the `SettingsRadioButton`
// component.
export const setWindowModeWrapped = mode => dispatch => () => {
  return setWindowMode(mode)(dispatch)
}

export const GET_ICON_VISIBLE = "get icon visible"
export const getIconVisible = () => dispatch =>
  dispatch(GET({ url: iconVisibilityPath() }))
    .then(({ success, data }) =>
      success &&
      dispatch({
        type: GET_ICON_VISIBLE,
        success,
        data: data === "true",
      })
    )

export const setIconVisible = () => dispatch =>
  dispatch(POST({
    url: iconVisibilityPath(),
    options: {
      body: "true",
    },
  }))
    .then(({ success }) =>
      success &&
      dispatch(getIconVisible())
    )

export const setIconInvisible = () => dispatch =>
  dispatch(POST({
    url: iconVisibilityPath(),
    options: {
      body: "false",
    },
  }))
    .then(({ success }) =>
      success &&
      dispatch(getIconVisible())
    )


export const GET_SETUP_COMPLETED = "get setup completed"
export const getSetupCompleted = () => dispatch =>
  dispatch(GET({ url: setupCompletionPath() }))
    .then(({ success, data, response }) => {
      if (success) {
        return dispatch({
          type: GET_SETUP_COMPLETED,
          success,
          data: data === "true",
        })
      } else if (response) {
        return dispatch({
          type: GET_SETUP_COMPLETED,
          success,
          data: "notset",
        })
      }

    })

export const setSetupCompleted = () => dispatch =>
  dispatch(POST({
    url: setupCompletionPath(),
    options: {
      body: "true",
    },
  }))
    .then(({ success }) =>
      success &&
      dispatch(getSetupCompleted())
    )

export const setSetupNotCompleted = () => dispatch =>
  dispatch(POST({
    url: setupCompletionPath(),
    options: {
      body: "false",
    },
  }))
    .then(({ success }) =>
      success &&
      dispatch(getSetupCompleted())
    )

export const deleteSetupCompleted = () => dispatch =>
  dispatch(DELETE({
    url: setupCompletionPath(),
  }))
    .then(({ success }) =>
      success &&
      dispatch(getSetupCompleted())
    )


export const GET_AUTOINSTALL_PLUGINS_ENALBED = "get autoinstall plugins enabled"
export const getAutoInstallPluginsEnabled = () => dispatch =>
  dispatch(GET({ url: autoInstallPluginsPath() }))
    .then(({ success, data }) =>
      success &&
      dispatch({
        type: GET_AUTOINSTALL_PLUGINS_ENALBED,
        success,
        data: data === "true",
      })
    )

export const setAutoInstallPluginsEnabled = (enabled) => dispatch =>
  dispatch(POST({
    url: autoInstallPluginsPath(),
    options: {
      body: enabled ? "true" : "false",
    },
  }))
    .then(({ success }) =>
      success &&
      dispatch(getAutoInstallPluginsEnabled())
    )

export const GET_SERVER = "get server"
export const getServer = () => dispatch =>
  dispatch(GET({ url: serverPath() }))
    .then(({ success, data }) =>
      success &&
      dispatch({
        type: GET_SERVER,
        success,
        data,
      })
    )

export const setServer = server => dispatch =>
  dispatch(POST({
    url: serverPath(),
    options: {
      body: server,
    },
  }))
    .then(({ success, error }) =>
      success
        ? dispatch(getServer())
        : { success, error }
    )

export const getHasSeenKiteLocalNotification = () => dispatch =>
  dispatch(GET({ url: hasSeenKiteLocalPath() }))
    .then(({ success, data }) => {
      if (success) {
        return data === 'true'
      }
      return false
    })

export const setHasSeenKiteLocalNotification = hasSeen => dispatch =>
  dispatch(POST({
    url: hasSeenKiteLocalPath(),
    options: {
      body: hasSeen,
    },
  }))

export const getHaveShownWelcome = () => dispatch =>
  dispatch(GET({ url: haveShownWelcomePath() }))
    .then(({ success, data }) => success && data === 'true')

export const setHaveShownWelcome = () => dispatch =>
  dispatch(POST({
    url: haveShownWelcomePath(),
    options: {
      body: 'true',
    },
  }))

export const GET_PROXY_MODE = 'get proxy mode'
export const getProxyMode = () => dispatch =>
  dispatch(GET({ url: proxyModePath() }))
    .then(({ success, data, response }) => {
      if (success) {
        return dispatch({
          type: GET_PROXY_MODE,
          success,
          data: typeof data === 'string' ? data : 'environment',
        })
      } else if (response) {
        return dispatch({
          type: GET_PROXY_MODE,
          success,
          data: 'environment',
        })
      }
    })

export const setProxyMode = (mode) => dispatch => () => {
  return dispatch(POST({
    url: proxyModePath(),
    options: {
      body: mode,
    },
  })).then(({ success }) => {
    if (success) {
      ipcRenderer.send('set-proxy-mode', mode)
      dispatch(getProxyMode())
    }
  })
}

export const GET_PROXY_URL = 'get proxy url'
export const getProxyURL = () => dispatch =>
  dispatch(GET({ url: proxyURLPath() }))
    .then(({ success, data, response }) => {
      if (success) {
        return dispatch({
          type: GET_PROXY_URL,
          success,
          data: typeof data === 'string' ? data : "",
        })
      } else if (response) {
        return dispatch({
          type: GET_PROXY_URL,
          success,
          data: "",
        })
      }
    })

export const setProxyURL = (url) => dispatch => {
  return dispatch(POST({
    url: proxyURLPath(),
    options: {
      body: url,
    },
  })).then(({ success }) => {
    if (success) {
      ipcRenderer.send('set-proxy-url', url)
      return dispatch(getProxyURL())
    }
  })
}

export const GET_KITE_SERVER_URL = 'get Kite server url'
export const getKiteServerURL = () => dispatch => (
  dispatch(GET({ url: kiteServerURLPath() }))
    .then(({ success, data, response }) => {
      if (success) {
        return dispatch({
          type: GET_KITE_SERVER_URL,
          success,
          data: typeof data === 'string' ? data : '',
        })
      } else if (response) {
        return dispatch({
          type: GET_KITE_SERVER_URL,
          success,
          data: '',
        })
      }
    })
)

export const setKiteServerURL = url => dispatch => {
  return dispatch(POST({
    url: kiteServerURLPath(),
    options: {
      body: url,
    },
  })).then(({ success, error }) => {
    if (success) {
      return dispatch(getKiteServerURL())
    }
    throw error
  })
}

export const GET_KITE_SERVER_STATUS = 'get Kite server status'
export const getKiteServerStatus = () => dispatch =>
  dispatch(GET({ url: kiteServerStatusPath() }))
    .then(({ success, data }) => {
      if (success) {
        return dispatch({
          type: GET_KITE_SERVER_STATUS,
          success,
          data,
        })
      }
      return dispatch({
        type: GET_KITE_SERVER_STATUS,
        success,
        data: { available: false, ping: 0 },
      })
    })

export const GET_MAX_FILE_SIZE = 'get max file size'
export const getMaxFileSize = () => dispatch =>
  dispatch(GET({ url: maxFileSizePath() })).then(({ success, data }) => {
    if (success) {
      return dispatch({
        type: GET_MAX_FILE_SIZE,
        success,
        data,
      })
    } else {
      dispatch({
        type: GET_MAX_FILE_SIZE,
        success,
        data: "",
      })
    }
  })

export const SET_MAX_FILE_SIZE = 'set max file size'
export const setMaxFileSize = size => dispatch =>
  dispatch(POST({
    url: maxFileSizePath(),
    options: {
      body: size,
    },
  })).then(({ success }) => success && dispatch(getMaxFileSize()))

export const GET_SHOW_COMPLETIONS_CTA = 'get show completions cta'
export const getShowCompletionsCTA = () => async dispatch => {
  const { success, data } = await dispatch(GET({ url: showCompletionsCTAPath() }))
  return dispatch(
    {
      type: GET_SHOW_COMPLETIONS_CTA,
      success,
      data: success ? data === "true" : true,
    }
  )
}

export const SET_SHOW_COMPLETIONS_CTA = 'set show completions cta'
export const setShowCompletionsCTA = enabled => async dispatch => {
  const { success } = await dispatch(
    POST({
      url: showCompletionsCTAPath(),
      options: {
        body: enabled ? "true" : "false",
      },
    })
  )
  return success && dispatch(getShowCompletionsCTA())
}

export const GET_RC_DISABLED_COMPLETIONS_CTA = 'get rc disabled completions cta'
export const getRCDisabledCompletionsCTA = () => async dispatch => {
  const req = GET({ url: rcDisabledCompletionsCTAPath() })
  const { success, data } = await dispatch(req)
  return dispatch({
    type: GET_RC_DISABLED_COMPLETIONS_CTA,
    success,
    data: success ? data === 'true' : false,
  })
}

export const GET_AUTOSTART_ENABLED = 'get autostart enabled'
export const getAutostartEnabled = () => dispatch =>
  dispatch(GET({ url: autostartDisabledPath() }))
    .then(({ success, data }) => {
      if (success) {
        return dispatch({
          type: GET_AUTOSTART_ENABLED,
          success,
          data: data === 'false',
        })
      } else {
        dispatch({
          type: GET_AUTOSTART_ENABLED,
          success,
          data: true,
        })
      }
    })

export const SET_AUTOSTART_ENABLED = 'set autostart enabled'
export const setAutostartEnabled = enabled => dispatch =>
  dispatch(POST({
    url: autostartDisabledPath(),
    options: {
      body: enabled ? "false" : "true",
    },
  }))
    .then(({ success }) => success && dispatch(getAutostartEnabled()))

export const getShowChooseEngine = () => dispatch =>
  dispatch(GET({ url: showChooseEnginePath() }))
    .then(({ success, data }) => success && data === "true")

export const setSelectedEngine = engineID => async dispatch => {
  const { success } = await dispatch(
    POST({
      url: selectedEnginePath(),
      options: {
        body: engineID,
      },
    }))
  return success
}

export const getEmailRequired = () => async (dispatch) => {
  const { success, data } = await dispatch(GET({ url: emailRequiredPath() }))
  if (!success || !data['required']) {
    return false
  }
  return data['required']
}
