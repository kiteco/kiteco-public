import React from 'react';

import { Layout } from 'antd';
import { BasicProps } from 'antd/lib/layout/layout';

import './index.less';
const { Header } = Layout;

class CustomHeader extends React.PureComponent<BasicProps> {
  render() {
    return (
      <Header {...this.props as BasicProps}>
        {this.props.children}
      </Header>
    );
  }
}

export { CustomHeader as Header };
