import React from 'react';
import { connect } from 'react-redux'
import { History } from 'history';
import queryString from 'query-string';
import { loadStripe } from '@stripe/stripe-js';
import { Elements } from "@stripe/react-stripe-js";
import { noop } from "lodash";

import BasicLayout from '../../components/Layouts/Basic';
import { Title } from '../../ui-library/Typography/Title';
import { Text } from '../../ui-library/Typography/Text';
import { Link } from '../../ui-library/Typography/Link';
import { Paragraph } from '../../ui-library/Typography/Paragraph';
import { Space } from '../../ui-library/Space';
import { Row } from '../../ui-library/Grid/Row';
import { Col } from '../../ui-library/Grid/Col';
import { FormInstance } from '../../ui-library/Forms/Form';
import { RadioGroup, RadioChangeEvent } from '../../ui-library/Forms/RadioGroup';
import { Radio } from '../../ui-library/Forms/Radio';
import { Skeleton } from '../../ui-library/Feedback/Skeleton';
import { message } from '../../ui-library/Feedback/Message';

import { fetchAccountInfo, logIn, createNewAccount } from '../../redux/actions/account'
import {
  fetchLicenseInfo,
  LicenseInfo,
  ProLicenseInfo,
  Plan,
  getIsProSubscriber,
  IIsProSubscriber,
} from '../../redux/store/license';
import { ICoupon, retrieveCoupon, IPrice, retrievePrice, retrieveDefaultPlans, IDefaultPlans } from '../../utils/stripe.util';
import { ITaxDetails, retrieveTaxDetails } from '../../utils/octobat.util';
import { track } from '../../utils/analytics';
import { Domains } from '../../utils/domains'
import { Emails } from '../../utils/emails';

import './index.less';
import styles from './index.module.less';

import PaymentDetailsForm, { IPaymentDetailFormData } from './components/PaymentDetailsForm';
import InlineLoginForm, { ILoginCredentials, ILoginFormData } from './components/InlineLoginForm';
import CouponModal from './components/CouponModal';

import kiteIntellisense from './images/kite-intellisense.jpg';
import paymentVendors from './images/payment-vendors.jpg';
import { ReactComponent as CheckLogo } from './images/check.svg';

enum BillingCycle {
  ANNUAL = 'annual',
  MONTHLY = 'monthly',
}

interface IServerProSubscription {
  customer: {
    email: string,
    name: string,
    payment_method_id: string,
    address: {
      country: string,
      postal_code: string,
    }
  },
  subscription: {
    user_id: number,
    coupon_code: string | undefined,
    billing_cycle: string,
    monthly_plan_id?: string,
    annual_plan_id?: string,
    trial_days?: number,
  },
  tax_details: ITaxDetails | null,
}

interface IComponentState {
  selectedBillingCycle: BillingCycle,
  annualPrice: number,
  monthlyPrice: number,
  loggedInUser: {
    id: number;
    email: string;
  } | null | undefined, // undefined represents pending verification (shows loading until user verification is done)
  coupon: ICoupon | null,
  isPromoModalVisible: boolean,
  loginFormData: ILoginFormData | null,
  paymentDetailFormData: IPaymentDetailFormData | null,
  taxDetails: ITaxDetails | null,
  isPaymentInProgress: boolean,
  paymentErrMessage: string | null,
  isPlanFromUrl: boolean,
  isPriceLoading: boolean,
  configs?: {
    monthly_plan_id?: string,
    annual_plan_id?: string,
    trial_days?: number,
  },
}

interface IRouteParams {
  plan: BillingCycle,
  coupon: string,
  ap: string, // Annual Price ID
  mp: string, // Monthly Price ID
  td: number, // Trial Days
}

interface IReduxStateProps {
  account: any;
}

interface IReduxDispatchProps {
  fetchAccountInfo: () => Promise<void>;
  login: (loginCredentials: ILoginCredentials) => Promise<void>;
  createNewAccount: (signUpCredentials: ILoginCredentials) => Promise<any>;
  fetchLicenseInfo: () => Promise<LicenseInfo>,
  getIsProSubscriber: () => Promise<any>,
}

interface IComponentProps extends IReduxStateProps, IReduxDispatchProps {
  location: Location,
  history: History,
}

const STRIPE_PUBLISHABLE_KEY: string = process.env.REACT_APP_STRIPE_PUBLISHABLE_KEY as string;
const stripePromise = loadStripe(STRIPE_PUBLISHABLE_KEY);

class Checkout extends React.Component<IComponentProps, IComponentState> {
  state: IComponentState = {
    selectedBillingCycle: BillingCycle.ANNUAL,
    annualPrice: 0,
    monthlyPrice: 0,
    loggedInUser: undefined,
    coupon: null,
    isPromoModalVisible: false,
    loginFormData: null,
    paymentDetailFormData: null,
    taxDetails: null,
    isPaymentInProgress: false,
    paymentErrMessage: null,
    isPlanFromUrl: false,
    isPriceLoading: true,
    configs: {},
  };
  inlineloginFormInstance: FormInstance = null as any;
  pageErrorsRef: React.RefObject<HTMLDivElement>

  constructor(props: IComponentProps) {
    super(props);
    this.pageErrorsRef = React.createRef();
  }

  async componentDidMount(): Promise<void> {
    const { account, fetchAccountInfo, location } = this.props;
    const params: IRouteParams = queryString.parse(location.search) as any;

    await this.canCheckout();
    this.mapQueryParamsToState();

    if(!params.ap && !params.mp) {
      const defaultPlans: IDefaultPlans = await retrieveDefaultPlans();

      this.setState({
        isPriceLoading: false,
        annualPrice: defaultPlans.yearly,
        monthlyPrice: defaultPlans.monthly,
      })
    }

    if (!account.data) {
      fetchAccountInfo()
        .then((res: any): void => {
          // success is handled in `componentDidUpdate` because we also need to show email which comes after store update
          if (!res.success) {
            this.setState({ loggedInUser: null });
          }
        });
    }
  }

  componentDidUpdate(prevProps: IComponentProps): void {
    if (this.props?.account?.data !== prevProps?.account?.data) {
      const { id, email }: { id: number, email: string } = this.props.account.data;

      this.setState({
        loggedInUser: { id, email }
      });
    }
  }

  async mapQueryParamsToState(): Promise<void> {
    const params: IRouteParams = queryString.parse(this.props.location.search) as any;

    if (params.plan) {
      this.setState({
        selectedBillingCycle: params.plan,
        isPlanFromUrl: true,
      });
    }

    if (params.coupon) {
      const couponResponse: ICoupon = await retrieveCoupon(params.coupon);

      if (couponResponse.valid) {
        this.setState({
          coupon: couponResponse,
        });
      }
    }

    if (params.ap || params.mp || params.td) {
      this.setState({
        configs: {
          annual_plan_id: params.ap || undefined,
          monthly_plan_id: params.mp || undefined,
          trial_days: params.td ? parseInt(String(params.td), 10) : undefined,
        }
      });

      if (params.mp || params.ap) {
        const promises: Promise<IPrice>[] = [];

        if (params.ap) promises.push(retrievePrice(params.ap));
        if (params.mp) promises.push(retrievePrice(params.mp));

        const priceResponses: IPrice[] = await Promise.all(promises);
        const annualResponse: IPrice = priceResponses[0];
        const monthlyResponse: IPrice = priceResponses[params.ap ? 1 : 0];

        this.setState({
          isPriceLoading: false,
          annualPrice: (params.ap && annualResponse.amount) ? annualResponse.amount : this.state.annualPrice,
          monthlyPrice: (params.mp && monthlyResponse.amount) ? monthlyResponse.amount : this.state.monthlyPrice,
        });
      }
    }
  }

  /**
   * Whether user can continue checkout or not. Only Free and Trial users are allowed to checkout.
   *
   * @param plan string
   */
  async canCheckout(): Promise<boolean> {
    const [fetchLicenseInfoResponse, subscriptionResponse] = await Promise.all([
      this.props.fetchLicenseInfo() as Promise<ProLicenseInfo>,
      this.props.getIsProSubscriber() as Promise<IIsProSubscriber>,
    ]);

    const plan = fetchLicenseInfoResponse && fetchLicenseInfoResponse.plan;
    if (!plan) return true;

    const canCheckout = (
      (plan.indexOf(Plan.Free) >= 0 || plan.indexOf(Plan.Trial) >= 0)
      && subscriptionResponse.isProSubscriber === false
    );
    if (canCheckout) return true;

    if (!this.state.loginFormData) {
      // represents first load i.e info got from session verification (not via login)
      this.props.history.push('/settings/account');
      return false;
    }

    this.showErrorMessage(
      `This account already has a Kite Pro license. Please log into Kite on your desktop to access Pro features.`
    );
    return false;
  }

  onBillingCycleChange = (radioEvent: RadioChangeEvent): void => {
    const selectedPlan = radioEvent.target.value as BillingCycle;

    this.setState({
      selectedBillingCycle: selectedPlan,
    });

    track({
      event: 'checkout: clicked checkout plan chooser',
      props: {
        plan: selectedPlan,
      },
    });
  }

  onLoginClick = (): void => {
    this.setState({
      loggedInUser: null,
    });

    track({ event: 'checkout: clicked login button' });
  }

  onLoginFormInitialize = (loginFormInstance: FormInstance): void => {
    this.inlineloginFormInstance = loginFormInstance;
  }

  onRedeemCouponClick = (): void => {
    this.setState({
      isPromoModalVisible: true,
    });

    track({ event: 'checkout: clicked redeem coupon' });
  }

  onCancelPromoModal = (): void => {
    this.setState({
      isPromoModalVisible: false,
    });
  }

  isAnnualBillingCycleSelected(): boolean {
    return this.state.selectedBillingCycle === BillingCycle.ANNUAL;
  }

  getKiteProPricing(): number {
    return this.isAnnualBillingCycleSelected() ? this.state.annualPrice : this.state.monthlyPrice;
  }

  getTaxAmount(): number {
    if (!this.state.taxDetails?.rate) return 0;

    const amountToTax: number = this.getDiscountedAmount(this.getKiteProPricing());
    return amountToTax * (this.state.taxDetails.rate / 100);
  }

  getOffAmount(amount: number): number {
    if (!this.state.coupon) return 0;
    return amount * (this.state.coupon.percentOff / 100);
  }

  getDiscountedAmount(amount: number): number {
    return amount - this.getOffAmount(amount);
  }

  getTotalBillingAmount(): number {
    const total: number = this.getDiscountedAmount(this.getKiteProPricing()) + this.getTaxAmount();
    return Math.round(total * 100) / 100;
  }

  onLoginFormData = (loginFormData: ILoginFormData | null): void => {
    this.setState({ loginFormData });
  }

  onValidCoupon = (coupon: ICoupon | null): void => {
    this.setState({
      coupon: coupon,
      isPromoModalVisible: false,
    } as IComponentState);
  }

  onCountryChange = async (countryISOcode: string): Promise<void> => {
    const taxDetails: ITaxDetails | null = await retrieveTaxDetails(countryISOcode);

    this.setState({ taxDetails });
  }

  onSubmitCheckoutForm = async (paymentDetailFormData: IPaymentDetailFormData | null): Promise<void> => {
    if (!paymentDetailFormData) return;

    this.setState({ paymentDetailFormData });

    if (!this.inlineloginFormInstance) {
      this.handleCheckout();
      return;
    }

    this.inlineloginFormInstance.validateFields()
      .then(async (): Promise<void> => this.handleCheckout())
      .catch(noop);
  }

  async handleCheckout(): Promise<void> {
    track({ event: 'checkout: clicked login button' });

    if (this.state.loginFormData?.hasEmptyPassword) {
      message.error('Please reset password from given link before continuing checkout')
      return;
    }

    if (this.state.loggedInUser || this.state.loginFormData) {
      let isUserLoggedIn: boolean = !!this.state.loggedInUser;

      track({ event: 'checkout: clicked purchase button' });

      if (!isUserLoggedIn && this.state.loginFormData) {
        const { email, password } = this.state.loginFormData;
        const credentials = { email, password };

        this.setState({ isPaymentInProgress: true });
        const response = this.state.loginFormData?.isNewAccount ?
          await this.props.createNewAccount(credentials) :
          await this.props.login(credentials);

        isUserLoggedIn = response.success;
        if (!isUserLoggedIn) {
          this.showErrorMessage(response.error);
          return;
        }

        const canCheckout: boolean = await this.canCheckout();
        if (!canCheckout) return;
      }

      this.handlePurchase();
      return;
    }
  }

  /**
   * Handle subscription of the user and navigates to confirmation page if purchase is successful.
   */
  async handlePurchase(): Promise<void> {
    const body: IServerProSubscription = {
      customer: {
        email: this.state.loggedInUser?.email as string,
        name: this.state.paymentDetailFormData?.name as string,
        payment_method_id: this.state.paymentDetailFormData?.creditCardInfo?.id as string,
        address: {
          country: this.state.paymentDetailFormData?.countryISOCode as string,
          postal_code: this.state.paymentDetailFormData?.creditCardInfo?.billing_details.address?.postal_code as string,
        }
      },
      subscription: {
        user_id: this.state.loggedInUser?.id as number,
        coupon_code: this.state.coupon?.id,
        billing_cycle: this.state.selectedBillingCycle,
        ...this.state.configs,
      },
      tax_details: this.state.taxDetails,
    };

    this.setState({ isPaymentInProgress: true });

    await fetch(
      `${process.env.REACT_APP_BACKEND}/api/checkout/pro-subscription`,
      {
        method: "POST",
        body: JSON.stringify(body),
      }
    )
      .then((res: Response) => res.json())
      .then((data): void => {
        if (data.success) {
          this.setState({
            isPaymentInProgress: false,
            paymentErrMessage: null,
          });

          message.success('Payment succeed, redirecting to activation page');

          setTimeout((): void => {
            window.location.href = `https://${Domains.WWW}/pro/confirmation`;
          }, 1500);

          return;
        }

        this.showErrorMessage(data.message);
      })
      .catch((_: Error): void => {
        this.showErrorMessage('Please try again.');
      });
  }

  showErrorMessage(msg: string): void {
    this.setState({
      paymentErrMessage: msg,
      isPaymentInProgress: false,
    });

    message.error('Please resolve errors.');
    this.pageErrorsRef.current?.scrollIntoView();

    track({
      event: 'checkout: error shown',
      props: {
        error_detail: msg,
      },
    });
  }

  render(): JSX.Element {
    const PriceOption = (props: { price: number }): JSX.Element => {
      return (
        <div className="d-flex">
          {
            this.state.coupon &&
            <Text className="highlight" delete>${(props.price).toFixed(2)}&nbsp;</Text>
          }

          <Text>${this.getDiscountedAmount(props.price).toFixed(2)} / month</Text>
        </div>
      );
    };

    const AnnualCard = (
      <div className={`detail-card ${styles['detail-card']}`}>
        <Skeleton loading={this.state.isPriceLoading} paragraph={{ rows: 4 }} round={true} active>
          <Title className={styles['detail-title']} level={3}>
            Kite Pro,
          <Text className="selected-package"> {this.isAnnualBillingCycleSelected() ? 'Annual' : 'Monthly'} </Text>
          </Title>

          <div className={styles['bill-calculation']}>
            <table>
              <tbody>
                <tr>
                  <td>Kite Pro</td>
                  <td>${this.getKiteProPricing().toFixed(2)}</td>
                </tr>
                {
                  this.state.coupon &&
                  <tr className="coupon-discount">
                    <td>Discounted</td>
                    <td>-${this.getOffAmount(this.getKiteProPricing())?.toFixed(2)}</td>
                  </tr>
                }
                {
                  (!!this.state.taxDetails &&
                    !!this.state.taxDetails?.rate &&
                    !!this.state.taxDetails?.name
                  ) &&
                  <tr className="tax-details">
                    <td>{`${this.state.taxDetails.name} (${this.state.taxDetails.rate}%)`}</td>
                    <td>${this.getTaxAmount()?.toFixed(2)}</td>
                  </tr>
                }
                <tr className="bill-total">
                  <td>Total</td>
                  <td>${this.getTotalBillingAmount().toFixed(2)}</td>
                </tr>
              </tbody>
            </table>

            <Paragraph className={styles['bill-desc']}>
              We will charge you ${this.getTotalBillingAmount().toFixed(2)}
              {this.isAnnualBillingCycleSelected() ? ' yearly' : ' monthly'} until you cancel your Kite Pro
              subscription.
              <br /><br />
              Your payment details are encrypted and secure. All amounts are shown in USD.
            </Paragraph>
          </div>
        </Skeleton>
      </div>
    );

    return (
      <BasicLayout>
        <section className="container kite-pro-checkout">
          <Title className={styles['page-title']} level={1}> Kite Pro Checkout </Title>

          {
            this.state?.paymentErrMessage &&
            <div ref={this.pageErrorsRef} className="page-error">
              <h4 className="title"> Oops. Something went wrong </h4>
              <p className="desc">
                {this.state.paymentErrMessage}
                <br />
                For assistance on payment related errors, please contact your payment provider. For Kite related errors, please email 
                <a href={`mailto:${Emails.Support}`}> {Emails.Support} </a>.
              </p>
            </div>
          }

          <Row className={styles['checkout-content']} justify="space-between">
            <Col md={24} lg={13}>
              <ol className={`checkout-steps ${styles['checkout-steps']}`}>
                {
                  !this.state.isPlanFromUrl &&
                  <li className={styles['checkout-step']}>
                    <Title className={styles['step-title']} level={2}> Select a billing cycle </Title>
                    <Skeleton loading={this.state.isPriceLoading} title={false} paragraph={{ rows: 3 }} round={true}
                      active
                    >
                      <RadioGroup className={styles['package-selector']} value={this.state.selectedBillingCycle}
                        onChange={this.onBillingCycleChange}
                      >
                        <Radio value='annual'>
                          <Space direction="vertical">
                            <Text strong>ANNUAL BILLING</Text>
                            <PriceOption price={this.state.annualPrice / 12} />
                          </Space>
                        </Radio>
                        <Radio value='monthly'>
                          <Space direction="vertical">
                            <Text strong>MONTHLY BILLING</Text>
                            <PriceOption price={this.state.monthlyPrice} />
                          </Space>
                        </Radio>
                      </RadioGroup>
                    </Skeleton>

                    <div className="d-md-block">
                      {AnnualCard}
                    </div>
                  </li>
                }

                <li className={styles['checkout-step']}>
                  <Title className={styles['step-title']} level={2}> Enter account information </Title>

                  {
                    this.state.loggedInUser === undefined &&
                    <Skeleton title={false} paragraph={{ rows: 2 }} round={true} active></Skeleton>
                  }

                  {
                    this.state.loggedInUser === null &&
                    <div className={styles['step-desc']}>
                      <Paragraph>
                        Please enter your login credentials. A new account will be created if no existing account is found.
                      </Paragraph>
                      <InlineLoginForm onFormInitialize={this.onLoginFormInitialize} onChange={this.onLoginFormData} />
                    </div>
                  }

                  {
                    this.state.loggedInUser !== null && this.state.loggedInUser !== undefined &&
                    <Paragraph className={styles['step-desc']}>
                      The Kite Pro subscription will be unlocked for <Text strong>{this.state.loggedInUser?.email}</Text>
                      <br />
                      Not you? <Link onClick={this.onLoginClick} tabIndex={0}> Log in here </Link> to your existing account.
                    </Paragraph>
                  }
                </li>

                <li className={styles['checkout-step']}>
                  <Title className={styles['step-title']} level={2}>
                    Enter payment details
                  <img src={paymentVendors} alt="payment vendors" />
                  </Title>
                  <Paragraph className={styles['step-desc']}>
                    {
                      this.state.coupon ?
                        <span>
                          Coupon applied: <Link onClick={this.onRedeemCouponClick} tabIndex={0}> {this.state.coupon.name} </Link>
                        </span>
                        :
                        <span>
                          Have a promo coupon?
                        <Link onClick={this.onRedeemCouponClick} tabIndex={0}> Redeem your coupon </Link>
                        </span>
                    }
                    <CouponModal visible={this.state.isPromoModalVisible} onValidCoupon={this.onValidCoupon}
                      onCancel={this.onCancelPromoModal} />
                  </Paragraph>

                  <Elements stripe={stripePromise}>
                    <PaymentDetailsForm isPaymentInProgress={this.state.isPaymentInProgress}
                      onCountryChange={this.onCountryChange}
                      onSubmitPaymentDetailsForm={this.onSubmitCheckoutForm}
                    />
                  </Elements>
                </li>
              </ol>
            </Col>

            <Col md={24} lg={10}>
              <div className="d-md-none">
                {AnnualCard}
              </div>
              <div className={styles['kite-pro-detail']}>
                <Title className={styles['kite-pro-title']} level={3}> WHAT’S INCLUDED WITH KITE PRO? </Title>
                <img src={kiteIntellisense} alt="Kite Powered Intellisense" />
                <ul className={styles['kite-pro-features']}>
                  <li> <CheckLogo /> Unlimited Pro completions</li>
                  <li> <CheckLogo /> Line-of-code completions</li>
                  <li> <CheckLogo /> Multi-line completions</li>
                  <li> <CheckLogo /> Intelligent Snippets</li>
                  <li> <CheckLogo /> Import Alias completions</li>
                  <li> <CheckLogo /> Dictionary Key completions</li>
                  <li> <CheckLogo /> Premium support</li>
                </ul>
              </div>
            </Col>
          </Row>
        </section>
      </BasicLayout>
    );
  }
}

const mapStoreStateToProps = (storeState: any): IReduxStateProps => ({
  account: storeState.account,
});

const mapStoreDispatchToProps = (storeDispatch: any): IReduxDispatchProps => ({
  fetchAccountInfo: () => storeDispatch(fetchAccountInfo()),
  login: (loginCredentials: ILoginCredentials) => storeDispatch(logIn(loginCredentials)),
  createNewAccount: (signUpCredentials: ILoginCredentials) => storeDispatch(createNewAccount(signUpCredentials)),
  fetchLicenseInfo: () => storeDispatch(fetchLicenseInfo()),
  getIsProSubscriber: () => storeDispatch(getIsProSubscriber()),
});

export default connect(mapStoreStateToProps, mapStoreDispatchToProps)(Checkout)
