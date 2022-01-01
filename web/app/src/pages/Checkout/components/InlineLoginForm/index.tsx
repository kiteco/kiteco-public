import React, { useEffect, useState } from 'react';
import { connect } from 'react-redux';
import { debounce } from 'lodash';

import { checkNewEmail, ICheckEmailResponse } from '../../../../redux/actions/account';

import { Row } from '../../../../ui-library/Grid/Row';
import { Col } from '../../../../ui-library/Grid/Col';
import { Form, FieldData, FormInstance, useForm } from '../../../../ui-library/Forms/Form';
import { FormItem } from '../../../../ui-library/Forms/FormItem';
import { Input } from '../../../../ui-library/Forms/Input';
import { InputPassword } from '../../../../ui-library/Forms/InputPassword';
import { Link } from '../../../../ui-library/Typography/Link';
import { Spin } from '../../../../ui-library/Feedback/Spin';

import './index.less';

export interface ILoginFormData extends ILoginCredentials {
  isNewAccount: boolean,
  hasEmptyPassword: boolean,
}

export interface ILoginCredentials {
  email: string,
  password: string,
}

interface IReduxDispatchProps {
  checkEmail?: (email: string) => Promise<ICheckEmailResponse>;
}

interface IComponentProps extends IReduxDispatchProps {
  onFormInitialize: (form: FormInstance) => void,
  onChange: (loginFormData: ILoginFormData | null) => void;
}

type IFormErrors = string[];

const InlineLoginForm: React.FC<IComponentProps> = (props): JSX.Element => {
  const [loginCredentials, setLoginCredentials] = useState<ILoginCredentials>();
  const [isNewAccount, setIsNewAccount] = useState<boolean>(false);
  const [hasEmptyPassword, setHasEmptyPassword] = useState<boolean>(false);
  const [isEmailVerificationInProgress, setIsEmailVerificationInProgress] = useState<boolean>(false);
  const [form] = useForm();

  useEffect((): void => {
    props.onFormInitialize(form);
  }, [props, form]);

  function onFieldsChange(changedFields: FieldData[], allFields: FieldData[]): void {
    const formDataReducer = (previousValue: object, current: FieldData) => {
      return {
        ...previousValue,
        [current.name[0]]: current.value,
      }
    };

    const formErrorsReducer = (previousValue: string[], current: FieldData): IFormErrors => {
      return [
        ...previousValue,
        ...current.errors
      ]
    };

    const formErrors: IFormErrors = allFields.reduce(formErrorsReducer, []) as IFormErrors;
    const formData: ILoginCredentials = allFields.reduce(formDataReducer, {}) as ILoginCredentials;

    if (formErrors.length) {
      setHasEmptyPassword(false);
      setIsNewAccount(false);
      props.onChange(null);

      return;
    }

    setLoginCredentials(formData);
    if (loginCredentials?.email !== formData.email) verifyEmail(formData.email);

    props.onChange({
      ...formData,
      isNewAccount,
      hasEmptyPassword,
    });
  }

  function verifyEmail(email: string): void {
    if (!props.checkEmail) return;

    setIsEmailVerificationInProgress(true);

    props.checkEmail(email)
      .then(({ success, data, error }: ICheckEmailResponse): void => {
        setIsEmailVerificationInProgress(false);

        if (typeof (data) === "object") {
          setIsNewAccount(!data.account_exists);
          setHasEmptyPassword(!data.has_password);
        } else if (typeof (data) === "string") { // no existing account is found against current email
          setHasEmptyPassword(false);
          setIsNewAccount(true);
        }
      });
  }

  return (
    <Form className="inline-login-form" form={form}
      onFieldsChange={debounce(onFieldsChange, 500) as any /** ant design Field Data interface is incomplete */}
    >
      <Row>
        <Col sm={24} md={12}>
          <FormItem name="email" label="Email" rules={[
            { required: true, message: 'Field Missing' },
            { type: 'email', message: 'Invalid Email Address' },
          ]}>
            <Input placeholder="Enter Email" />
          </FormItem>

          {
            isEmailVerificationInProgress &&
            <Spin className="validating" />
          }
        </Col>

        <Col sm={24} md={12}>
          <FormItem name="password" label="Password" rules={[{ required: true, message: 'Field Missing' }]}>
            <InputPassword placeholder="Enter Password" />
          </FormItem>
        </Col>

        {
          isNewAccount && !hasEmptyPassword &&
          <p className="help-message"> * New account will be created with Pro subscription. </p>
        }

        {
          !isNewAccount && hasEmptyPassword &&
          <p className="help-message has-error">
            * You have not previously set a password for this account. Please
              <Link href={`/reset-password/email=${loginCredentials?.email}`} target="_label"> set your password here </Link>
              and then continue the checkout process.
          </p>
        }
      </Row>
    </Form>
  );
}

const mapStoreStateToProps = (_: any): any => ({});

const mapStoreDispatchToProps = (storeDispatch: any): IReduxDispatchProps => ({
  checkEmail: (email: string): Promise<ICheckEmailResponse> => storeDispatch(checkNewEmail(email)),
});

export default connect(mapStoreStateToProps, mapStoreDispatchToProps)(InlineLoginForm)
