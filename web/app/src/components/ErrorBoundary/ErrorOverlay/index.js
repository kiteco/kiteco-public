import React from "react";
import Helmet from "react-helmet";

import { Emails } from '../../../utils/emails'

const ErrorOverlay = ({
  title,
  subtitle,
  subtitle2,
  supportEmail,
  handler,
  btnText
}) => {
  return (
    <div className="app__error-overlay">
      <Helmet>
        <meta name="robots" content="noindex,nofollow" />
      </Helmet>
      <h1 className="app__error-title">{title}</h1>
      <h3 className="app__error-subtitle">{subtitle}</h3>
      {subtitle2 && <h3 className="app__error-subtitle">{subtitle2}</h3>}
      {supportEmail && (
        <h3 className="app__error-subtitle">
          If this error persists, please email us at{" "}
          <a href={`mailto:${Emails.Support}`}>{Emails.Support}</a>
        </h3>
      )}
      <button className="app__error-btn" onClick={handler}>
        {btnText}
      </button>
    </div>
  );
};

export default ErrorOverlay;
