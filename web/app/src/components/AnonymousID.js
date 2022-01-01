import React from 'react'
import { connect } from 'react-redux'

import { RESET_ANONYMOUS_ID } from '../redux/actions/promotions'

class AnonymousID extends React.Component {
  componentDidMount() {
    const { anonymousID, resetAnonID } = this.props
    if ( !anonymousID ) {
      resetAnonID()
    }
  }
  render() {
    return null
  }
}

const mapStateToProps = (state, { identifier }) => ({
  anonymousID: state.promotions.anonymousID,
})

const mapDispatchToProps = dispatch => ({
  resetAnonID: () => dispatch({ type: RESET_ANONYMOUS_ID }),
})

export default connect(mapStateToProps, mapDispatchToProps)(AnonymousID)
