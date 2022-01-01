import * as actions from '../actions/style'

/**
 * Sometimes, the Breadcrumbs component in a Page header will run over a single line
 * This displaces downwards the grid column of which it is a part, which,
 * without modification, misaligns it to elements in adjacent columns
 * This module of the store intends to coordinate the setting of the height
 * based on readings of the height of the Breadcrumbs component
 */

const defaultState = {
  extraIntroHeight: 0,
  extraIntroClass: ""
}

const getExtraHeightClass = (height) => {
  if(height === 0) return ""
  if(height <= 18) return "extra-1" //var(--font-size-small) - 18px
  if(height <= 36) return "extra-2"
  if(height <= 54) return "extra-3"
  return "extra-4"
}

const setExtraIntroHeight = (state, action) => {
  return {
    ...state,
    extraIntroHeight: action.height,
    extraIntroClass: getExtraHeightClass(action.height)
  }
}

const styles = (state = defaultState, action) => {
  switch(action.type) {
    case actions.SET_EXTRA_INTRO_HEIGHT:
      return setExtraIntroHeight(state, action)
    default:
      return state
  }
}

export default styles
