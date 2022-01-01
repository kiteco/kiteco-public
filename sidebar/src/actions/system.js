import {
  GET, POST
} from './fetch'

import {
  systemInfoPath,
  versionPath,
  checkOnlinePath,
  onlinePath,
  kitedReadyPath,
  spyderOptimalSettingsPath,
} from '../utils/urls'

export const GET_KITED_READY = 'get kited ready'
export const getKitedReady = () => dispatch =>
  dispatch(GET({ url: kitedReadyPath() }))
    .then(({ success, data }) => {
      if(success) {
        return dispatch({
          type: GET_KITED_READY,
          success,
          kitedReady: data.online
        })
      }
    })

export const FORCE_CHECK_ONLINE = 'force check online'
export const forceCheckOnline = () => dispatch =>
    dispatch(GET({ url: checkOnlinePath() }))
      .then(({ success, data }) => {
        if(success) {
          return dispatch({
            type: CHECK_IF_ONLINE,
            success,
            isOnline: data.online
          })
        }
      })

export const CHECK_IF_ONLINE = 'check online'
export const checkIfOnline = () => dispatch =>
    dispatch(GET({ url: onlinePath() }))
      .then(({ success, data }) => {
        if(success) {
          return dispatch({
            type: CHECK_IF_ONLINE,
            success,
            isOnline: data.online
          })
        } else {
          //what does not success imply here?
          //at the very least, it implies that 
          //there was a response from kited, as
          //network error would have been caught
        }
      })

export const UPDATE_SYSTEM_INFO = 'update system info'
export const getSystemInfo = () => dispatch =>
  dispatch(GET({ url: systemInfoPath() } ))
    .then(({ success, data }) =>
      success && dispatch({
        type: UPDATE_SYSTEM_INFO,
        success,
        data,
      })
    )

export const GET_VERSION = "get client version"
export const getVersion = () => dispatch =>
  dispatch(GET({ url: versionPath() }))
    .then(({ success, data, error }) => {
      if (success) {
        return dispatch({
          type: GET_VERSION,
          data: data.version,
          success,
        })
      }
      return { success, error }
    })


export const GET_HAS_SPYDER_OPTIMAL_SETTINGS = 'spyder optimized settings'
export const getSpyderOptimalSettings = () => dispatch =>
  dispatch(GET({ url: spyderOptimalSettingsPath() }))
  .then(({ success, data }) => {
    return dispatch({
      type: GET_HAS_SPYDER_OPTIMAL_SETTINGS,
      data: data,
      success,
    })
  })

export const setSpyderOptimalSettings = () => dispatch => {
  return dispatch(POST({
    url: spyderOptimalSettingsPath(),
  })).then(() => dispatch({
    type: GET_HAS_SPYDER_OPTIMAL_SETTINGS,
    success: true,
    data: true,
  }))
}

export const SET_HAS_SEEN_SPYDER_NOTIFICATION = 'set has seen spyder settings notification'
export const setHasSeenSpyderNotification = () => dispatch =>
  dispatch({
    type: SET_HAS_SEEN_SPYDER_NOTIFICATION,
    data: true
  })