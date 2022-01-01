import React from 'react'
import ReactDOM from 'react-dom'

import '../assets/modal.css'

class Modal extends React.Component {
  outerClick = e => {
    const { dismiss } = this.props
    if (dismiss) {
      if (ReactDOM.findDOMNode(this) === e.target) {
        dismiss()
      }
    }
  }

  render() {
    const { children } = this.props
    return <div onClick={this.outerClick} className="modal">
      <div className="modal__inner">
        { children }
      </div>
    </div>
  }
}

export default Modal
