// these editors require an internet connection to install since they are stored with their editor plugin managers/stores i.e. Atom Package Manager
export const REMOTE_INSTALL_EDITORS = [{ id: 'atom', name: 'Atom' }, { id: 'vscode', name: 'Visual Studio Code' }]

export const getInstalledPlugins = (plugins) =>
  plugins.filter(plugin =>
    plugin.editors && plugin.editors.length
  ).filter(editor =>
    editor.editors.some((plugin) => plugin.plugin_installed === true)
  )

export const isAnyEditorInstalled = (plugins) =>
  plugins.some(plugin => plugin.editors && plugin.editors.length)

export const runningInstallDisable = ({ running, install_while_running }) => running && install_while_running === false

export const isFullyInstalled = (editor) => editor.plugin_installed === true && !editor.compatibility

export const getDefaultPlugins = ({ plugins, system }) => {
  let defaultPlugins = {}
  const installed_editors = (plugins || []).filter(plugin => {
    if(!system.networkConnected) {
      const remote = REMOTE_INSTALL_EDITORS.find(ed => ed.id === plugin.id)
      if(remote) {
        return false
      }
    }
    return plugin.editors
      && plugin.editors.length
      && plugin.editors.some(editor => !editor.compatibility)
      && plugin.editors.some(editor => !isFullyInstalled(editor))
  })
  installed_editors.forEach((plugin) => {
    const networkDisabled = !system.networkConnected && system.haveCheckedNetworkConnection && REMOTE_INSTALL_EDITORS.includes(plugin.id)
    const displayExpanded = (plugin.multiple_install_locations && plugin.editors.length > 1)
                            || plugin.editors.some((editor) => editor.hasOwnProperty('compatibility'))
    if (displayExpanded) {
      if (!runningInstallDisable(plugin)) {
        plugin.editors.map((editor) => {
          if (!editor.compatibility) {
            defaultPlugins[editor.path] = networkDisabled ? null : plugin.id
          }
        })
      }
    } else {
      if (!runningInstallDisable(plugin)) {
        if (!plugin.editors[0].compatibility) {
          defaultPlugins[plugin.editors[0].path] = networkDisabled ? null : plugin.id
        }
      }
    }
  })
  return defaultPlugins
}

// get plugins that cannot be installed because their editors are running right now
export const getRunningInstallDisablePlugins = ({ plugins, system }) => {
  let defaultPlugins = {}
  const installed_editors = (plugins || []).filter(plugin => {
    if(!system.networkConnected) {
      const remote = REMOTE_INSTALL_EDITORS.find(ed => ed.id === plugin.id)
      if(remote) {
        return false
      }
    }
    return plugin.editors && plugin.editors.length && plugin.editors.some((editor) => !editor.compatibility)
  })
  installed_editors.forEach((plugin) => {
    const networkDisabled = !system.networkConnected && system.haveCheckedNetworkConnection && REMOTE_INSTALL_EDITORS.includes(plugin.id)
    const displayExpanded = (plugin.multiple_install_locations && plugin.editors.length > 1)
                            || plugin.editors.some((editor) => editor.hasOwnProperty('compatibility'))
    if (displayExpanded) {
      if (runningInstallDisable(plugin)) {
        plugin.editors.map((editor) => {
          if (!editor.compatibility) {
            defaultPlugins[editor.path] = networkDisabled ? null : plugin
          }
        })
      }
    } else {
      if (runningInstallDisable(plugin)) {
        if (!plugin.editors[0].compatibility) {
          defaultPlugins[plugin.editors[0].path] = networkDisabled ? null : plugin
        }
      }
    }
  })
  return defaultPlugins
}

export const installMultiple = async ({
  forceCheckOnline,
  toInstall,
  install,
}) => {
  const installed = []
  const { success, isOnline } = await forceCheckOnline()
  const shouldFilter = !success || !isOnline

  let filtered = []
  let keys = shouldFilter
    ? Object.keys(toInstall).filter(path => {
      if(toInstall[path]) {
        const remote = REMOTE_INSTALL_EDITORS.find(ed => toInstall[path] === ed.id)
        // then push to filtered so we can display alert on next slide
        if(remote) {
          filtered.push(remote)
          return false
        }
        return true
      }
      return false
    })
    : Object.keys(toInstall)

  let results = await Promise.all(keys.map((path) => {
    if (toInstall[path]) {
      installed.push(toInstall[path])
      return install({
        path,
        id: toInstall[path]
      })
    }
    return null
  }))

  results = results.filter(result => result)
  const successes = results.map(result => result.success)
  const errors = results.map(result => result.error).filter(e => e)
  return {
    results,
    installed,
    filtered,
    successes,
    errors,
  }
}
