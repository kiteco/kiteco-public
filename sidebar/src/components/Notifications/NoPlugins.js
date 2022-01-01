import React from 'react'
import { Link } from 'react-router-dom'

const NoPlugins = ({ dismiss }) =>
  <div className="notifications__plugins">
    <div className="notifications__plugins__header">
      <div className="notifications__plugins__title">
        NO EDITOR PLUGINS DETECTED
      </div>
      <div className="notifications__plugins__hide"
        onClick={dismiss}
      >
        Hide
      </div>
    </div>
    <div className="notifications__plugins__content">
      <div className="notifications__plugins__p">
        Kite is built to boost your programming environment in your favorite editor.<br/>
      </div>
      <Link 
        className="notifications__plugins__install"
        to="/settings/plugins"
      >Install Editor Plugins</Link>  
    </div>
  </div>

export default NoPlugins