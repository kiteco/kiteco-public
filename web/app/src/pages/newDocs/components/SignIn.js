import React from 'react'
import { connect } from 'react-redux'

import LoadingSpinner from './LoadingSpinner'
import { setLoading } from '../../../redux/actions/loading'
import * as accountActions from '../../../redux/actions/account'

class SignIn extends React.Component {

  constructor(props) {
    super(props)
    this.state = {
      error: "",
      isLogin: true,
      email: "",
      password: "",
      hasRequestedReset: false
    }
  }

  componentDidMount() {
    this._isMounted = true
    if(!this.props.account.status) {
      this.props.setLoading(true)
      this.props.fetchAccountInfo().then(() => {
        this.props.setLoading(false)
      })
    }
  }

  componentWillUnmount() {
    this._isMounted = false
  }

  submit = (e) => {
    e.preventDefault()
    this.props.setLoading(true)
    if (!this.state.email || !this.state.password) {
      this.setState({
        error: "All fields are required",
      })
      return
    }
    this.props.checkEmail(this.state.email).then(({ success, data, error }) => {
      //no more actions if no data or invalid email
      if(error && (!data || data.email_invalid)) {
        this.props.setLoading(false)
        this.setState({ error })
      } else {
        if(data && !data.has_password) {
          this.props.requestPasswordReset(this.state.email).then(() => {
            this.props.setLoading(false)
            this.setState({
              error: "Your account does not have a password! We just sent you an email to set a password. Please set it before trying to sign in."
            })
          })
        } else {
          const action = this.state.isLogin
            ? this.props.logIn({
              email: this.state.email,
              password: this.state.password
            })
            : this.props.createNewAccount({
              email: this.state.email,
              password: this.state.password
              //TODO: do we want to add any tracking info here that indicates that we are
              //getting an account creation from the docs?
            })
          action.then(({ success, error }) => {
            this.props.setLoading(false)
            if(this._isMounted) {
              this.setState({
                error: success ? "" : error
              })
              if(success) this.props.submitSuccessCb()
            }
          })
        }
      }
    })
  }

  toggleTrue = () => this.setState({ isLogin: true })
  toggleFalse = () => this.setState({ isLogin: false })

  toggleState = (isLogin) => isLogin
    ? this.toggleTrue
    : this.toggleFalse

  isSubmitDisabled = () => {
    return this.props.loading.isLoading || !this.state.email || !this.state.password
  }

  setEmail = (e) => {
    this.setState({ email: e.target.value, hasRequestedReset: false })
  }

  setPassword = (e) => {
    this.setState({ password: e.target.value })
  }

  resetPasswordHandler = () => {
    this.setState({ hasRequestedReset: true })
    this.props.requestPasswordReset(this.state.email)
  }

  render() {
    const { className = "sign-in-form", loading } = this.props
    const { error, isLogin, email, password } = this.state
    return (
      <div className={`${className}__main`}>
        { loading.isLoading && <LoadingSpinner /> }
        { !loading.isLoading && <form
          onSubmit={this.submit}
          className={`${className}__main-form`}>
          { error && <div className={`${className}__error`}>
            { error }
          </div> }

          <div className={`${className}__toggle-type`}>
            <button
              type="button"
              onClick={this.toggleState(true)}
              className={ `${className}__toggle-btn
                ${ isLogin ? 'selected' : 'unselected' }
                ${ loading.isLoading ? `disabled` : '' }` }
              disabled={loading.isLoading}
            >Log in</button>
            <button
              type="button"
              onClick={this.toggleState(false)}
              className={ `${className}__toggle-btn
                ${ !isLogin ? 'selected' : 'unselected'}
                ${ loading.isLoading ? `disabled` : '' }` }
              disabled={loading.isLoading}
            >Sign up</button>
          </div>
          
          <div className={`${className}__fields`}>
            <Email
              className={className}
              email={email}
              setEmail={this.setEmail}
            />
            <Password
              className={className}
              password={password}
              setPassword={this.setPassword}
            />
            { this.state.email && <p className={`${className}__password-reset`}>
              Forgot password?
              <button 
                type="button" 
                onClick={this.resetPasswordHandler} 
                className={`${className}__password-reset__btn`} >Send email to reset</button>
            </p> }
          </div>

          <div className={`${className}__submit`}>
            <button
              className={`${className}__submit__btn${ this.isSubmitDisabled() ? ' disabled' : '' }`}
              disabled={this.isSubmitDisabled()}
              type="submit"
            >
              { isLogin ? 'Log in' : 'Sign up' }
          </button>
          </div>
        </form> }
      </div>
    )
  }
}

const Email = ({ className, email, setEmail }) => {
  return (<div className={`${className}__email`}>
    <input
      type="email"
      name="email"
      placeholder="email"
      value={email}
      onChange={setEmail}
    />
  </div>)
}

const Password = ({ className, password, setPassword }) => {
  return (<div className={`${className}__password`}>
    <input
      type="password"
      name="password"
      placeholder="Password"
      value={password}
      onChange={setPassword}
    />
  </div>)
}

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
  loading: state.loading,
  account: state.account
})

const mapDispatchToProps = dispatch => ({
  setLoading: (isLoading) => dispatch(setLoading(isLoading)),
  logIn: (credentials) => dispatch(accountActions.logIn(credentials)),
  createNewAccount: (account) => dispatch(accountActions.createNewAccount(account)),
  requestPasswordReset: (email) => dispatch(accountActions.requestPasswordReset(email)),
  checkEmail: (email) => dispatch(accountActions.checkNewEmail(email)),
  fetchAccountInfo: () => dispatch(accountActions.fetchAccountInfo()),
})

export default connect(mapStateToProps, mapDispatchToProps)(SignIn)
