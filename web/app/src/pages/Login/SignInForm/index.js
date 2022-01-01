import { push } from 'connected-react-router'
import React from 'react'
import { connect } from 'react-redux'
import { Link } from 'react-router-dom'
import queryString from 'query-string'

import * as account from '../../../redux/actions/account'

import './assets/sign-in-form.css'

const stages = Object.freeze({
  email: "email",
  password: "password",
})

class SignInForm extends React.Component {
  _password = null

  constructor(props) {
    super(props)
    this.state = {
      title: "Enter your email",
      email: "",
      password: "",
      status: "enabled",
      isNewUser: props.source === "new",
      error: "",
      stage: stages.email,
    }

    this.postRedirect = this.postRedirect.bind(this)
  }

  componentDidMount() {
    const { email, account, fetchAccountInfo } = this.props;
    // since some parent components unmount SignInForm right
    // after successful login/signup and others will
    // keep me around, we should track if we are mounted
    this._isMounted = true
    if (!account.status) {
      fetchAccountInfo()
    }

    this.setState({ title: this.getTitle(this.state.stage) })
    if (email) {
      this.setState({ email }, this.checkEmail);
    }
  }

  componentWillUnmount() {
    this._isMounted = false
  }

  setPassword = (e) => {
    this.setState({
      password: e.target.value,
    })
  }

  setEmail = (e) => {
    this.setState({
      email: e.target.value,
    })
  }

  postRedirect = () => {
    if (this.props.redirect) {
      window.location.href = this.props.redirect
      return
    }
    this.props.push("/settings")
  }

  submit = async (e) => {
    e.preventDefault()
    this.setState({ status: "disabled" })

    const { email, password } = this.state;
    if (!email || !password) {
      this.setState({
        status: "enabled",
        error: "All fields are required",
      })
      return
    }

    const { isNewUser } = this.state;
    const { referral, privilege, source } = this.props;
    const { login, createNewAccount, submit } = this.props;

    let action = null;
    if (isNewUser) {
      action = await createNewAccount({
        email,
        password,
        referral,
        privilege,
        channel: source || "",
      });
    } else {
      action = await login({ email, password });
    }

    submit && submit(action);

    if (this._isMounted) {
      this.setState({
        status: "enabled",
        error: action.success ? "" : action.error,
      })
    }
    if (action.success) {
      this.postRedirect()
    }
  }

  checkEmail = e => {
    e && e.preventDefault()

    this.setState({
      status: "disabled",
      error: "",
    })

    const { email, isNewUser } = this.state
    const { checkEmail, requestSetPassword } = this.props

    if (!email) {
      this.setState({
        status: "enabled",
        error: "Email required",
      })
      return
    }

    checkEmail(email).then(async ({ success, data, error }) => {
      if (!data) {
        const newState = {
          stage: stages.password,
          status: "enabled",
          error: "",
          isNewUser: success ? true : false,
          title: this.getTitle(stages.password, !success),
        }
        this.setState(newState)
        return
      }

      if (data.email_invalid) {
        this.setState({
          error,
          status: "enabled",
        })
        return
      }

      const newState = {
        title: this.getTitle(stages.password, data.account_exists),
        stage: "password",
        status: "enabled",
        error: "",
      }

      if (!isNewUser && !data.account_exists) {
        newState.isNewUser = true;
      }

      if (isNewUser && data.account_exists) {
        newState.isNewUser = false;
        newState.error = "You already have an account with this email!" +
          " Please login with your existing password."
      }

      // request set new password for passwordless accounts
      if (!isNewUser && data.account_exists && !data.has_password) {
        await requestSetPassword(email)
        newState.error = "You already have an account with this email but you haven't set your password!" +
          " We just sent you an email to set a password. Please set it before trying to sign in."
      }

      this.setState(newState)
      if (this._password) {
        this._password.focus()
      }
    })
  }

  editEmail = e => {
    e && e.preventDefault();
    this.setState({
      stage: stages.email,
      title: this.getTitle(stages.email),
    })
  }

  getTitle = (stage, accountExists) => {
    const { pathname, pro } = this.props;

    let title;
    switch (true) {
      case stage === "email":
        title = "Enter your email";
        break;
      case stage === "password" && accountExists:
        title = "Enter your password";
        break;
      case stage === "password" && !accountExists:
        title = "Create a password";
        break;
      default:
        break;
    }

    // Since the backend does not pass account information over,
    // we request account info in this component to determine the
    // first part of the title. Rather than pass the second part
    // depending on the route, determine it here for clarity.
    const segments = pathname.split('/');
    switch (true) {
      case segments.includes("start-trial"):
        title += " to start your Kite Pro trial";
        break;
      // Support previous implementation of ?pro=1 query param.
      case segments.includes("upgrade-pro") || pro === "1":
        title += " to upgrade to Kite Pro";
        break;
      default:
        break;
    }

    return title;
  }

  render() {
    const { className="sign-in-form" } = this.props
    const { isNewUser, email, error, stage, title } = this.state

    if (this.props.account.status === "loading") {
      return <div className={`${className}`}>Loading...</div>
    }

    if (this.props.account.status === "logged-in") {
      this.postRedirect()
      return (
        <div className={`${className}`}>
          Redirecting...
        </div>
      )
    }

    return (
      <div className={`${className}__wrapper`}>
        <h1>{title}</h1>
        <div className={`${className}`}>
          <form
            className={`${className}__main`}
            onSubmit={stage === "email" ? this.checkEmail : this.submit}
          >
            { error &&
              <div className={`${className}__error`}>
                { error }
              </div>
            }

            <div className={`${className}__fields`}>
              <Email
                email={email}
                set={stage === "email" && this.setEmail}
                className={className}
                onClick={this.editEmail}
              />
              <Password
                password={this.state.password}
                set={this.setPassword}
                className={className}
                hide={stage === "email"}
                assignRef={input => this._password = input}
              />
            </div>
            <div className={`${className}__extra`}>
              {
                (!isNewUser && stage === stages.password)
                ?
                  <Link to={`/reset-password?email=${email}`}>Reset password</Link>
                :
                  // eslint-disable-next-line
                  <a className={`${className}__toggle`} href="#" onClick={this.editEmail}>
                    { stage === stages.password ? "Back" : null }
                  </a>
              }
              <button
                className={`
                  ${className}__button
                  ${className}__button--${this.state.status}
                `}
                type="submit"
              >
                { stage === stages.email && "Continue" }
                { !isNewUser && stage === stages.password && "Sign In" }
                { isNewUser && stage === stages.password && "Create Account" }
              </button>
            </div>
          </form>
        </div>
      </div>
    )
  }
}

const Email = ({ email, set, className, onClick }) =>
  <div className={`${className}__email`}>
    <input
      className={`${className}__input`}
      readOnly={!set}
      onClick={!set ? onClick : () => {}}
      onFocus={!set ? onClick : () => {}}
      name="email"
      type="email"
      value={email}
      placeholder="Email"
      onChange={set}
    />
  </div>

const Password = ({ password, set, className, hide, assignRef }) =>
  <div className={`${className}__password ${hide ? className : ""}${hide ? "__password--hide" : ""}`}>
    <input
      className={`${className}__input`}
      name="password"
      type="password"
      value={password}
      placeholder="Password"
      onChange={set}
      ref={assignRef}
    />
  </div>

const mapStateToProps = (state, ownProps) =>
  ({
    account: state.account,
    email: queryString.parse(state.router.location.search).email,
    source: queryString.parse(state.router.location.search).source,
    redirect: queryString.parse(state.router.location.search).redirect,
    pro: queryString.parse(state.router.location.search).pro,
    pathname: state.router.location.pathname,
    ...ownProps,
  })

const mapDispatchToProps = dispatch => ({
  fetchAccountInfo: () => dispatch(account.fetchAccountInfo()),
  login: (cred) => dispatch(account.logIn(cred)),
  createNewAccount: cred => dispatch(account.createNewAccount(cred)),
  logout: () => dispatch(account.logOut()),
  checkEmail: email => dispatch(account.checkNewEmail(email)),
  requestSetPassword: email => dispatch(account.requestPasswordReset(email)),
  push: url => dispatch(push(url)),
})

export default connect(mapStateToProps, mapDispatchToProps)(SignInForm)
