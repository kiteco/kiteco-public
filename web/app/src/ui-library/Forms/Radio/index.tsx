import React from 'react';

import { Radio } from 'antd';
import { RadioProps } from 'antd/lib/radio';

class CustomRadio extends React.PureComponent<RadioProps> {
  render() {
    return (
      <Radio {...this.props as RadioProps}>
        {this.props.children}
      </Radio>
    );
  }
}

export { CustomRadio as Radio };
