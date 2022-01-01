import React from 'react';

import '../assets/return-value.css'

class ReturnValue extends React.Component {

  render() {
    const { returnValues } = this.props
    return <div className="docs__return-value">
      <h2 className="docs__return-value__label">
        Returns
      </h2>
      <span className="docs__return-value__type">
        {returnValues.map((value, index) =>
          value.type + (index < returnValues.length - 1 ? " | " : "")
        )}
      </span>
    </div>;
  }
}

export default ReturnValue;
