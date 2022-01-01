import { GET } from './fetch'
import { normalizeValueReportFromSymbol } from '../utils/value-report'
import { membersPath, symbolReportPath } from '../utils/urls'

export const LOAD_DOCS = 'load docs'
export const loadDocs = (language, identifier) => ({
  type: LOAD_DOCS,
  identifier,
  language,
  meta: {
    analytics: {
      event: LOAD_DOCS,
      props: {
        identifier,
        language,
      },
    },
  },
})

export const LOAD_DOCS_FAILED = 'loading docs failed'
export const loadDocsFailed = (error, language, identifier) => ({
  type: LOAD_DOCS_FAILED,
  error,
  meta: {
    analytics: {
      event: LOAD_DOCS_FAILED,
      props: {
        language,
        identifier,
      },
    },
  },
})

export const SHOW_DOCS = 'show docs'
export const showDocs = (data) => ({
  type: SHOW_DOCS,
  data,
  meta: {
    analytics: {
      event: SHOW_DOCS,
      props: {
        full_name: data.full_name,
      },
    },
  },
})

export const fetchDocs = (language, identifier) => dispatch => {
  if (language && identifier) {
    dispatch(loadDocs(language, identifier))
    //TODO: REMOVE THIS HACK AFTER API IS FIXED
    if (!identifier.startsWith(`${language};`)){
      identifier = `${language};${identifier}`
    }
    const url = symbolReportPath(identifier)
    return dispatch(GET({ url }))
      .then( ({ success, data, response }) => {
        if (success) {
          return dispatch(showDocs(normalizeValueReportFromSymbol(data)))
        } else {
          return dispatch(loadDocsFailed(
            (response && response.status) || 500,
            language,
            identifier
          ))
        }
      })
  } else {
    return dispatch(loadDocsFailed(
      `Tried to fetch invalid docs: ${language}/${identifier}`,
      language,
      identifier
    ))
  }
}

export const LOAD_MEMBERS = 'load members'
export const loadMembers = (language, identifier) => ({
  type: LOAD_MEMBERS,
  language,
  identifier,
  meta: {
    analytics: {
      event: LOAD_MEMBERS,
      props: {
        language,
        identifier,
      },
    },
  },
})

export const LOAD_MEMBERS_FAILED = 'loading members failed'
export const loadMembersFailed = (error, language, identifier) => ({
  type: LOAD_MEMBERS_FAILED,
  error,
  meta: {
    analytics: {
      event: LOAD_MEMBERS_FAILED,
      props: {
        language,
        identifier,
      },
    },
  },
})

export const SHOW_MEMBERS = 'show members'
export const showMembers = (members, identifier) => ({
  type: SHOW_MEMBERS,
  members,
  meta: {
    analytics: {
      event: SHOW_MEMBERS,
      props: {
        total: members.total,
        identifier: identifier,
      },
    },
  },
})

export const fetchMembers = (language, identifier, page = 0, limit = 999) => dispatch => {
  if(language && identifier) {
    //TODO: REMOVE THIS HACK AFTER API IS FIXED
    dispatch(loadMembers(language, identifier))
    if (!identifier.startsWith(`${language};`)){
      identifier = `${language};${identifier}`
    }
    const url = membersPath(identifier, page, limit)
    return dispatch(GET({ url }))
      .then( ({ success, data, response }) => {
        if(success) {
          return dispatch(showMembers(data, identifier))
        } else {
          return dispatch(loadMembersFailed(
            (response && response.status) || 500,
            language,
            identifier
          ))
        }
      })
  } else {
    return dispatch(loadMembersFailed(
      `Tried to fetch invalid members: ${language}/${identifier}`,
      language,
      identifier
    ))
  }
}
