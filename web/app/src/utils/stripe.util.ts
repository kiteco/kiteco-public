export interface IDefaultPlans {
  monthly: number;
  yearly: number;
}

export async function retrieveDefaultPlans(): Promise<IDefaultPlans> {
  const response = await fetch(`${process.env.REACT_APP_BACKEND}/api/checkout/default-plans`);
  const defaultPlans: IDefaultPlans = await response.json();

  return {
    monthly: toDollars(defaultPlans.monthly),
    yearly: toDollars(defaultPlans.yearly),
  }
}

interface IResponseCoupon {
  id: string,
  name: string,
  message: string,
  percent_off: number,
  valid: boolean,
}

export interface ICoupon {
  id: string,
  name: string,
  percentOff: number,
  valid: boolean,
  message?: string,
}

export async function retrieveCoupon(couponCode: string): Promise<ICoupon> {
  const response = await fetch(`${process.env.REACT_APP_BACKEND}/api/checkout/verify-coupon/${couponCode}`);
  const stripeCoupon: IResponseCoupon = await response.json();

  return {
    id: stripeCoupon.id,
    name: stripeCoupon.name,
    message: stripeCoupon.message,
    percentOff: stripeCoupon.percent_off,
    valid: stripeCoupon.valid,
  }
}

export interface IResponsePrice {
  unit_amount: number,
}

export interface IPrice {
  amount: number,
}

export async function retrievePrice(priceId: string): Promise<IPrice> {
  const response = await fetch(`${process.env.REACT_APP_BACKEND}/api/checkout/prices/${priceId}`);
  const stripePrice: IResponsePrice = await response.json();

  return {
    amount: toDollars(stripePrice.unit_amount),
  }
}

function toDollars(priceInCents: number): number {
  return priceInCents / 100;
}
