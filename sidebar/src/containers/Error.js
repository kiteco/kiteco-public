import React from 'react'
import { connect } from 'react-redux'

import '../assets/error.css'

class Error extends React.Component {
  render() {
    return (
      <div className={`
        global-error
        ${this.props.show ? "global-error--show" : "global-error--hide"}
      `}>
        { this.props.message }
      </div>
    )
  }
}

const mapStateToProps = (state, ownProps) => ({
  ...state.errors,
})

export default connect(mapStateToProps)(Error)
