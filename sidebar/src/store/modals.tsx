import { Action as BaseAction } from 'redux'
import { ThunkAction } from 'redux-thunk'
import { Goal, Goals } from '../components/LoginModal'

/* Modals */

export type ModalName = ModalNames.LoginModal

export enum ModalNames {
  LoginModal = "LoginModal"
}

export interface ILoginModalData {
  onSuccess: () => any,
  goal: Goal,
}

/* Reducer */

interface State {
  active: ModalName | null,
  loginModalData: ILoginModalData,
}

const initLoginData: ILoginModalData = {
  onSuccess: () => {},
  goal: Goals.init,
}

const initialState: State = {
  active: null,
  loginModalData: initLoginData,
}

export function reducer(state = initialState, action: Action): State {
  switch (action.type) {
    case ActionType.SetActive:
      return { ...state, active: action.active }
    case ActionType.SetLoginModalData:
      return {
        ...state,
        loginModalData: {
          onSuccess: action.onSuccess,
          goal: action.goal,
        },
      }
    default:
      return state
  }
}

/* Actions */

enum ActionType {
  SetActive = "modals.SetActive",
  SetLoginModalData = "modals.SetLoginModalData",
}

interface SetActive extends BaseAction {
  type: ActionType.SetActive
  active: ModalName | null
}

interface SetLoginModalData extends BaseAction {
  type: ActionType.SetLoginModalData
  onSuccess: () => any
  goal: Goal
}

type Action = SetActive | SetLoginModalData

function setActive(active: ModalName | null): ThunkAction<void, any, {}, Action> {
  return function(dispatch) {
    dispatch({ type: ActionType.SetActive, active })
  }
}

export function deactivate(): ThunkAction<void, any, {}, Action> {
  return function(dispatch) {
    // Reset data, since onSuccess may refer to a component method, and
    // component may unmount after deactivating the modal.
    dispatch(setLoginModalData(initLoginData.onSuccess, initLoginData.goal))
    dispatch(setActive(null))
  }
}

function setLoginModalData(onSuccess: () => any, goal: Goal): ThunkAction<void, any, {}, Action> {
  return function(dispatch) {
    dispatch({ type: ActionType.SetLoginModalData, onSuccess, goal })
  }
}

export function requireLogin(data : ILoginModalData): ThunkAction<void, any, {}, Action> {
  return async function(dispatch, getState) {
    if (getState().account.status !== "logged-in") {
      dispatch(setLoginModalData(data.onSuccess, data.goal))
      dispatch(setActive(ModalNames.LoginModal))
    } else {
      data.onSuccess()
    }
  }
}
