import { GET, batchFetchSafe } from '../actions/fetch'
import { curationExamplesPath } from '../utils/urls'

export const LOAD_EXAMPLES= 'load examples'
export const loadExamples = (language, identifiers, data) => ({
  type: LOAD_EXAMPLES,
  identifiers,
  language,
  data,
  meta: {
    analytics: {
      event: LOAD_EXAMPLES,
      props: {
        identifiers,
        language,
      },
    },
  },
})

export const REGISTER_EXAMPLES= 'register examples'
export const registerExamples = (language, identifiers, data) => ({
  type: REGISTER_EXAMPLES,
  identifiers,
  language,
  data,
  meta: {
    analytics: {
      event: REGISTER_EXAMPLES,
      props: {
        identifiers,
        language,
      },
    },
  },
})

export const LOAD_EXAMPLES_FAILED = 'loading examples failed'
export const loadExamplesFailed = (error, language, identifiers) => ({
  type: LOAD_EXAMPLES_FAILED,
  error,
  meta: {
    analytics: {
      event: LOAD_EXAMPLES_FAILED,
      props: {
        language,
        identifiers,
        error,
      },
    },
  },
})

export const fetchExamples = (language, identifiers) => (dispatch, getStore) => {
  if (language && identifiers && identifiers.length) {
    const { examples: { data } } = getStore();

    const identifiersToLoad = identifiers.filter(id => !data[id]);

    if (identifiersToLoad.length > 0) {
      dispatch(loadExamples(language, identifiersToLoad))

      return dispatch(batchFetchSafe(identifiersToLoad.map(id =>
        GET({ url: curationExamplesPath(language, id) })
      ))).then(({ success, data, response }) => {
        if (success) {
          const map = data.reduce((m, e) => {
            m[e.id] = e;
            return m;
          }, {})

          return dispatch(registerExamples(language, identifiersToLoad, map))
        } else {
          return dispatch(loadExamplesFailed(
            (response && response.status) || 500,
            language,
            identifiersToLoad,
          ))
        }
      })
    } else {
      return dispatch(registerExamples(language, [], {}));
    }
  } else {
    return dispatch(loadExamplesFailed(
      `Tried to fetch invalid example: ${language}/${identifiers}`,
      language,
      identifiers,
    ))
  }
}
