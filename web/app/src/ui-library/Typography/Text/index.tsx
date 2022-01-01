import React from 'react';

import { Typography } from 'antd';
import { TextProps } from 'antd/lib/typography/Text';

import './index.less';
const { Text } = Typography;

class CustomText extends React.PureComponent<TextProps> {
  render() {
    return (
      <Text {...this.props as TextProps} />
    );
  }
}

export { CustomText as Text };
