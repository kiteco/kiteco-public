import React from 'react';
import { connect } from 'react-redux'

import { Redirect } from 'react-router-dom'


/* This component routes the application to the appropriate
 * component after first login
 */
const LoginRedirector = ({ setupCompleted }) => {
  if (setupCompleted) {
    return <Redirect to="/home"/>
  } else {
    return <Redirect to="/setup"/>
  }
}

const mapStateToProps = (state, ownProps) => ({
  setupCompleted: state.settings.setupCompleted,
})

const mapDispatchToProps = dispatch => ({
})

export default connect(mapStateToProps, mapDispatchToProps)(LoginRedirector)
