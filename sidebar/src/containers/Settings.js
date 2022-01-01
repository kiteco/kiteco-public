import React from 'react'
import { connect } from 'react-redux'
import { Route, Switch } from 'react-router-dom'

import Nav from './Nav'

import SettingsHome from './SettingsHome'
import Plugins from './Plugins'

import '../assets/home.css'

class Settings extends React.Component {
  render() {
    const { location, shouldBlur } = this.props
    return (
      <div className={`main ${shouldBlur ? 'main--blur' : ''}`}>
        <Nav path={location.pathname} />
        <Switch>
          <Route path="/settings/plugins" component={Plugins} />
          <Route render={props => <SettingsHome {...props} />} />
        </Switch>
      </div>
    )
  }
}

const mapStateToProps = (state, ownProps) => ({
  system: state.system,
  ...ownProps,
})

export default connect(mapStateToProps, null)(Settings)
