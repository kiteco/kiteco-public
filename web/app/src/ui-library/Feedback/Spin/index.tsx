import React from 'react';

import { Spin } from 'antd';
import { SpinProps } from 'antd/lib/spin';
import { LoadingOutlined } from '@ant-design/icons';

class CustomSpin extends React.PureComponent<SpinProps> {
  render(): JSX.Element {
    const antIcon = <LoadingOutlined style={{ fontSize: 24 }} spin />;

    return (
      <Spin indicator={antIcon} {...this.props as Spin}>
        {this.props.children}
      </Spin>
    );
  }
}

export { CustomSpin as Spin };
