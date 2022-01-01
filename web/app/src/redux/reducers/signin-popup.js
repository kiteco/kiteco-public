import * as actions from '../actions/signin-popup'

const defaultState = {
  visible: false
}

const toggleSigninPopup = (state, action) => {
  return {
    visible: action.show
  }
}

const signinPopup = (state = defaultState, action) => {
  switch(action.type) {
    case actions.TOGGLE_SIGNIN_POPUP:
      return toggleSigninPopup(state, action)
    default:
      return state
  }
}

export default signinPopup
