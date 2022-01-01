import React from 'react'
import { connect } from 'react-redux'
import { metrics } from '../utils/metrics'
import { getIconForEditor } from '../utils/editorInfo'

import * as actions from '../actions/plugins'
import * as system from '../actions/system'

import { notify, dismiss } from '../store/notification'

import '../assets/plugins.css'
import Spinner from '../components/Spinner'
import LoadingButton from '../components/LoadingButton'
import { Domains } from '../utils/domains'

const REMOTE_INSTALL_EDITORS = ['atom', 'vscode']

class Plugins extends React.Component {
  constructor(props) {
    super(props)
    this.pluginsList = React.createRef()
    this.state = {
      loading: true,
      error: null,
    }
  }

  componentDidMount() {
    if (!this.props.separator) {
      this.props.updateSystem()
    }
    this.props.getPlugins()
      .then(() => this.setState({ loading: false }))

    metrics.incrementCounter('sidebar_settings_plugins_opened')

    this.props.checkIfOnline()
  }

  componentWillUnmount() {
    clearInterval(this.onlinePoller)
  }

  static getDerivedStateFromProps(props) {
    const plugins = props.plugins || []
    const installed_editors = plugins.filter(plugin =>
      plugin.editors && plugin.editors.length
    ).map(plugin => plugin.id)
    props.updateEncounteredEditors(installed_editors)
    return null
  }

  install = params => () => {
    const {
      notify,
      installPlugin,
      getPlugins,
      dismiss,
      forceCheckOnline,
    } = this.props
    this.setState({
      loading: true,
      error: null,
    })
    forceCheckOnline().then(({ success, isOnline }) => {
      if (!(success && isOnline) && REMOTE_INSTALL_EDITORS.includes(params.id)) {
        return getPlugins() //?
      } else {
        return installPlugin(params)
          .then(({ success, error, data }) => {
            if (success) {
              const { id } = params
              const { name, requires_restart, running } = data
              if (requires_restart === true && running === true) {
                notify({
                  id: `plugins--${id}`,
                  component: 'plugins',
                  payload: {
                    ...params,
                    name,
                  },
                })
              }
              dismiss('noplugins')
              return getPlugins()
            } else {
              this.setState({ error })
            }
          })
      }
    })
      .then(() => this.setState({ loading: false }))
  }

  scrollToTop = () => {
    if (this.pluginsList.current.scrollTop > 0) {
      const step = Math.min(20, this.pluginsList.current.scrollTop)
      this.pluginsList.current.scrollTop -= step
      setTimeout(() => {
        this.scrollToTop()
      }, 5)
    }
  }

  refresh = () => {
    this.scrollToTop()
    const { getPlugins } = this.props
    this.setState({
      loading: true,
      error: null,
    })
    getPlugins().then(() => this.setState({ loading: false }))
  }

  uninstall = params => () => {
    this.setState({
      loading: true,
      error: null,
    })
    this.props.uninstallPlugin(params)
      .then(({ success, error }) => {
        if (success) {
          return this.props.getPlugins()
        } else {
          this.setState({ error })
        }
      })
      .then(() => this.setState({ loading: false }))
  }

  render() {
    const plugins = this.props.plugins || []
    const { networkConnected } = this.props
    const installed_editors = plugins.filter(plugin =>
      (plugin.editors && plugin.editors.length) || plugin.manual_install_only
    )

    const installed_plugins = installed_editors.filter(editor =>
      editor.editors.some((plugin) => plugin.plugin_installed === true)
    )

    const not_installed_plugins = installed_editors.filter(editor => {
      return editor.editors.every((plugin) => plugin.plugin_installed === false)
    })

    const uninstalled_editors = plugins.filter(plugin =>
      !(plugin.editors && plugin.editors.length) && !plugin.manual_install_only
    )

    return (
      <div className="main__sub">
        <div className="plugins">
          <div className={`plugins__error ${this.state.error ? "" : "plugins__error--hide"}`}>
            <h3>{ this.state.error && this.state.error.title }</h3>
            <p>{ this.state.error && this.state.error.detail }</p>
          </div>
          <div ref={this.pluginsList} className="plugins__list">
            { this.state.loading &&
              <Spinner theme={`light ${plugins.length > 0 ? 'compact': ''}`}/>
            }
            { installed_plugins.length > 0 && <div>
              <h3 className="section__title">Installed editor plugins</h3>
              {installed_plugins.map(plugin => (
                <Plugin
                  className="plugins__plugin--installed"
                  key={plugin.id}
                  install={this.install}
                  uninstall={this.uninstall}
                  separator={this.props.separator}
                  loading={this.state.loading}
                  refresh={this.refresh}
                  {...plugin}
                />
              ))}
            </div> }
            { not_installed_plugins.length > 0 && <div>
              <h3 className="section__title">Available editor plugins</h3>
              {not_installed_plugins.map(plugin => {
                let networkDisabled = false
                if (!networkConnected) {
                  if (REMOTE_INSTALL_EDITORS.includes(plugin.id)) {
                    networkDisabled = true
                  }
                }
                return plugin.manual_install_only ? <ManualEditor key={plugin.id} icon={getIconForEditor(plugin.id)} {...plugin} /> : (
                  <Plugin
                    className="plugins__plugin--not-installed"
                    key={plugin.id}
                    install={this.install}
                    uninstall={this.uninstall}
                    separator={this.props.separator}
                    loading={this.state.loading}
                    refresh={this.refresh}
                    networkDisabled={networkDisabled}
                    {...plugin}
                  />
                )
              })}
            </div> }
            { !this.state.loading && installed_editors.length === 0 &&
              <p> We could not find any supported editors on your system. If this is an error, please <a rel="noopener noreferrer" target="_blank" href="https://github.com/kiteco/issue-tracker">report it</a>.</p>
            }
            { uninstalled_editors.length > 0 &&
              <div className="plugins__also">
                <h3 className="section__title">Undetected Editors</h3>
                { uninstalled_editors.map( editor =>
                  <UninstalledEditor key={editor.id} icon={getIconForEditor(editor.id)} refresh={this.refresh} id={editor.id} name={editor.name}/>
                )}
              </div>
            }
          </div>
        </div>
      </div>
    )
  }
}

class UninstalledEditor extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      expanded: false,
    }
    this.urls = {
      'atom': 'https://github.com/kiteco/atom-plugin#installing-the-kite-assistant-for-atom',
      'intellij': `https://${Domains.Help}/article/62-managing-editor-plugins#custom-installations`,
      'sublime3': 'https://github.com/kiteco/KiteSublime#installing-the-kite-assistant-for-sublime',
      'vscode': 'https://github.com/kiteco/vscode-plugin#installing-the-kite-assistant-for-visual-studio-code',
      'spyder': `https://${Domains.Help}/article/123-kite-cannot-detect-spyder-installation`,
    }
  }

  expand = () => {
    this.setState({ expanded: true })
  }

  render() {
    const { id, name, icon, refresh } = this.props

    return <div className={`plugins__plugin plugins__plugin--expanded-false`}>
      <div className="plugins__plugin__title">
        {icon && <img className="plugins__plugin-icon" src={icon} alt={id}/>}
        <h3 className="plugins__plugin__title__h3">{name}</h3>
      </div>

      {!this.state.expanded &&
        <span><p>Kite could not automatically detect {name}. If you have {name} installed, <a onClick={this.expand}>fix this</a>.</p></span>}

      {this.state.expanded &&
        <span>
          <p>Kite could not automatically detect {name}.</p>
          <p>Make sure {name} is running and then <a className="plugins__try-again" href="#try-again" onClick={refresh}>try again</a>.</p>
          {this.urls[id] &&
            <p>If this doesn't work, you can manually install this plugin. <a href={this.urls[id]} target="_blank">Learn how</a></p>
          }
        </span>
      }
    </div>
  }
}

class ManualEditor extends React.Component {
  constructor(props) {
    super(props)
    this.descriptions = {
      'jupyterlab': props => <JupyterLab {...props} />,
    }
  }

  render() {
    const { id, name, icon } = this.props

    return (
      <div className="plugins__plugin plugins__plugin--manual">
        <div className="plugins__plugin__title">
          {icon && <img className="plugins__plugin-icon" src={icon} alt={id} />}
          <h3 className="plugins__plugin__title__h3">{name}</h3>
        </div>
        {this.descriptions[id](this.props)}
      </div>
    )
  }
}

class JupyterLab extends React.Component {
  render() {
    return (
      <div>
        <p>
          Kite supports JupyterLab v2.2 and above.
          See our <a href="https://github.com/kiteco/jupyterlab-kite#installing-the-kite-extension-for-jupyterlab" target="_blank">installation guide</a> for more info.
        </p>
      </div>
    )
  }
}

const Plugin = ({
  install,
  uninstall,
  name,
  id,
  editors,
  requires_restart,
  running,
  install_while_running,
  uninstall_while_running,
  separator,
  className,
  loading,
  multiple_install_locations,
  refresh,
  networkDisabled,
}) => {
  const displayExpanded = (multiple_install_locations && editors.length > 1)
                          || editors.some((plugin) => plugin.hasOwnProperty('compatibility'))


  const installed = editors.some(p => p.plugin_installed === true)
  const uninstalled = editors.some(p => p.plugin_installed === false)
  const mustClose = (running &&
    ((installed === true && uninstall_while_running === false) ||
    (uninstalled === true && install_while_running === false)))
  return <div className={`plugins__plugin plugins__plugin--expanded-${displayExpanded} ${className}`}>
    <div className="plugins__plugin__title">
      { getIconForEditor(id) &&
          <img className="plugins__plugin-icon" src={getIconForEditor(id)} alt={id}/>
      }
      <h3 className="plugins__plugin__title__h3">{name}</h3>
      { !displayExpanded &&
          <SingleEditor
            separator={separator}
            install={install}
            uninstall={uninstall}
            loading={loading}
            id={id}
            name={name}
            running={running}
            networkDisabled={networkDisabled}
            install_while_running={install_while_running}
            uninstall_while_running={uninstall_while_running}
            {...editors[0]}
          />
      }
    </div>
    { mustClose &&
        <div className="plugins__warning">
          Please close all running instances of { name } before&nbsp;
          { installed && !uninstalled && "uninstalling" }
          { !installed && uninstalled && "installing" }
          { installed && uninstalled && "installing or uninstalling" }.&nbsp;
          <a className="plugins__try-again" href="#try-again" onClick={refresh}>
            Try again
          </a>
        </div>
    }
    { running && !mustClose && requires_restart &&
        <div className="plugins__note">
          {installed
            ? `You may need to restart ${name} after uninstalling for the changes to take effect.`
            : `You may need to restart ${name} after installing for Kite to activate.`
          }
        </div>
    }
    { id === "spyder" && !mustClose && !installed &&
        <p className="plugins__also">
          Kite will automatically update Spyder's autocompletions settings so that they are optimized to work with Kite.
          <a href={`https://${Domains.Help}/article/90-using-the-spyder-plugin#spyder-setup`} target="_blank" rel="noopener noreferrer">Learn more</a>.
        </p>
    }
    { displayExpanded &&
        <div className="plugins__editor-list">
          { editors.map((editor, i) =>
            <Editor
              key={i}
              separator={separator}
              install={install}
              uninstall={uninstall}
              loading={loading}
              id={id}
              name={name}
              running={running}
              networkDisabled={networkDisabled}
              install_while_running={install_while_running}
              uninstall_while_running={uninstall_while_running}
              {...editor}
            />
          )}
        </div>
    }
    { !installed && networkDisabled &&
        <div className="plugins__warning">
          {name} needs a network connection to install
        </div>
    }
  </div>
}

//Make SingleEditor and Editor class based components - we need each thing to have its own state (if button was clicked)
class SingleEditor extends React.Component {
  constructor(props) {
    super(props)
    this.state = { isClicked: false }
    this.handleClick = this.handleClick.bind(this)
  }

  static getDerivedStateFromProps(props, state) {
    if (!props.loading && state.isClicked) {
      return { isClicked: false }
    }
    return null
  }

  handleClick() {
    const {
      plugin_installed,
      uninstall,
      install,
      id,
      name,
      path,
    } = this.props
    if (!this.state.isClicked) {
      this.setState({ isClicked: true })
      plugin_installed ?
        uninstall({ id, path, name })() :
        install({ id, path, name })()
    }
  }

  render() {
    const {
      loading,
      plugin_installed,
      compatibility,
      running,
      install_while_running,
      uninstall_while_running,
      networkDisabled,
    } = this.props
    const disabled = (running && (
      (install_while_running === false && plugin_installed === false) ||
      (uninstall_while_running === false && plugin_installed === true)
    )) || loading
    return (
      <div className="plugins__editor plugins__single">
        { !compatibility && !(networkDisabled && !plugin_installed) &&
          <LoadingButton
            className={`plugins__install plugins__install--installed-${plugin_installed} ${disabled ? "plugins__install--disabled" : ""}`}
            onClick={this.handleClick}
            isDisabled={loading}
            text={plugin_installed ? "Uninstall" : "Install"}
            isClicked={this.state.isClicked}
          />
        }
      </div>
    )
  }
}

class Editor extends React.Component {
  constructor(props) {
    super(props)
    this.state = { isClicked: false }
    this.handleClick = this.handleClick.bind(this)
  }

  static getDerivedStateFromProps(props, state) {
    if (!props.loading && state.isClicked) {
      return { isClicked: false }
    }
    return null
  }

  handleClick() {
    const {
      plugin_installed,
      uninstall,
      install,
      name,
      id,
      path,
    } = this.props
    if (!this.state.isClicked) {
      this.setState({ isClicked: true })
      plugin_installed ?
        uninstall({ id, path, name })() :
        install({ id, path, name })()
    }
  }

  render() {
    const {
      separator,
      path,
      loading,
      running,
      install_while_running,
      uninstall_while_running,
      plugin_installed,
      compatibility,
      version,
      networkDisabled,
    } = this.props
    const disabled = (running && (
      (install_while_running === false && plugin_installed === false) ||
      (uninstall_while_running === false && plugin_installed === true)
    )) || loading
    return (
      <div className="plugins__editor plugins__editor--multiple">
        <div className={`
          plugins__indicator
          ${plugin_installed ? "plugins__indicator--installed" : ""}
          ${compatibility ? "plugins__indicator--incompatible" : ""}
        `}/>
        <div className="plugins__editor-detail">
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
          { compatibility &&
            <p className="plugins__compatibility-message">
              Incompatible: { compatibility  }
            </p>
          }
        </div>
        { !compatibility && !(networkDisabled && !plugin_installed) &&
          <LoadingButton
            className={`plugins__install plugins__install--installed-${plugin_installed} ${disabled ? "plugins__install--disabled" : ""}`}
            onClick={this.handleClick}
            isDisabled={loading}
            text={plugin_installed ? "Uninstall" : "Install"}
            isClicked={this.state.isClicked}
          />
        }
      </div>
    )
  }
}

const mapStateToProps = (state) => ({
  plugins: state.plugins,
  separator: state.system.path_separator,
  networkConnected: state.system.networkConnected,
})

const mapDispatchToProps = dispatch => ({
  getPlugins: () => dispatch(actions.getPlugins()),
  installPlugin: params => dispatch(actions.installPlugin(params)),
  uninstallPlugin: params => dispatch(actions.uninstallPlugin(params)),
  updateEncounteredEditors: params => dispatch(actions.updateEncounteredEditors(params)),
  updateSystem: () => dispatch(system.getSystemInfo()),
  forceCheckOnline: () => dispatch(system.forceCheckOnline()),
  checkIfOnline: () => dispatch(system.checkIfOnline()),
  notify: params => dispatch(notify(params)),
  dismiss: id => dispatch(dismiss(id)),
})

export default connect(mapStateToProps, mapDispatchToProps)(Plugins)
