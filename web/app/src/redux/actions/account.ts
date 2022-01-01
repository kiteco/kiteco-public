import { Dispatch } from 'redux';
import { replace, push } from 'connected-react-router'

import { record } from './utils'

import { identify, reset, alias } from '../../utils/analytics'

import { GET, POST } from './fetch'

import {
  createFormData,
  createJson,
} from "../../utils/fetch"

import {
  userAccountPath,
  loginPath,
  logoutPath,
  checkEmailPath,
  checkPasswordPath,
  createAccountPath,
  forumLoginPath,
  verifyEmailPath,
  unsubscribePath,
  invitePath,
  passwordResetPath,
  newsletterSignUp,
  mobileDownloadPath,
  pyconSignupPath,
} from "../../utils/urls"

/* ==={ DATA FETCHING }=== */

/**
 * fetchAccountInfo attempts to get user account information
 * based on the current session.
 * Returns a promise that will resolve into either account data
 * or an error message about the account
 *
 * Use this if you want to try to get the account information, but
 * if the user is not authenticated, then it is okay to fail
 */
export const REQUEST_ACCOUNT = 'request account user'
export const RECEIVE_ACCOUNT_INFO = 'receive account user'
export const FAILED_ACCOUNT_FETCH = 'request account user failed'
export const fetchAccountInfo = (aliasFirst: any = null) => (dispatch: any, getState: any): Promise<any> => {
  const { account } = getState()
  if (account.status === "loading") {
    return account.promise
  } else {
    const promise = dispatch(GET({ url: userAccountPath }))
      .then(({ success, data, error, response }: any) => {
        const { account } = getState()
        if (success) {
          const action = {
            type: RECEIVE_ACCOUNT_INFO,
            success,
            error,
            data,
          }
          if (account.status === "loading") {
            if (aliasFirst) {
              alias(data.id)
            }
            identify(data.id)
            dispatch(action)
          }
          return action
        } else {
          if (account.status === "loading") {
            dispatch({
              type: FAILED_ACCOUNT_FETCH,
            })
          }
          return { success, error }
        }
      });

    dispatch({
      type: REQUEST_ACCOUNT,
      promise,
    });

    return promise;
  }
}

/* ==={ LOGIN/LOGOUT ACTIONS }=== */

/**
 * Logs a user into kite. Sends a request with
 * user credentials as JSON. If this succeeds,
 * the server will send back session cookies that
 * all subsequent requests will rely on.
 */
export const LOG_IN = 'log in'
export const logIn = (credentials: any) => (dispatch: any) =>
  dispatch(POST({
    url: loginPath,
    options: createFormData(credentials),
    reportNetworkFailure: true,
  }))
    .then(({ success, data, error }: any) => {
      if (success) {
        dispatch({
          ...record(LOG_IN),
          data,
        })
        identify(data.id)
        return { success }
      } else {
        return { success, error }
      }
    })

/**
 * Logs a user out.
 * Resets analytics
 *
 * NOTE: odd backend bug right now where after logout,
 * we can still hit the /api/account/user endpoint and
 * receive information about the logged-out user
 * Use the `ensureAuthentication` method below to catch
 * this case for now.
 * TODO: figure out why the above happens and fix it
 */
export const LOG_OUT = 'log out'
export const logOut = (redirect: any = null) => (dispatch: any) =>
  dispatch(GET({
    url: logoutPath,
    reportNetworkFailure: true,
  }))
    .then(({ success, error }: any) => {
      dispatch(record(LOG_OUT))
      reset()
      if (redirect) {
        dispatch(push("/login"))
      }
      return { success, error }
    })

/**
 * Boot to login screen
 * and record the path to go to after login, if any
 */
export const BOOT_TO_LOGIN = 'boot to login'
export const bootToLogin = (nextLocation: any) => (dispatch: any) => {
  dispatch({
    ...record(BOOT_TO_LOGIN),
    location: nextLocation,
  })
  dispatch(replace("/login"))
}

/**
 * Boot to login screen and record the current requested location
 */
export const bootToLoginAndSaveRequestedLocation = () => (dispatch: any, getState: any) => {
  const { router } = getState()
  dispatch(bootToLogin({ ...router.location }))
}

/**
 * Use this if you want to ensure that a user is logged-in,
 * and to force the user to log in if they are not otherwise logged in.
 */
export const fetchAccountInfoOrRedirect = (aliasFirst: any) => (dispatch: any) =>
  dispatch(fetchAccountInfo(aliasFirst))
    .then(({ type, success, error }: any) => {
      if (!type || type !== RECEIVE_ACCOUNT_INFO) {
        dispatch(bootToLoginAndSaveRequestedLocation())
      }
      return { success, error }
    })

/* ==={ ACCOUNT CREATION ACTIONS }=== */

export interface ICheckEmailResponse {
  success: boolean,
  error: string,
  data: {
    fail_reason: string,
    email_invalid: boolean,
    account_exists: boolean,
    has_password: boolean,
  } | string,
}

/**
 * Checks to see if this email is a valid potential
 * user email.
 */
export const checkNewEmail = (email: string) => (dispatch: Dispatch<any>): Promise<ICheckEmailResponse> =>
  dispatch(POST({
    url: checkEmailPath,
    options: createJson({ email }),
  }))

/**
 * Checks to see if this password is a valid
 * potential user password.
 */
export const checkNewPassword = (password: any) => (dispatch: any) =>
  dispatch(POST({
    url: checkPasswordPath,
    options: createJson({ password }),
  }))

/**
 * Checks if account details (email, password),
 * combine to make a valid new user account.
 *
 * Returns an object with `success` signaling
 * the result of the check and an object `error`
 * with errors for password and email.
 */
export const checkNewAccount = (account: any) => (dispatch: any) => {
  return Promise.all([
    dispatch(checkNewEmail(account.email)),
    dispatch(checkNewPassword(account.password)),
  ]).then(([email, password]) => {
    return {
      success: email.success && password.success,
      error: {
        email: email.error,
        password: password.error,
      }
    }
  })
}

export const CREATE_NEW_ACCOUNT = 'create new account'
export const createNewAccount = (account: any) => (dispatch: any) =>
  dispatch(POST({
    url: createAccountPath,
    options: createFormData(account),
    reportNetworkFailure: true,
  }))
    .then(({ success, data, error }: any) => {
      if (success) {
        identify(data.id)
        return dispatch({
          type: CREATE_NEW_ACCOUNT,
          data,
          success,
          error,
        })
      } else {
        return { success, data, error }
      }
    })


/* ==={ RESET PASSWORD }=== */

export const requestPasswordReset = (email: any) => (dispatch: any) =>
  dispatch(POST({
    url: passwordResetPath("request"),
    options: createFormData({ email }),
    reportNetworkFailure: true,
  }))

export const performPasswordReset = (submission: any) => (dispatch: any) =>
  dispatch(POST({
    url: passwordResetPath("perform"),
    options: createFormData(submission),
    reportNetworkFailure: true,
  }))

/* ==={ VERIFY EMAIL }=== */

export const verifyEmail = (submission: any) => (dispatch: any) =>
  dispatch(POST({
    url: verifyEmailPath,
    options: createFormData(submission),
    reportNetworkFailure: true,
  }))

/* ==={ UNSUBSCRIBE }=== */

export const unsubscribe = (email: any) => (dispatch: any) =>
  dispatch(POST({
    url: unsubscribePath,
    options: createFormData({ email }),
    reportNetworkFailure: true,
  }))

/* ==={ SLACK INVITE } === */
export const slackInvite = (email: any) => (dispatch: any) =>
  dispatch(POST({
    url: invitePath("slack"),
    options: createJson({ email }),
  }))

/* ==={ INVITE }=== */

export const inviteEmails = (submission: any) => (dispatch: any) =>
  dispatch(POST({
    url: invitePath("emails"),
    options: createJson(submission),
  }))


/* ==={ FORUM LOGIN }=== */

export const forumLogin = (submission: any) => (dispatch: any) =>
  dispatch(POST({
    url: forumLoginPath,
    options: createJson(submission),
  }))

/* ==={ NEWSLETTER }=== */

export const newsletter = (submission: any) => (dispatch: any) =>
  dispatch(POST({
    url: newsletterSignUp,
    options: createJson(submission),
  }))

/* ==={ MOBILE DOWNLOAD }=== */

export const mobileDownload = (submission: any) => (dispatch: any) =>
  dispatch(POST({
    url: mobileDownloadPath,
    options: createJson(submission),
  }))

/* ==={ PYCON SIGNUP }=== */

export const pyconSignUp = (submission: any) => (dispatch: any) =>
  dispatch(POST({
    url: pyconSignupPath,
    options: createJson(submission),
  }))
