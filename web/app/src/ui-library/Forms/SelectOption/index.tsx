import React from 'react';

import { Select } from 'antd';
import { OptionProps } from 'antd/lib/select';

const { Option } = Select;

class CustomOption extends React.PureComponent<OptionProps> {
  render() {
    return (
      <Option {...this.props as OptionProps}>
        {this.props.children}
      </Option>
    );
  }
}

export { CustomOption as Option };
