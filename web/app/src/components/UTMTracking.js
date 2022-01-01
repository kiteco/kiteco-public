import React from 'react'
import { connect } from 'react-redux'
import { withRouter } from 'react-router'
import queryString from 'query-string'

import { SAVE_UTM } from '../redux/actions/promotions'

const utmParameters = [
  "utm_source",
  "utm_medium",
  "utm_campaign",
  "utm_term",
  "utm_content",
]

class UTMTracking extends React.Component {
  componentDidMount() {
    const { save, qs } = this.props
    const results = utmParameters.reduce((all, param) => ({
      ...all,
      [param]: qs[param],
    }), {})
    save(results)
  }
  render() {
    return null
  }
}

const mapStateToProps = (state, ownProps) => ({
  qs: queryString.parse(ownProps.location.search),
})

const mapDispatchToProps = dispatch => ({
  save: props => dispatch({ type: SAVE_UTM, props }),
})

export default withRouter(connect(mapStateToProps, mapDispatchToProps)(UTMTracking))
