import { GET } from "./fetch";
import { answersPath } from "../../utils/urls";

export const LOAD_POST = "load post";
export const loadPost = slug => ({
  type: LOAD_POST,
  slug
});

export const LOAD_POST_SUCCESS = "load post success";
export const loadPostSuccess = content => ({
  type: LOAD_POST_SUCCESS,
  content
});

export const LOAD_POST_FAILED = "load post failed";
export const loadPostFailed = error => ({
  type: LOAD_POST_FAILED,
  error
});

export const fetchPost = slug => dispatch => {
  dispatch(loadPost(slug));
  return dispatch(GET({ url: answersPath(slug) })).then(
    ({ success, data, response }) => {
      if (success) {
        return dispatch(loadPostSuccess(data));
      } else {
        return dispatch(loadPostFailed((response && response.status) || 500));
      }
    }
  );
};
