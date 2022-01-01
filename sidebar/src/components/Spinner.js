import React from 'react';

import '../assets/spinner.css';

const Spinner = ({theme, text}) =>
  <div className={"loading__spinner " + (theme || 'dark').split(/\s+/g).map(t => `loading__spinner--${t}`).join(' ')}>
    {text && <p className="loading__spinner__text">{text}</p>}
    <br/>
    <div className="bounce1"></div>
    <div className="bounce2"></div>
    <div className="bounce3"></div>
  </div>;

export default Spinner;
