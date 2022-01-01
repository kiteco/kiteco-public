import { push } from 'connected-react-router'
import { Path } from 'history'
import React from 'react'
import { connect } from 'react-redux'
import { Link } from 'react-router-dom'
import { AnyAction } from 'redux'
import { ThunkDispatch } from 'redux-thunk'

import { Product, Plan, LicenseInfo, fetchLicenseInfo, getIsProSubscriber, cancelSubscription } from '../../redux/store/license';
import { Domains } from '../../utils/domains'
import { Emails } from '../../utils/emails'

interface AccountProps {
  push: (p: Path) => void

  getIsProSubscriber: () => Promise<void>
  cancelSubscription: () => Promise<boolean>
  fetchLicenseInfo: () => Promise<LicenseInfo>
  licenseInfo?: LicenseInfo
  isProSubscriber?: number

  loggedUserEmail: string
}

class Account extends React.Component<AccountProps, {cancelError: boolean}> {
  constructor(props: AccountProps) {
    super(props)
    this.cancel = this.cancel.bind(this)
    this.state = {cancelError: false}
  }

  componentDidMount() {
    this.props.fetchLicenseInfo()
    this.props.getIsProSubscriber()
  }

  cancel() {
    this.props.cancelSubscription().then((ok) => this.setState({cancelError: !ok}))
  }

  render() {
    const {
      loggedUserEmail,
      licenseInfo,
      isProSubscriber,
    } = this.props;
    const { cancelError } = this.state;

    let licenseBody;
    if (isProSubscriber === undefined || licenseInfo === undefined) {
    } else if (cancelError) {
      licenseBody = (
        <p>
          We encountered an error canceling your subscription.
          <a href={`mailto:${Emails.Feedback}`}>
            Please contact us for assistance.
          </a>
        </p>
      )
    } else if (isProSubscriber) {
      licenseBody = (
        <p>
          {/* eslint-disable-next-line */}
          <a href="#" onClick={this.cancel}>
            Cancel your Kite Pro subscription.
          </a>
        </p>
      )
    } else if (licenseInfo.product === Product.Free) {
      if (!licenseInfo.trial_available) {
        licenseBody = (
          <p>
            <a href={`https://${Domains.WWW}/pro`}>
              Subscribe to Kite Pro!
            </a>
          </p>
        )
      }
    } else {
      switch (licenseInfo.plan) {
        case Plan.Trial:
          licenseBody = (
            <p>
              Your Kite Pro trial will end in {licenseInfo.days_remaining} days.
              {' '}
              <a href={`https://${Domains.WWW}/pro`}>
                Subscribe now!
              </a>
            </p>
          )
          break
        case Plan.Temp:
          licenseBody = (
            <p>
              Your Kite Pro payment is still pending.
              {' '}
              <a href={`mailto:${Emails.Feedback}`}>
                Contact us to cancel your subscription.
              </a>
            </p>
          )
          break
        case Plan.Education:
          licenseBody = (
            <p>
              Your Kite Pro educational subscription is on us!
              It will expire in {licenseInfo.days_remaining} days.
            </p>
          )
          break
        default:
          licenseBody = (
            <p>
              Your Kite Pro subscription is no longer active,
              and your license will expire in {licenseInfo.days_remaining} days.
              {' '}
              <a href={`https://${Domains.WWW}/pro`}>
                Resubscribe
              </a>
            </p>
          )
          break
      }
    }

    return (
      <div>
        <p>
          <Link to={`/reset-password?email=${loggedUserEmail}`}>Reset password</Link>
        </p>

        <p>
          <a href={`mailto:${Emails.Privacy}`}>
            Contact us to delete your account
          </a>
        </p>

        { licenseBody }
      </div>
    );
  }
}

function mapStateToProps(state: any) {
  return {
    loggedUserEmail: state.account.data.email,
    licenseInfo: state.license.licenseInfo,
    isProSubscriber: state.license.isProSubscriber,
  }
}

function mapDispatchToProps(dispatch: ThunkDispatch<any, {}, AnyAction>) {
  return {
    push: (p: Path) => dispatch(push(p)),
    getIsProSubscriber: () => dispatch(getIsProSubscriber()),
    cancelSubscription: () => dispatch(cancelSubscription()),
    fetchLicenseInfo: () => dispatch(fetchLicenseInfo()),
  }
}

export default connect(mapStateToProps, mapDispatchToProps)(Account)
