import {
  GET,
  POST,
} from './fetch'
import { reset as resetNotifications } from '../store/notification'
import { createFormData, createJson } from '../utils/fetch'
import { identify as cioIdentify } from '../utils/customer-io'
import { identify as mpIdentify } from '../utils/mixpanel'
import { identify as doorbellIdentify, deidentify as doorbellDeidentify } from '../utils/doorbell'
import {
  userAccountPath,
  defaultEmailPath,
  loginPath,
  logoutPath,
  createAccountPath,
  checkEmailPath,
  createPasswordlessAccountPath,
  passwordResetPath,
  metricsIDPath,
  emailVerificationPath,
} from '../utils/urls'

import { fetchLicenseInfo } from '../store/license'

export const FAILED_ACCOUNT_FETCH = 'request account user failed'
export const LOAD_USER = "load account/user data"
export const getUser = () => dispatch =>
  dispatch(GET({ url: userAccountPath() }))
    .then(({ success, data, error }) => {
      if (success && data) {
        dispatch(fetchLicenseInfo())

        doorbellIdentify(data)
        dispatch(identifyMetricsID())
        return dispatch({
          type: LOAD_USER,
          data,
        })
      } else if (error) {
        return dispatch({
          type: FAILED_ACCOUNT_FETCH,
          data,
        })
      }
    })

export const DEFAULT_EMAIL = "default email"
export const defaultEmail = () => dispatch =>
  dispatch(GET({ url: defaultEmailPath() }))
    .then(({ success, data }) => {
      if (success) {
        return dispatch({ type: DEFAULT_EMAIL, data })
      } else {
        return dispatch({ type: DEFAULT_EMAIL })
      }
    })


export const LOG_IN = "log in"
export const LOG_IN_FAILED = "log in failed"
export const logIn = credentials => {
  return async (dispatch) => {
    const req = POST({ url: loginPath(), options: createFormData(credentials) })
    const { success, data, error } = await dispatch(req)
    if (success && data) {
      dispatch(fetchLicenseInfo())

      // Correct identity is important for onboarding tracking
      // used to send remote messages for cohorting.
      await dispatch(identifyMetricsID())
      return dispatch({
        type: LOG_IN,
        data,
      })
    } else if (error) {
      return dispatch({
        type: LOG_IN_FAILED,
        error,
      })
    }
  }
}

export const LOG_OUT = "log out"
export const logOut = () => dispatch =>
  dispatch(GET({ url: logoutPath() }))
    .then(({ success, error }) => {
      if (success) {
        dispatch(resetNotifications())
        dispatch(fetchLicenseInfo())

        // NOTE(Daniel): We don't reset the user ID here to keep state if the
        // user continues to use Kite without logging in.
        doorbellDeidentify()
        dispatch(identifyMetricsID())
        return dispatch({
          type: LOG_OUT,
          success,
        })
      } else {
        return { success, error }
      }
    })

export const verifyEmail = (email, dispatch) =>
  dispatch(POST({
    url: emailVerificationPath,
    options: createJson({ email }),
  }))

const createAccountHelper = async (accountPath, credentials, dispatch) => {
  const { success, error, data } = await verifyEmail(credentials.email, dispatch)
  if (success && data) {
    if (data.verified) {
      const req = POST({ url: accountPath, options: createFormData(credentials) })
      const { success, data, error } = await dispatch(req)
      if (success && data) {
        dispatch(fetchLicenseInfo())

        // Correct identity is important for onboarding tracking
        // used to send remote messages for cohorting.
        await dispatch(identifyMetricsID())
        return dispatch({
          type: CREATE_ACCOUNT,
          success,
          data,
        })
      } else if (error) {
        return { success, error }
      }
    } else {
      return { succes: false, error: "Email address is invalid" }
    }
  } else if (error) {
    return { success, error }
  }
}

export const CREATE_ACCOUNT = "create account"
export const createAccount = credentials => dispatch =>
  createAccountHelper(createAccountPath(), credentials, dispatch)

export const createPasswordlessAccount = credentials => dispatch =>
  createAccountHelper(createPasswordlessAccountPath(), credentials, dispatch)

export const NO_PASSWORD = "user has not set password"
export const REQUEST_PASSWORD_RESET = "request password reset"
export const requestPasswordReset = email => dispatch =>
  dispatch(POST({
    url: passwordResetPath(),
    options: createFormData(email),
  }))

export const IDENTIFY_METRICS_ID = "identify metrics ID"
export const identifyMetricsID = () => dispatch =>
  dispatch(GET({
    url: metricsIDPath(),
  }))
    .then(({ success, data, error }) => {
      if (success && data) {
        cioIdentify(data["forgetful_metrics_id"])
        mpIdentify(data["metrics_id"])
        return dispatch({
          type: IDENTIFY_METRICS_ID,
          success,
          data: data["metrics_id"],
        })
      } else if (error) {
        return { success, error }
      }
    })

/**
 * Checks to see if this email is a valid potential
 * user email.
 */
export const checkNewEmail = email => dispatch =>
  dispatch(POST({
    url: checkEmailPath,
    options: createJson({ email }),
  }))

