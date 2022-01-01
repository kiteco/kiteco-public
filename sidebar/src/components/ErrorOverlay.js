import React from 'react'
import Spinner from './Spinner'

const ErrorOverlay = ({ 
  title, 
  subtitle, 
  handler, 
  btnText, 
  linkText, 
  link, 
  spinner,
  isSeeThrough,
}) => {
  return <div>
    <div className={`app__error-overlay ${isSeeThrough ? "app__error-overlay--see-through" : ""}`}>
      <div>
        <p className={"app__error-title"}>{title}</p>
        <p className="app__error-subtitle">{subtitle}</p>
      </div>
      { spinner && <Spinner/> }
      { btnText && handler &&
        <div>
          <div className="app__error-button" onClick={handler}>{btnText}</div>
        </div>
      }
      { linkText && link &&
        <div>
            <a href={link} target="_blank" rel="noopener noreferrer" className="app__error-button">{linkText}</a>
        </div>
      }
    </div>
  </div>
}

export default ErrorOverlay