import React from "react";

import Prism from "prismjs/components/prism-core";
import "prismjs/components/prism-python.min.js";

class LinedBlock extends React.Component {
  render() {
    return (
      <div>
        {this.props.language === "python" && (
          <code className="lang-python">{this.props.code}</code>
        )}
        {!this.props.language && <code>{this.props.code}</code>}
      </div>
    );
  }
  // Highlight on initial load
  componentDidMount() {
    Prism.highlightAll();
  }
}

//assume tokenization of all punctuation
//Make sure to run through Prism beforehand...
const StructuredCodeBlock = ({ code }) => {
  return (
    <code className="with-syntax-highlighting code">
      {code.map((token, index) => {
        if (token.tokenType === "punctuation") {
          if (token.token === ",") {
            return (
              <span key={index} className="punctuation">
                {token.token}{" "}
              </span>
            );
          }
          return (
            <span key={index} className="punctuation">
              {token.token}
            </span>
          );
        }
        return (
          <span key={index} className={token.tokenType ? token.tokenType : ""}>
            {token.token}
          </span>
        );
      })}
    </code>
  );
};

const Statement = ({ code }) => {
  return (
    <code className="with-syntax-highlighting code">
      {code &&
        code.map((token, i) => {
          let className;
          switch (token.type) {
            case "punctuation":
            case "keyword":
            case "string":
              className = token.type;
              break;
            default:
              className = "";
              break;
          }
          if (className) {
            return (
              <span key={i} className={className}>
                {token.content}
              </span>
            );
          }
          return <span key={i}>{token.content}</span>;
        })}
    </code>
  );
};

//assumes param.default_value is accessible
const Parameter = ({ param, includeComma }) => {
  return (
    <div className="parameter">
      {param.name}
      {param.default_value && <span className="punctuation">: </span>}
      {param.default_value && param.default_value.type && (
        <span className="keyword">{param.default_value.type}</span>
      )}
      {param.default_value && param.default_value.repr && (
        <span className="punctuation">=</span>
      )}
      {param.default_value && param.default_value.repr && (
        <span className="literal">{param.default_value.repr}</span>
      )}
      {includeComma && <span className="punctuation">,</span>}
    </div>
  );
};

export { StructuredCodeBlock, Parameter, Statement, LinedBlock };
