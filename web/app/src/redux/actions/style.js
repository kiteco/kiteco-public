/**
 * Sometimes, the Breadcrumbs component in a Page header will run over a single line
 * This displaces downwards the grid column of which it is a part, which,
 * without modification, misaligns it to elements in adjacent columns
 * This module of the store intends to coordinate the setting of the height
 * based on readings of the height of the Breadcrumbs component
 */

export const SET_EXTRA_INTRO_HEIGHT = 'set intro height'
export const setExtraIntroHeight = height => dispatch => dispatch({
  type: SET_EXTRA_INTRO_HEIGHT,
  height
})