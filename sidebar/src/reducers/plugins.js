import * as actions from '../actions/plugins'

const defaultState = null

const updatePlugins = (state, action) => {
  const { plugins } = action.data
  plugins.sort((a, b) => (a.id > b.id) ? 1 : -1)
  return plugins
}

const updateAutoInstalledPlugins = (state, action) => {
  const pluginIDs = action.data
  pluginIDs.sort((a, b) => (a.id > b.id) ? 1 : -1)
  return pluginIDs
}

const defaultMruState = {
  mruEditor: "",
}

// Use an object that can be extended for any future functionality
const pluginsInfo = (state = defaultMruState, action) => {
  if (action.type !== actions.GET_MRU_EDITOR)
    return state

  return {
    ...state,
    mruEditor: (action.data && action.data.editor) || "",
  }
}

const plugins = (state = defaultState, action) => {
  switch(action.type) {
    case actions.UPDATE_PLUGINS:
      return updatePlugins(state, action)
    case actions.UPDATE_AUTO_INSTALLED_PLUGINS:
      return updateAutoInstalledPlugins(state, action)
    default:
      return state
  }
}

export { plugins, pluginsInfo }
