import React from "react";
import ReactMarkdown from "react-markdown";
import { connect } from "react-redux";
import { Link } from "react-router-dom";
import { StarHowToHeader, stripBackticks } from "../../../util/Titles";
import { popHowtoHistory } from "../../../../../../../../redux/actions/history";
import {
  addStarred,
  removeStarred
} from "../../../../../../../../redux/actions/starred";
import DocsSpinner from "../../../util/DocsSpinner";
import { getIdFromLocation } from "../../../../../../../../utils/route-parsing";

const IndexLink = ({ link, text }) => {
  return (
    <li>
      <Link to={link}>
        <ReactMarkdown source={text} />
      </Link>
    </li>
  );
};

const HowToBreadcrumbs = ({ history, clickCb }) => {
  const prev = history.howToHistory.length > 1 ? history.howToHistory[1] : null;
  if (prev) {
    prev.name = stripBackticks(prev.name);
    return (
      <div className="breadcrumbs">
        <span className="back-bracket">&#10550; </span>
        {/* Do we want the Link below to 'replace', as opposed to push?
        Semantically, it seems to make sense*/}
        <Link
          to={{
            pathname: prev.path,
            state: { howToBreadcrumb: true }
          }}
          onClick={clickCb}
        >
          <span className={prev.pageType === "identifier" ? "code" : ""}>
            {prev.name}
          </span>
        </Link>
      </div>
    );
  } else {
    return null;
  }
};

const HowToNav = ({
  name,
  examples,
  history,
  location,
  language,
  popHowtoHistory,
  starred,
  addStarred,
  removeStarred,
  loading
}) => {
  const exampleId = getIdFromLocation(location);
  const example = examples.data[exampleId];
  const title = example ? example.title : "";

  const isStarred =
    starred && starred.starredMap[location.pathname] ? true : false;
  const clickHandler = isStarred
    ? removeStarred(location.pathname)
    : addStarred(title, location.pathname, "howto");
  return (
    <nav className="nav" data-mock-page-type="HOWTO">
      {loading.isDocsLoading && (
        <div className="intro fixed-height-intro">
          <DocsSpinner />
        </div>
      )}
      {!loading.isDocsLoading && (
        <div className="intro fixed-height-intro">
          <HowToBreadcrumbs history={history} clickCb={popHowtoHistory} />
          <StarHowToHeader
            type="howto"
            title={title}
            isStarred={isStarred}
            clickHandler={clickHandler}
          />
        </div>
      )}

      {example && example.related && (
        <div className="index">
          <section>
            <h3>Related</h3>
            <div>
              <ul className="how-to">
                {example.related.map((howto, i) => (
                  <IndexLink
                    key={i}
                    link={`/${language}/examples/${howto.id}/${
                      howto.package
                      }-${howto.title
                        .toLowerCase()
                        .replace(/\s/g, "-")
                        .replace("%", "")}`}
                    text={howto.title}
                  />
                ))}
              </ul>
            </div>
          </section>
        </div>
      )}
    </nav>
  );
};

const mapStateToProps = (state, ownProps) => {
  return {
    ...ownProps,
    examples: state.examples,
    history: state.history,
    location: state.router.location,
    language: state.docs.language,
    starred: state.starred,
    loading: state.loading
  };
};

const mapDispatchToProps = dispatch => ({
  popHowtoHistory: () => dispatch(popHowtoHistory()),
  addStarred: (name, path, pageType) => () =>
    dispatch(addStarred(name, path, pageType)),
  removeStarred: path => () => dispatch(removeStarred(path))
});

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(HowToNav);
