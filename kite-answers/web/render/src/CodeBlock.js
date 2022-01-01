import React from "react";
import Prism from "prismjs/components/prism-core";
import "prismjs/components/prism-python.min.js";

import "./styles/CodeBlock.css";

import CodeOutput from "./CodeOutput";
import CodeAnnotation from "./CodeAnnotation";

class CodeBlock extends React.Component {
  render() {
    const codeBlock = this.props.codeBlock;
    return (
      <div className="codeBlock">
        {codeBlock.map((value, index) => {
          if (value.code_line === "") {
            value.code_line = "\n";
          }
          return (
            <div className="container" key={index}>
              <div className="leftChild">
                {value.code_line && (
                  <pre className={"lang-" + value.lang}>
                    <code className={"lang-" + value.lang}>
                      {value.code_line}
                    </code>
                  </pre>
                )}
                {value.output && <CodeOutput output={value.output} />}
              </div>
              {value.annotation_block && (
                <CodeAnnotation annotationBlock={value.annotation_block} />
              )}
            </div>
          );
        })}
      </div>
    );
  }
  // Highlight on initial load
  componentDidMount() {
    Prism.highlightAll();
  }
  // Highlight again on page update.
  componentDidUpdate() {
    Prism.highlightAll();
  }
}

export default CodeBlock;
