import * as actions from "../actions/answers";

export const LOADING = "loading";
const defaultState = {
  status: LOADING,
  slug: null,
  content: {},
  error: null
};

const loadPost = (state, action) => {
  return {
    ...state,
    status: LOADING,
    slug: action.slug
  };
};

export const SUCCESS = "success";
const loadPostSuccess = (state, action) => {
  return {
    ...state,
    status: SUCCESS,
    content: { ...state.content, ...action.content }
  };
};

export const FAILED = "failed";
const loadPostFailed = (state, action) => {
  return {
    ...state,
    status: FAILED,
    error: action.error
  };
};

const answers = (state = defaultState, action) => {
  switch (action.type) {
    case actions.LOAD_POST:
      return loadPost(state, action);
    case actions.LOAD_POST_SUCCESS:
      return loadPostSuccess(state, action);
    case actions.LOAD_POST_FAILED:
      return loadPostFailed(state, action);
    default:
      return state;
  }
};

export default answers;
