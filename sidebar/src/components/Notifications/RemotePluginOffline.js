import React from 'react'

import { Link } from 'react-router-dom'

import { getIconForEditor } from '../../utils/editorInfo'

const RemotePluginOffline = ({ dismiss, id, name }) =>
  <div className="notifications__plugins">
    <div className="notifications__plugins__header">
      <div className="notifications__plugins__title">
        Unable to Install Plugin
      </div>
      <div className="notifications__plugins__hide"
        onClick={dismiss}
      >
        Hide
      </div>
    </div>
    <div className="notifications__plugins__content">
      <div className="notifications__plugins__p">
        Could not install&nbsp;
        <span className="notifications__plugins__name">{ name }</span>
        { getIconForEditor(id)
          && <img
            className="notifications__plugins__icon"
            src={getIconForEditor(id)} alt={id}
          />
        }
        due to a network connection problem. Please reconnect and <Link className="notifications__plugins__a--bold" to="/settings/plugins">try again</Link>.
      </div>
    </div>
  </div>

export default RemotePluginOffline
