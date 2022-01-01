import React from 'react';

import { Layout } from 'antd';
import { BasicProps } from 'antd/lib/layout/layout';

import './index.less';

class CustomLayout extends React.PureComponent<BasicProps> {
  render() {
    return (
      <Layout {...this.props as BasicProps}>
        {this.props.children}
      </Layout>
    );
  }
}

export { CustomLayout as Layout };
