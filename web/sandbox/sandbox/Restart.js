import React from 'react'

import './playback-buttons.css'

class RestartButton extends React.PureComponent {

  restart = () => {
    clearTimeout(this.restartHandle)
    this.restartHandle = setTimeout(() => {
      this.props.restartPlayback()
    }, 150)
  }

  render() {
    return (
      <div 
        className="playback-buttons" 
      >
        <div
          className="playback-buttons__button"
          onClick={this.restart}
        >&#10226;</div>
      </div>
    )
  }
}

export default RestartButton