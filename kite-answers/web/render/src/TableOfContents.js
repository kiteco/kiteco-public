import React from "react";
import ReactMarkdown from "react-markdown/with-html";

import "./styles/TableOfContents.css";

class TableOfContents extends React.Component {
  render() {
    const toc = this.props.toc;
    return (
      <div className="tableOfContents">
        {toc.map(item => {
          return (
            <a key={item.anchor} href={"#" + item.anchor}>
              <ReactMarkdown source={item.header}></ReactMarkdown>
            </a>
          );
        })}
      </div>
    );
  }
}

export default TableOfContents;
