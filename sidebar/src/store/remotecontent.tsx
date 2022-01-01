import { Action as BaseAction } from 'redux'
import { ThunkAction } from 'redux-thunk'
import { GET, POST } from '../actions/fetch'
import {
  fetchRemoteContentPath,
  remoteContentPath,
} from '../utils/urls'

// types

export interface RemoteContentItem {
  content: string,
  link: string,
}

const emptyRemoteContentItem: RemoteContentItem = {
  content: "",
  link: "",
}

// actions

enum ActionType {
    SetContent = 'remotecontent.SetContent',
}

interface SetContent extends BaseAction {
    type: ActionType.SetContent,
    data: RemoteContentState
}

type Action = SetContent

// reducer

export interface RemoteContentState {
  docs_dashboard_paragraph: RemoteContentItem
  dashboard_header: RemoteContentItem
}

let init: RemoteContentState = {
  docs_dashboard_paragraph: emptyRemoteContentItem,
  dashboard_header: emptyRemoteContentItem,
}

export function reducer (state: RemoteContentState = init, action: Action): RemoteContentState {
  switch (action.type) {
    case ActionType.SetContent:
      const newState = {
        ...state,
        docs_dashboard_paragraph: action.data.docs_dashboard_paragraph,
        dashboard_header: action.data.dashboard_header,
      }
      return newState
    default:
      return state
  }
}

// action creators

export function getRemoteContent(): ThunkAction<Promise<void>, any, {}, Action> {
  return async (dispatch): Promise<void> => {
    const { success, data } = await dispatch(GET({ url: remoteContentPath() }))
    dispatch({
      type: ActionType.SetContent,
      success,
      data,
    })
  }
}

export function fetchRemoteContent(): ThunkAction<Promise<void>, any, {}, Action> {
  return async (dispatch): Promise<void> => {
    return await dispatch(POST({ url: fetchRemoteContentPath() }))
  }
}
