const APP_EXCEPTION = 'APP_EXCEPTION'

const defaultState = {
  appException: false
}

const appException = (state, action) => {
  return {
    ...state,
    appException: true
  }
}

const errors = (state = defaultState, action) => {
  switch (action.type) {
    case APP_EXCEPTION:
      return appException(state, action)
    default:
      return state
  }
}

export default errors