export const parseRoute = (route) => {
  const url = new URL(route)
  const params = {}
  for (let param of url.searchParams.entries()) {
    params[param[0]] = param[1]
  }
  return {
    path: url.pathname.substr(1), //to take out extra preceding '/'
    params,
  }
}

export const localhostProxy = (suffix) => {
  return `http://localhost:46624${suffix}`
}

export const kitedReadyPath = () => {
  return '/clientapi/kited_online'
}

export const checkOnlinePath = () => {
  return '/clientapi/checkonline'
}

export const onlinePath = () => {
  return '/clientapi/online'
}

export const membersPath = (identifier, page, limit) => {
  return [
    `/api/editor/value/${identifier}/members`,
    [
      `offset=${page}`,
      `limit=${limit}`,
    ].join('&'),
  ].join('?')
}

export const symbolReportPath = (identifier) => {
  return `/api/editor/symbol/${identifier}`
}

export const userAccountPath = () => {
  return '/clientapi/user'
}

export const defaultEmailPath = () => {
  return '/clientapi/default-email'
}

export const loginPath = () => {
  return '/clientapi/login'
}

export const logoutPath = () => {
  return '/clientapi/logout'
}

export const createAccountPath = () => {
  return '/clientapi/create-account'
}

export const createPasswordlessAccountPath = () => {
  return '/clientapi/create-passwordless'
}

export const passwordResetPath = () => {
  return '/api/account/reset-password/request'
}

export const curationExamplesPath = (language, id) => {
  return `/api/${language}/curation/${id}`
}

const DEFAULT_OFFSET = 0
const DEFAULT_LIMIT = 6
export const searchQueryCompletionPath = ({ query, offset = DEFAULT_OFFSET, limit = DEFAULT_LIMIT }) => {
  return `/api/editor/search?q=${query}&offset=${offset}&limit=${limit}`
}

export const kitedStatusPath = () => {
  return '/clientapi/health'
}

export const userNodeHealthPath = () => {
  return '/ping'
}

export const licenseInfoPath = () => {
  return '/clientapi/license-info'
}

export const pluginsPath = (id) => {
  if (!id) return '/clientapi/plugins'
  return `/clientapi/plugins/${id}`
}

export const autoInstalledPluginsPath = () => {
  return '/clientapi/plugins/auto_installed'
}

export const encounteredEditorsPath = () => {
  return '/clientapi/plugins/encountered'
}

export const mostRecentEditor = () => {
  return '/clientapi/plugins/most_recent'
}

export const iconVisibilityPath = () => {
  return '/clientapi/settings/show_status_icon'
}

export const setupCompletionPath = () => {
  return '/clientapi/settings/setup_completed'
}

export const serverPath = () => {
  return '/clientapi/settings/server'
}

export const systemInfoPath = () => {
  return '/clientapi/systeminfo'
}

export const versionPath = () => {
  return '/clientapi/version'
}

export const syncerStatePath = () => {
  return '/clientapi/syncer/state'
}

export const countersPath = () => {
  return '/clientapi/metrics/counters'
}

export const autosearchDefaultPath = () => {
  return '/clientapi/settings/autosearch_default'
}

export const themeDefaultPath = () => {
  return '/clientapi/settings/theme_default'
}

export const hasSeenKiteLocalPath = () => {
  return '/clientapi/settings/has_seen_kite_local'
}

export const windowModePath = () => {
  return '/clientapi/settings/window_mode'
}

export const setupStagePath = () => {
  return '/clientapi/settings/setup_stage'
}

export const completionsDisabledPath = () => {
  return '/clientapi/settings/completions_disabled'
}

export const metricsDisabledPath = () => {
  return '/clientapi/settings/metrics_disabled'
}

export const autoInstallPluginsPath = () => {
  return '/clientapi/settings/auto_install_new_editor_plugins'
}

export const haveShownWelcomePath = () => {
  return '/clientapi/settings/have_shown_welcome'
}

export const metricsIDPath = () => {
  return '/clientapi/metrics/id'
}

export const proxyModePath = () => {
  return '/clientapi/settings/proxy_mode'
}

export const proxyURLPath = () => {
  return '/clientapi/settings/proxy_url'
}

export const kiteServerURLPath = () => {
  return '/clientapi/settings/kite_enterprise_server'
}

export const kiteServerStatusPath = () => {
  return '/clientapi/settings/kite_enterprise_server/status'
}

export const spyderOptimalSettingsPath = () => {
  return '/clientapi/plugins/spyder/optimalSettings'
}

export const maxFileSizePath = () => {
  return '/clientapi/settings/max_file_size_kb'
}

export const checkEmailPath = '/api/account/check-email'

export const emailVerificationPath = '/api/account/verify-newsletter'

export const uploadLogsPath = () => {
  return '/clientapi/logupload'
}

export const capturePath = () => {
  return '/clientapi/capture'
}

export const autostartDisabledPath = () => {
  return '/clientapi/settings/autostart_disabled'
}

export const showCompletionsCTAPath = () => {
  return '/clientapi/settings/show_completions_cta'
}

export const rcDisabledCompletionsCTAPath = () => {
  return '/clientapi/settings/rc_disabled_completions_cta'
}

export const conversionCohortPath = () => {
  return '/clientapi/settings/conversion_cohort'
}

export const paywallCompletionsRemainingPath = () => {
  return '/clientapi/settings/paywall_completions_remaining'
}

export const fetchCohortPath = () => {
  return '/clientapi/cohort/fetch'
}

export const remoteContentPath = () => {
  return '/clientapi/remotecontent/get'
}

export const fetchRemoteContentPath = () => {
  return '/clientapi/remotecontent/fetch'
}

export const showChooseEnginePath = () => {
  return '/clientapi/settings/test_choose_engine'
}

export const selectedEnginePath = () => {
  return '/clientapi/settings/selected_engine'
}

export const allFeaturesProPath = () => {
  return '/clientapi/settings/all_features_pro'
}

export const emailRequiredPath = () => {
  return '/clientapi/cohort/email_required'
}
