import { GET, batchFetchSafe } from './fetch'
import { examplesPath, multipleExamplesPath } from '../../utils/urls'

export const LOAD_EXAMPLES = 'load examples'
export const loadExamples = (language, identifiers, data) => ({
  type: LOAD_EXAMPLES,
  identifiers,
  language,
  data
})

export const REGISTER_EXAMPLES = 'register examples'
export const registerExamples = (language, identifiers, data) => ({
  type: REGISTER_EXAMPLES,
  identifiers,
  language,
  data
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
        GET({ url: examplesPath(language, id) })
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

export const newFetchExamples = (language, identifiers) => (dispatch, getStore) => {
  if (language && identifiers && identifiers.length) {
    const { examples: { data } } = getStore();

    const identifiersToLoad = identifiers.filter(id => !data[id]);

    if (identifiersToLoad.length > 0) {
      dispatch(loadExamples(language, identifiersToLoad))

      return dispatch(GET({ url: multipleExamplesPath(language, identifiersToLoad) })).then(({ success, data, response }) => {
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
      return Promise.resolve(dispatch(registerExamples(language, [], {})));
    }
  } else {
    return dispatch(loadExamplesFailed(
      `Tried to fetch invalid examples: ${language}/${identifiers}`,
      language,
      identifiers,
    ))
  }
}

export const fetchExample = (language, id) => (dispatch, getStore) => {
  if (language && id) {
    const { examples: { data } } = getStore()
    if (!data[id]) {
      //then still need to load
      dispatch(loadExamples(language, [id]))
      return dispatch(GET({ url: examplesPath(language, id) }))
        .then(({ success, data, response }) => {
          if (success) {
            const map = {
              [data.id]: data
            }
            return dispatch(registerExamples(language, [id], map))
          } else {
            return dispatch(loadExamplesFailed(
              (response && response.status) || 500,
              language,
              [id],
            ))
          }
        })
    } else {
      //so caller can use data in promise resolution
      return Promise.resolve({ data })
    }
  } else {
    return dispatch(
      `Loading an example failed: we need both language and id to be provided`,
      language,
      id
    )
  }
}
