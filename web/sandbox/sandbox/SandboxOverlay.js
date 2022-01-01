import React from 'react'
import { connect } from 'react-redux'

import './sandbox-overlay.css'
import { focusEditor } from '../actions/sandbox-editors'

class SandboxOverlay extends React.PureComponent {

  edit = (e) => {
    e.stopPropagation()
    this.props.focusEditor(this.props.editorId)
    this.props.clickCb && this.props.clickCb()
  }

  editInPlace = (e) => {
    e.stopPropagation()
    this.props.editScripting()
    this.props.focusEditor(this.props.editorId)
  }

  render () {
    return <div className={`sandbox-overlay ${this.props.cursorClass ? `${this.props.cursorClass}` : ''}`}>
      <div className="sandbox-overlay__buttons">
        <div 
          className="sandbox-overlay__button"
          onClick={this.edit}
        >
          Let me try typing!
        </div>
      </div>
    </div>
  }
}

const mapDispatchToProps = dispatch => ({
  focusEditor: editorId => dispatch(focusEditor(editorId)),
})

export default connect(null, mapDispatchToProps)(SandboxOverlay)