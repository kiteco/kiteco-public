export const TOGGLE_SIGNIN_POPUP = 'toggle signin popup'
export const toggleSigninPopup = show => dispatch => () => dispatch({
  type: TOGGLE_SIGNIN_POPUP,
  show
})