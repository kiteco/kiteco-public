import React from "react";
import { connect } from "react-redux";
import { Helmet } from "react-helmet";

import { stripBackticks } from "../../../util/Titles";
import { getIdFromLocation } from "../../../../../../../../utils/route-parsing";

import CuratedExample from "../CuratedExample";
import Promo from "../Promo";

class HowToPage extends React.Component {
  constructor(props) {
    super(props);

    this.state = {
      examplesToShow: [],
      currentPath: "",
      primaryExample: null
    };
  }

  static getDerivedStateFromProps(props, state) {
    const { location = {}, examples } = props;
    if (location.pathname !== state.currentPath) {
      //will want to split
      const exampleId = getIdFromLocation(location);
      //derive examplesToShow from examples
      const example = examples.data[exampleId];
      if (example) {
        return {
          currentPath: location.pathname,
          examplesToShow: [example],
          primaryExample: { ...example }
        };
      }
    }
    return null;
  }

  getTitle = () => {
    const { primaryExample } = this.state;
    if (primaryExample) {
      return `${
        primaryExample.package
        } - ${primaryExample.title} - Python code example - Kite`;
    }
    return "Python code example - Kite";
  };

  getMetaDescription = () => {
    const { primaryExample } = this.state;
    if (primaryExample) {
      return `Python code example '${primaryExample.title}' for the package ${
        primaryExample.package
        }, powered by Kite`;
    }
    return "Illustrative Python code examples, powered by Kite";
  };

  render() {
    const { language, loggedIn } = this.props;
    const { examplesToShow } = this.state;
    const { primaryExample } = this.state;
    if (primaryExample) {
      primaryExample.title = stripBackticks(primaryExample.title);
    }
    return (
      <div className="items">
        <Helmet>
          <title>{this.getTitle()}</title>
          <meta name="description" content={this.getMetaDescription()} />
        </Helmet>
        <div className="item">
          <div className="intro fixed-height-intro fixed-height-intro-spacing" />
          <div className="docs">
            <h3>Details</h3>
            {examplesToShow.map(example => {
              return (
                <CuratedExample
                  key={example.id}
                  example={example}
                  full_name={example.package}
                  id={example.id}
                  language={language}
                  linkTitle={false}
                />
              );
            })}
          </div>
          {!loggedIn && (
            <div className="extra">
              <Promo />
            </div>
          )}
        </div>
      </div>
    );
  }
}

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
  examples: state.examples,
  location: state.router.location
});

export default connect(mapStateToProps)(HowToPage);
