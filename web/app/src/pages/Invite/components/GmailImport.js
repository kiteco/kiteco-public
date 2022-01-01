import React from 'react'
import ReactDOM from 'react-dom'
import Fuse from 'fuse.js'
import { connect } from 'react-redux'

import {
  signIn,
  getContacts,
} from '../../../utils/google'
import { track } from '../../../utils/analytics'

import '../assets/gmail-import.css'

/**
 * GmailImport pops up a modal window and asks the user to sign
 * into their Google account.
 *
 * NOTE: this component must be a child of a component that is
 * wrapped inside utils/google/wrapGoogle with the google client
 * e.g. components/Invite/index.js
 */
class GmailImport extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      emails: null,
      error: null,
    }
  }

  componentDidMount() {
    this.getContacts()
  }

  getContacts = () => {
    signIn()
      .then(getContacts)
      .then(({ success, data, error }) => {
        if (success) {
          const noTitle = data.filter(e => !e.title)
          const withTitle = data.filter(e => e.title)
          const sort = field => (a, b) => {
            if ( a[field] < b[field] ) return -1
            if ( b[field] < a[field] ) return 1
            return 0
          }
          withTitle.sort(sort("title"))
          noTitle.sort(sort("email"))
          this.setState({ emails: [...withTitle, ...noTitle]})
        } else {
          this.setState({ error })
        }
      })
      .catch(error => this.setState({ error }))
  }

  refresh = () => {
    this.setState({
      emails: null,
      error: null,
    })
    this.getContacts()
  }

  render() {
    const { emails, error } = this.state
    const { email } = this.props
    return <div className="invite__gmail">
      { emails !== null &&
        <EmailSelect
          emails={emails}
          email={email}
        />
      }
      { emails === null && error === null &&
        <div className="invite__gmail__loading">
          Loading your contacts...
        </div>
      }
      { error &&
        <div className="invite__gmail__error">
          <p>There was an error connecting to Gmail.</p>
          <div className="invite__gmail__error-btn" onClick={this.refresh}>Try Again</div>
        </div>
      }
    </div>
  }
}

class EmailSelectComponent extends React.Component {
  fuse = null
  list = null

  constructor(props) {
    super(props)
    this.state = {
      name: props.name || "",
      search: "",
      selected: {},
      success: false,
      sending: false,
      error: null,
    }
  }

  componentDidMount() {
    // TODO(Daniel): This is a hack since it assumes the parent component
    // is the invite component.
    track({
      event: "webapp: invite: gmail contacts loaded"
    })

    const { emails } = this.props
    this.fuse = new Fuse(emails, { keys: ['email', 'title'] })
  }

  setName = e => this.setState({name: e.target.value})

  search = e => {
    this.setState({search: e.target.value})
    const list = ReactDOM.findDOMNode(this.list)
    list.scrollTop = 0
  }

  select = email => e => this.setState(state => {
    const { selected } = state
    return {
      ...state,
      selected: {
        ...selected,
       [email]: !selected[email],
      },
    }
  })

  send = () => {
    this.setState({ sending: true })
    const { email } = this.props
    const { selected, name } = this.state
    const selectedEmails = Object.keys(selected).filter(e => selected[e])
    if (!name) {
      this.setState({
        error: "name required",
        sending: false,
      })
      return
    }
    if (selectedEmails.length > 0) {
      // TODO(Daniel): This is a hack since it assumes the parent component
      // is the invite component.
      track({
        event: "webapp: invite: gmail invites attempted",
        props: {
          num_invites: selectedEmails.length
        }
      })
      email({
        name,
        emails: selectedEmails,
      }).then(({ success, error }) => {
        if (!success) {
          this.setState({ error, sending: false })
        } else {
          // TODO(Daniel): This is a hack since it assumes the parent component
          // is the invite component.
          track({
            event: "webapp: invite: gmail invites sent",
            props: {
              num_invites: selectedEmails.length
            }
          })
          this.setState({ success, sending: false })
        }
      })
    } else {
      this.setState({
        error: "no contacts selected",
        sending: false,
      })
    }
  }

  render() {
    const { emails } = this.props
    const {
      name,
      search,
      selected,
      error,
      success,
      sending,
    } = this.state
    const searchedEmails = search ? this.fuse.search(search) : emails
    const numberSelected = Object.values(selected).filter(x => x).length

    if (success) {
      return <div className="invite__gmail__form--success">
        <div className="invite__gmail__form--success__inner">
          Invites sent!
        </div>
      </div>
    }

    return <div className="invite__gmail__form">
      <div className="invite__gmail__from">
        <input
          className="invite__gmail__from__input"
          type="text"
          value={name}
          placeholder="Your name"
          onChange={this.setName}
        />
      </div>

      <input
        className="invite__gmail__search"
        type="text"
        value={search}
        placeholder="Search your contacts"
        onChange={this.search}
      />

      <ul
        className="invite__gmail__emails"
        ref={ref => this.list = ref}
      >
        { searchedEmails.map(e =>
          <li
            key={ e.email }
            className="invite__gmail__list-item"
            onClick={this.select(e.email)}
          >
            <div className="invite__gmail__checkbox">
              <input
                type="checkbox"
                checked={selected[e.email] || false}
                readOnly
              />
              <span className="invite__gmail__checkmark" />
              <div
                className="invite__gmail__name"
                title={e.title}
              >
                { e.title }
              </div>
            </div>
            <div
              className="invite__gmail__email"
              title={e.email}
            >
              { e.email }
            </div>
          </li>
        )}
        { searchedEmails.length === 0 &&
          <li className="invite__gmail__no-emails">
            No contacts
          </li>
        }
      </ul>

      <div className="invite__gmail__buttons">
        <button
          onClick={this.send}
          className={`
            invite__gmail__submit
            ${(numberSelected > 0 && !sending) ? "invite__gmail__submit--active": ""}
          `}
        >
          Invite { numberSelected || "" } friend{ numberSelected === 1 ? "" : "s" }
        </button>
      </div>

      { error &&
        <div className="invite__gmail__error--small">
          <p>{ error }</p>
        </div>
      }
    </div>
  }
}

const EmailSelect = connect((state, props) => ({
  name: state.account.data ? state.account.data.name : null,
  ...props,
}))(EmailSelectComponent)

export default GmailImport
