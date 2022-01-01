import React from 'react';

import { Radio } from 'antd';
import { RadioGroupProps, RadioChangeEvent as AntRadioEventChange } from 'antd/lib/radio';

class CustomRadioGroup extends React.PureComponent<RadioGroupProps> {
  render() {
    return (
      <Radio.Group {...this.props as RadioGroupProps}>
        {this.props.children}
      </Radio.Group>
    );
  }
}

export { CustomRadioGroup as RadioGroup };
export type RadioChangeEvent = AntRadioEventChange;
