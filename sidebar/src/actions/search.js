import { metrics } from '../utils/metrics'
import { setAutosearchDefaultOff, setAutosearchDefaultOn } from './settings'

export const ENABLE_AUTOSEARCH = "enable autosearch"
export const enableAutosearch = () => dispatch => {
  metrics.incrementCounter('sidebar_autosearch_enabled')
  dispatch(setAutosearchDefaultOn())
  return dispatch({
    type: ENABLE_AUTOSEARCH,
  })
}

export const DISABLE_AUTOSEARCH = "disable autosearch"
export const disableAutosearch = () => dispatch => {
  dispatch(setAutosearchDefaultOff())
  return dispatch({
    type: DISABLE_AUTOSEARCH,
  })
}

export const AUTOSEARCH_EVENT = "autosearch event"
export const autosearchEvent = ({ id }) => dispatch => {
  return dispatch({
    type: AUTOSEARCH_EVENT,
    data: id,
  })
}

