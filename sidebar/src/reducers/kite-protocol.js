import * as actions from '../actions/kite-protocol'

const defaultState = {
  path: "",
  params: {},
  needsAction: false,
}

const needsAction = ({ path, params }) => {
  return false
}

const addRoute = (state, action) => {
  return {
    ...state,
    path: action.path,
    params: action.params,
    needsAction: needsAction(action)
  }
}

const actionHandled = (state, action) => {
  return {
    ...defaultState
  }
}

const kiteProtocol = (state = defaultState, action) => {
  switch(action.type) {
    case actions.ADD_ROUTE:
      if(state.needsAction) {
        //only one action at a time
        return state
      }
      return addRoute(state, action)
    case actions.ACTION_HANDLED:
      return actionHandled(state, action)
    default:
      return state
  }
}

export default kiteProtocol