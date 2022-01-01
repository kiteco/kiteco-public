import { parseRoute } from '../utils/urls'

export const ADD_ROUTE = 'add route'
export const addRoute = route => dispatch => dispatch({
  ...parseRoute(route),
  type: ADD_ROUTE
})

export const ACTION_HANDLED = 'action handled'
export const actionHandled = () => dispatch => dispatch({
  type: ACTION_HANDLED
})