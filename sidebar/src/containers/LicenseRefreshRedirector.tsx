import React from 'react'
import { connect } from 'react-redux'
import { Redirect } from 'react-router-dom'
import { AnyAction } from 'redux'
import { ThunkDispatch } from 'redux-thunk'
import { fetchLicenseInfo, ConversionCohorts, ConversionCohort } from '../store/license'
import { logOut } from '../actions/account'

/* This component routes the application to the appropriate
 * component after the first launch
 */

interface Props {
  location: { search: string }
  status: string
  fetchLicenseInfo: () => void
  conversionCohort: ConversionCohort
  logOut: () => void
  user: { email: string }
}

class LicenseRefreshRedirector extends React.Component<Props, {}> {
  getQueryEmail() {
    const { location } = this.props
    if (!location || !location.search)
      return ""
    return new URLSearchParams(this.props.location.search).get("email")
  }

  render() {
    const { status, user, fetchLicenseInfo, conversionCohort, logOut } = this.props
    const queryEmail = this.getQueryEmail()
    const differentEmail = queryEmail && !(queryEmail === user.email)

    fetchLicenseInfo()
    // Anonymous users who can autostart trials should not need to login.
    if ((conversionCohort !== ConversionCohorts.OptIn && status !== "logged-in") || (status === "logged-in" && !differentEmail)) {
      return <Redirect to="/"/>
    } else {
      logOut()
      return (
        <Redirect
          to={{ pathname: "/login", state: { queryEmail, activateLicense: true }}}
        />
      )
    }
  }
}

function mapDispatchToProps (dispatch: ThunkDispatch<any, {}, AnyAction>) {
  return {
    fetchLicenseInfo: () => dispatch(fetchLicenseInfo()),
    logOut: () => dispatch(logOut()),
  }
}

function mapStateToProps (state: any, ownProps?: {}) {
  return {
    ...ownProps,
    status: state.account.status,
    user: state.account.user,
    conversionCohort: state.license.conversionCohort,
  }
}

export default connect(mapStateToProps, mapDispatchToProps)(LicenseRefreshRedirector)
