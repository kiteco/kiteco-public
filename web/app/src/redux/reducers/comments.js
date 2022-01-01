import * as actions from '../actions/comments'

/*
 * This store is persisted in localStorage
 * so these defaults may not be used on first load
 */
const defaultState = {}

const addComment = (state, action) => ({
  ...state,
  [action.comment.uuid]: action.comment,
})

const deleteComment = (state, action) => {
  const newState = { ...state }
  delete newState[action.uuid]
  return newState
}

const comments = (state = {...defaultState}, action) => {
  switch (action.type) {
    case actions.ADD_COMMENT:
      return addComment(state, action)
    case actions.DELETE_COMMENT:
      return deleteComment(state, action)
    default:
      return state
  }
}

export default comments
