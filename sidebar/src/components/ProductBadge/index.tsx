import React from 'react'
import { connect } from 'react-redux'
import { AnyAction } from 'redux'
import { ThunkDispatch } from 'redux-thunk'
import pluralize from 'pluralize'

import { Product, Plan, LicenseInfo, CTASource, ConversionCohort, ConversionCohorts } from '../../store/license'
import {
  fetchLicenseInfo,
  startTrial,
  getPaywallCompletionsRemaining,
  getConversionCohort,
  getAllFeaturesPro,
} from '../../store/license'
import { ILoginModalData, requireLogin } from '../../store/modals'
import { Goals } from '../../components/LoginModal'
import { localhostProxy } from '../../utils/urls'
const { shell } = window.require("electron")

import styles from './index.module.css'

interface ProductBadgeProps {
  className: string
  licenseInfo: LicenseInfo
  conversionCohort: ConversionCohort
  paywallCompletionsRemaining: number
  allFeaturesPro: boolean

  fetchLicenseInfo: () => Promise<LicenseInfo>
  getAllFeaturesPro: () => Promise<Boolean>
  getPaywallCompletionsRemaining: () => Promise<number | undefined>
  getConversionCohort: () => Promise<ConversionCohort | undefined>
  startTrial: () => void
  requireLogin: (d: ILoginModalData) => void
  kiteServerAvailable: boolean
}

interface ProductBadgeState {
  inFlight: boolean
}

interface IAction {
  copy: string,
  do: () => void,
  // Existence of loginData indicates onClick action requires login.
  loginData?: ILoginModalData,
}

class ProductBadge extends React.Component<ProductBadgeProps, ProductBadgeState> {
  constructor(props: ProductBadgeProps) {
    super(props)
    this.upgradeAction = this.upgradeAction.bind(this)
    this.settingsAction = this.settingsAction.bind(this)
    this.computeAction = this.computeAction.bind(this)
    this.computeOptInAction = this.computeOptInAction.bind(this)
    this.state = {
      inFlight: false,
    }
  }

  componentDidMount() {
    this.props.getConversionCohort()
    this.props.getPaywallCompletionsRemaining()
    this.props.getAllFeaturesPro()
    this.props.fetchLicenseInfo()
  }

  upgradeAction() {
    shell.openExternal(localhostProxy('/clientapi/desktoplogin?d=%2Fpro%3Floc%3Dcopilot_badge%26src%3Dlimit'))
  }

  settingsAction() {
    shell.openExternal(localhostProxy('/clientapi/desktoplogin?d=/settings'))
  }

  computeAction(licenseInfo: LicenseInfo, conversionCohort: ConversionCohort): IAction | null  {
    if (!licenseInfo) {
      return null
    }

    switch (conversionCohort) {
      case ConversionCohorts.UsagePaywall:
        if (licenseInfo.product === Product.Free) {
          const { allFeaturesPro, paywallCompletionsRemaining } = this.props
          if (allFeaturesPro && paywallCompletionsRemaining === 0) {
            return {
              copy: "No completions left today",
              do: this.upgradeAction,
            }
          }

          const singularComplName = allFeaturesPro ? "completion" : "Pro completion"
          return {
            copy: `${paywallCompletionsRemaining} ${pluralize(singularComplName, paywallCompletionsRemaining)} left today`,
            do: this.upgradeAction,
          }
        } else {
          return {
            copy: "Pro",
            do: this.settingsAction,
          }
        }
      case ConversionCohorts.Autostart:
      case ConversionCohorts.QuietAutostart:
        if (licenseInfo.product === Product.Free && licenseInfo.trial_available) {
          // Badage is invisible for autostart cohort before trial start
          return null
        }
        return this.computeOptInAction(licenseInfo)
      case ConversionCohorts.OptIn:
        return this.computeOptInAction(licenseInfo)
      case ConversionCohorts.Unset:
        return null
    }
  }

  computeOptInAction(licenseInfo: LicenseInfo): IAction | null {
    if (!licenseInfo) {
      return null
    }

    switch (licenseInfo.product) {
      case Product.Free:
        if (licenseInfo.trial_available_duration) {
          return {
            copy: "Start your free Kite Pro trial",
            do: this.props.startTrial,
            loginData: {
              goal: Goals.startTrial,
              // onSuccess is a stub here. Set at render time to include appropriate guards.
              onSuccess: () => {},
            },
          }
        } else {
          return {
            copy: "Upgrade to Kite Pro",
            do: this.upgradeAction,
            loginData: {
              goal: Goals.upgrade,
              // onSuccess is a stub here. Set at render time to include appropriate guards.
              onSuccess: () => {},
            },
          }
        }
      case Product.Pro:
        if (licenseInfo.plan === Plan.Trial) {
          if (licenseInfo.days_remaining <= 7) {
            return {
              copy: `Pro: Trialing (${licenseInfo.days_remaining} days left)`,
              do: this.upgradeAction,
              loginData: {
                goal: Goals.upgrade,
                // onSuccess is a stub here. Set at render time to include appropriate guards.
                onSuccess: () => {},
              },
            }
          } else {
            return {
              copy: "Pro: Trialing",
              do: this.upgradeAction,
              loginData: {
                goal: Goals.upgrade,
                // onSuccess is a stub here. Set at render time to include appropriate guards.
                onSuccess: () => {},
              },
            }
          }
        } else {
          return {
            copy: "Pro",
            do: this.settingsAction,
          }
        }
    }
  }

  staleActionGuard = (oldAction: IAction, guarded: () => void) : (() => void) => {
    const { fetchLicenseInfo, conversionCohort } = this.props
    return () => {
      fetchLicenseInfo().then((newLicenseInfo) => {
        // dispatch the passed action iff it hasn't changed due to an updated license.
        // otherwise, React will update the component.
        let newAction = this.computeAction(newLicenseInfo, conversionCohort)
        if (oldAction === null || newAction === null)
          return

        if (oldAction.do !== newAction.do)
          return

        guarded()
      })
    }
  }

  getOnClick = (action: IAction) : (() => void) => {
    let onClick = () => {}

    const guardedAction = this.staleActionGuard(action, action.do)

    const loginData = action.loginData
    if (loginData) {
      // Prevent opening the modal if the action is stale, and prevent modal from executing a stale action.
      loginData.onSuccess = guardedAction
      onClick = this.staleActionGuard(action, () => this.props.requireLogin(loginData))
    } else {
      onClick = guardedAction
    }

    return onClick
  }

  combineStylesForCohort = (style: string, add: string) => {
    const stys = [style]
    const { conversionCohort, allFeaturesPro, paywallCompletionsRemaining } = this.props
    if (conversionCohort === ConversionCohorts.UsagePaywall) {
      if (allFeaturesPro && paywallCompletionsRemaining === 0) {
        stys.push(add)
      }
    }
    return stys.join(" ")
  }

  componentDidUpdate(prevProps: ProductBadgeProps) {
    if (this.props.kiteServerAvailable !== prevProps.kiteServerAvailable) {
      this.props.fetchLicenseInfo()
    }
  }

  render() {
    let { licenseInfo, conversionCohort } = this.props

    let action = this.computeAction(licenseInfo, conversionCohort)

    if (action === null) {
      return (
        <div className={this.props.className}>
          <div className={styles.container}>
            <div>
              <div className={styles.logo} />
            </div>
          </div>
        </div>
      )
    }

    const onClick = this.getOnClick(action)

    return (
      <div className={this.props.className}>
        <div className={styles.container}>
          <div className={this.combineStylesForCohort(styles.logo, styles.error)}/>
          { action.copy ?
            <button
              className={this.combineStylesForCohort(styles.button, styles.errorWithHover)}
              disabled={this.state.inFlight}
            >
              { action.copy }
            </button>
            : null
          }
        </div>
      </div>
    )
  }
}

function mapDispatchToProps (dispatch: ThunkDispatch<any, {}, AnyAction>) {
  return {
    fetchLicenseInfo: () => dispatch(fetchLicenseInfo()),
    getPaywallCompletionsRemaining: () => dispatch(getPaywallCompletionsRemaining()),
    getAllFeaturesPro: () => dispatch(getAllFeaturesPro()),
    getConversionCohort: () => dispatch(getConversionCohort()),
    startTrial: () => dispatch(startTrial(CTASource.ProductBadge)),
    requireLogin: (d: ILoginModalData) => dispatch(requireLogin(d)),
  }
}

function mapStateToProps (state: any) {
  return {
    licenseInfo: state.license.licenseInfo,
    conversionCohort: state.license.conversionCohort,
    allFeaturesPro: state.license.allFeaturesPro,
    paywallCompletionsRemaining: state.license.paywallCompletionsRemaining,
    kiteServerAvailable: state.settings.kiteServerAvailable,
  }
}

export default connect(mapStateToProps, mapDispatchToProps)(ProductBadge)
