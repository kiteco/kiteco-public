import * as actions from '../actions/tooltips'

const defaultState = {
  show: false,
  kind: "",
  bounds: {},
  data: {},
}

const showToolTip = (state, action) => {
  return {
    ...state,
    show: true,
    kind: action.kind,
    bounds: action.bounds,
    data: action.data,
  }
}

const hideToolTip = (state, action) => {
  if (action.data === state.data &&
    action.bounds === state.bounds &&
    action.kind === state.kind) {
    return {
      ...state,
      show: false,
    }
  } else {
    return state
  }
}

const tooltips = (state = defaultState, action) => {
  switch (action.type) {
    case actions.SHOW_TOOLTIP:
      return showToolTip(state, action)
    case actions.HIDE_TOOLTIP:
      return hideToolTip(state, action)
    default:
      return state;
  }
}

export default tooltips
