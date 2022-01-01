import React from 'react'
import { connect } from 'react-redux'
import { push } from 'connected-react-router'

import * as stylePopup from '../../../../../../redux/actions/style-popup'
import * as signinPopup from '../../../../../../redux/actions/signin-popup'
import * as fetch from '../../../../../../redux/actions/fetch'
import { logOut } from '../../../../../../redux/actions/account'
import { setLoading } from '../../../../../../redux/actions/loading'
import { navigatorOs } from "../../../../../../utils/navigator"

import Search from '../util/Search'
import { getTrackingParamsFromURL } from '../../../../../../utils/analytics'

const os = navigatorOs()

class Header extends React.Component<any> {
  logOutWrapper = () => {
    this.props.setLoading(true)
    this.props.logOut().then(() => {
      this.props.setLoading(false)
    })
  }

  isNotRoot = () => {
    return this.props.location.pathname !== '/docs' && this.props.location.pathname !== '/docs/'
  }

  getTrackingParams(): string {
    const clickCTA: string = 'navbar';

    return getTrackingParamsFromURL('python/docs/', clickCTA)
      || getTrackingParamsFromURL('python/answers/', clickCTA)
      || getTrackingParamsFromURL('python/examples/', clickCTA);
  }

  render() {
    const { account } = this.props
    const loggedIn = account.status === 'logged-in'
    return (
      <header>
        <div className='brand' onClick={() => window.open("/", "_blank")}>
          <div className='logo' />
          <div className='tagline'>Your programming copilot</div>
        </div>
        {this.isNotRoot() && <Search get={this.props.get} push={this.props.push} />}
        <div className='status'>
          <button
            onClick={() => window.open(`/download${this.getTrackingParams()}`, "_blank")}
            className='install-kite-app-promo'
          >
            {os === 'linux' ? "Install Kite for Free!" : "Download Kite for Free!"}
          </button>
          {loggedIn && <button className='settings-button' onClick={() => { window.open("/settings", "_blank") }} />}
        </div>
        <div className='style-popup-button' onClick={this.props.toggleStylePopup(true)} />
      </header>
    );
  }
}

const mapDispatchToProps = (dispatch: any) => ({
  toggleStylePopup: (show: any) => dispatch(stylePopup.toggleStylePopup(show)),
  toggleSigninPopup: (show: any) => dispatch(signinPopup.toggleSigninPopup(show)),
  get: (params: any) => dispatch(fetch.GET(params)),
  push: (params: any) => dispatch(push(params)),
  logOut: () => dispatch(logOut()),
  setLoading: (isLoading: boolean) => dispatch(setLoading(isLoading))
})

const mapStateToProps = (state: any, ownProps: any) => ({
  ...ownProps,
  account: state.account,
  location: state.router.location,
})

export default connect(mapStateToProps, mapDispatchToProps)(Header)
