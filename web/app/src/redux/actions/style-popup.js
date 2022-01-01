export const TOGGLE_STYLE_POPUP = 'toggle style popup'
export const toggleStylePopup = show => dispatch => () => {
  return dispatch({
    type: TOGGLE_STYLE_POPUP,
    data: show
  })
}

export const CHANGE_THEME = 'change theme'
export const changeTheme = theme => dispatch => {
  return dispatch({
    type: CHANGE_THEME,
    data: theme
  })
}

export const CHANGE_FONT = 'change font'
export const changeFont = font => dispatch => {
  return dispatch({
    type: CHANGE_FONT,
    data: font
  })
}

export const CHANGE_ZERO = 'change zero'
export const changeZero = zero => dispatch => {
  return dispatch({
    type: CHANGE_ZERO,
    data: zero
  })
}