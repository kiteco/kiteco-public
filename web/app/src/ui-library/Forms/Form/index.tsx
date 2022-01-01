import React from 'react';

import { Form } from 'antd';
import { FormProps } from 'antd/lib/form';
import { useForm as AntUseForm, FormInstance as AntFormInstance } from 'antd/lib/form/Form';

export interface FieldData {
  name: string[];
  value: any;
  touched: boolean;
  validating: boolean;
  errors: string[];
}

class CustomForm extends React.PureComponent<FormProps> {
  render() {
    return (
      <Form {...this.props as FormProps}>
        {this.props.children}
      </Form>
    );
  }
}

export {
  CustomForm as Form,
  AntUseForm as useForm,
};
export type FormInstance = AntFormInstance;
