import React from 'react';

import { Space } from 'antd';
import { SpaceProps } from 'antd/lib/space';

import './index.less';

class CustomSpace extends React.PureComponent<SpaceProps> {
  render() {
    return (
      <Space {...this.props as SpaceProps} />
    );
  }
}

export { CustomSpace as Space };
