import React from 'react';

import { Layout } from 'antd';
import { BasicProps } from 'antd/lib/layout/layout';

import './index.less';
const { Content } = Layout;

class CustomContent extends React.PureComponent<BasicProps> {
  render() {
    return (
      <Content {...this.props as BasicProps}>
        {this.props.children}
      </Content>
    );
  }
}

export { CustomContent as Content };
