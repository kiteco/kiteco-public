import React from 'react'
import { ENTERPRISE } from '../utils/enterprise'

const TopBar = () => {
  return <div className="header">
    { ENTERPRISE
      ? <div className="kite-enterprise-logo"/>
      : <div className="kite-logo"/>
    }
  </div>
}

export default TopBar
