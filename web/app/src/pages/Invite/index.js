import React from 'react'
import { connect } from 'react-redux'
import Helmet from 'react-helmet'
import queryString from 'query-string'

import ScrollToTop from '../../components/ScrollToTop'
import Header from '../../components/Header'

import TwitterLink from './components/TwitterLink'
import LinkCopy from './components/LinkCopy'
import MessageCopy from './components/MessageCopy'
import GmailImport from './components/GmailImport'

import * as account from '../../redux/actions/account'
import { Domains } from '../../utils/domains'

import {
  wrapGoogleLoad,
  initGoogle,
} from '../../utils/google'
import { track } from '../../utils/analytics'

import './assets/invite.css'
import coworkers from './assets/coworkers.svg'

class Invite extends React.Component {

  constructor(props) {
    super(props)
    this.state = {
      showGmailImport: false,
      showSlackMessage: false,
      showLinkCopy: false,
      showTextCopy: false,
    }
  }

  componentDidMount() {
    initGoogle({ props: this.props })
    const { tpt } = this.props
    track({
      event: "webapp: invite page loaded",
      props: {
        tpt,
      },
    })
  }

  componentDidUpdate(prevProps) {
    initGoogle({ props: this.props, prevProps })
  }

  inviteFromGmailClicked = () => {
    track({
      event: "webapp: invite: invite from gmail clicked"
    })
    this.setState({ showGmailImport: true })
  }

  slackCoworkersClicked = () => {
    track({
      event: "webapp: invite: slack coworkers clicked"
    })
    this.setState({ showSlackMessage: true })
  }

  copyReferralLinkClicked = () => {
    track({
      event: "webapp: invite: copy referral link clicked"
    })
    this.setState({ showLinkCopy: true })
  }

  moreActionsClicked = () => {
    track({
      event: "webapp: invite: more actions clicked"
    })
    this.setState({ showTextCopy: true })
  }

  openSlack = () => {
    window.open('https://slack.com/app_redirect?channel=random')
    track({event: "webapp: invite: slack opened"})
  }

  render() {
    const { showGmailImport, showLinkCopy, showSlackMessage, showTextCopy } = this.state
    const { referralCode, email } = this.props

    // This was used in the referral link CTA, which we've removed below
    // const url = `${window.location.origin}/ref/${referralCode}`

    return (
      <div className="invite-wrapper">
        <ScrollToTop/>
        <Header className="header__dark invite-header" type="root" downloadButton={false} />
        <div className="invite">
          <Helmet>
            <title>Invite friends and coworkers to Kite!</title>
          </Helmet>
          <h1 className="invite__tagline invite__appear">
            Invite friends &amp; coworkers!
          </h1>
          <img
            className="invite__header-logo"
            src={coworkers}
            alt="Invite friends and coworkers"
          />
          <div className="invite__options invite__appear">
            <div className="invite-option gmail-option">

              { showGmailImport ?
                <div className="invite__gmail__wrapper invite__code invite__appear">
                  <p className="invite__top-text">Invite from your Gmail</p>
                  <GmailImport email={email}/>
                </div> :
                <div className="invite__show-gmail__wrapper">
                  <button
                    className="invite__options__button"
                    onClick={this.inviteFromGmailClicked}
                  >
                    <div className="invite__options__gmail">
                      Invite from your Gmail
                    </div>
                  </button>
                </div>
              }
              <p className='invite-option__bottom-text'>We never store your contacts</p>
            </div>
            <div className="invite-option">
              { showLinkCopy && referralCode &&
              <div className="invite__code invite__appear">
                <LinkCopy
                  referralCode={referralCode}
                />
              </div>
              }
              <TwitterLink
                onClick={() => track({ event: "webapp: invite: twitter clicked" })}
                className="invite__options__button"
                message={`Kite is the best autocompletions engine available for Python, powered by AI. Check it out — It's free! https://${Domains.PrimaryHost} @kitehq`}
              >
                <div className="invite__options__twitter">
                  Share on twitter
                </div>
              </TwitterLink>
            </div>
            <div className="invite-option">
              { showSlackMessage ?
                <div className="invite__code invite__appear">
                  <p className="invite__top-text">Slack your coworkers</p>
                  <MessageCopy
                    className='slack__copy-message'
                    eventPlace='slack'
                    buttonText="Copy Message"
                    buttonTextCopied="Copied! Click Again to Open Slack."
                    onSecondClick={this.openSlack}
                    messageBlock={
                      <div>
                        Hey team! I've been using Kite, an AI-powered autocompletions engine for Python & JavaScript, to boost my productivity. You can get it for free at https://{Domains.PrimaryHost}.
                        <br/><br/>
                        Check it out in action!
                        https://giphy.com/gifs/python-kite-Y4c5RiOeUGfcX4mgee
                      </div>
                    }
                    messageValue={`Hey team! I've been using Kite, an AI-powered autocompletions engine for Python & JavaScript, to boost my productivity. You can get it for free at https://${Domains.PrimaryHost}.

Check it out in action!
https://giphy.com/gifs/python-kite-Y4c5RiOeUGfcX4mgee`}
                  />
                </div> :
                <button
                  className="invite__options__button"
                  onClick={this.slackCoworkersClicked}
                >
                  <div className="invite__options__slack">
                    Slack your coworkers
                  </div>
                </button>
              }
            </div>
            <div className="invite-option">
              {showTextCopy ?
                <div className="invite__code invite__appear">
                  <p className="invite__top-text">Share Kite</p>
                  <MessageCopy
                    eventPlace='share button'
                    buttonText="Copy Message"
                    buttonTextCopied="Message Copied - Share Kite Now!"
                  />
                </div> :
                <div
                  onClick={this.moreActionsClicked}
                  className='invite__more-options'
                >
                  More options
                </div>
              }
            </div>
          </div>
        </div>
      </div>
    )
  }
}

const mapStateToProps = (state, ownProps) => ({
  tpt: queryString.parse(ownProps.location.search).tpt,
  // TODO: Bring this back when the referral homepage works again
  // referralCode: state.account.planDetails.referral_code,
})

const mapDispatchToProps = dispatch => ({
  email: sub => dispatch(account.inviteEmails(sub)),
})

export default wrapGoogleLoad(connect(mapStateToProps, mapDispatchToProps)(Invite))
