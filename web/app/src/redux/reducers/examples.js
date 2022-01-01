import * as actions from '../actions/examples'
import { LOG_OUT } from '../actions/account'

const defaultState = {
  status: "loading",
  data: {},
  language: null,
  identifier: null,
  error: null,
}

const registerExamples = (state, action) => {
  return {
    ...state,
    status: "success",
    data: { ...state.data, ...action.data },
  }
}

const loadExamplesFailed = (state, action) => {
  return {
    ...state,
    status: "failed",
    error: action.error,
  }
}

const loadExamples = (state, action) => {
  return {
    ...state,
    status: "loading",
    language: action.language,
    identifiers: action.identifiers,
  }
}

const logOut = () => ({ ...defaultState })

const examples = (state = defaultState, action) => {
  switch (action.type) {
    case actions.LOAD_EXAMPLES:
      return loadExamples(state, action);
    case actions.REGISTER_EXAMPLES:
      return registerExamples(state, action);
    case actions.LOAD_EXAMPLES_FAILED:
      return loadExamplesFailed(state, action);
    case LOG_OUT:
      return logOut();
    default:
      return state;
  }
}

export default examples
