import React from 'react';
import { connect } from 'react-redux'

import { Redirect } from 'react-router-dom'

/* This component routes the application to the appropriate
 * component after the first launch
 */
const LaunchRedirector = ({ status, setupCompleted }) => {
  if (status === "logged-in" || (setupCompleted && setupCompleted !== 'notset')) {
    return <Redirect to="/login-redirector"/>
  } else {
    return <Redirect to="/login"/>
  }
}

const mapStateToProps = (state, ownProps) => ({
  status: state.account.status,
  setupCompleted: state.settings.setupCompleted,
})

const mapDispatchToProps = dispatch => ({
})

export default connect(mapStateToProps, mapDispatchToProps)(LaunchRedirector)
