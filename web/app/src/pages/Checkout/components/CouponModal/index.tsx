import React, { useState } from 'react';

import { Modal } from '../../../../ui-library/Feedback/Modal';
import { message } from '../../../../ui-library/Feedback/Message';
import { Form } from '../../../../ui-library/Forms/Form';
import { FormItem } from '../../../../ui-library/Forms/FormItem';
import { Input } from '../../../../ui-library/Forms/Input';

import { ICoupon, retrieveCoupon } from '../../../../utils/stripe.util';

import './index.less';

interface IComponentProps {
  visible: boolean;
  onValidCoupon: (coupon: ICoupon | null) => void;
  onCancel(): void;
}

const CouponModal = (props: IComponentProps) => {
  const [couponCode, setCouponCode] = useState<string>();
  const [isFetching, setIsFetching] = useState<boolean>(false);

  const onCouponChange = (e: React.ChangeEvent<HTMLInputElement>): void => {
    setCouponCode(e.target.value);
  }

  const verifyCoupon = async (): Promise<void> => {
    if (!couponCode) {
      props.onValidCoupon(null);
      return;
    }

    setIsFetching(true);
    const couponResponse: ICoupon = await retrieveCoupon(couponCode);
    setIsFetching(false);

    if (!couponResponse.valid) {
      message.error(couponResponse.valid === false ? `Coupon expired` : couponResponse.message);
    } else {
      props.onValidCoupon(couponResponse);
      message.success(`Coupon successfuly applied`);
    }
  }

  return (
    <Modal className="promo-coupon-modal" title="Add Promo Coupon" visible={props.visible} okText="Apply Coupon"
      okButtonProps={{ loading: isFetching }} onOk={verifyCoupon} onCancel={props.onCancel} centered
    >
      <Form>
        <FormItem name="coupon">
          <Input placeholder="Enter coupon code" value={couponCode} autoFocus={true} allowClear={true}
            onChange={onCouponChange}
          />
        </FormItem>
      </Form>
    </Modal>
  );
}

export default CouponModal;
