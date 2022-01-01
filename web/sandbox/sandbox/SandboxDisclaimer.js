import React from 'react'

import './sandbox-disclaimer.css'

const SandboxDisclaimer = ({ text, theme }) => {
  return <div className={`${theme ? `${theme} ` : ''}sandbox__disclaimer`}>
    { text }
  </div>
}

export default SandboxDisclaimer