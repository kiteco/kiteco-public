import React from 'react'
import { Link } from 'react-router-dom'
import { connect } from 'react-redux'
import { push } from 'connected-react-router'

import * as accountActions from '../../redux/actions/account'
import { GET } from '../../redux/actions/fetch'
import { Domains } from '../../utils/domains'

import Search from './components/Search'
import Settings from './components/Settings'
import TrialPeriod from './components/TrialPeriod'
import DownloadButton from '../DownloadButton'

import { isUseragentMobile, navigatorOs } from "../../utils/navigator"

import './assets/header.css'

/*
* todo rework this class to be more structured. Especially class management
* */

const os = navigatorOs()

class Header extends React.Component {
  componentDidMount() {
    if (!this.props.status) {
      this.props.fetchAccountInfo()
    }
  }

  render() {
    const { type, status, className, downloadButton, expandedLogo, downloadButtonText } = this.props
    const logoClass = (
      className &&
      className.includes('header__dark')
    ) ? 'logo__light' : 'logo__dark'
    if (type === "root") {
      return <RootHeader
        className={this.props.className}
        logoClass={logoClass}
        status={this.props.status}
        downloadButton={downloadButton}
        downloadButtonText={downloadButtonText}
      />
    }
    else if (
      type === "app" &&
      status === "logged-in"
    ) {
      return <AppHeader
            plan={this.props.plan}
            get={this.props.get}
            push={this.props.push}
            logout={this.props.logout}
            account={this.props.account}
          />
    } else if (type === "logo") {
      return <LogoHeader/>
    } else if (
      status &&
      status !== "loading"
    ) {
      return <ContentHeader
            className={this.props.className}
            logoClass={logoClass}
            status={this.props.status}
            headerTitle={this.props.headerTitle}
            headerTitlePath={this.props.headerTitlePath}
            expandedLogo={expandedLogo}
          />
    } else {
      return null
    }
  }
}

const AppHeader = ({
  plan,
  get,
  push,
  logout,
  account,
}) => (
  <div className="header">
    <a href={`https://${Domains.WWW}`}>
      <div className="header__logo logo__dark header__logo--lower-opacity"/>
    </a>
    <div className="header__nav"/>
    { plan.status &&
      <TrialPeriod
        { ...plan }
      />
    }
    <Search get={get} push={push} />
    <Settings
      logout={logout}
      { ...account }
    />
  </div>
)

const LogoHeader = () =>
  <div className="header">
    <div className="header__logo logo__dark header__logo--lower-opacity"/>
  </div>

const RootHeader = ({
  logoClass,
  className,
  downloadButton,
  downloadButtonText
}) => <div className={`header header__relative ${className}`}>
  <div className="header__wrapper">
    <a href={`https://${Domains.WWW}`}>
      <div className={"header__logo--expanded expanded__" + logoClass}>
      </div>
    </a>
    <ul className="header__homepage__nav">
      { downloadButton && !isUseragentMobile() &&
        <DownloadButton
          className="header__download-button"
          text={
            (downloadButtonText && typeof downloadButtonText === 'string') ?
              downloadButtonText : (os === 'linux' ? 'Install Kite Free!' : 'Download Kite Free!')
          }
          subText=""
        />
      }
    </ul>
  </div>
</div>

const ContentHeader = ({
  headerTitle,
  headerTitlePath,
  logoClass,
  className,
  expandedLogo,
}) => <div className={`header header__relative ${className}`}>
  <div className="header__wrapper">
    <div className="header__logo-wrapper">
      <a href={`https://${Domains.WWW}`}>
        <div className={expandedLogo ? `header__logo--expanded expanded__${logoClass}` : `header__logo ${logoClass}`}>
        </div>
      </a>
    </div>
    {headerTitle &&
      <div className="header__title-wrapper">
        {headerTitlePath ?
          <Link to={headerTitlePath}>
            <div className='header__title'>{headerTitle}</div>
          </Link>
          :
          <div className='header__title'>{headerTitle}</div>
        }
      </div>
    }
    <div className="header__homepage__nav-wrapper">
      <ul className="header__homepage__nav">
        {!isUseragentMobile() &&
          <DownloadButton
            className="header__download-button"
            text={os === 'linux' ? 'Install Kite Free!' : 'Download Kite Free!'}
            subText=""
          />
        }
      </ul>
    </div>
  </div>
</div>

const mapStateToProps = (state, ownProps) => ({
  plan: state.account.plan,
  account: state.account.data,
  status: state.account.status,
  ...ownProps
})

const mapDispatchToProps = dispatch => ({
  get: params => dispatch(GET(params)),
  push: params => dispatch(push(params)),
  logout: redirect => dispatch(accountActions.logOut(redirect)),
  fetchAccountInfo: () => dispatch(accountActions.fetchAccountInfo()),
})

export default connect(mapStateToProps, mapDispatchToProps)(Header)
