export const LOAD_CACHED_COMPLETIONS = 'load cached completions'

const defaultState = {
  cachedCompletions: null
}

const loadCachedCompletions = (state, action) => {
  return {
    cachedCompletions: action.completions
  }
}

const cachedCompletions = (state = defaultState, action) => {
  switch(action.type) {
    case LOAD_CACHED_COMPLETIONS:
      return loadCachedCompletions(state, action)
    default:
      return state
  }
}

export default cachedCompletions