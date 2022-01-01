import {
  GET,
  POST,
  DELETE,
} from './fetch'

import { createQueryURL, createJson } from '../utils/fetch'
import { pluginsPath, encounteredEditorsPath, autoInstalledPluginsPath, mostRecentEditor } from '../utils/urls'

export const UPDATE_PLUGINS = "update plugins"
export const getPlugins = () => dispatch =>
  dispatch(POST({ url: pluginsPath() }))
    .then(({ success, data }) => {
      if ( success ) {
        return dispatch({
          type: UPDATE_PLUGINS,
          success,
          data,
        })
      }
    })

export const installPlugin = ({ id, path }) => dispatch =>
  dispatch(POST({
    url: createQueryURL(pluginsPath(id), { path }),
  }))

export const uninstallPlugin = ({ id, path }) => dispatch =>
  dispatch(DELETE({
    url: createQueryURL(pluginsPath(id), { path }),
  }))

export const updateEncounteredEditors = editors => dispatch =>
  dispatch(POST({
    url: encounteredEditorsPath(),
    options: createJson(editors)
  }))


export const UPDATE_AUTO_INSTALLED_PLUGINS = "update auto installed plugins"
export const getAutoInstalledPlugins = () => dispatch =>
dispatch(GET({ url: autoInstalledPluginsPath() }))
.then(({ success, data }) => {
  if ( success ) {
    return dispatch({
      type: UPDATE_AUTO_INSTALLED_PLUGINS,
      success,
      data,
    })
  }
})
export const resetAutoInstalledPlugins = () => dispatch =>
dispatch(DELETE({ url: autoInstalledPluginsPath() }))
.then(({ success, data }) => {
  if ( success ) {
    return dispatch({
      type: UPDATE_AUTO_INSTALLED_PLUGINS,
      success,
      data,
    })
  }
})

export const GET_MRU_EDITOR = "update mru plugins"
export const getMRUEditor = () => dispatch =>
  dispatch(GET({ url: mostRecentEditor() }))
    .then(({ success, data }) => {
      if ( success ) {
        return dispatch({
          type: GET_MRU_EDITOR,
          success,
          data,
        })
      }
    })

