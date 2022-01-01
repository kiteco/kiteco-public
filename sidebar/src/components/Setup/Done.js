import React from 'react'

import { connect } from 'react-redux'
import { track } from '../../utils/analytics'
import { reloadSidebar } from '../../utils/app-lifecycle'
import { completionFlow } from '../../utils/setup'
import * as settings from '../../actions/settings.js'
import { forceCheckOnline } from '../../actions/system.js'
import '../../assets/setup/done.css'

class Done extends React.Component {
  componentDidMount() {
    track({ event: "onboarding_done_step_mounted" })
  }

  done = () => {
    const {
      setSetupCompleted,
      forceCheckOnline,
      getHaveShownWelcome,
      setHaveShownWelcome,
      metricsId,
      installId,
    } = this.props
    completionFlow({
      setSetupCompleted,
      forceCheckOnline,
      getHaveShownWelcome,
      setHaveShownWelcome,
      metricsId,
      installId,
    })
  }

  restartKite = () => {
    reloadSidebar(300)
  }

  setAutoInstallPluginsEnabled = () => {
    this.props.setAutoInstallPluginsEnabled(true)
  }

  setAutoInstallPluginsDisabled = () => {
    this.props.setAutoInstallPluginsEnabled(false)
  }

  render() {
    return <div className="setup__done showup__animation">
      <h2 className="setup__title">Kite is ready!</h2>
      <p className="setup__text">
        <span>Welcome to the future of programming.</span>
        <br/><br/>
        <span>Restart your editor to activate Kite. If you haven’t installed the Kite plugin yet, you can do so from settings.</span>
      </p>
      { this.props.system && this.props.system.os !== "linux" &&
        <SettingsCheckbox
          setTo={this.props.settings.iconVisible}
          setOn={this.props.setIconVisible}
          setOff={this.props.setIconInvisible}
        >
          <p className="setup__text">Show the Kite icon in your&nbsp;
            { this.props.system.os === "darwin" && "menu bar" }
            { this.props.system.os === "linux" && "notification center" }
            { this.props.system.os === "windows" && "system tray" }
            { !this.props.system.os && "menu bar" }.
          </p>
        </SettingsCheckbox>
      }
      <SettingsCheckbox
        setTo={this.props.settings.pluginsAutoInstall}
        setOn={this.setAutoInstallPluginsEnabled}
        setOff={this.setAutoInstallPluginsDisabled}
      >
        <p className="setup__text">Automatically integrate Kite when new editors are installed</p>
      </SettingsCheckbox>
      <button className="setup__button" onClick={this.done}>
        Let&#39;s go!
      </button>
    </div>
  }
}

const SettingsCheckbox = ({ setTo, setOn, setOff, children }) =>
  <div className="setup__icon">
    <button
      className="setup__icon__check"
      onClick={ setTo ? setOff : setOn }
    >
      { setTo ? "✓" : "" }
    </button>
    {children}
  </div>

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
  system: state.system,
  settings: state.settings,
  plugins: state.plugins,
  metricsId: state.account.metricsId,
  installId: state.account.installId,
})

const mapDispatchToProps = dispatch => ({
  setAutoInstallPluginsEnabled: enabled => dispatch(settings.setAutoInstallPluginsEnabled(enabled)),
  setIconVisible: () => dispatch(settings.setIconVisible()),
  setIconInvisible: () => dispatch(settings.setIconInvisible()),
  setHaveShownWelcome: () => dispatch(settings.setHaveShownWelcome()),
  getHaveShownWelcome: () => dispatch(settings.getHaveShownWelcome()),
  forceCheckOnline: () => dispatch(forceCheckOnline()),
})

export default connect(mapStateToProps, mapDispatchToProps)(Done)
