import React from 'react'
import { connect } from 'react-redux'

import { focusEditor } from '../actions/sandbox-editors'

import './playback-buttons.css'

class PlaybackButtons extends React.PureComponent {
  
  edit = () => {
    this.props.invalidatePlaybackPause()
    this.props.focusEditor(this.props.editorId)
  }

  reset = () => {
    this.props.resetPlayback()
  }

  pause = () => {
    this.props.pausePlayback()
  }

  resume = () => {
    this.props.resumePlayback()
  }

  restart = () => {
    //maybe should be reset+start?
    this.props.restartPlayback()
  }

  isStopped = () => {
    return this.props.isPlaybackPaused()
      || this.props.isPlaybackReset()
      || this.props.isEditing()
  }

  render() {
    if(this.props.isDev) {
      return (
        <div className="playback-buttons--dev">
          <div 
            className="playback-buttons__button--dev"
            onClick={this.reset}
          >
            Reset
          </div>
          <div 
            className="playback-buttons__button--dev"
            onClick={this.isStopped() ? this.resume : this.pause}
          >
            { this.isStopped()
              ? "Play"
              : "Pause"
            }
          </div>
          <div 
            className="playback-buttons__button--dev"
            onClick={this.edit}
          >
            Edit
          </div>
          <div 
            className="playback-buttons__button--dev"
            onClick={this.restart}
          >
            &#10226; Restart
          </div>
        </div>
      )
    }
    return (
      <div className="playback-buttons">
        <div
          className="playback-buttons__button"
          onClick={this.restart}
        >&#10226;</div>
      </div>
    )
  }
}

const mapDispatchToProps = dispatch => ({
  focusEditor: (editorId) => dispatch(focusEditor(editorId)),
})

export default connect(null, mapDispatchToProps)(PlaybackButtons)