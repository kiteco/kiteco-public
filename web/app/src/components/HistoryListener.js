import React from 'react'

class HistoryListener extends React.Component {

  componentWillUnmount() {
    if(this.props.unlisten) this.props.unlisten()
  }

  render() {
    return null
  }
}

export default HistoryListener