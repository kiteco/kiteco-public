import { GET } from './fetch'
import { usagesPath } from '../../utils/urls'
import { normalizeLocalCodeExamples } from '../../utils/data-normalization'

export const LOAD_USAGES = 'load usages'
export const loadUsages = (language, identifier) => ({
  type: LOAD_USAGES,
  identifier,
  language
})

export const REGISTER_USAGES = 'register usages'
export const registerUsages = (data) => ({
  type: REGISTER_USAGES,
  data
})

export const LOAD_USAGES_FAILED = 'loading usages failed'
export const loadUsagesFailed = (error) => ({
  type: LOAD_USAGES_FAILED,
  error
})

export const fetchUsages = (language, identifier) => dispatch => {
  if(language && identifier) {
    dispatch(loadUsages(language, identifier))
    return dispatch(GET({ url: usagesPath(identifier) }))
      .then(({ success, data, response }) => {
        if(success) {
          return dispatch(registerUsages(normalizeLocalCodeExamples(data, true)))
        } else {
          return dispatch(loadUsagesFailed(
            (response && response.status) || 500,
            identifier,
          ))
        }
      })
  } else {
    return dispatch(loadUsagesFailed('To fetch usages, need a language and identifier'))
  }
}
