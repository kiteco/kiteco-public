import React from "react";

import "./styles/StaticPreviewer.css";

import { render } from "./utils/api";
import { RENDER_URL_PATH_DEV, RENDER_URL_PATH_PROD } from "./utils/constants";
import AnswersContainer from "./AnswersContainer";

class StaticPreviewer extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      error: null,
      isLoaded: false,
      input: null
    };
  }

  render() {
    const { error, isLoaded } = this.state;
    if (error) {
      return <div>{error.name + ": " + error.message}</div>;
    } else if (!isLoaded) {
      return <div>Loading...</div>;
    } else {
      return (
        <div className="staticAnswersContainer">
          <AnswersContainer input={this.state.input} />
        </div>
      );
    }
  }
  componentDidMount() {
    let url =
      process.env.NODE_ENV === "development"
        ? RENDER_URL_PATH_DEV
        : RENDER_URL_PATH_PROD;
    render(this, new Request(url));
  }
}

export default StaticPreviewer;
