import React from 'react'

import '../assets/button-loading.css'

const LoadingButton = ({ className, onClick, isDisabled, text, isClicked }) =>
  isClicked
    ? <div className="lds-ring">
      <div></div>
      <div></div>
      <div></div>
      <div></div>
    </div>
  : <button
    className={className}
    onClick={onClick}
    disabled={isDisabled}
    data-clicked={isClicked}
  >
    {text}
  </button>

export default LoadingButton
