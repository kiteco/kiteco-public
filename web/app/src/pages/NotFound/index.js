import React from "react";
import Helmet from "react-helmet";
import { Link } from "react-router-dom";

import Header from "../../components/Header";
import Section from "../../components/Section";

import "./assets/not-found.css";

class NotFound extends React.Component {
  render() {
    const { history } = this.props;

    return (
      <div className="not-found">
        <Helmet>
          <title>Kite 404</title>
        </Helmet>
        <div className="not-found__screen">
          <div className="not-found__bg-image not-found__meteor" />
          <div className="not-found__bg-image not-found__earth" />
          <div className="not-found__bg-image not-found__kite" />
          <Header className="header__dark not-found__header" type="root" />
          <Section className="not-found__content">
            <div className="not-found__info">
              <h1 className="not-found__title">404</h1>
              <h2 className="not-found__subtitle">Lost in Space</h2>
              <p className="not-found__text">
                Ooops, looks like the page you are trying to find is no longer
                available.
              </p>
              <div className="not-found__navigation-btns">
                {history && history.length >= 2 && (
                  <div
                    onClick={() => history.goBack()}
                    className="not-found__btn not-found__btn"
                  >
                    Back
                  </div>
                )}
                <Link to="/" className="not-found__btn not-found__btn--fill">
                  Go to Homepage
                </Link>
              </div>
            </div>
          </Section>
        </div>
      </div>
    );
  }
}

export default NotFound;
