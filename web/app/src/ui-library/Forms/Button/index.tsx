import React from 'react';

import { Button } from 'antd';
import { ButtonProps } from 'antd/lib/button';

import './index.less';

class CustomButton extends React.PureComponent<ButtonProps> {
  render() {
    return (
      <Button {...this.props as ButtonProps}>
        {this.props.children}
      </Button>
    );
  }
}

export { CustomButton as Button };
