import React from 'react'
import { connect } from 'react-redux'

import * as actions from '../../../redux/actions/account'

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
  
  // using UNSAFE to have this called before componentDidMount of children
  UNSAFE_componentWillMount() {
    const { status, check, aliasFirst } = this.props
    if ( !status || status === "logged-out") {
      check(aliasFirst)
    }
  }

  render() {
    const { children, status } = this.props
    if (status === "logged-in") {
      if (children) {
        if (typeof children === 'function') {
          return children()
        } else {
          return React.Children.only(children)
        }
      }
    }
    return null
  }
}

const mapStateToProps = (state, ownProps) => ({
  status: state.account.status,
})

const mapDispatchToProps = dispatch => ({
  check: aliasFirst => dispatch(actions.fetchAccountInfoOrRedirect(aliasFirst)),
})

export default connect(mapStateToProps, mapDispatchToProps)(AccountGate)
