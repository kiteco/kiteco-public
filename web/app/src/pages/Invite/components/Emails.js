import React from 'react'
import ReactTags from 'react-tag-autocomplete'
import { connect } from 'react-redux'

import { validateEmail } from '../../../utils/validation'
import { track } from '../../../utils/analytics'

import './assets/emails.css'

const TagComponent = ({ tag: { name, valid }, onDelete }) =>
  <div className={`
    invite__emails__tag-component
    ${ valid ? "" : "invite__emails__tag-component--invalid" }
  `}>
    { name }
    <button
      className="invite__emails__tag-component__delete"
      onClick={onDelete}
    >
      ✕
    </button>
  </div>

class Emails extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      name: props.name || "",
      emails: [],
      error: null,
      sending: false,
      success: false,
    }
  }

  setName = e => this.setState({name: e.target.value})

  deletion = i => {
    const { emails: oldEmails } = this.state
    const emails = [ ...oldEmails ]
    emails.splice(i, 1)
    this.setState({ emails })
  }

  addition = ({ name }) => {
    const { emails: oldEmails } = this.state
    const oldEmailArray = oldEmails.map(e => e.name)
    const emails = name.split(",")
      .reduce((prev, e) => {
        const et = e.trim()
        if (prev.includes(et) || !et ) {
          return prev
        }
        return [ ...prev, et ]
      }, [])
      .map((e, i) => ({ name: e, id: e, valid: validateEmail(e) }))
    const nodup = emails.filter(e => !oldEmailArray.includes(e.name))
    this.setState({
      emails: [...oldEmails, ...nodup],
    })
  }

  send = () => {
    const { email } = this.props
    const { emails, name } = this.state
    if ( !name ) {
      this.setState({
        error: "name required",
      })
      return
    }
    if ( emails.length > 0 ) {
      if (emails.filter(e => !e.valid).length > 0) {
        this.setState({
          error: "there are invalid emails in the list",
        })
        return
      }
      this.setState({ sending: true })
      const selectedEmails = emails.map(e => e.name)
      track({
        event: "webapp: invite: manual email invites attempted",
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
          track({
            event: "webapp: invite: manual email invites sent",
            props: {
              num_invites: selectedEmails.length
            }
          })
          this.setState({ success, sending: false })
        }
      })
    } else {
      this.setState({
        error: "no emails to send",
      })
    }
  }

  render() {
    const { name, emails, error, sending, success } = this.state
    if ( success ) {
      return <div className="invite__emails--success">
        <div className="invite__emails--success__inner">
          Invites sent!
        </div>
      </div>
    }
    return <div className="invite__emails">
      <div className="invite__emails__from">
        <label htmlFor="name">From</label>
        <input
          className="invite__emails__from__input"
          type="text"
          value={name}
          placeholder="Your name"
          onChange={this.setName}
        />
      </div>

      { error &&
        <div className="invite__emails__error">
          { error }
        </div>
      }

      <ReactTags
        tags={emails}
        handleDelete={this.deletion}
        handleAddition={this.addition}
        allowNew={true}
        tagComponent={TagComponent}
        placeholder="Email addresses"
        delimiters={[9, 13, 188]}
        autoresize={false}
        classNames={{
          root: "invite__emails__entry",
          selected: "invite__emails__selected",
          search: "invite__emails__search",
          searchInput: "invite__emails__search-input",
        }}
      />

      <p className="invite__emails__info">Press tab or comma after each email to complete it</p>

      <div className="invite__emails__buttons">
        <button
          onClick={this.send}
          className={`
            invite__emails__submit
            ${(!sending && emails.length > 0) ? "invite__emails__submit--active" : ""}
          `}
        >
          Send invites
        </button>
      </div>
    </div>
  }
}

const mapStateToProps = (state, props) => ({
  name: state.account.data ? state.account.data.name : null,
  ...props,
})

export default connect(mapStateToProps)(Emails)
