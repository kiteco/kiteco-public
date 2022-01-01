import { connect } from 'react-redux'

import React from 'react'
import Helmet from 'react-helmet'
import { Route, Redirect, Switch } from 'react-router-dom'

import Nav from './components/Nav'
import Header from './components/Header'
import Footer from './components/Footer'
import Page from './components/Page'
import Root from './Root'
import StylePopup from './components/StylePopup'
import SignInPopup from './components/SignInPopup'
import StyleWatcher from './StyleWatcher'

import '../../assets/index.css'

const VALID_LANGUAGES = 'python|js'

class Docs extends React.Component {

  render() {
    const {match} = this.props
    return (
      <div className='Documentation'>
        <StyleWatcher/>
        <Helmet>
          <title>Kite Docs</title>
        </Helmet>
        {/* TODO(dane): uncomment below once this is merged: https://github.com/Pomax/react-onclickoutside/pull/318 */}
        <StylePopup disableOnClickOutside={false/* !this.props.stylePopup.visible */} />
        <SignInPopup disableOnClickOutside={false/* !this.props.signinPopup.visible */} />
        <Header />
        <Nav />
        <Switch>
          <Route
            exact
            path={`${match.url}/examples/:exampleId`}
            render={(props) => <Page {...props} language={match.params.language}/>}
          />
           <Route
            exact
            path={`${match.url}/examples/:exampleId/:exampleTitle`}
            render={(props) => <Page {...props} language={match.params.language}/>}
          />
          <Route
            path={`${match.url}/docs/:valuePath+`}
            render={(props) => <Page {...props} language={match.params.language}/>
            }
          />
          <Route
            path={`/:language(${VALID_LANGUAGES})`}
            render={() => <Root />}
          />
          <Route
            exact
            path={`${match.url}/:garbage+`}
            render={() => <Redirect to={`/${match.params.language}`} />}
          />
        </Switch>
        <Footer />
      </div>
    )
  }
}

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
  stylePopup: state.stylePopup,
  signinPopup: state.signinPopup
})

export default connect(mapStateToProps)(Docs)
