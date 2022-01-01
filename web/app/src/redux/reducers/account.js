import * as actions from '../actions/account'

const defaultState = {
  status: null,
  promise: null,
  data: null,
  pricing: null,
  organizations: null,
  planDetails: {},
  organizationInvoices: {},
  personalInvoices: null,
  invoices: {},
  plan: {},
}

const saveAccount = (state, action) =>
  ({
    ...state,
    data: action.data,
    status: "logged-in",
    promise: null,
  })

const requestAccount = (state, action) =>
  ({
    ...state,
    status: "loading",
    promise: action.promise,
  })

const logOut = (state, action) =>
  ({
    ...defaultState,
    status: "logged-out",
  })

const failedAccountFetch = (state, action) =>
  ({
    ...defaultState,
    status: "logged-out",
  })

const account = (state = defaultState, action) => {
  switch (action.type) {
    case actions.REQUEST_ACCOUNT:
      return requestAccount(state, action)
    case actions.LOG_IN:
    case actions.CREATE_NEW_ACCOUNT:
    case actions.RECEIVE_ACCOUNT_INFO:
      return saveAccount(state, action)
    case actions.FAILED_ACCOUNT_FETCH:
      return failedAccountFetch(state, action)
    case actions.BOOT_TO_LOGIN:
    case actions.LOG_OUT:
      return logOut(state, action)
    default:
      return state
  }
}

export default account
