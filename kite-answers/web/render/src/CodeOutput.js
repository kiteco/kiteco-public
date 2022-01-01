import React from "react";

import "./styles/CodeOutput.css";

class Output extends React.Component {
  render() {
    const output = this.props.output;
    switch (output.type) {
      case "text":
        return (
          <div className="outputBlock">
            <div className="outputTitle">{output.title}</div>
            <pre className="output">
              <code>{output.data}</code>
            </pre>
          </div>
        );
      case "image":
        return (
          <div className="outputBlock">
            <div className="outputTitle">{output.title}</div>
            <img className="output" src={output.data} alt=""></img>
          </div>
        );
      default:
        return null;
    }
  }
}

export default Output;
