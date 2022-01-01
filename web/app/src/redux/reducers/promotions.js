import uuid from 'uuid/v4'

import * as docs from '../actions/docs'
import * as actions from '../actions/promotions'
import * as comments from '../actions/comments'

/*
 * This store is persisted in localStorage
 * so these defaults may not be used on first load
 */
const defaultState = {
  docsViews: 0,
  docsPromoGIFcollapsed: false,
  anonymousID: uuid(),
  email: "",
  utm: {},
}

const incrementDocsViews = state => ({
  ...state,
  docsViews: state.docsViews + 1,
})

const saveUTM = (state, action) => {
  const newUTM = { ...action.props }
  Object.keys(newUTM).forEach(key => {
    if (newUTM[key] === undefined || newUTM[key] === null) {
      delete newUTM[key]
    }
  })
  return {
    ...state,
    utm: {
      ...state.utm,
      ...newUTM,
    },
  }
}

const resetDocsViews = state => ({
  ...state,
  docsViews: 0,
})

const resetAnonymousID = state => ({
  ...state,
  anonymousID: uuid(),
})

const updateEmail = (state, action) => ({
  ...state,
  email: action.comment.email,
})

const toggleGIFCollapse = (state, action) => ({
  ...state,
  docsPromoGIFcollapsed: !state.docsPromoGIFcollapsed,
})

const promotions = (state = defaultState, action) => {
  switch (action.type) {
    case docs.SHOW_DOCS:
      return incrementDocsViews(state)
    case actions.RESET_DOCS_VIEWS:
      return resetDocsViews(state)
    case actions.RESET_ANONYMOUS_ID:
      return resetAnonymousID(state)
    case actions.SAVE_UTM:
      return saveUTM(state, action)
    case actions.TOGGLE_GIF_COLLAPSE:
      return toggleGIFCollapse(state, action)
    case comments.ADD_COMMENT:
      return updateEmail(state, action)
    default:
      return state
  }
}

export default promotions
