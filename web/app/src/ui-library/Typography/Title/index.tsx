import React from 'react';

import { Typography } from 'antd';
import { TitleProps } from 'antd/lib/typography/Title';

const { Title } = Typography;

class CustomTitle extends React.PureComponent<TitleProps> {
  render() {
    return (
      <Title {...this.props as TitleProps} />
    );
  }
}

export { CustomTitle as Title };
