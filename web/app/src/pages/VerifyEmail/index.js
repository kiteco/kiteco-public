import React from 'react'
import { connect } from 'react-redux'
import { Link } from 'react-router-dom'
import queryString from 'query-string'

import ScrollToTop from '../../components/ScrollToTop'
import * as account from '../../redux/actions/account'

import './assets/verify-email.css'

class VerifyEmail extends React.Component {

  constructor(props) {
    super(props)
    this.state = {
      success: false,
      error: "",
      data: "",
    }
  }

  componentDidMount() {
    this.verify()
  }

  verify = () => {
    this.props.verify({
      code: this.props.code,
      email: this.props.email,
    })
      .then(({ success, error="", data="" }) =>
        this.setState({ success, error, data })
      )
  }

  render() {
    return <div className="verify-email">
      <ScrollToTop/>
      <h1>Verify your email</h1>
      { this.state.error &&
        <p className="verify-email__error">
          { this.state.error }
        </p>
      }
      { this.state.success &&
        <p>Thank you for verifying your email! We hope you enjoy using Kite.</p>
      }
      { /* If we get any data back,
           assume that it is a token to set a user's password */
        this.state.data &&
        <Link
          className="verify-email__set-password"
          to={`/reset-password?token=${encodeURIComponent(this.state.data)}&email=${encodeURIComponent(this.props.email)}`}
        >
          Set your password
        </Link>
      }
    </div>
  }
}

const mapStateToProps = (state, ownProps) => ({
  ...queryString.parse(ownProps.location.search),
})

const mapDispatchToProps = dispatch => ({
  verify: submission => dispatch(account.verifyEmail(submission)),
})

export default connect(mapStateToProps, mapDispatchToProps)(VerifyEmail)
