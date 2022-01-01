import { POST } from './fetch'

import {
  createJson,
} from "../../utils/fetch"

import { commentPath } from '../../utils/urls'

export const ADD_COMMENT = 'add comment'
export const addComment = comment => dispatch => {
  dispatch(POST({
    url: commentPath,
    options: createJson(comment),
  }))
  return dispatch({
    type: ADD_COMMENT,
    comment,
  })
}


export const DELETE_COMMENT = 'delete comment'
export const deleteComment = uuid => ({
  type: DELETE_COMMENT,
  uuid,
})
