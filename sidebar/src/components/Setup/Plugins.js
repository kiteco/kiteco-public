import React from 'react'
import { connect } from 'react-redux'

import { Domains } from '../../utils/domains'
import { track } from '../../utils/analytics'
import { timeoutAfter } from '../../utils/fetch'
import { fetchConversionCohort, fetchCohortTimeoutMS } from '../../store/license'
import { forceCheckOnline, checkIfOnline } from '../../actions/system'
import * as plugins from '../../actions/plugins'

import '../../assets/setup/plugins.css'

import { getIconForEditor } from '../../utils/editorInfo'
import {
  runningInstallDisable,
  REMOTE_INSTALL_EDITORS,
  getDefaultPlugins,
  installMultiple,
  isFullyInstalled,
} from '../../utils/plugins'

import Spinner from '../Spinner'

class SetupPlugins extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      loading: false,
      error: "",
      install: {},
      defaultedInstalls: false,
    }
  }

  getPathsById(object, value) {
    return Object.keys(object).filter(key => object[key] === value)
  }

  markAllEditorsByDefault(forceUpdate) {
    const { plugins, system } = this.props
    if (this.props.plugins && (!this.state.defaultedInstalls || (forceUpdate && !this.state.loading))) {
      const defaultPlugins = getDefaultPlugins({ plugins, system })
      this.setState({
        defaultedInstalls: true,
        install: { ...defaultPlugins },
      })
    }
  }

  componentDidMount() {
    this.markAllEditorsByDefault()
    track({ event: "onboarding_plugins_step_mounted" })
    this.props.checkIfOnline()
  }

  componentDidUpdate(prevProps) {
    const forceUpdate = prevProps.system.networkConnected !== this.props.system.networkConnected
    this.markAllEditorsByDefault(forceUpdate)
  }

  static getDerivedStateFromProps(props) {
    const plugins = props.plugins || []
    const installed_editors = plugins.filter(plugin =>
      plugin.editors && plugin.editors.length
    ).map(plugin => plugin.id)
    props.updateEncounteredEditors(installed_editors)
    return null
  }

  uninstall = ({ path }) => () => {
    this.setState({
      install: {
        ...this.state.install,
        [path]: null,
      },
    })
  }

  install = ({ path, id }) => () => {
    this.setState({
      install: {
        ...this.state.install,
        [path]: id,
      },
    })
  }

  submit = async () => {
    this.setState({ loading: true })
    const [{ filtered, installed }] = await Promise.all([
      this.installSelected(),
      timeoutAfter(this.props.fetchConversionCohort, fetchCohortTimeoutMS),
    ])
    this.setState({ loading: false })
    return this.props.advance({ filtered, installed })
  }

  installSelected = async () => {
    const { results, installed, filtered, successes, errors } = await installMultiple({
      forceCheckOnline: this.props.forceCheckOnline,
      toInstall: this.state.install,
      install: this.props.installPlugin,
    })
    track({
      event: "onboarding_plugins_step_install_clicked",
      props: {
        num_editors_rendered: results.length,
        num_plugins_installed: installed.length,
        plugins_installed: installed,
      },
    })
    if ( !successes.every(s => s) ) {
      this.setState({ error: errors.pop() })
      // TODO: surface these errors to the user/backend metrics somehow. Notification?
    }
    const { success, error } = await this.props.getPlugins()
    if ( !success ) {
      this.setState({ error })
    }
    return { filtered, installed }
  }

  refresh = () => {
    const { getPlugins } = this.props
    this.setState({
      loading: true,
      error: null,
    })
    getPlugins().then(() => this.setState({ loading: false }))
  }

  render() {
    const plugins = this.props.plugins || []
    const { system } = this.props
    const installed_editors = plugins.filter(plugin =>
      plugin.editors && plugin.editors.length
        && plugin.editors.some(editor =>  !editor.compatibility)
        && plugin.editors.some(editor => !editor.plugin_installed )
    ).map(plugin => {
      // sort incompatible editors to the end
      plugin.editors.sort((a, b) => {
        if (a.version_required && !b.version_required) {
          return 1
        }
        if (!a.version_required && b.version_required) {
          return -1
        }
        return 0
      })
      return plugin
    })

    const fully_installed_editors = plugins.filter(plugin =>
      plugin.editors && plugin.editors.length && plugin.editors.every(isFullyInstalled)
    )
    const uninstalled_editors = plugins.filter(plugin =>
      !(plugin.editors && plugin.editors.length)
    )
    return (
      <div className="setup__plugins__container">
        { !this.props.plugins &&
          <Spinner text="Detecting Kite support on your computer..."/>
        }
        { this.state.loading &&
          <Spinner text="This may take up to a minute..."/>
        }
        <div className={`
          setup__plugins
          ${this.state.loading || !this.props.plugins ? "setup__plugins--loading" : ""}
        `}>
          <h2 className="setup__title">Kite Setup</h2>
          { this.state.error &&
            <p className="setup__plugins__error">
              { this.state.error.detail || this.state.error }
            </p>
          }
          <p className="setup__text showup__animation">To get started, let’s choose which editor plugins to install on this machine.</p>

          <div className="setup__plugins__list showup__animation">
            { installed_editors.map(plugin => {
              let networkDisabled = false
              if (!system.networkConnected) {
                if (REMOTE_INSTALL_EDITORS.find(ed => ed.id === plugin.id)) {
                  networkDisabled = true
                }
              }
              return <Plugin
                key={plugin.id}
                install={this.install}
                uninstall={this.uninstall}
                {...plugin}
                installedList={this.state.install}
                separator={this.props.separator}
                refresh={this.refresh}
                networkDisabled={networkDisabled}
              />
            }
            )}
            { installed_editors.length === 0 && uninstalled_editors.length === 0 && this.props.plugins &&
              <p>We were unable to find any editors on your system that Kite supports.</p>
            }
            { fully_installed_editors.length > 0 && <p>Kite is already integrated with the following editors on your system:
              &nbsp;{ fully_installed_editors.map(e => e.name).join(", ") }.</p>}
            <p>
              Kite also supports JupyterLab v2.2 and above. Learn how to <a className="plugins__more-info" href={`https://${Domains.Help}/article/143-how-to-install-the-jupyterlab-plugin`} target="_blank">integrate Kite with JupyterLab</a>.
            </p>
            { uninstalled_editors.length > 0 &&
              <p>Kite could not detect the following editors on your system:&nbsp;
                { uninstalled_editors.map(e => e.name).join(", ") }.
                <br/>If you have these editors installed, please make sure they are running
                and <a className="plugins__try-again" href="#try-again" onClick={this.refresh}>try again</a>.
              </p>
            }
            {
              uninstalled_editors.some(e => e.id === 'spyder') && (
                <p>
                  If you installed Kite from Spyder directly, you can ignore
                  this message. Kite has automatically been integrated with
                  Spyder.
                </p>
              )
            }
          </div>
          <p className="setup__text showup__animation">Note: Installation can take up to a minute.</p>
          <button
            className="setup__button showup__animation"
            onClick={this.submit}
          >
            Install
          </button>
          <div className="setup__links-container setup__links-container--centered">
            <a
              href="#"
              className="home__link__settings"
              onClick= {() => this.props.advance()}
            >
              Skip this step
            </a>
          </div>
        </div>
      </div>
    )
  }
}

const Plugin = ({
  id,
  name,
  editors,
  install,
  uninstall,
  separator,
  installedList,
  multiple_install_locations,
  running,
  install_while_running,
  refresh,
  networkDisabled,
}) => {
  const displayExpanded = (multiple_install_locations && editors.length > 1)
                          || editors.some((plugin) => plugin.hasOwnProperty('compatibility'))

  const disabled = runningInstallDisable({ running, install_while_running })
  return (
    <div className="setup__plugins__plugin">
      <img
        className="setup__plugins__icon"
        src={getIconForEditor(id)}
        alt={name}
      />
      <div className="setup__plugins__editor-list">
        <h3 className="setup__subtitle">{name}</h3>
        { disabled &&
          <div className="setup__plugins__warning">
            Please close all running instances of { name } before installing.&nbsp;
            <a className="setup__plugins__try-again" href="#try-again" onClick={refresh}>
              (Try again)
            </a>
          </div>
        }
        { networkDisabled &&
          <p className="setup__plugins__warning">
            {name} needs a network connection to install
          </p>
        }
        {displayExpanded && editors.map((editor, i) =>
          <Editor
            key={i}
            separator={separator}
            install={install}
            uninstall={uninstall}
            id={id}
            installed={installedList[editor.path]}
            disabled={disabled}
            networkDisabled={networkDisabled}
            refresh={refresh}
            {...editor}
          />
        )}
      </div>
      {!displayExpanded && <SingleEditor
        install={install}
        uninstall={uninstall}
        id={id}
        installed={installedList[editors[0].path]}
        disabled={disabled}
        networkDisabled={networkDisabled}
        refresh={refresh}
        {...editors[0]}
      />}
    </div>
  )
}

const SingleEditor = ({
  id,
  path,
  install,
  uninstall,
  installed,
  version_required,
  disabled,
  networkDisabled,
  refresh,
}) => (
  <div className="plugin__single__editor">
    { !version_required && !(networkDisabled && !installed) &&
      <button
        className={`setup__plugins__checkbox setup__plugins__checkbox--installed-${installed} ${ (disabled || networkDisabled) ? "setup__plugins__checkbox--disabled" : "" } `}
        onClick={ installed ? uninstall({ id, path }) : install({ id, path })}
      >
        ✓
      </button>
    }
    { version_required && <p className="plugins__compatibility-message setup__plugin__compatibility">
      Please upgrade to version {version_required} or above and <a className="setup__plugins__try-again" href="#try-again" onClick={refresh}>try again</a>.
    </p>
    }
  </div>
)

const Editor = ({
  separator,
  install,
  uninstall,
  installed,
  path,
  compatibility,
  version,
  version_required,
  disabled,
  networkDisabled,
  refresh,
  id, // temp
}) => (
  <div className="plugins__editor">
    <div className="plugins__editor-detail setup__plugins__detail">
      { version &&
        <p>Version: { version }</p>
      }
      { path &&
        <p>Path: { path.split(separator).map((component, i) => {
          if (i) {
            return <span key={i}>
              { separator }<wbr/>{ component }
            </span>
          } else {
            return <span key={i}>
              { component }
            </span>
          }
        })}</p>
      }
      { version_required &&
        <p className="plugins__compatibility-message setup__plugin__compatibility">
          Please upgrade to version {version_required} or above and <a className="setup__plugins__try-again" href="#try-again" onClick={refresh}>try again</a>.
        </p>
      }
    </div>
    { !compatibility &&
      <button
        className={`setup__plugins__checkbox ${ (disabled || networkDisabled) ? "setup__plugins__checkbox--disabled" : ""}`}
        onClick={ installed ? uninstall({ id, path }) : install({ id, path })}
      >
        { installed ? "✓" : "" }
      </button>
    }
  </div>
)

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
  plugins: state.plugins,
  system: state.system,
})

const mapDispatchToProps = dispatch => ({
  fetchConversionCohort: () => dispatch(fetchConversionCohort()),
  forceCheckOnline: () => dispatch(forceCheckOnline()),
  checkIfOnline: () => dispatch(checkIfOnline()),
  installPlugin: id => dispatch(plugins.installPlugin(id)),
  uninstallPlugin: id => dispatch(plugins.uninstallPlugin(id)),
  updateEncounteredEditors: ids => dispatch(plugins.updateEncounteredEditors(ids)),
  getPlugins: () => dispatch(plugins.getPlugins()),
})

export default connect(mapStateToProps, mapDispatchToProps)(SetupPlugins)
