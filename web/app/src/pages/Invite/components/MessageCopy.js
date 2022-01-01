import React from 'react'

import { track } from '../../../utils/analytics'
import { Domains } from '../../../utils/domains'

class MessageCopy extends React.Component {
  state = {
    copied: false,
  }

  highlight = () => {
    if (this.codeInput && this.codeInput.select) {
      this.codeInput.select()
      track({
        event: `webapp: invite: ${this.props.eventPlace || ''} message highlighted`,
      })
    }
  }

  copy = () => {
    if (
      this.codeInput &&
      this.codeInput.select
    ) {
      this.codeInput.select()
      document.execCommand('copy')

      track({
        event: `webapp: invite: ${this.props.eventPlace || ''} message copied`,
      })
      this.setState({
        copied: true,
      })
    }
  }

  handleSecondClick = () => {
    if(typeof this.props.onSecondClick === 'function') {
      this.setState({ copied: false })
      this.props.onSecondClick()
    }
  }

  render() {
    const { copied } = this.state
    const { buttonText, buttonTextCopied, messageBlock, messageValue, className } = this.props

    return <div className={`invite__link${className ? ` ${className}` : ''}`}>
      <textarea
        className="invite__message--hidden"
        value={messageValue || `I've been using Kite, an AI-powered autocompletions engine for Python & JavaScript, to boost my productivity. You can get it for free at https://${Domains.PrimaryHost}.\r\n\r\nCheck it out in action! https://giphy.com/gifs/python-kite-Y4c5RiOeUGfcX4mgee`}
        readOnly
        ref={i => this.codeInput = i}
      />
      <div className="invite__code__input">
        {messageBlock ||
          <div>
            I've been using Kite, an AI-powered autocompletions engine for Python & JavaScript, to boost my productivity. You can get it for free at https://{Domains.PrimaryHost}. <br/><br/>
            Check it out in action!
            https://giphy.com/gifs/python-kite-Y4c5RiOeUGfcX4mgee
          </div>
        }
      </div>
      <div
        onClick={copied ? this.handleSecondClick : this.copy}
        ref={i => this.copyButton = i}
        className={"invite__button " + (copied ? 'invite__button--copied' : '')}
      >
        {!copied
          ? buttonText || "Copy message"
          : <div>{buttonTextCopied || 'Copied!'}</div>
        }
      </div>
    </div>
  }
}

export default MessageCopy
