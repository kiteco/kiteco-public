import React from "react";
import Helmet from "react-helmet";
import { connect } from "react-redux";

import ModuleNav from "./components/ModuleNav";
import HowToNav from "./components/HowToNav";
import ModulePage from "./components/ModulePage";
import TypePage from "./components/TypePage";
import FunctionPage from "./components/FunctionPage";
import InstancePage from "./components/InstancePage";
import HowToPage from "./components/HowToPage";
import GenericPage from "./components/GenericPage";

import * as actions from "../../../../../../redux/actions/docs";
import { fetchExample } from "../../../../../../redux/actions/examples";
import { setPageNameForPath } from "../../../../../../redux/actions/history";
import { setDocsLoading } from "../../../../../../redux/actions/loading";

import { Domains } from '../../../../../../utils/domains'

import {
  idFromPath,
  parsePageKind,
  oldToNewPath
} from "../../../../../../utils/route-parsing";
import constants from "../../../../../../utils/theme-constants";
const { PAGE_KIND } = constants;

//import './assets/page.css'

class Page extends React.Component {
  parseRoute = () => {
    const {
      match,
      setPageKind,
      setExample,
      setPageNameForPath,
      newFetchDocs,
      newFetchMembers,
      fetchExample,
      docs,
      setDocsLoading,
      language,
      history
    } = this.props;
    switch (parsePageKind(match)) {
      case PAGE_KIND.HOWTO:
        if (docs.kind !== PAGE_KIND.HOWTO) {
          setPageKind(PAGE_KIND.HOWTO);
        }
        const { exampleId, exampleTitle } = match.params;
        if (docs.language !== language || docs.exampleId !== exampleId) {
          setExample(language, exampleId);
          fetchExample(language, exampleId)
            //use the exampleId to map in and dispatch the register name
            .then(({ data }) => {
              window.scrollTo(0, 0);
              if (!exampleTitle) {
                history.replace(
                  `/${language}/examples/${exampleId}/${
                    data[exampleId].package
                  }-${data[exampleId].title.toLowerCase().replace(/\s/g, "-")}`
                );
              }
              setPageNameForPath({
                name: data[exampleId].title,
                path: `/${language}/examples/${exampleId}/${
                  data[exampleId].package
                }-${data[exampleId].title
                  .toLowerCase()
                  .replace(/\s/g, "-")
                  .replace("%", "")}`,
                id: `${language}-example-${exampleId}`,
                packageName: data[exampleId].package
              });
            });
        }
        break;
      case PAGE_KIND.IDENTIFIER:
        if (docs.kind !== PAGE_KIND.IDENTIFIER) {
          setPageKind(PAGE_KIND.IDENTIFIER);
        }
        const valueId = idFromPath(language, match.params.valuePath);
        if (docs.language !== language || docs.identifier !== valueId) {
          setDocsLoading(true);
          Promise.all([
            newFetchDocs(valueId, language),
            newFetchMembers(language, valueId)
          ]).then(() => {
            window.scrollTo(0, 0);
            setDocsLoading(false);
            window.dataLayer.push({
              event: "optimize.activate"
            });
          });
        }
        break;
      default:
        //TODO: figure out default behavior here - just route to Root??
        break;
    }
  };

  componentDidMount() {
    this.parseRoute();
  }

  componentDidUpdate() {
    this.parseRoute();
  }

  pageSwitch({
    type,
    name,
    doc,
    loggedIn,
    language,
    identifier,
    status,
    examplesStatus,
    members
  }) {
    switch (type) {
      case "module":
        return (
          <ModulePage
            doc={doc}
            name={name}
            language={language}
            type={type}
            loggedIn={loggedIn}
            full_name={identifier}
            members={members}
          />
        );
      case "type":
        return (
          <TypePage
            doc={doc}
            name={name}
            language={language}
            type={type}
            loggedIn={loggedIn}
            members={members}
          />
        );
      case "function":
        return (
          <FunctionPage
            doc={doc}
            name={name}
            language={language}
            type={type}
            loggedIn={loggedIn}
          />
        );
      case "instance":
        return (
          <InstancePage
            doc={doc}
            name={name}
            language={language}
            type={type}
            loggedIn={loggedIn}
          />
        );
      case "howto":
        const { examples, full_name } = doc;
        if (examplesStatus === "failed") {
          return <DocsNotFoundPage isHowTo={true} identifier="that id" />;
        }
        return (
          <HowToPage
            examples={examples}
            full_name={full_name}
            language={language}
            loggedIn={loggedIn}
          />
        );
      default:
        // TODO: unknown/generic page
        if (type) {
          return (
            <GenericPage
              doc={doc}
              name={name}
              type={type}
              language={language}
              loggedIn={loggedIn}
            />
          );
        }
        if (status === "failed") {
          return <DocsNotFoundPage identifier={identifier} />;
        }
        return null;
    }
  }

  render() {
    const { account } = this.props;
    const loggedIn = account.status === "logged-in";
    const { kind, language, identifier, status } = this.props.docs;
    const { status: examplesStatus } = this.props.examples;
    const type = kind === PAGE_KIND.HOWTO ? "howto" : this.props.docs.data.type;
    const {
      name,
      ancestors,
      members,
      parameters,
      returnValues,
      exampleIds,
      patterns,
      kwargs,
      localCodeExamples,
      totalLocalCodeUsages,
      types,
      documentation_str,
      description_html,
      canonical_link,
      answers_links,
    } = this.props.docs.data;
    return (
      <div className={`Page ${type}-page`}>
        <Helmet>
          {canonical_link && (
            <link
              rel="canonical"
              href={`https://${Domains.WWW}` + oldToNewPath(canonical_link)}
            />
          )}
          {!canonical_link && this.props.docs.kind !== PAGE_KIND.HOWTO && (
            <meta name="robots" content="noindex, nofollow" />
          )}
        </Helmet>
        {type === "howto" && examplesStatus !== "failed" && (
          <HowToNav name={name} ancestors={ancestors} />
        )}
        {(type === "module" || type === "type") && (
          <ModuleNav
            type={type}
            name={name}
            ancestors={ancestors}
            members={members}
          />
        )}
        <div className="items">
          {this.pageSwitch({
            type,
            name,
            doc: {
              description_html,
              documentation_str,
              exampleIds,
              ancestors,
              parameters,
              returnValues,
              patterns,
              kwargs,
              types,
              localCodeExamples,
              totalLocalCodeUsages,
              members,
              answers_links,
            },
            loggedIn,
            language,
            identifier,
            status,
            examplesStatus
          })}
        </div>
      </div>
    );
  }
}

const DocsNotFoundPage = ({ identifier, isHowTo }) => {
  return (
    <div className="items">
      <div className="item">
        <div className="intro">
          <h2 className="not-found">404</h2>
        </div>
        <div className="docs">
          <section>
            We couldn't locate any {isHowTo ? "example" : "docs"} for
            {!isHowTo && <span className="code"> {identifier}</span>}
            {isHowTo && <span> {identifier}</span>}
          </section>
        </div>
      </div>
    </div>
  );
};

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
  docs: state.docs,
  account: state.account,
  examples: state.examples
});

const mapDispatchToProps = dispatch => ({
  setPageKind: kind => dispatch(actions.setPageKind(kind)),
  setExample: (language, exampleId) =>
    dispatch(actions.setExample(language, exampleId)),
  newFetchDocs: (identifier, language) =>
    dispatch(actions.newFetchDocs(identifier, language)),
  newFetchMembers: (language, identifier) =>
    dispatch(actions.newFetchMembers(language, identifier)),
  fetchExample: (language, id) => dispatch(fetchExample(language, id)),
  setPageNameForPath: ({ name, path, packageName, id }) =>
    dispatch(setPageNameForPath({ name, path, packageName, id })),
  setDocsLoading: isLoading => dispatch(setDocsLoading(isLoading))
});

export default connect(
  mapStateToProps,
  mapDispatchToProps
)(Page);
