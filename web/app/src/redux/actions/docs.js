import { GET } from './fetch'
import { registerUsages, loadUsages } from './usages'
import { normalizeValueReport, normalizeMembersReport } from '../../utils/value-report'
import { membersPath, symbolPath } from '../../utils/urls'
import { normalizeSymbolReport, normalizeLocalCodeExamples } from '../../utils/data-normalization'

export const SET_PAGE_KIND = 'set page kind'
export const setPageKind = (kind) => ({
  type: SET_PAGE_KIND,
  kind,
})

export const SET_EXAMPLE = 'set example'
export const setExample = (language, exampleId) => ({
  type: SET_EXAMPLE,
  language,
  exampleId
})

export const LOAD_DOCS = 'load docs'
export const loadDocs = (identifier, language) => ({
  type: LOAD_DOCS,
  identifier,
  language,
})

export const LOAD_DOCS_FAILED = 'loading docs failed'
export const loadDocsFailed = (error, identifier) => ({
  type: LOAD_DOCS_FAILED,
  error,
  meta: {
    analytics: {
      event: LOAD_DOCS_FAILED,
      props: {
        identifier,
      },
    },
  },
})

export const SHOW_DOCS = 'show docs'
export const showDocs = (data) => {
  return {
    type: SHOW_DOCS,
    data
  }
}

export const newFetchDocs = (identifier, language) => dispatch => {
  if (identifier) {
    dispatch(loadDocs(identifier, language))
    return dispatch(GET({ url: symbolPath(identifier) }))
      .then(({ success, data, response }) => {
        if (success) {
          dispatch(loadUsages(language, identifier))
          dispatch(registerUsages(normalizeLocalCodeExamples(data)))
          return dispatch(showDocs(normalizeSymbolReport(data)))
        } else {
          return dispatch(loadDocsFailed(
            (response && response.status) || 500,
            identifier
          ))
        }
      })
  } else {
    return dispatch(loadDocsFailed(
      'Cannot fetch docs without an identifier'
    ))
  }
}

export const fetchDocs = (identifier) => dispatch => {
  if (identifier) {
    dispatch(loadDocs(identifier))
    // const url = `/api/${language}/value/${identifier}`
    const url = symbolPath(identifier);
    return dispatch(GET({ url }))
      .then(({ success, data, response }) => {
        if (success) {
          return dispatch(showDocs(normalizeValueReport(data)))
        } else {
          return dispatch(loadDocsFailed(
            (response && response.status) || 500,
            identifier,
          ))
        }
      })
  } else {
    return dispatch(loadDocsFailed(
      `Tried to fetch invalid docs: ${identifier}`,
      identifier,
    ))
  }
}

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
  members
})

export const LOAD_MEMBERS = 'load members'
export const loadMembers = (language, identifier) => ({
  type: LOAD_MEMBERS,
  language,
  identifier
})

export const SORT_MEMBERS = 'sort members'
export const sortMembers = criteria => ({
  type: SORT_MEMBERS,
  criteria,
})

export const newFetchMembers = (language, identifier, page = 0, limit = 999) => dispatch => {
  if (language && identifier) {
    dispatch(loadMembers(language, identifier))
    return dispatch(GET({ url: membersPath(identifier, page, limit) }))
      .then(({ success, data, response }) => {
        if (success) {
          return data.members
            ? dispatch(showMembers(data.members, identifier))
            : dispatch(showMembers([], identifier))
        } else {
          dispatch(loadMembersFailed(
            (response && response.status) || 500,
            identifier,
            language
          ))
        }
      })
  } else {
    return dispatch(loadMembersFailed(
      'Cannot fetch members without an identifier'
    ))
  }
}

export const fetchMembers = (language, identifier, page = 0, limit = 999) => dispatch => {
  if (language && identifier) {
    //TODO: REMOVE THIS HACK AFTER API IS FIXED
    dispatch(loadMembers(language, identifier))
    if (!identifier.startsWith(`${language};`)) {
      identifier = `${language};${identifier}`
    }
    const url = membersPath(identifier, page, limit)
    return dispatch(GET({ url }))
      .then(({ success, data, response }) => {
        if (success) {
          return dispatch(showMembers(normalizeMembersReport(data), identifier))
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
