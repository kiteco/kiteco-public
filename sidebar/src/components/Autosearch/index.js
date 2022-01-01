import React from 'react'
import { connect } from 'react-redux'
import { push } from 'react-router-redux'

import { notify, dismiss } from '../../store/notification'

// Autosearch pays attention to the autosearchId property and will push to that page if autosearch is enabled
class Autosearch extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      notificationId: null,
    }
  }

  componentDidUpdate(props) {
    const { id: prevId, enabled: prevEnabled } = props
    const { id: nextId, enabled: nextEnabled, push, notify, dismiss } = this.props
    const { notificationId } = this.state
    if (
      nextId !== ""
      && prevId !== nextId
    ) {
      if (nextEnabled) {
        // autosearch the next id
        push(`/docs/${nextId}`)
      } else if (notificationId === null) {
        // send out a notification but only if we haven't yet notified
        const { data } = notify({ id: 'autosearch', component: 'autosearch', docsOnly: true })
        this.setState({
          notificationId: data.id,
        })
      }
    }
    // if we enable autosearch, dismiss the notification
    if (
      prevEnabled === false
      && nextEnabled === true
    ) {
      if (nextId) {
        push(`/docs/${nextId}`)
      }
      if (notificationId !== null) {
        dismiss(notificationId)
        this.setState({
          notificationId: null,
        })
      }
    }
  }

  render() {
    return null
  }
}

const mapDispatchToProps = dispatch => ({
  push: params => dispatch(push(params)),
  notify: params => dispatch(notify(params)),
  dismiss: id => dispatch(dismiss(id)),
})

const mapStateToProps = (state, ownProps) => ({
  enabled: state.search.autosearchEnabled,
  id: state.search.id,
})

export default connect(mapStateToProps, mapDispatchToProps)(Autosearch)
