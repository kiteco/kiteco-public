import React from "react";

import "./styles/CodeAnnotation.css";

class CodeAnnotation extends React.Component {
  render() {
    const annotationBlock = this.props.annotationBlock;
    return (
      <div className="annotationBlock">
        {annotationBlock.map((value, index) => {
          return <p key={index}>{value}</p>;
        })}
      </div>
    );
  }
}

export default CodeAnnotation;
