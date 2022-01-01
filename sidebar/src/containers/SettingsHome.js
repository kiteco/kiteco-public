import React from 'react'
import { connect } from 'react-redux'
import { push } from 'react-router-redux'
import { metrics } from '../utils/metrics'
import { track } from '../utils/analytics'
import { reloadSidebar } from '../utils/app-lifecycle'
import { Product, ConversionCohorts, checkIfUserNodeAvailable } from '../store/license'
import { localhostProxy } from '../utils/urls'
import { WindowMode } from '../utils/settings'

import AccountBanner from '../components/AccountBanner'
import KiteServerInput from '../components/KiteServerInput'
import MaxFileSizeInput from '../components/MaxFileSizeInput'

import * as settings from '../actions/settings'
import * as system from '../actions/system'
import { getConversionCohort } from '../store/license'

import '../assets/home.css'

const { shell } = window.require("electron")

class SettingsHome extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      protocol: "http",
      hostname: "",
      port: "",
      enterpriseServerURL: "",
    }
  }

  componentDidMount() {
    const {
      getSystemInfo,
      getIconVisible,
      getAutoInstallPluginsEnabled,
      getServer,
      getVersion,
      getDefaultTheme,
      getWindowMode,
      checkIfOnline,
      getCompletionsDisabled,
      getMetricsDisabled,
      getProxyMode,
      getProxyURL,
      getShowCompletionsCTA,
      getRCDisabledCompletionsCTA,
      getConversionCohort,
      getAutostartEnabled,
      settings,
      checkIfUserNodeAvailable,
    } = this.props

    getDefaultTheme()
    getWindowMode()
    getSystemInfo()
    getIconVisible()
    getAutoInstallPluginsEnabled()
    getShowCompletionsCTA()
    getRCDisabledCompletionsCTA()
    getConversionCohort()
    getServer()
    getVersion()
    getProxyMode()
    checkIfOnline()
    getCompletionsDisabled()
    checkIfUserNodeAvailable()
    getMetricsDisabled().then(({ success, data }) => {
      if (success) {
        this.originalMetricsDisabled = data
      } else {
        this.originalMetricsDisabled = false
      }
    })

    if (settings.autostartEnabled === undefined) {
      getAutostartEnabled()
    }

    metrics.incrementCounter('sidebar_settings_home_opened')

    getProxyURL().then(response => {
      // parsing the url here because JavaScript's URL is unable to handle unknown protocol values like "socks5"
      let regexp = new RegExp("^([^:]+)://(?:([^:@]+)(?::([^@]+))?@)?([^:]+):(.+)$")
      let match = regexp.exec(response.data)
      if (match && match.length === 6) {
        this.setState({
          protocol: match[1] || "",
          hostname: match[4] || "",
          port: match[5] || "",
        })
      }
    })
  }

  disableCompletions = () => {
    track({ event: 'copilot_settings_completions_disabled' })
    this.props.disableCompletions()
  }

  enableCompletions = () => {
    this.props.enableCompletions().then(() => {
      track({ event: 'copilot_settings_completions_enabled' })
    })
  }

  disableMetrics = () => {
    track({ event: 'copilot_settings_metrics_disabled' })
    this.props.disableMetrics()
  }

  enableMetrics = () => {
    this.props.enableMetrics().then(() => {
      track({ event: 'copilot_settings_metrics_enabled' })
    })
  }

  disableAutoInstall = () => {
    track({ event: 'copilot_settings_autoinstall_disabled' })
    this.props.setAutoInstallPluginsEnabled(false)
  }

  enableAutoInstall = () => {
    this.props.setAutoInstallPluginsEnabled(true).then(() => {
      track({ event: 'copilot_settings_autoinstall_enabled' })
    })
  }

  disableAutostart = () => {
    this.props.setAutostartEnabled(false)
    track({ event: 'copilot_settings_autostart_disabled' })
  }

  enableAutostart = () => {
    this.props.setAutostartEnabled(true)
    track({ event: 'copilot_settings_autostart_enabled' })
  }

  restartKite = () => {
    reloadSidebar(300)
  }

  redoSetup = () => {
    this.props.setCurrentSetupStage()
    this.props.setSetupNotCompleted()
    this.props.push("/setup")
  }

  onCustomProxyChange = (event) => {
    const name = event.target.name
    const value = event.target.value

    this.setState((prevState, props) => {
      // this needs to happen in two steps because of
      // https://github.com/microsoft/TypeScript/issues/38175
      const obj = { ...prevState, [name]: value }
      const { protocol, hostname, port } = obj
      if (protocol && hostname && port) {
        props.setProxyURL(`${protocol}://${hostname}:${port}`)
      }

      return {
        [name]: value,
      }
    })
  }

  upgradeAction = () => {
    shell.openExternal(localhostProxy('/clientapi/desktoplogin?d=%2Fpro%3Floc%copilot_settings%26src%3Dlearn_more'))
  }

  render() {
    const {
      status,
      system,
      settings,
      setIconVisible,
      setIconInvisible,
      setDefaultTheme,
      setWindowMode,
      setProxyMode,
      userNodeAvailable,
    } = this.props

    return (
      <div className="main__sub">
        <div className="home">
          {userNodeAvailable && <div className="home__section">
            <h2 className="section__title">
              Account
            </h2>
            <AccountBanner />
          </div>}
          <div className="home__section">
            <h2 className="section__title">Options</h2>
            <SettingsRadioButton
              name='theme'
              label='Theme'
              options={[{ value: 'dark', label: 'Dark' }, { value: 'light', label: 'Light' }, { value: 'high-contrast', label: 'High contrast' }]}
              setTo={settings.themeDefault}
              setFunc={setDefaultTheme}
            />

            <SettingsRadioButton
              name='window_mode'
              label='Window'
              options={[
                { value: WindowMode.NORMAL, label: 'Normal' },
                { value: WindowMode.FOCUS_ON_DOCS, label: 'Focus on docs' },
                { value: WindowMode.ALWAYS_ON_TOP, label: 'Always on top' }]}
              setTo={settings.windowMode}
              setFunc={setWindowMode}
            />

            <SettingsCheckbox
              setTo={settings.pluginsAutoInstall}
              setOn={this.enableAutoInstall}
              setOff={this.disableAutoInstall}
            >
              <p>Automatically install the Kite plugin for all supported editors</p>
            </SettingsCheckbox>
            {system && system.os !== "linux" &&
              <SettingsCheckbox
                setTo={settings.iconVisible}
                setOn={setIconVisible}
                setOff={setIconInvisible}
              >
                <p>Show Kite icon in your&nbsp;
                  {system.os === "darwin" && "menu bar"}
                  {system.os === "linux" && "notification center"}
                  {system.os === "windows" && "system tray"}
                  {!system.os && "menubar"}
                </p>
              </SettingsCheckbox>
            }
            <MultiwordInKiteFreeCheckbox
              conversionCohort={this.props.conversionCohort}
              licenseInfo={this.props.licenseInfo}
              rcDisabledCompletionsCTA={this.props.rcDisabledCompletionsCTA}
              showCompletionsCTA={settings.showCompletionsCTA}
              enableCompletionsCTA={() => {
                this.props.setShowCompletionsCTA(true)
                track({ event: 'copilot_settings_completions_cta_enabled' })
              }}
              disableCompletionsCTA={() => {
                this.props.setShowCompletionsCTA(false)
                track({ event: 'copilot_settings_completions_cta_disabled' })
              }}
            />

            <React.Fragment>
              <SettingsCheckbox
                setTo={settings.completionsDisabled}
                setOn={this.disableCompletions}
                setOff={this.enableCompletions}
              >
                <p>Disable all completions from Kite</p>
              </SettingsCheckbox>
            </React.Fragment>

            <React.Fragment>
              <SettingsCheckbox
                setTo={settings.metricsDisabled}
                setOn={this.disableMetrics}
                setOff={this.enableMetrics}
              >
                <p>Disable sending usage metrics</p>
              </SettingsCheckbox>
              {
                typeof this.originalMetricsDisabled !== 'undefined' &&
                (settings.metricsDisabled !== this.originalMetricsDisabled) &&
                <p className="home__disclaimer"><span
                  className="home__disclaimer--link"
                  onClick={this.restartKite}>
                  Restart Kite</span> for this to take effect</p>
              }
            </React.Fragment>

            <React.Fragment>
              <SettingsCheckbox
                setTo={settings.autostartEnabled}
                setOn={this.enableAutostart}
                setOff={this.disableAutostart}
              >
                <p>Kite automatically starts on boot</p>
              </SettingsCheckbox>
            </React.Fragment>

            <Labeled label="Proxy" className="home__proxy">
              <SettingsRadioButton
                name='proxy_mode'
                options={[{ value: 'direct', label: 'None' }, { value: 'environment', label: 'System' }, { value: 'manual', label: 'Custom' }]}
                setTo={settings.proxyMode}
                setFunc={setProxyMode} />

            </Labeled>
            {
              settings.proxyMode === 'manual' &&
              <CustomProxy className="home__proxy home__proxy__custom"
                disabled={false}
                protocol={this.state.protocol}
                hostname={this.state.hostname}
                port={this.state.port}
                isValid={this.state.protocol !== "" && this.state.hostname !== "" && this.state.port !== ""}
                onChange={this.onCustomProxyChange}
              />
            }

            <Labeled label="Server" className="home__server">
              <KiteServerInput/>
            </Labeled>

            <Labeled label="Max File Size (KB)" className="home__maxFileSize">
              <MaxFileSizeInput/>
            </Labeled>

            <div className="home__redo home__row"><div>Redo Setup:</div>
              <div
                className="home__redo__btn"
                onClick={this.redoSetup}></div>
            </div>
          </div>
        </div>
        <div className="home__version">Version: {system.version}</div>
      </div>
    )
  }
}

function MultiwordInKiteFreeCheckbox({
  conversionCohort,
  licenseInfo,
  rcDisabledCompletionsCTA,
  showCompletionsCTA,
  enableCompletionsCTA,
  disableCompletionsCTA,
}) {
  switch (conversionCohort) {
    case ConversionCohorts.OptIn:
    case ConversionCohorts.Autostart:
    case ConversionCohorts.QuietAutostart:
      break
    default:
      return null
  }
  if (rcDisabledCompletionsCTA) {
    return null
  }
  if (!licenseInfo || licenseInfo.product !== Product.Free || licenseInfo.trial_available) {
    return null
  }
  return (
    <SettingsCheckbox
      setTo={showCompletionsCTA}
      setOn={enableCompletionsCTA}
      setOff={disableCompletionsCTA}
    >
      <p>Show multi-word completions in Kite Free</p>
    </SettingsCheckbox>
  )
}

const SettingsCheckbox = ({ setTo, setOn, setOff, children }) =>
  <div className="home__row">
    <button
      className="home__check"
      onClick={setTo ? setOff : setOn}
    >
      {setTo ? "✓" : ""}
    </button>
    <div className="home__column">
      {children}
    </div>
  </div>

const SettingsRadioButton = ({ name, label, options, setTo, setFunc }) =>
  <div className="home__row home__radio">{label && <div>{label}:</div>}<div>
    {options.map((option, i) =>
      <div key={i}>
        <input name={name} type='radio' value={option.value} id={name + option.value}
          checked={option.value === setTo} onChange={setFunc(option.value)}></input>
        <label htmlFor={name + option.value}>{option.label}</label>
      </div>
    )}</div></div>

const Labeled = ({ label, className, children }) =>
  <div className={`home__row ${className}`}><div>{label}:</div>{children}</div>

const CustomProxy = ({ className, onChange, disabled, protocol, hostname, port, isValid }) => <div className={className}>
  <select name="protocol" disabled={disabled} title="proxy type" onChange={onChange} className="home__proxy__proto">
    <option value="http" selected={protocol === 'http'}>http://</option>
    <option value="socks5" selected={protocol === 'socks5'}>socks5://</option>
  </select>
  <input type="text" name="hostname" disabled={disabled} value={hostname} onChange={onChange} className="home__proxy__host" placeholder="hostname" required={true} title="hostname of the proxy server" />
  <span>:</span>
  <input type="text" name="port" disabled={disabled} value={port} onChange={onChange} className="home__proxy__port" placeholder="port" required={true} title="port of the proxy server" />
  {isValid && <span className="home__proxy__state" title="Your proxy settings are complete and applied to Kite">✓</span>}
</div>

const mapStateToProps = (state, ownProps) => ({
  settings: state.settings,
  system: state.system,
  status: state.account.status,
  ...ownProps,
  userNodeAvailable: state.license.userNodeAvailable,
  licenseInfo: state.license.licenseInfo,
  rcDisabledCompletionsCTA: state.settings.rcDisabledCompletionsCTA,
  conversionCohort: state.license.conversionCohort,
})

const mapDispatchToProps = dispatch => ({
  getDefaultTheme: () => dispatch(settings.getDefaultTheme()),
  setDefaultTheme: (theme) => dispatch(settings.setDefaultTheme(theme)),
  getWindowMode: () => dispatch(settings.getWindowMode()),
  setWindowMode: mode => dispatch(settings.setWindowModeWrapped(mode)),
  getIconVisible: () => dispatch(settings.getIconVisible()),
  setIconVisible: () => dispatch(settings.setIconVisible()),
  setIconInvisible: () => dispatch(settings.setIconInvisible()),
  getAutoInstallPluginsEnabled: () => dispatch(settings.getAutoInstallPluginsEnabled()),
  setAutoInstallPluginsEnabled: (enabled) => dispatch(settings.setAutoInstallPluginsEnabled(enabled)),
  getShowCompletionsCTA: () => dispatch(settings.getShowCompletionsCTA()),
  setShowCompletionsCTA: enabled => dispatch(settings.setShowCompletionsCTA(enabled)),
  getRCDisabledCompletionsCTA: () => dispatch(settings.getRCDisabledCompletionsCTA()),
  getConversionCohort: () => dispatch(getConversionCohort()),
  getAutosearchDefault: () => dispatch(settings.getAutosearchDefault()),
  getServer: () => dispatch(settings.getServer()),
  getSystemInfo: () => dispatch(system.getSystemInfo()),
  getVersion: () => dispatch(system.getVersion()),
  checkIfOnline: () => dispatch(system.checkIfOnline()),
  checkIfUserNodeAvailable: () => dispatch(checkIfUserNodeAvailable()),
  enableCompletions: () => dispatch(settings.setCompletionsDisabled("false")),
  disableCompletions: () => dispatch(settings.setCompletionsDisabled("true")),
  getCompletionsDisabled: () => dispatch(settings.getCompletionsDisabled()),
  enableMetrics: () => dispatch(settings.setMetricsDisabled("false")),
  disableMetrics: () => dispatch(settings.setMetricsDisabled("true")),
  getMetricsDisabled: () => dispatch(settings.getMetricsDisabled()),
  setSetupNotCompleted: () => dispatch(settings.setSetupNotCompleted()),
  setCurrentSetupStage: stage => dispatch(settings.setCurrentSetupStage(stage)),
  getProxyMode: () => dispatch(settings.getProxyMode()),
  setProxyMode: (mode) => dispatch(settings.setProxyMode(mode)),
  getProxyURL: () => dispatch(settings.getProxyURL()),
  setProxyURL: (url) => dispatch(settings.setProxyURL(url)),
  getAutostartEnabled: () => dispatch(settings.getAutostartEnabled()),
  setAutostartEnabled: (enabled) => dispatch(settings.setAutostartEnabled(enabled)),
  push: params => dispatch(push(params)),
})

export default connect(mapStateToProps, mapDispatchToProps)(SettingsHome)
