import React from 'react'
import { connect } from 'react-redux'

// utils
import { track } from '../../../utils/analytics'
import { emailVerificationPath } from '../../../utils/urls'
import {createJson} from "../../../utils/fetch";

// actions
import { newsletter } from '../../../redux/actions/account'
import { POST } from "../../../redux/actions/fetch";

// assets
import './newsletter.css'

class Newsletter extends React.Component {
  constructor(props) {
    super(props)
    const { email } = this.props
    this.state = {
      email: email || "",
      complete: false,
      error: "",
    }
  }

  changeEmail = event => {
    this.setState({
      email: event.target.value,
      error: "",
    })
  }

  submit = event => {
    event.preventDefault()
    const { add, post } = this.props
    const { email } = this.state
    if (email.trim() === "") {
      this.setState({
        error: "Your email is required",
      })
      return
    }
    post({
      url: emailVerificationPath,
      options: createJson({
        email,
      }),
    }).then(res => {
      if(res.success) {
        if (res.data.verified) {
          track({
            event: "website: newsletter signup",
            props: {
              referrer: window.location.href,
            }
          })
          add({
            email,
            newsletter: true,
            channel: "newsletter",
          })
          this.setState({
            complete: true,
            error: "",
          })
        } else {
          this.setState({
            error: "Email is invalid",
          })
        }
      } else {
        this.setState({
          error: "No response from server, try later",
        })
      }
    })
  }

  render() {
    const { email, error, complete } = this.state
    if (complete) {
      return <Completed/>
    }
    return <View
      email={email}
      changeEmail={this.changeEmail}
      submit={this.submit}
      error={error}
    />
  }
}

const Completed = () => <div>
  Thanks for signing up!
</div>

const View = ({ email, changeEmail, submit, error }) =>
  <div className="newsletter">
    <form onSubmit={submit} className="newsletter__new">
      <input
        type="email"
        className={`newsletter__email`}
        value={email}
        placeholder="email"
        onChange={changeEmail}
      />
      <button className="newsletter__submit" type="submit" alt="send">
        Join the tribe
      </button>
    </form>
    { error &&
      <div className="newsletter__error">
        { error }
      </div>
    }
  </div>

const mapStateToProps = (state, props) => ({
})

const mapDispatchToProps = dispatch => ({
  add: submission => dispatch(newsletter(submission)),
  post: params => dispatch(POST(params)),
})

export default connect(mapStateToProps, mapDispatchToProps)(Newsletter)
