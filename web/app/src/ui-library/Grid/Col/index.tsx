import React from 'react';

import { Col } from 'antd';
import { ColProps } from 'antd/lib/col';

import './index.less';

class CustomCol extends React.PureComponent<ColProps> {
  render() {
    return (
      <Col {...this.props as ColProps} />
    );
  }
}

export { CustomCol as Col };
