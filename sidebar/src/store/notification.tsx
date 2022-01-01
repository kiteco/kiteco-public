import { Action as BaseAction } from 'redux'
import { ThunkAction } from 'redux-thunk'

// - types

export enum NotifType {
  Default = 'default',
  Autosearch = 'autosearch',
  NoPlugins = 'noplugins',
  Offline = 'offline',
  Plugins = 'plugins',
  PluginsAutoInstalled = 'plugin-auto-installed',
  RemotePluginOffline = 'remote-plugin-offline',
  RunningPluginInstallFailure = 'running-plugin-install-failure',
  SpyderSettings = 'spyder-settings',
}

type ID = any

export interface Notification {
  id: ID
  payload?: any
  component?: NotifType
  timeout?: number
  // if true, only show on the Docs dashboard
  docsOnly: boolean
}

// - actions

enum ActionType {
  Notify = 'notification.Notify',
  Dismiss = 'notification.Dismiss',
  Reset = 'notification.Reset',
}

interface NotifyAction extends BaseAction {
  type: ActionType.Notify
  data: Notification
}

interface DismissAction extends BaseAction {
  type: ActionType.Dismiss
  data: ID
}

interface ResetAction extends BaseAction {
  type: ActionType.Reset
}

type Action = NotifyAction | DismissAction | ResetAction

// - reducer

type State = Notification[];

const init: State = []

export function reducer(state = init, action: Action): State {
  switch (action.type) {
    case ActionType.Notify:
      const newState = state.filter(n => n.id !== action.data.id)
      newState.push(action.data)
      return newState

    case ActionType.Dismiss:
      return state.filter(n => n.id !== action.data)

    case ActionType.Reset:
      return [ ...init ]

    default:
      return state
  }
}

// action creators

export function notify(notif: Notification) : ThunkAction<void, {}, {}, Action> {
  notif.id = notif.id || new Date().getTime()
  return function(dispatch) {
    if (notif.timeout) {
      setTimeout(() => dispatch(dismiss(notif.id)), notif.timeout)
    }
    return dispatch({
      type: ActionType.Notify,
      data: notif,
    })
  }
}

export function dismiss(id: any): ThunkAction<void, {}, {}, Action> {
  return function(dispatch) {
    return dispatch({
      type: ActionType.Dismiss,
      data: id,
    })
  }
}

export function reset(): Action {
  return {
    type: ActionType.Reset,
  }
}
