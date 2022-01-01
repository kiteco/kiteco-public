import React from 'react';

import { Layout } from 'antd';
import { BasicProps } from 'antd/lib/layout/layout';

import './index.less';
const { Footer } = Layout;

class CustomFooter extends React.PureComponent<BasicProps> {
  render() {
    return (
      <Footer {...this.props as BasicProps}>
        {this.props.children}
      </Footer>
    );
  }
}

export { CustomFooter as Footer };
