import { GET } from './fetch'

import { uploadLogsPath, capturePath } from '../utils/urls'

export const UPLOAD_LOGS = 'upload logs'

export const uploadLogs = () => dispatch => {
  return dispatch(GET({url: uploadLogsPath()})).then(({success, data}) => {
    if (success) {
      return dispatch({
        type: UPLOAD_LOGS,
        success,
        data,
      })
    }
  })
}

export const CAPTURE = 'capture'

export const capture = () => dispatch => {
  return dispatch(GET({url: capturePath()})).then(({success, data}) => {
    if (success) {
      return dispatch({
        type: CAPTURE,
        success,
        data,
      })
    }
  })
}
