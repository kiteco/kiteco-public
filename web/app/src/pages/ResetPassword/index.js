import React from 'react'
import { connect } from 'react-redux'
import { Link } from 'react-router-dom'
import queryString from 'query-string'

import ScrollToTop from '../../components/ScrollToTop'

import * as account from '../../redux/actions/account'
import './assets/reset-password.css'

class ResetPassword extends React.Component {
  /**
   * If we find a token in the url, assume that we are trying
   * to perform the password reset. Otherwise, assume that we
   * are still requesting the password reset.
   */
  render() {
    if (this.props.token && this.props.email) {
      return <Perform
        token={this.props.token}
        email={this.props.email}
        perform={this.props.perform}
      />
    } else {
      return <Request
        email={this.props.email}
        request={this.props.request}
      />
    }
  }
}

class Request extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      email: props.email || "",
      error: "",
      success: false,
    }
  }

  submit = e => {
    e.preventDefault()
    this.setState({ error: "" })
    if (this.state.email) {
      this.props.request(this.state.email)
        .then(({ success, error="" }) =>
          this.setState({ success, error })
        )
    } else {
      this.setState({
        error: "Please enter an email address",
      })
    }
  }

  render() {
    if (this.state.success) {
      return <div
        className="reset-password"
      >
        <ScrollToTop/>
        <h1>Reset your password for Kite</h1>
        <p>Great! We've sent an email to <strong>{ this.state.email }</strong> with a link to reset your password.</p>
      </div>
    }
    return <div
      className="reset-password"
    >
      <ScrollToTop/>
      <h1>Reset your password for Kite</h1>
      <p>Type your email below and we will send you a link to reset your password.</p>
      <form
        onSubmit={this.submit}
        className="reset-password__form"
      >
        <div>
          <label htmlFor="email">Email</label>
          <input
            className="reset-password__input"
            type="email"
            name="email"
            value={this.state.email}
            onChange={e => this.setState({email: e.target.value})}
          />
        </div>
        <button
          className="reset-password__button"
          type="submit"
        >
          Reset Password
        </button>
      </form>
      { this.state.error &&
        <div className="reset-password__error">
          { this.state.error }
        </div>
      }
    </div>
  }
}

class Perform extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      password: "",
      passwordConfirm: "",
      error: "",
      success: false,
      submitted: false,
    }
  }

  submit = e => {
    e.preventDefault()
    if (this.state.password !== this.state.passwordConfirm) {
      this.setState({
        error: "Passwords do not match. Please try again."
      })
    } else {
      this.setState({
        error: "",
        submitted: true,
      })
      this.props.perform({
        password: this.state.password,
        token: this.props.token,
        email: this.props.email,
      })
        .then(({ success, error="" }) =>
          this.setState({
            success,
            error,
            submitted: false,
          })
        )
    }
  }

  render() {
    if (this.state.success) {
      return <div className="reset-password">
        <h1>Set your new password</h1>
        <p>Great! You've successfully set the password for <strong>{ this.props.email }</strong>. <Link to="/login">Sign in now</Link>.</p>
      </div>
    }
    const { submitted } = this.state

    return <div className="reset-password">
      <h1>Set your new password</h1>
      <p>Fill in the form below to set the password for <strong>{ this.props.email }</strong>.</p>
      <form
        onSubmit={this.submit}
        className="reset-password__form"
      >
        <div>
          <label htmlFor="password">New Password</label>
          <input
            className="reset-password__input"
            type="password"
            name="password"
            value={this.state.password}
            onChange={e => this.setState({password: e.target.value})}
          />
        </div>
        <div>
          <label htmlFor="password-confirm">Confirm New Password</label>
          <input
            className="reset-password__input"
            type="password"
            name="password-confirm"
            value={this.state.passwordConfirm}
            onChange={e => this.setState({passwordConfirm: e.target.value})}
          />
        </div>
        { !submitted &&
          <button
            className="reset-password__button"
            type="submit"
          >
            Set Password
          </button>
        }
      </form>
      { this.state.error &&
        this.state.error !== "expired reset token\n" &&
        <div className="reset-password__error">
          { this.state.error }
        </div>
      }
      { this.state.error === "expired reset token\n" &&
        <div className="reset-password__error">
          This request has expired.&nbsp;
          To generate a new password reset request:&nbsp;
          <Link to={`/reset-password?email=${this.props.email}`}>
            Click here
          </Link>.
        </div>
      }
    </div>
  }
}

const mapStateToProps = (state, ownProps) => ({
  ...queryString.parse(ownProps.location.search),
})

const mapDispatchToProps = dispatch => ({
  request: email => dispatch(account.requestPasswordReset(email)),
  perform: email => dispatch(account.performPasswordReset(email)),
})

export default connect(mapStateToProps, mapDispatchToProps)(ResetPassword)
