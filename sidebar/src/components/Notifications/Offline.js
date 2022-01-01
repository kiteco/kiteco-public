import React from 'react'

const Offline = ({ dismiss, copy }) => {
  return <div className="notifications__plugins">
    <div className="notifications__plugins__header">
      <div className="notifications__plugins__title">
        KITE OFFLINE
      </div>
      <div className="notifications__plugins__hide"
        onClick={dismiss}
      >
        Hide
      </div>
    </div>
    <div className="notifications__plugins__content">
      <div className="notifications__plugins__p">
          {copy &&
            <React.Fragment>{copy}<br/></React.Fragment>
          }
        Unfortunately, we can't reach our backend right now.
      </div>  
    </div>
  </div>
}

export default Offline