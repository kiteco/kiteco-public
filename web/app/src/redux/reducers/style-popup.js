import * as actions from '../actions/style-popup'
import constants from '../../utils/theme-constants'

const { FONTS, THEMES, ZEROES } = constants

const defaultState = {
  visible: false,
  font: FONTS[0],
  zero: ZEROES.SLASHED,
  theme: THEMES.LIGHT
}

const toggleStylePopup = (state, action) => {
  return {
    ...state,
    visible: action.data ? true : false
  }
}

const changeFont = (state, action) => {
  const font = FONTS.find(font => font.value === action.data)
  return {
    ...state,
    font: font,
    zero: ZEROES[Object.keys(ZEROES).find(key => {
      return font.zeroes[ZEROES[key]]
    })]
  }
}

const changeTheme = (state, action) => {
  return {
    ...state,
    theme: action.data
  }
}

const changeZero = (state, action) => {
  return {
    ...state,
    zero: action.data
  }
}

const stylePopup = (state = defaultState, action) => {
  switch(action.type) {
    case actions.TOGGLE_STYLE_POPUP:
      return toggleStylePopup(state, action)
    case actions.CHANGE_FONT:
      return changeFont(state, action)
    case actions.CHANGE_THEME:
      return changeTheme(state, action)
    case actions.CHANGE_ZERO:
      return changeZero(state, action)
    default:
      return state
  }
}

export default stylePopup
