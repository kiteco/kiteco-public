import React, { useState } from 'react';
import { useStripe, useElements, CardElement } from '@stripe/react-stripe-js';
import { PaymentMethod, StripeCardElement, StripeCardElementOptions } from '@stripe/stripe-js';

import { CountrySelect } from '../../../../components/Selects/CountrySelect';

import { Row } from '../../../../ui-library/Grid/Row';
import { Col } from '../../../../ui-library/Grid/Col';
import { Form } from '../../../../ui-library/Forms/Form';
import { FormItem } from '../../../../ui-library/Forms/FormItem';
import { Input } from '../../../../ui-library/Forms/Input';
import { Button } from '../../../../ui-library/Forms/Button';

import './index.less'

interface IComponentProps {
  isPaymentInProgress: boolean;
  onCountryChange: (countryISOCode: string) => void,
  onSubmitPaymentDetailsForm: (paymentDetailFormData: IPaymentDetailFormData | null) => void;
}

export interface IPaymentDetailFormData {
  name: string;
  countryISOCode: string;
  creditCardInfo: PaymentMethod | undefined;
}

const PaymentDetailsForm: React.FC<IComponentProps> = (props): JSX.Element => {
  const [isRequestingPaymentID, setIsRequestingPaymentID] = useState<boolean>(false);

  // Stripe specific attributes
  const [error, setError] = useState<string | null>();
  const stripe = useStripe();
  const elements = useElements();
  const stripeCardOptions: StripeCardElementOptions = {
    style: {
      base: {
        color: 'rgba(0, 0, 0, 0.75)',
        fontFamily: 'Arial',
        fontSize: '14px',

        '::placeholder': {
          color: 'rgba(0, 0, 0, 0.45)',
        }
      },
      invalid: {
        color: '#ff4d4f',
        iconColor: '#ff4d4f'
      }
    }
  };

  const onCardElementChange = async (event: any) => {
    setError(event.error ? event.error.message : null);
  };

  const onFinish = async (values: IPaymentDetailFormData): Promise<void> => {
    if (!stripe || !elements) return;

    const cardElement = elements.getElement(CardElement);

    setIsRequestingPaymentID(true);
    const { error, paymentMethod } = await stripe.createPaymentMethod({
      type: 'card',
      card: cardElement as StripeCardElement,
    });

    if (error) {
      setError(error.message);
      props.onSubmitPaymentDetailsForm(null);
      return;
    }

    props.onSubmitPaymentDetailsForm({
      ...values,
      creditCardInfo: paymentMethod,
    });
    setIsRequestingPaymentID(false);
  }

  return (
    <div className="payment-details-form-wrapper">
      <Form name="payment-details-form" onFinish={onFinish}>
        <Row>
          <Col sm={24} md={12}>
            <FormItem name="name" label="Name" rules={[{ required: true, message: 'Field Missing' }]}>
              <Input placeholder="Name of cardholder" />
            </FormItem>
          </Col>
          <Col sm={24} md={12}>
            <FormItem name="countryISOCode" label="Country" initialValue="US" rules={[{ required: true, message: 'Field Missing' }]}>
              <CountrySelect onChange={props.onCountryChange} />
            </FormItem>
          </Col>
        </Row>
        <Row>
          <Col span={24}>
            <FormItem className="custom-required" label="Card Information">
              <CardElement className="stripe-card-element" options={stripeCardOptions} onChange={onCardElementChange} />
              {
                !!error &&
                <div className="ant-form-item-explain card-error-message">
                  <div role="alert">{error}</div>
                </div>
              }
            </FormItem>
          </Col>
        </Row>
        <Row className="submit-row">
          <Col span={24}>
            <Button type="primary" htmlType="submit" loading={props.isPaymentInProgress || isRequestingPaymentID}>
              Buy Now
            </Button>
          </Col>
        </Row>
      </Form>
    </div>
  );
}

export default PaymentDetailsForm;
