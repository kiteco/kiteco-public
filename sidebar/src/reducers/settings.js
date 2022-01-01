import * as actions from '../actions/settings'
import { setCanUse } from '../utils/analytics'
import { WindowMode, validateWindowMode } from '../utils/settings'
import { Domains } from '../utils/domains'

const defaultState = {
  iconVisible: true,
  autosearchDefault: null,
  themeDefault: 'dark',
  setupCompleted: "notset",
  server: "",
  webapp: "",
  windowMode: WindowMode.FOCUS_ON_DOCS,
  pluginsAutoInstall: false,
  completionsDisabled: false,
  metricsDisabled: false,
  proxyMode: 'environment',
  proxyURL: '',
  enterpriseServerURL: '',
  kiteServerStatus: false,
  showCompletionsCTA: true,
  rcDisabledCompletionsCTA: false,
  maxFileSize: 1024,
}

const updateServer = (state, action) => {
  const server = action.data.replace(/\/+$/, "")
  let webapp = server
  switch (webapp) {
    case `https://${Domains.Alpha}`:
      webapp = `https://${Domains.PrimaryHost}`
      break
    case `https://${Domains.Staging}`:
      webapp = `https://${Domains.GaStaging}`
      break
    default:
  }
  return {
    ...state,
    server,
    webapp,
  }
}

const updateMetricsDisabled = (state, action) => {
  /**
   * NB: the code below is somewhat of an anti-pattern, but it provides a
   * decent way to keep the gating of sending analytics to Segment in sync with
   * our notion of metricsDisabled
   */
  setCanUse(!action.data)
  return {
    ...state,
    metricsDisabled: action.data,
  }
}

const settings = (state = defaultState, action) => {
  switch (action.type) {
    case actions.GET_ICON_VISIBLE:
      return { ...state, iconVisible: action.data }
    case actions.GET_AUTOSEARCH_DEFAULT:
      return { ...state, autosearchDefault: action.data }
    case actions.GET_SERVER:
      return updateServer(state, action)
    case actions.GET_SETUP_COMPLETED:
      return { ...state, setupCompleted: action.data }
    case actions.GET_DEFAULT_THEME:
      return { ...state, themeDefault: action.data ? action.data : 'dark' }
    case actions.GET_WINDOW_MODE:
      return { ...state, windowMode: validateWindowMode(action.data) }
    case actions.GET_COMPLETIONS_DISABLED:
      return { ...state, completionsDisabled: action.data }
    case actions.GET_METRICS_DISABLED:
      return updateMetricsDisabled(state, action)
    case actions.GET_AUTOINSTALL_PLUGINS_ENALBED:
      return { ...state, pluginsAutoInstall: action.data }
    case actions.GET_PROXY_MODE:
      return { ...state, proxyMode: action.data }
    case actions.GET_PROXY_URL:
      return { ...state, proxyMode: action.data }
    case actions.GET_KITE_SERVER_URL:
      return { ...state, enterpriseServerURL: action.data }
    case actions.GET_KITE_SERVER_STATUS:
      return { ...state, kiteServerAvailable: action.data.available }
    case actions.GET_AUTOSTART_ENABLED:
      return { ...state, autostartEnabled: action.data }
    case actions.GET_SHOW_COMPLETIONS_CTA:
      return { ...state, showCompletionsCTA: action.data }
    case actions.GET_RC_DISABLED_COMPLETIONS_CTA:
      return { ...state, rcDisabledCompletionsCTA: action.data }
    case actions.GET_MAX_FILE_SIZE:
      return { ...state, maxFileSize: action.data }
    default:
      return state
  }
}

export default settings
