import React from 'react'

// assets
import './assets/download-button.css'

// utils
import { track } from '../../utils/analytics'
import { event } from '../../utils/ga'
import { navigatorOs } from '../../utils/navigator'
import { FormButton } from './FormButton'

export const sendDownloadAnalytics = (props = {}) => {
  // get current time
  let now = new Date()
  const time = now.getTime()

  // set 'bannerExpireTime' cookie value and time delay on a year
  const expireTime = time + 1000 * 60 * 60 * 24 * 365
  now.setTime(expireTime)
  document.cookie = `bannerExpireTime=true;expires=${now.toUTCString()}`

  // analytics
  track({
    event: typeof props.eventText === 'string' ?
      `website: ${props.eventText}` :
      "website: user clicked download button",
    props: {
      ...props,
      referrer: window.location.href,
      windowScroll: window.pageYOffset,
    }
  })
  event({
    category: "Download",
    action: "click",
    label: window.location.href,
  })

}

class DownloadButton extends React.Component {

  clickDownload = () => {
    const { onClick } = this.props
    sendDownloadAnalytics(this.props)
    if ( onClick ) {
      onClick()
    }
  }

  render() {
    const {
      className,
      os = navigatorOs(),
      text,
      subText = "It's Free",
      to = "/download",
      disabled = false,
      newTab = false,
    } = this.props

    return (
      <FormButton
        modifiers={[disabled ? 'disabled' : undefined, 'big']}
        url={to}
        onClick={this.clickDownload}
        className={`download-button ${className}`}
        newTab={newTab}
      >
        <div className={`
            download-button__icon
            download-button__icon--${os}
          `}/>
        <div className="download-button__text-elements">
            <span className={`download-button__text`}>
              { text || (os === 'linux' ? "Install Kite Now" : "Download Kite Now") }
            </span>
          { subText &&
          <span className={`download-button__subtext`}>
                &nbsp;– { subText }
              </span>
          }
        </div>
      </FormButton>
    )
  }
}

export default DownloadButton
