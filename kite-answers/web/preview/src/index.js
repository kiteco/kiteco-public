import React from "react";
import ReactDOM from "react-dom";

import LivePreviewer from "./LivePreviewer";
import StaticPreviewer from "./StaticPreviewer";

import "./styles/index.css";

import * as serviceWorker from "./serviceWorker";

class PreviewApp extends React.Component {
  shouldUseLive() {
    const regex = /\/live(|\/)$/;
    return regex.test(window.location.pathname);
  }
  render() {
    if (this.shouldUseLive()) {
      return <LivePreviewer />;
    }
    return <StaticPreviewer />;
  }
}

ReactDOM.render(<PreviewApp />, document.getElementById("root"));

// If you want your app to work offline and load faster, you can change
// unregister() to register() below. Note this comes with some pitfalls.
// Learn more about service workers: https://bit.ly/CRA-PWA
serviceWorker.unregister();
