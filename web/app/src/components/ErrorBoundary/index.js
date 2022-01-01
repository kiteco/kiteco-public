import React from 'react'
import { connect } from 'react-redux'

import Rollbar from '../../utils/rollbar'

import ErrorOverlay from './ErrorOverlay'

class ErrorBoundary extends React.Component {

  componentDidCatch(error, info) {
    this.props.sendAppException()
    Rollbar.handleErrorBoundaryCatch({ error, info })
  }

  render() {
    if(this.props.errors.appException) {
      return <ErrorOverlay
        title="Huh... that's weird"
        subtitle="Something unexpected occurred. We'll investigate what happened."
        subtitle2="Some visitors report that this error screen can occur due to their ad blocker and browser version, and that upgrading their browser or disabling their ad blocker helps."
        supportEmail={true}
        handler={() => { window.location.reload(true) }}
        btnText="Refresh the page"
      />
    }
    return this.props.children
  }
}

const mapDispatchToProps = dispatch => ({
  sendAppException: () => dispatch({
    type: 'APP_EXCEPTION'
  })
})

const mapStateToProps = (state, ownProps) => ({
  errors: state.errors,
})

export default connect(mapStateToProps, mapDispatchToProps)(ErrorBoundary)
