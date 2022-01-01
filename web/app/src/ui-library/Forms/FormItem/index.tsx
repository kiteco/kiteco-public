import React from 'react';

import { Form } from 'antd';
import { FormItemProps } from 'antd/lib/form';

import './index.less'

class CustomFormItem extends React.PureComponent<FormItemProps> {
  render() {
    return (
      <Form.Item validateTrigger="onBlur" {...this.props as FormItemProps}>
        {this.props.children}
      </Form.Item>
    );
  }
}

export { CustomFormItem as FormItem };
