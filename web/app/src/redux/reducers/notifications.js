import * as actions from '../actions/notifications'
import * as fetch from '../actions/fetch'

const defaultState = {
  notifications: [],
  count: 0,
  offline: false,
}

const add = (state, {
  message="",
  error,
  kind="standard",
  timeout=5000
}) =>
  message // if no message, don't add as a notification
  ? { ...state,
      count: state.count + 1,
      notifications: [
        ...state.notifications,
        {
          id: state.count,
          message,
          error,
          kind,
          timeout,
        },
      ],
    }
  : state

const remove = (state, { id }) => ({
  ...state,
  notifications: state.notifications.filter(n => n.id !== id ),
})

const reportNetworkFail = state => (
  state.offline // if already offline, don't show another notification
  ? state
  : {
      ...state,
      ...add(state, {
        kind: "network-error",
        message: "A network error has occured",
        timeout: 0,
      }),
      offline: true,
    }
)

const reportNetworkConnected = state => ({
  ...state,
  offline: false,
  notifications: state.notifications.filter(n => n.kind !== "network-error"),
})

const notifications = (state = defaultState, action) => {
  switch (action.type) {
    case actions.REPORT_NOTIFICATION:
      return add(state, action)
    case actions.HIDE_NOTIFICATION:
      return remove(state, action)
    case fetch.REPORT_NETWORK_FAIL:
      return reportNetworkFail(state)
    case fetch.REPORT_NETWORK_CONNECTED:
      return reportNetworkConnected(state)
    default:
      return state
  }
}

export default notifications
