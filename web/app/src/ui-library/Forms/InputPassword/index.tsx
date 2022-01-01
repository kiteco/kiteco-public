import React from 'react';

import { Input } from 'antd';
import { PasswordProps } from 'antd/lib/input';

import './index.less';

class CustomInputPassword extends React.PureComponent<PasswordProps> {
  render() {
    return (
      <Input.Password {...this.props as PasswordProps} />
    );
  }
}

export { CustomInputPassword as InputPassword };
