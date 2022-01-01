import React from 'react';

import { Select } from 'antd';
import { SelectProps, SelectValue as AntSelectValue } from 'antd/lib/select';

import './index.less';

class CustomSelect extends React.PureComponent<SelectProps<SelectValue>> {
  render() {
    return (
      <Select {...this.props as SelectProps<SelectValue>}>
        {this.props.children}
      </Select>
    );
  }
}

export { CustomSelect as Select }
export type SelectValue = AntSelectValue;
