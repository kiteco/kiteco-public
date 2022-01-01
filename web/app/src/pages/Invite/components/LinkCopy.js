import React from 'react'

import { track } from '../../../utils/analytics'

class LinkCopy extends React.Component {

  constructor(props) {
    super(props)
    this.state = {
      copied: false,
    }
  }

  componentDidMount() {
    this.copy()
  }

  highlight = () => {
    if (this.codeInput && this.codeInput.select) {
      this.codeInput.select()
      track({
        event: "webapp: invite: invite link highlighted",
      })
    }
  }

  copy = () => {
    if (
      this.codeInput &&
      this.codeInput.select &&
      this.codeInput.blur
    ) {
      this.codeInput.select()
      document.execCommand('copy')
      this.codeInput.blur()

      track({
        event: "webapp: invite: invite link copied",
      })
      this.setState({
        copied: true,
      })
      setTimeout(() => this.setState({ copied: false }), 3000)
    }
  }

  render() {
    const { copied } = this.state
    const { referralCode } = this.props
    const url = `${window.location.origin}/ref/${referralCode}`
    return <div className="invite__link">
      <input
        className="invite__code__input"
        value={url}
        readOnly
        ref={i => this.codeInput = i}
        onClick={this.highlight}
      />
      <div
        onClick={this.copy}
        ref={i => this.copyButton = i}
        className={"invite__button " + (copied ? 'invite__button--copied' : '')}
      >
      </div>
    </div>
  }
}

export default LinkCopy
