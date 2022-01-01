import React from 'react';

import { Row } from 'antd';
import { RowProps } from 'antd/lib/row';

class CustomRow extends React.PureComponent<RowProps> {
  render() {
    return (
      <Row {...this.props as RowProps} />
    );
  }
}

export { CustomRow as Row };
