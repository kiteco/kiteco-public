import React from 'react'

import { getIconForEditor } from '../../utils/editorInfo'

const Plugins = ({ dismiss, id, name }) =>
  <div className="notifications__plugins">
    <div className="notifications__plugins__header">
      <div className="notifications__plugins__title">
        New plugin installed
      </div>
      <div className="notifications__plugins__hide"
        onClick={dismiss}
      >
        Hide
      </div>
    </div>
    <div className="notifications__plugins__content">
      <div className="notifications__plugins__p">
        Please restart&nbsp;
        <span className="notifications__plugins__name">{ name }</span>
        { getIconForEditor(id)
          && <img
            className="notifications__plugins__icon"
            src={getIconForEditor(id)} alt={id}
          />
        }
        to activate Kite.
      </div>
    </div>
  </div>

export default Plugins
