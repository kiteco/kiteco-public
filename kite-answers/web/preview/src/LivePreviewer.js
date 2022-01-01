import React from "react";

import "./styles/LivePreviewer.css";

import { render } from "./utils/api";
import { RENDER_URL_PATH_DEV, RENDER_URL_PATH_PROD } from "./utils/constants";
import AnswersContainer from "./AnswersContainer";
import MarkdownContainer from "./MarkdownContainer";

class LivePreviewer extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      error: null,
      isLoaded: false,
      input: null
    };
    this.renderAnswersPage = this.renderAnswersPage.bind(this);
  }

  renderAnswersPage(input) {
    if (!input) {
      this.setState({ input: null });
      return;
    }
    let url =
      process.env.NODE_ENV === "development"
        ? RENDER_URL_PATH_DEV
        : RENDER_URL_PATH_PROD;
    const request = new Request(url, {
      method: "POST",
      body: input
    });
    render(this, request);
  }

  render() {
    return (
      <div className="livePreviewer">
        <MarkdownContainer renderAnswersPage={this.renderAnswersPage} />
        {this.state.input && (
          <div className="liveAnswersContainer">
            <AnswersContainer input={this.state.input} />
          </div>
        )}
        {!this.state.input && (
          <div className="liveAnswersContainer">
            <b>This page will auto-refresh</b>
            <br />
            <br />
            <b>Hit 'Render' to preview the page on demand</b>
          </div>
        )}
      </div>
    );
  }
}

export default LivePreviewer;
