import React from 'react'
import { connect } from 'react-redux'
import CSSTransitionGroup from 'react-transition-group/CSSTransitionGroup'

import './assets/notifications.css'

import * as notifications from "../../redux/actions/notifications"

class Notification extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      timeout: null,
    }
  }

  componentDidMount() {
    this.start()
  }

  dismiss = () => {
    this.stop()
    this.hide()
  }

  hide = () => {
    const { id, dismiss } = this.props
    dismiss(id)
  }

  stop =  () => {
    const { timeout } = this.state
    if (timeout) {
      clearTimeout(timeout)
    }
  }

  start = () => {
    const { timeout } = this.props
    if (timeout) {
      this.setState({
        timeout: setTimeout(this.hide, timeout),
      })
    }
  }

  render() {
    const { message, kind } = this.props
    return <div
      className={`notification notification--${kind}`}
      onMouseEnter={this.stop}
      onMouseLeave={this.start}
    >
      <div className="notification__message">
        { message }
      </div>
      <button
        className="notification__button"
        onClick={this.dismiss}
      >
        dismiss
      </button>
    </div>
  }
}

const Notifications = ({ notifications, dismiss }) =>
  <CSSTransitionGroup
    className="notification-holder"
    transitionName="notification__transition-"
    transitionEnterTimeout={500}
    transitionLeaveTimeout={500}
  >
    { notifications.map(n =>
      <Notification key={n.id} dismiss={dismiss} {...n} />
    )}
  </CSSTransitionGroup>

function mapStateToProps(state, ownProps) {
  return {
    ...state.notifications,
  }
}

const mapDispatchToProps = dispatch => ({
  dismiss: id => dispatch(notifications.hideNotification(id)),
})

export default connect(mapStateToProps, mapDispatchToProps)(Notifications)
