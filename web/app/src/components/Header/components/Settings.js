import React from 'react'
import ReactDOM from 'react-dom'
import { Link } from 'react-router-dom'

import '../assets/settings.css'

class Settings extends React.Component {
  constructor(props) {
    super(props)

    this.state = {
      open: false,
    }
  }

  componentDidMount() {
    // Hacky solution to detect clicks outside of box
    document.addEventListener('click', this.handleClick, false)
  }

  componentWillUnmount() {
    // Hacky solution to detect clicks outside of box
    document.removeEventListener('click', this.handleClick, false)
  }

  // Hacky solution to detect clicks outside of box
  handleClick = (e) => {
    if (!ReactDOM.findDOMNode(this).contains(e.target)) {
      this.setState({
        open: false,
      })
    }
  }

  // toggle modal visibility
  toggleShow = () => {
    this.setState({
      open: !this.state.open,
    })
  }

  close = () => {
    this.setState({
      open: false,
    })
  }

  logout = (e) => {
    e.preventDefault()
    this.close()
    this.props.logout(true)
  }

  render() {
    const { light } = this.props
    return (
      <div className="header__settings">
        <button
          onClick={this.toggleShow}
          className={`header__settings__icon ${light ? "header__settings__icon--light" : ""}`}
        >
        </button>
        <div
          className={`
            header__settings__modal
            ${this.state.open ? "": "header__settings__modal--closed"}
          `}
        >
          <div className="header__settings__description">
            Signed in as
            <span className="header__settings__email">
              { this.props.email }
            </span>
          </div>
          {/* Note(Daniel): Taking out referral link until we figure out what to do with Pro */}
          {/* { !ENTERPRISE &&
            <div className="header__settings__actions">
              <Link to="/invite" onClick={this.close}>
                Share Kite and earn free Kite Pro 🎁
              </Link>
            </div>
          } */}
          <div className="header__settings__actions">
            <Link to="/settings" onClick={this.close}>
              Settings
            </Link>
            {/* eslint-disable-next-line */}
            <a href="#" onClick={this.logout}>
              Sign out
            </a>
          </div>
        </div>
      </div>
    )
  }
}

export default Settings
