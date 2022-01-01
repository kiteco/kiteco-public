/**
 * NB: This reducer has a pretty significant side effect - 
 *     namely the calling of a registered loadFn for a given
 *     3rd party (like segment) script. The motivation for
 *     this was to, in conjunction with the ThirdPartyScripts
 *     component, enable the loading of the scripts in the event
 *     of a network connection (since, in a Kite Local world, we
 *     cannot anticipate being online on startup)
 */

import * as actions from '../actions/scripts'
import { SCRIPTS } from '../utils/scripts'

const scriptDict = SCRIPTS.reduce((dict, script) => {
  dict[script.name] = {
    loaded: false,
    loadFn: script.loadFn
  }
  return dict
}, {})

const defaultState = { scriptDict }

/**
 * NB: the state of "loaded" does not necessarily correspond to whether
 *     the script actually loaded. A more robust solution would hook this
 *     notion to the markers of completion in the script itself.
 *     However, this approximation should be sufficient for our purposes
 */
const loadScript = (state, action) => {
  let loaded = false
  if(state.scriptDict[action.name]) {
    loaded = state.scriptDict[action.name].loadFn()
  }
  if(loaded) {
    return {
      ...state,
      scriptDict: {
        ...state.scriptDict,
        [action.name]: {
          loaded,
          loadFn: state.scriptDict[action.name].loadFn,
        }
      }
    }
  }
  return state
}

const scripts = (state = defaultState, action) => {
  switch(action.type) {
    case actions.LOAD_SCRIPT:
      return loadScript(state, action)
    default:
      return state
  }
}

export default scripts