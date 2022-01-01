import React from 'react'
import { connect } from 'react-redux'
import { Link, Route, Switch, Redirect } from 'react-router-dom'
import { replace } from 'connected-react-router'
import Helmet from 'react-helmet'

import './assets/settings.css'

import AccountGate from './components/AccountGate'
import Header from '../../components/Header'
import ScrollToTop from '../../components/ScrollToTop'

import Account from './Account'

const MenuLink = ({ label, to, exact }) =>
  <Route
    path={to}
    exact={exact}
    children={({ match }) =>
      <li className={ match ? "selected" : ""}>
        <Link to={to}>
          { label }
        </Link>
      </li>
  }/>

class Settings extends React.Component {
  // Remove this redirect soon
  componentDidMount() {
    const { section, replace } = this.props
    if (section) {
      replace(`/settings/${section}`)
      return null
    }
  }

  render() {
    const { match } = this.props
    return <AccountGate>
      <div className="settings__wrapper">
        <Header type="app" />
        <div className="settings">
          <ScrollToTop/>
          <Helmet>
            <title>Kite Settings</title>
          </Helmet>
          <h1>Settings</h1>
          <div className="settings-container">
            <ul className="settings-nav">
              <MenuLink label="Account" to="/settings/account"/>
            </ul>

            <div className="settings-content">
              <Switch>
                <Route path={`${match.url}/account`} component={Account}/>
                <Redirect to={`${match.url}/account`}/>
              </Switch>
            </div>
          </div>
        </div>
      </div>
    </AccountGate>
  }
}

function mapStateToProps(state, ownProps) {
  return {
    section: ownProps.location.hash.replace("#", ""),
    match: ownProps.match,
  }
}

const mapDispatchToProps = dispatch => ({
  replace: location => dispatch(replace(location)),
})

export default connect(mapStateToProps, mapDispatchToProps)(Settings)
