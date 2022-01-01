import * as actions from '../actions/active-file'

const defaultState = {
  filename: "",
}

const activeFileEvent = (state, action) => {
  return {
    ...state,
    filename: action.data,
  }
}

const activeFile = (state = defaultState, action) => {
  switch (action.type) {
    case actions.ACTIVE_FILE_EVENT:
      return activeFileEvent(state, action)
    default:
      return state;
  }
}

export default activeFile
