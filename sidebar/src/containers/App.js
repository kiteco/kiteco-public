import React from 'react'
import { connect } from 'react-redux'
import { push } from 'react-router-redux'
import { Route, Switch } from 'react-router-dom'
import PropTypes from 'prop-types'

import * as plugins from '../actions/plugins'
import * as settings from '../actions/settings'
import * as polling from '../actions/polling'
import * as kiteProtocol from '../actions/kite-protocol'
import * as system from '../actions/system'
import * as account from '../actions/account'

import ErrorBoundary from './ErrorBoundary'
import LoginContainer from './LoginContainer'
import Settings from './Settings'
import ChooseEngine from './ChooseEngine'
import Setup from './Setup'
import Sidebar from './Sidebar'
import AppErrors from './AppErrors'
import ColorTheme from './ColorTheme'
import StatePoller from './StatePoller'
import ThirdPartyScripts from './ThirdPartyScripts'
import KitedWaiter from './KitedWaiter'
import LoginRedirector from './LoginRedirector'
import LicenseRefreshRedirector from './LicenseRefreshRedirector.tsx'
import LaunchRedirector from './LaunchRedirector'
import RelatedCodeRedirector from "./Sidebar/related-code/RelatedCodeRedirector.tsx"
import Help from './Help'
import LoginModal from '../components/LoginModal'
import { ModalNames } from '../store/modals'

import { ShortcutManager, Shortcuts } from 'react-shortcuts'
import keymap from '../keymap'
import { handleShortcuts } from '../actions/shortcuts'
import { fetchLicenseInfo, getConversionCohort, getAllFeaturesPro } from '../store/license'
import {
  addElectronAppEventListeners,
  handleDisconnectedCase,
  handleUnresponsiveCase,
  kiteNotWorking,
  reloadSidebar,
  refreshSidebar,
} from '../utils/app-lifecycle'

import {
  handleSetupTransition,
  setElectronWindowSettings,
} from '../utils/app-init'
import { load as loadAnalytics } from '../utils/analytics'
import { fetchRemoteContent, getRemoteContent } from "../store/remotecontent"

const shortcutManager = new ShortcutManager(keymap)

const MAX_POLL_COUNT = 3

class App extends React.Component {

  constructor(props) {
    super(props)
    this.state = {
      SHOULD_BLUR: false,
    }
    this.props.fetchLicenseInfo()
    this.props.fetchRemoteContent()
      .then(() => this.props.getRemoteContent())
    this.props.getSetupCompleted()
      .then(({ success, data }) => {
        if (success && data) {
          this.props.getConversionCohort()
          this.props.getAllFeaturesPro()
        }
      })
  }

  getChildContext() {
    return { shortcuts: shortcutManager }
  }

  reloadKite = () => {
    this.props.reportAttemptRestart().then(reloadSidebar)
  }

  refreshCopilot = () => {
    refreshSidebar()
  }

  componentDidUpdate() {
    const { errors, polling } = this.props
    const { SHOULD_BLUR } = this.state
    const notWorking = kiteNotWorking(errors, polling)
    if (notWorking !== SHOULD_BLUR) {

      this.setState({ SHOULD_BLUR: notWorking })
    }
    if (notWorking) {
      if (!errors.online) {
        //if navigator.onLine, then we'll assume that something is wrong
        //connection wise on our (kited) end
        if (navigator.onLine) {
          //disconnected case - e.g. TypeError from fetch
          if (errors.pollCount < MAX_POLL_COUNT) {
            handleDisconnectedCase(this.props)
          } else {
            this.reloadKite()
          }
        }
      } else {
        //unresponsive case - e.g. call to clientapi/health 503's
        handleUnresponsiveCase(this.props)
      }
    }
  }

  /**
   * We use the UNSAFE method here so as to set up the environment for
   * child components. If we instead used the more idiomatic componentDidMount,
   * we wouldn't be able to use environmental level variables set here in child
   * componentDidMount methods
   */
  UNSAFE_componentWillMount() {
    const { identifyMetricsID, getMetricsDisabled } = this.props
    /*
     * getMetricsDisbaled enables canUse in analytics, which
     * allows loadAnalytics to run and associates future events with the correct identity
     * Identify and store relevant ids in state in the correct order (MetricsID overrides for existing users)
     * then transition for users without setup setting stored
     */
    getMetricsDisabled()
      .then(loadAnalytics)
      .then(identifyMetricsID)
      .then(() => handleSetupTransition(this.props))

    addElectronAppEventListeners(this.props)
    /* Setup Electron main process based on stored user settings */
    setElectronWindowSettings(this.props)

  }

  render() {
    const { errors, theme, activeModal } = this.props
    const { SHOULD_BLUR } = this.state

    return (
      <ErrorBoundary
        handler={this.refreshCopilot}
        alreadyError={errors.appException}
      >
        <KitedWaiter>
          <StatePoller />
          <ThirdPartyScripts />
          <AppErrors reloadHandler={this.reloadKite} />
          <ColorTheme theme={theme}>
            <Shortcuts
              name='App'
              global={true}
              alwaysFireHandler={true}
              handler={this.props.handleShortcuts}
            >
              <RelatedCodeRedirector />
              <Switch>
                <Route exact path="/" component={LaunchRedirector} />
                <Route path="/license-refresh" component={LicenseRefreshRedirector} />
                <Route path="/login-redirector" component={LoginRedirector} />
                <Route path="/login" render={props => <LoginContainer shouldBlur={SHOULD_BLUR} {...props} />} />
                <Route path="/choose-engine" render={props => <ChooseEngine shouldBlur={SHOULD_BLUR} {...props} />} />
                <Route path="/setup" render={props => <Setup shouldBlur={SHOULD_BLUR} {...props} />} />
                <Route path="/settings" render={props => <Settings shouldBlur={SHOULD_BLUR} {...props} />} />
                <Route render={props => <Sidebar shouldBlur={SHOULD_BLUR} {...props} />} />
              </Switch>
              {
                activeModal === ModalNames.LoginModal &&
                <LoginModal/>
              }
            </Shortcuts>
          </ColorTheme>
        </KitedWaiter>
      </ErrorBoundary>
    )
  }
}

App.childContextTypes = {
  shortcuts: PropTypes.object.isRequired,
}

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
  errors: state.errors,
  polling: state.polling,
  theme: state.settings.themeDefault,
  metricsId: state.account.metricsId,
  installId: state.account.installId,
  settings: state.settings,
  licenseInfo: state.license.licenseInfo,
  activeModal: state.modals.active,
})

const mapDispatchToProps = dispatch => ({
  getPlugins: () => dispatch(plugins.getPlugins()),
  addRoute: (route) => dispatch(kiteProtocol.addRoute(route)),
  getDefaultTheme: () => dispatch(settings.getDefaultTheme()),
  getSetupCompleted: () => dispatch(settings.getSetupCompleted()),
  setSetupCompleted: () => dispatch(settings.setSetupCompleted()),
  setSetupNotCompleted: () => dispatch(settings.setSetupNotCompleted()),
  deleteSetupCompleted: () => dispatch(settings.deleteSetupCompleted()),
  getHaveShownWelcome: () => dispatch(settings.getHaveShownWelcome()),
  setHaveShownWelcome: () => dispatch(settings.setHaveShownWelcome()),
  getWindowMode: () => dispatch(settings.getWindowMode()),
  getKitedStatus: () => dispatch(polling.getKitedStatus()),
  getUser: () => dispatch(account.getUser()),
  getProxyMode: () => dispatch(settings.getProxyMode()),
  getProxyURL: () => dispatch(settings.getProxyURL()),
  getMetricsDisabled: () => dispatch(settings.getMetricsDisabled()),
  identifyMetricsID: () => dispatch(account.identifyMetricsID()),
  checkIfOnline: () => dispatch(system.checkIfOnline()),
  forceCheckOnline: () => dispatch(system.forceCheckOnline()),
  reportPolling: (isPolling) => dispatch(polling.reportPolling(isPolling)),
  reportPollingSuccessful: () => dispatch(polling.reportPollingSuccessful()),
  reportAttemptRestart: () => dispatch(polling.reportAttemptRestart()),
  reportRestartSuccessful: () => dispatch(polling.reportRestartSuccessful()),
  reportRestartErrored: () => dispatch(polling.reportRestartErrored()),
  reportNoSupport: () => dispatch(polling.reportNoSupport()),
  push: params => dispatch(push(params)),
  handleShortcuts: handleShortcuts(dispatch),
  fetchLicenseInfo: () => dispatch(fetchLicenseInfo()),
  getAllFeaturesPro: () => dispatch(getAllFeaturesPro()),
  getConversionCohort: ()=> dispatch(getConversionCohort()),
  fetchRemoteContent: () => dispatch(fetchRemoteContent()),
  getRemoteContent: () => dispatch(getRemoteContent()),
})

export default connect(mapStateToProps, mapDispatchToProps)(App)
