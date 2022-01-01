import React from 'react'
import './loading-spinner.css'

const LoadingSpinner = () => {
  return (
    <div>
      <div className="lds-ripple">
        <div></div>
        <div></div>
      </div>
    </div>
  )
}

export default LoadingSpinner
