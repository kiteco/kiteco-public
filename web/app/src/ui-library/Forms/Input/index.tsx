import React from 'react';

import { Input } from 'antd';
import { InputProps } from 'antd/lib/input';

import './index.less';

class CustomInput extends React.PureComponent<InputProps> {
  render() {
    return (
      <Input {...this.props as InputProps} />
    );
  }
}

export { CustomInput as Input };
