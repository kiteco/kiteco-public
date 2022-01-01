import React from 'react'

import { track } from '../../utils/analytics'

import '../../assets/setup/plugins.css'

import { getIconForEditor } from '../../utils/editorInfo'

class SetupRestartPlugins extends React.Component {

  componentDidMount() {
    const { advance } = this.props
    const needsRestart = this.needsRestart()
    const couldNotInstall = this.couldNotInstall()
    track({ event: "onboarding_restart-plugins_step_mounted" })
    if (needsRestart.length === 0 && couldNotInstall.length === 0) {
      advance()
    }
  }

  couldNotInstall = () => {
    const { location } = this.props
    return (location.state && location.state.filtered) || []
  }

  needsRestart = () => {
    const plugins = this.props.plugins || []
    const location = this.props.location
    const installed = (location && location.state && location.state.installed) || []
    if (plugins.length === 0 || installed.length === 0) {
      return []
    }
    const installed_plugins = plugins.filter(p => installed.includes(p.id))
    return installed_plugins.filter(p => p.running && p.requires_restart)
  }

  render() {
    const { advance } = this.props
    const needsRestart = this.needsRestart()
    const couldNotInstall = this.couldNotInstall()
    return <div className={`setup__plugins`}>
      { needsRestart.length > 0 &&
        <React.Fragment>
          <h2 className="setup__title">
            Restart your editors
          </h2>
          <p className="setup__text showup__animation">
            Just a heads up that you'll need to restart these editors to activate Kite.
          </p>
          <div className="setup__flex showup__animation">
            { needsRestart.map(plugin =>
              <Plugin
                key={plugin.id}
                {...plugin}
              />
            )}
          </div>
        </React.Fragment>
      }
      { needsRestart.length === 0 && couldNotInstall.length > 0 &&
        <h2 className="setup__title">
          Installation Failure
        </h2>
      }
      { couldNotInstall.length > 0 &&
        <p className="setup__text showup__animation">
          We weren't able to install plugins for {
            couldNotInstall.map((ed, i) => {
              if (i === couldNotInstall.length - 1) {
                return 'or ' + ed.name + ' '
              }
              return couldNotInstall.length <= 2
                ? ed.name + ' '
                : ed.name +', '
            })
          }due to a network connection problem. Try again later in settings when
          a connection is restored
        </p>
      }
      <button
        className="setup__button showup__animation"
        onClick={advance}
      >
        Continue
      </button>
    </div>
  }
}

const Plugin = ({ id, name, icon }) =>
  <div className="setup__plugins__plugin">
    <img
      className="setup__plugins__icon"
      src={getIconForEditor(id)}
      alt={name}
    />
    <div className="setup__plugins__editor-list">
      <h3 className="setup__subtitle">{name}</h3>
    </div>
  </div>

export default SetupRestartPlugins
