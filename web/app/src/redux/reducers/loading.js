import * as actions from '../actions/loading'

const defaultState = {
  isLoading: false,
  isDocsLoading: false,
  docsLoadingCount: 0,
  loadingCount: 0
}

const setDocsLoading = (state, action) => {
  let count = state.docsLoadingCount
  if(action.isLoading) count++
  else if(count > 0) count--
  return {
    ...state,
    isDocsLoading: count > 0,
    docsLoadingCount: count
  }
}

const setLoading = (state, action) => {
  let count = state.loadingCount
  if(action.isLoading) count++
  else if(count > 0) count--
  return {
    ...state,
    isLoading: count > 0,
    loadingCount: count
  }
}

const loading = (state = defaultState, action) => {
  switch(action.type) {
    case actions.SET_DOCS_LOADING:
      return setDocsLoading(state, action)
    case actions.SET_LOADING:
      return setLoading(state, action)
    default:
      return state
  }
}

export default loading
