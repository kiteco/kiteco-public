import React from "react";
import ReactDOM from "react-dom";
import Helmet from "react-helmet";
import { connect } from "react-redux";

import { Domains } from "../../utils/domains"

import * as actions from "../../redux/actions/answers";
import { SUCCESS, FAILED } from "../../redux/reducers/answers";
import NotFound from "../NotFound";
import Nav from "../newDocs/components/Docs/components/Nav";
import Header from "../newDocs/components/Docs/components/Header";
import Footer from "../newDocs/components/Docs/components/Footer";
import Promo from "../newDocs/components/Docs/components/Page/components/Promo";
import StyleWatcher from "../newDocs/components/Docs/StyleWatcher";

// this must be imported before kite-answers-renderer,
// since the latter defines styles that must  the defaults.
import "../newDocs/assets/index.css";

import AnswersPage from "@kiteco/kite-answers-renderer";
import "./assets/index.css";

class AnswersContainer extends React.Component {
  render() {
    const content = this.props.answers.content;
    const status = this.props.answers.status;
    if (status === FAILED) {
      return <NotFound />;
    }
    if (status === SUCCESS) {
      const title = content.title || "";
      const canonicalSlug = content.canonical;
      return (
        <div className="Documentation" id="KiteAnswers">
          <StyleWatcher />
          <Helmet>
            {title !== "" && <title>{title}</title>}
            {title === "" && <title>Kite Answers</title>}
            {canonicalSlug && (
              <link
                rel="canonical"
                href={`https://${Domains.WWW}/python/answers/` + canonicalSlug}
              />
            )}
          </Helmet>
          <Header />
          <Nav />
          <div className="Page">
            <div className="items">
              <div className="items">
                <div className="item">
                  <div className="intro" />
                  {/* Nest AnswersPage in same sequence as docs page, to reuse its CSS. */}
                  <AnswersPage source={content} />
                  <div className="extra">
                    <Promo />
                  </div>
                </div>
              </div>
            </div>
          </div>
          <Footer />
        </div>
      );
    } else {
      return <div />;
    }
  }
  componentDidMount() {
    const slug = this.props.match.params.slug;
    this.props.fetchPost(slug);
  }

  componentDidUpdate() {
    this.markupExternalLinks();
  }

  markupExternalLinks() {
    const links = ReactDOM.findDOMNode(this).querySelectorAll(".answersPage a");
    links.forEach(link => {
      let path = link.getAttribute("href");
      if (path && !this.isInternalLink(path)) {
        link.setAttribute("rel", "nofollow noopener noreferrer");
      }
    });
  }

  isInternalLink(path) {
    return (
      path[0] === "#" ||
      path[0] === "/" ||
      path.toLowerCase().includes(Domains.PrimaryHost)
    );
  }
}

const mapStateToProps = state => ({
  answers: state.answers
});

const mapDispatchToProps = dispatch => ({
  fetchPost: slug => dispatch(actions.fetchPost(slug))
});

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(AnswersContainer);
