import React from "react";

import "./styles/index.css";

import Description from "./Description";
import CodeBlock from "./CodeBlock";
import TableOfContents from "./TableOfContents";

class AnswersPage extends React.Component {
  render() {
    const content = this.props.source.content;
    document.title = this.props.source.title;
    return (
      <div className="answersPage">
        {content.map((value, index) => {
          return (
            <React.Fragment key={index}>
              {value.description && (
                <Description description={value.description} />
              )}
              {value.toc && <TableOfContents toc={value.toc} />}
              {value.code_block && <CodeBlock codeBlock={value.code_block} />}
            </React.Fragment>
          );
        })}
      </div>
    );
  }
}

export default AnswersPage;
