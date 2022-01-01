import { Action as BaseAction } from 'redux'
import { ThunkAction } from 'redux-thunk'

import { GET, DELETE } from '../actions/fetch'
import { licenseInfoPath, subscriptionsPath } from '../../utils/urls'

// Types

export enum Product {
  Hide = "hide",
  Pro = "pro",
  Free = "free",
}

export enum Plan {
  Free = "free",
  Education = "pro_education",
  Monthly = "pro_monthly",
  Yearly = "pro_yearly",
  Server = "pro_server",
  Temp = "pro_temp",
  Trial = "pro_trial",
  TrialMonthly = "pro_trial_monthly",
  TrialYearly = "pro_trial_yearly",
}

export interface ProLicenseInfo {
  product: Product.Pro
  plan: Plan
  days_remaining: number
}

interface FreeLicenseInfo {
  product: Product.Free
  trial_available: boolean
}

export type LicenseInfo = ProLicenseInfo | FreeLicenseInfo

// Actions

enum ActionType {
  SetLicenseInfo = 'license.SetLicenseInfo',
  SetIsProSubscriber = 'license.SetIsProSubscriber',
}

interface SetLicenseInfoAction extends BaseAction {
  type: ActionType.SetLicenseInfo
  data: LicenseInfo
}

interface SetIsProSubscriber extends BaseAction {
  type: ActionType.SetIsProSubscriber
  data: boolean
}

type Action = SetLicenseInfoAction | SetIsProSubscriber

// reducer

// TODO(naman) this is hack, eventually when we port the reducer combinator to TS, we can remove it.
interface State {
  licenseInfo?: LicenseInfo;
  isProSubscriber?: boolean;
}

let init: State = {}

export function reducer(state: State = init, action: Action): State {
  switch (action.type) {
    case ActionType.SetLicenseInfo:
      return { ...state, licenseInfo: action.data }
    case ActionType.SetIsProSubscriber:
      return { ...state, isProSubscriber: action.data }
    default:
      return state
  }
}

// action creators

export function fetchLicenseInfo(): ThunkAction<Promise<LicenseInfo>, any, {}, Action> {
  return function (dispatch): Promise<LicenseInfo> {
    return dispatch(GET({ url: licenseInfoPath }))
      .then((result: any) => {
        if (result.success && result.data) {
          let data: LicenseInfo = result.data
          dispatch({
            type: ActionType.SetLicenseInfo,
            data,
          })
          return result.data
        }
      })
  }
}

export interface IIsProSubscriber {
  isProSubscriber: boolean;
}

export function getIsProSubscriber(): ThunkAction<Promise<void>, any, {}, Action> {
  return function (dispatch): Promise<void> {
    return dispatch(GET({ url: subscriptionsPath }))
      .then((result: any) => {
        if (result.data) {
          dispatch({
            type: ActionType.SetIsProSubscriber,
            data: result.data.isProSubscriber,
          });

          return result.data;
        }
      })
  }
}

export function cancelSubscription(): ThunkAction<Promise<boolean>, any, {}, Action> {
  return function (dispatch): Promise<boolean> {
    return dispatch(DELETE({ url: subscriptionsPath }))
      .then((result: any) => {
        dispatch(getIsProSubscriber());
        dispatch(fetchLicenseInfo());
        return result.success
      })
  }
}
