import React from 'react'
import { connect } from 'react-redux'
import { push } from 'connected-react-router'
import Helmet from 'react-helmet'

import Search from '../components/util/Search'
import * as fetch from '../../../../../redux/actions/fetch'
import './root.css'
import ScrollToTop from '../../../../../components/ScrollToTop'

class Root extends React.Component {
  render() {
    return <div className='Root'>
      <ScrollToTop/>
      <Helmet>
        <title>Kite Docs</title>
      </Helmet>
      <div className='Root__left-spacer'></div>
      <div className='Root__top-spacer'></div>
      <div className='Root__content'>
        <h3>Welcome to Kite Docs!</h3>
        <p>Use our Intelligent Search to find documentation for your favorite <span className='code'>python</span> packages</p>
        <p>We've tried to do the legwork of putting everything you need all in one easily searchable place</p>
        <p>So you can get back to what you (and we) love doing - creating awesome things</p>
        <Search get={this.props.get} push={this.props.push} placeholder="e.g. requests.api.get"/>
      </div>
    </div>
  }
}

const mapDispatchToProps = dispatch => ({
  //get: params => dispatch(GET(params)),
  push: params => dispatch(push(params)),
  get: params => dispatch(fetch.GET(params))
})

const mapStateToProps = (state, ownProps) => ({

})

export default connect(mapStateToProps, mapDispatchToProps)(Root)
