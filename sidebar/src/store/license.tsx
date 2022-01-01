import { Action as BaseAction, Dispatch } from 'redux'
import { ThunkAction, ThunkDispatch, ThunkMiddleware } from 'redux-thunk'
import pluralize from 'pluralize'
import _ from 'lodash'
import { GET, POST } from '../actions/fetch'
import * as notif from './notification'
import {
  licenseInfoPath,
  localhostProxy,
  conversionCohortPath,
  fetchCohortPath,
  allFeaturesProPath,
  userNodeHealthPath,
} from '../utils/urls'
import { paywallCompletionsRemainingPath } from '../utils/urls'
import { track } from "../utils/analytics"
const { shell } = window.require("electron")

// types

export enum Product {
  Hide = "hide",
  Pro = "pro",
  Free = "free",
}
export enum Plan {
  Trial = "pro_trial",
  Education = "pro_education",
}
interface ProLicenseInfo {
  product: Product.Pro
  plan: Plan
  days_remaining: number
}
interface FreeLicenseInfo {
  product: Product.Free
  trial_available: boolean
  trial_available_duration?: TrialAvailableDuration
}
interface TrialAvailableDuration {
  unit: string
  value: number
}
export type LicenseInfo = ProLicenseInfo | FreeLicenseInfo | null

export enum CTASource {
  CopilotNotif = "copilot_notif",
  // DesktopNotif is a special case, included for here completeness.
  // DesktopNotif = "desktop_notif",
  ProductBadge = "product_badge",
  SettingsButton = "settings_button",
  SettingsCheckbox = "settings_checkbox",
  SettingsLearnMore = "settings_learn_more",
}

export enum ConversionCohorts {
  Autostart = "autostart",
  OptIn = "opt-in",
  QuietAutostart = "quiet-autostart",
  UsagePaywall = "usage-paywall",
  Unset = "",
}

export type ConversionCohort =
  | ConversionCohorts.Autostart
  | ConversionCohorts.OptIn
  | ConversionCohorts.UsagePaywall
  | ConversionCohorts.QuietAutostart
  | ConversionCohorts.Unset

// actions

enum ActionType {
  SetUserNodeAvailability = 'license.SetUserNodeAvailability',
  SetLicenseInfo = 'license.SetLicenseInfo',
  GetConversionCohort = 'license.GetOnboardingCohort',
  GetPaywallCompletionsRemaining = 'license.GetPaywallCompletionsRemaining',
  GetAllFeaturesPro = 'license.GetAllFeaturesPro',
}

interface SetUserNodeAvailability extends BaseAction {
  type: ActionType.SetUserNodeAvailability
  data: boolean
}

interface SetLicenseInfoAction extends BaseAction {
  type: ActionType.SetLicenseInfo
  data: LicenseInfo
}

interface GetConversionCohort extends BaseAction {
  type: ActionType.GetConversionCohort
  data: ConversionCohort
}

interface GetPaywallCompletionsRemaining extends BaseAction {
  type: ActionType.GetPaywallCompletionsRemaining,
  data: number
}

interface GetAllFeaturesPro extends BaseAction {
  type: ActionType.GetAllFeaturesPro,
  data: boolean
}


type Action =
  SetUserNodeAvailability
  | SetLicenseInfoAction
  | GetConversionCohort
  | GetAllFeaturesPro
  | GetPaywallCompletionsRemaining

// reducer

// TODO(naman) this is hack, eventually when we port the reducer combinator to TS, we can remove it.
interface State {
  userNodeAvailable: boolean | null
  licenseInfo: LicenseInfo
  conversionCohort: ConversionCohort
  allFeaturesPro: boolean | null
  paywallCompletionsRemaining: number | null
}

let init: State = {
  userNodeAvailable: null,
  licenseInfo: null,
  conversionCohort: ConversionCohorts.Unset,
  allFeaturesPro: null,
  paywallCompletionsRemaining: null,
}

export function reducer (state: State = init, action: Action): State {
  switch (action.type) {
    case ActionType.SetUserNodeAvailability:
      return { ...state, userNodeAvailable: action.data }
    case ActionType.SetLicenseInfo:
      return { ...state, licenseInfo: action.data }
    case ActionType.GetConversionCohort:
      return { ...state, conversionCohort: action.data }
    case ActionType.GetAllFeaturesPro:
      return { ...state, allFeaturesPro: action.data }
    case ActionType.GetPaywallCompletionsRemaining:
      return { ...state, paywallCompletionsRemaining: action.data }
    default:
      return state
  }
}


// action creators

export function checkIfUserNodeAvailable(): ThunkAction<Promise<boolean>, any, {}, Action> {
  return async (dispatch): Promise<boolean> => {
    const { success, data } = await dispatch(GET({
      url: userNodeHealthPath(),
    }))
    dispatch({
      type: ActionType.SetUserNodeAvailability,
      data: success,
    })
    return success
  }
}

export function fetchLicenseInfo(queries: { refresh?: boolean } | undefined = undefined): ThunkAction<Promise<LicenseInfo>, any, {}, Action> {
  return async (dispatch): Promise<LicenseInfo> => {
    const { success, data } = await dispatch(GET({
      url: licenseInfoPath(),
      queries,
    }))
    if (success && data) {
      dispatch({
        type: ActionType.SetLicenseInfo,
        data,
      })
    }
    return data
  }
}

export function getConversionCohort(): ThunkAction<Promise<ConversionCohort>, any, {}, Action> {
  return async (dispatch, getState): Promise<ConversionCohort> => {
    const curCohort = getState().license.conversionCohort
    const { success, data } = await dispatch(GET({ url: conversionCohortPath() }))
    if (success && data) {
      let newCohort: ConversionCohort = data
      dispatch({
        type: ActionType.GetConversionCohort,
        data: newCohort,
      })
      return newCohort
    }
    return curCohort
  }
}

export function getAllFeaturesPro(): ThunkAction<Promise<boolean>, any, {}, Action> {
  return async (dispatch): Promise<boolean> => {
    const { success, data } = await dispatch(GET({ url: allFeaturesProPath() }))
    const newAllFeatPro = success ? data === 'true' : false
    dispatch({
      type: ActionType.GetAllFeaturesPro,
      success,
      data: newAllFeatPro,
    })
    return success ? data === 'true' : false
  }
}

export function getPaywallCompletionsRemaining(): ThunkAction<Promise<number | undefined>, any, {}, Action> {
  return async (dispatch): Promise<number | undefined> => {
    const { success, data } = await dispatch(GET({ url: paywallCompletionsRemainingPath() }))
    if (success && data) {
      let remaining = Number(data)
      if (Number.isNaN(remaining)) {
        return
      }

      dispatch({
        type:  ActionType.GetPaywallCompletionsRemaining,
        data: remaining.valueOf(),
      })
      return remaining.valueOf()
    }
  }
}

export const fetchCohortTimeoutMS = 6000
// This makes an external request to fetch the user's conversion cohort for the first time, during onboarding
// Not to be mistaken with getConversionCohort, which only queries kited and the local settings
export function fetchConversionCohort(): ThunkAction<Promise<void>, any, {}, Action> {
  return async (dispatch): Promise<void> => {
    return await dispatch(POST({ url: fetchCohortPath() }))
  }
}

export function startTrial(source: CTASource): ThunkAction<void, any, {}, Action> {
  return function(dispatch) {
    dispatch(dismissNotifs([
      NotificationIDs.AutostartTrial,
      NotificationIDs.OptInTrial,
    ]))

    // desktoplogin, and redirect to alpha.kite.com/web/account/start-trial
    shell.openExternal(localhostProxy(`/clientapi/desktoplogin/start-trial?cta-source=${source}`))

    // Refresh the license info after 10s.
    // If the user was already logged in, the trial should start by then.
    setTimeout(() => dispatch(fetchLicenseInfo()), 10000)
  }
}

export function getNotificationMiddleware(): ThunkMiddleware {
  return (
    midwareapi: {
      dispatch: ThunkDispatch<object, any, Action>,
      getState: () => { license: State }
    }
  ) =>  (next: Dispatch<BaseAction>) => (action: Action) => {
    let {
      allFeaturesPro,
      conversionCohort: cohort,
      licenseInfo,
      paywallCompletionsRemaining,
    } = midwareapi.getState().license

    /* These state changes may show or hide notifications
     * We only attempt to update the conversion notification
     * if the state has actually changed to something new.
     * Individual polling of means this is eventually
     * consistent, which is not ideal. Batching the requests
     * or exposing an endpoint returning all of this
     * would be the principled thing to do.
     */

    let update = false
    switch (action.type) {
      case ActionType.GetConversionCohort:
        update = action.data !== cohort
        cohort = action.data
        break
      case ActionType.GetAllFeaturesPro:
        update = action.data !== allFeaturesPro
        allFeaturesPro = action.data
        break
      case ActionType.GetPaywallCompletionsRemaining:
        const firstZero = (action.data !== paywallCompletionsRemaining && action.data === 0)
        const refreshed = paywallCompletionsRemaining !== null ? action.data > paywallCompletionsRemaining : false
        update = paywallCompletionsRemaining === null || firstZero || refreshed
        paywallCompletionsRemaining = action.data
        break
      case ActionType.SetLicenseInfo:
        update = !_.isEqual(action.data, licenseInfo)
        licenseInfo = action.data

        const canTrial = cohort === ConversionCohorts.OptIn || cohort === ConversionCohorts.Autostart || cohort === ConversionCohorts.QuietAutostart
        if (!licenseInfo || licenseInfo.product !== Product.Free || (!licenseInfo.trial_available_duration && canTrial)) {
          midwareapi.dispatch(dismissNotifs([
            NotificationIDs.AutostartTrial,
            NotificationIDs.OptInTrial,
            NotificationIDs.UsagePaywall,
            NotificationIDs.PaywallAllFeatPro,
          ]))
        }
        break
    }

    if (update) {
      midwareapi.dispatch(updateConversionNotif(
        cohort,
        allFeaturesPro,
        paywallCompletionsRemaining,
        licenseInfo,
      ))
    }

    return next(action)
  }


}

function updateConversionNotif(
  cohort: ConversionCohort,
  allFeaturesPro: boolean | null,
  complsRemaining: number | null,
  licenseInfo: LicenseInfo,
): ThunkAction<void, any, {}, Action> {
  return async (dispatch) => {
    // If any not set, store has not been fully and we're not ready to show a notification
    if (cohort === ConversionCohorts.Unset || allFeaturesPro === null || complsRemaining === null || licenseInfo === null) {
      return
    }

    switch (cohort) {
      case ConversionCohorts.Autostart:
        dispatch(dismissNotifs([
          NotificationIDs.OptInTrial,
          NotificationIDs.UsagePaywall,
          NotificationIDs.PaywallAllFeatPro,
        ]))
        if (licenseInfo && licenseInfo.product === Product.Free && licenseInfo.trial_available_duration) {
          dispatch(showAutostartTrialNotif(licenseInfo.trial_available_duration))
        }
        break
      case ConversionCohorts.OptIn:
        dispatch(dismissNotifs([
          NotificationIDs.AutostartTrial,
          NotificationIDs.UsagePaywall,
          NotificationIDs.PaywallAllFeatPro,
        ]))
        if (licenseInfo && licenseInfo.product === Product.Free && licenseInfo.trial_available_duration) {
          dispatch(showOptInTrialNotif(licenseInfo.trial_available_duration))
        }
        break
      case ConversionCohorts.UsagePaywall:
        dispatch(dismissNotifs([
          NotificationIDs.AutostartTrial,
          NotificationIDs.OptInTrial,
        ]))
        if (licenseInfo && licenseInfo.product === Product.Free) {
          if (!allFeaturesPro || complsRemaining > 0) {
            dispatch(showUsagePaywallNotif())
            dispatch(dismissNotifs([ NotificationIDs.PaywallAllFeatPro ]))
          } else {
            dispatch(showPaywallAllFeaturesProNotif())
          }
        }
        break
    }
  }
}

enum NotificationIDs {
  AutostartTrial = 'autostart-trial',
  OptInTrial = 'opt-in-trial',
  UsagePaywall = 'usage-paywall',
  PaywallAllFeatPro = 'paywall-all-features-pro',
}

type NotificationID =
  | NotificationIDs.AutostartTrial
  | NotificationIDs.OptInTrial
  | NotificationIDs.UsagePaywall
  | NotificationIDs.PaywallAllFeatPro

function showAutostartTrialNotif(duration: TrialAvailableDuration): ThunkAction<void, any, {}, Action> {
  return function(dispatch) {
    dispatch(notif.notify({
      id: NotificationIDs.AutostartTrial,
      component: notif.NotifType.Default,
      payload: {
        title: "Free Kite pro trial",
        text: `
          Kite Pro gives you ML-powered autocompletions for Python.
          Your ${duration.value} ${duration.unit} free trial will start automatically when you use your first ML-powered completion.
        `,
      },
      docsOnly: false,
    }))
  }
}

function showOptInTrialNotif(duration: TrialAvailableDuration): ThunkAction<void, any, {}, Action> {
  return function(dispatch) {
    track({
      event: "cta_shown",
      props: { cta_source: CTASource.CopilotNotif },
    })
    const { value, unit } = duration
    dispatch(notif.notify({
      id: NotificationIDs.OptInTrial,
      component: notif.NotifType.Default,
      payload: {
        title: "Start Your Free Kite Pro Trial",
        text: `
          Kite Pro gives you ML-powered autocompletions for Python.
          You can try it out for free for ${value} ${pluralize(unit, value)}.
          After the trial is over, you can continue to use the basic version of Kite for free.
        `,
        buttons: [{
          text: "Try Kite Pro for free",
          onClick: () => dispatch(startTrial(CTASource.CopilotNotif)),
        }],
      },
      docsOnly: false,
    }))
  }
}

function showUsagePaywallNotif(): ThunkAction<void, any, {}, Action> {
  return function(dispatch) {
    dispatch(notif.notify({
      id: NotificationIDs.UsagePaywall,
      component: notif.NotifType.Default,
      payload: {
        title: "Welcome to Kite Free",
        text: `
          Kite Free gives you a limited number of ★ completions to
          use per day. Upgrade to Kite Pro to code faster with
          unlimited completions powered by machine learning.`,
      },
      docsOnly: false,
    }))
  }
}

function showPaywallAllFeaturesProNotif(): ThunkAction<void, any, {}, Action> {
  return function(dispatch) {
    dispatch(notif.notify({
      id: NotificationIDs.PaywallAllFeatPro,
      component: notif.NotifType.Default,
      payload: {
        title: "You've Used Up All Your Completions Today",
        text: `
          Kite will unlock with more completions tomorrow. You
          can also upgrade to Kite Pro now to get unlimited
          completions powered by machine learning.`,
        noDismiss: true,
        buttons: [{
          text: 'Upgrade to Kite Pro',
          onClick: () => shell.openExternal(localhostProxy('/clientapi/desktoplogin?d=%2Fpro%3Floc%3Dcopilot_notif%26src%3Dlimit')),
          noDismiss: true,
        }],
      },
      docsOnly: false,
    }))
  }
}

function dismissNotifs(ids: NotificationID[]): ThunkAction<void, any, {}, Action> {
  return (dispatch) => {
    ids.forEach(id => dispatch(notif.dismiss(id)))
  }
}
