import React from 'react'
import { connect } from 'react-redux'
import { getUser } from '../actions/account'
import { getSetupCompleted } from '../actions/settings'

import Login from './Login'
import KiteLogo from '../components/KiteLogo'

import '../assets/login-container.css'

class LoginContainer extends React.Component {

  componentDidMount() {
    this.userPoll = setInterval(() => {
      this.props.getUser()
    }, 5000)
  }

  componentWillUnmount() {
    clearInterval(this.userPoll)
  }

  render() {
    const {
      shouldBlur,
      setupCompleted,
    } = this.props

    // if eligible to login
    return (
      <div className={`login__container__main ${shouldBlur ? 'main--blur' : ''}`}>
        <div className="login__header--invisible" />
        <div className={`login__container ${typeof (setupCompleted) === 'undefined' || setupCompleted === 'notset' ? 'login__container--checking' : ''}`}>
          <KiteLogo />
          <Login
            location={this.props.location}
            isSetup={!setupCompleted}
            className="login-container" />
        </div>
      </div>
    )
  }

}

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
  setupCompleted: state.settings.setupCompleted,
  status: state.account.status,
})

const mapDispatchToProps = dispatch => ({
  getUser: () => dispatch(getUser()),
  getSetupCompleted: () => dispatch(getSetupCompleted())
})

export default connect(mapStateToProps, mapDispatchToProps)(LoginContainer)
