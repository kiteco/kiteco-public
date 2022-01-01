import React from 'react'
import { connect } from 'react-redux'

import { focusEditor, hideHoverOverlay } from '../actions/sandbox-editors'

import './sandbox-hover-overlay.css'

class SandboxHoverOverlay extends React.Component {

  edit = (e) => {
    this.props.invalidatePlaybackPause()
    this.props.hideOverlay(this.props.editorId)
    this.props.focusEditor(this.props.editorId)
  }

  render() {
    return <div className="sandbox-hover-overlay" onClick={this.edit}>
      <div className="sandbox-hover-overlay__copy">
        Click anywhere to give Kite a fly!
      </div>
    </div>
  }
}

const mapDispatchToProps = dispatch => ({
  focusEditor: editorId => dispatch(focusEditor(editorId)),
  hideOverlay: editorId => dispatch(hideHoverOverlay(editorId)),
})

export default connect(null, mapDispatchToProps)(SandboxHoverOverlay)