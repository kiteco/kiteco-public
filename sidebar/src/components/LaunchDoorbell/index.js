import React from 'react'

import { show } from '../../utils/doorbell'

class LaunchDoorbell extends React.Component {

  componentDidMount() {
    const { history } = this.props
    show()
    history.goBack()
  }

  render() {
    return null
  }
}

export default LaunchDoorbell
