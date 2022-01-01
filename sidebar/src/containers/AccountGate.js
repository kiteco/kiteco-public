import React from 'react'
import { connect } from 'react-redux'
import { Redirect } from 'react-router-dom'

import * as actions from '../actions/account'

/** AccountGate acts as a wrapper for containers that
 *  require authentication to access.
 *  AccountGate is designed to be used declaratively as so:
 *
 *    <AccountGate>
 *      {
 *        a single component here which will be mounted
 *        if authentication succeeds
 *      }
 *    </AccountGate>
 *
 *  Alternatively, you may also pass in a function that
 *  returns a single component as the `children` prop if
 *  that is more convenient:
 *
 *    <AccountGate children={() => <div/>}/>
 *
 *  This function will not be evaluated until the account
 *  is authenticated
 *
 *  AccountGate when mounted will try to grab the account info
 *  and will boot to login screen if unauthenticated. It will also
 *  save the original path the user is trying to access and once
 *  authenticated, will push the user back to that path.
 *
 *  Child containers will not be mounted unless the user has been
 *  logged in.
 */
class AccountGate extends React.Component {

  checkStatus() {
    const {
      status,
      check,
      online,
      networkConnected,
    } = this.props
      /**
       * TODO(dane): the check below should be modified once
       * KiteLocal user management is implemented
       */
    if ( (networkConnected && online) && (!status || status === "logged-out" || status === "untried")) {
      check()
    }
  }

  // we use UNSAFE here to preempt componentDidMount methods of children
  UNSAFE_componentMount() {
    this.checkStatus()
  }

  componentDidUpdate() {
    this.checkStatus()
  }

  render() {
    const {
      children,
      status,
      setupCompleted,
      networkConnected,
    } = this.props
    if ((status === "logged-out" || status === "untried")) {
      return <Redirect to="/login" />
    }
    if ((status === "logged-in" && setupCompleted !== "") || status === 'untried') {
      if (children) {
        if (typeof children === 'function') {
          return children()
        } else {
          return React.Children.only(children)
        }
      }
    }

    return null;
  }
}

const mapStateToProps = (state, ownProps) => ({
  setupCompleted: state.settings.setupCompleted,
  status: state.account.status,
  online: state.errors.online,
  networkConnected: state.system.networkConnected,
})

const mapDispatchToProps = dispatch => ({
  check: () => dispatch(actions.fetchAccountInfoOrRedirect()),
})

export default connect(mapStateToProps, mapDispatchToProps)(AccountGate)
