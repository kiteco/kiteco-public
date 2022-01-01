import React from 'react';

import { Typography } from 'antd';
import { LinkProps } from 'antd/lib/typography/Link';

import './index.less';
const { Link } = Typography;

class CustomLink extends React.PureComponent<LinkProps> {
  render() {
    return (
      <Link {...this.props as LinkProps} />
    );
  }
}

export { CustomLink as Link };
