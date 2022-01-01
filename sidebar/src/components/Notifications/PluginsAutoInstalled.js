import React from 'react'

import { Link } from 'react-router-dom'

import { getIconForEditor } from '../../utils/editorInfo'

function uniqueIDs(ids) {
  let found = {}
  return ids.filter(id => {
    if (found[id] === true){
      return false
    }
    found[id] = true
    return true
  })
}

const PluginsAutoInstalled = ({ dismiss, id, pluginIDs }) =>
  <div className="notifications__plugins">
    <div className="notifications__plugins__header">
      <div className="notifications__plugins__title">
        New plugins installed
      </div>
      <div className="notifications__plugins__hide"
        onClick={dismiss}
      >
        Hide
      </div>
    </div>
    <div className="notifications__plugins__content">
      <div className="notifications__plugins__p">
        Kite has installed plugins for &nbsp;{ uniqueIDs(pluginIDs).map( (pluginID, index, data) => <span key={pluginID}>
          <span className="notifications__plugins__name">{ pluginID }</span>{ getIconForEditor(pluginID) && <img className="notifications__plugins__icon" src={getIconForEditor(pluginID)} alt={pluginID}/> }
          { data.length > 1 && index === data.length-2 && <span>, and &nbsp;</span> }
          { (index < data.length-2) && <span>,&nbsp;</span> }
        </span>
        )}
        .
        <br/>
        Please restart your editors to activate Kite. <Link className="notifications__plugins__a--bold" to="/settings/plugins" onClick={dismiss}>Undo</Link>.
      </div>
    </div>
  </div>

export default PluginsAutoInstalled
