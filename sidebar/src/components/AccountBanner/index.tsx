import React from 'react'
import { connect } from 'react-redux'
import { AnyAction } from 'redux'
import { ThunkDispatch } from 'redux-thunk'
import { Redirect } from 'react-router'

import pluralize from 'pluralize'
import Gravatar from 'react-gravatar'
import Spinner from '../Spinner'

import { LicenseInfo, Product, Plan, CTASource, ConversionCohort, ConversionCohorts } from '../../store/license'
import { fetchLicenseInfo, startTrial } from '../../store/license'
import { requireLogin, ILoginModalData } from '../../store/modals'
import { Goals } from '../../components/LoginModal'
import { logOut } from '../../actions/account'
import { localhostProxy } from '../../utils/urls'
import { timeoutAfter, errTimedOut } from '../../utils/fetch'

import styles from './index.module.css'
import { Domains } from '../../utils/domains'

const { shell } = window.require("electron")

interface AccountBannerProps {
  className: string
  licenseInfo: LicenseInfo
  conversionCohort: ConversionCohort
  email: string
  kiteServerAvailable: boolean
  fetchLicenseInfo: () => Promise<LicenseInfo>
  requireLogin: (d: ILoginModalData) => void
  startTrial: (src: CTASource) => void
  logOut: () => Promise<any>
}

interface AccountBannerState {
  loading: boolean
  redirectToLogin: boolean
  gravatarReachable: boolean
}

interface IProRowElement {
  get?: (onClick: () => void) => React.ReactElement
  text?: string
  onClick: () => void
  // Existence of loginData indicates onClick action requires login.
  loginData?: ILoginModalData
}

interface IProRowTextField extends IProRowElement {
  get: (onClick: () => void) => React.ReactElement
}

interface IProRowButton extends IProRowElement {
  text: string
}

interface IProRow {
  textField: IProRowTextField,
  button: IProRowButton,
}

class AccountBanner extends React.Component<AccountBannerProps, AccountBannerState> {
  constructor(props: AccountBannerProps) {
    super(props)
    this.state = {
      loading: true,
      redirectToLogin: false,
      gravatarReachable: false,
    }
    Promise.all([props.fetchLicenseInfo(), this._gravatarReachable()])
      .then( () => this.setState({ loading: false }) )
      .catch( () => this.setState({ loading: false }) )
  }

  async _gravatarReachable(): Promise<void> {
    // react-gravatar does not expose an onError method
    // so we roll our own check before using it
    try {
      const pingGravatar = () => fetch("https://www.gravatar.com/avatar", { cache: 'reload' })
      const resolved = await timeoutAfter(pingGravatar, 500)
      this.setState({ gravatarReachable: resolved !== errTimedOut && resolved.status < 400 })
    } catch (err) {
      this.setState({ gravatarReachable: false })
    }
  }

  upgradeAction() {
    shell.openExternal(localhostProxy('/clientapi/desktoplogin?d=%2Fpro%3Floc%3Dcopilot_settings%26src%3Dupgrade'))
  }

  helpAction() {
    shell.openExternal(`https://${Domains.Help}/article/128-kite-pro`)
  }

  settingsAction() {
    shell.openExternal(localhostProxy('/clientapi/desktoplogin?d=/settings'))
  }

  startTrialFromLearnMore = () => {
    this.props.startTrial(CTASource.SettingsLearnMore)
  }

  startTrialFromButton = () => {
    this.props.startTrial(CTASource.SettingsButton)
  }

  computeProRow(licenseInfo: LicenseInfo, conversionCohort: ConversionCohort): IProRow | null {
    if (!licenseInfo) {
      return null
    }

    switch (conversionCohort) {
      case ConversionCohorts.UsagePaywall:
        return this.computeUsagePaywallProRow(licenseInfo)
      case ConversionCohorts.Autostart:
      case ConversionCohorts.QuietAutostart:
        if (licenseInfo.product === Product.Free && licenseInfo.trial_available) {
          return null
        }
        return this.computeOptInProRow(licenseInfo)
      case ConversionCohorts.OptIn:
        return this.computeOptInProRow(licenseInfo)
      case ConversionCohorts.Unset:
        return null
    }
  }

  computeUsagePaywallProRow(licenseInfo: LicenseInfo): IProRow | null {
    if (!licenseInfo) {
      return null
    }

    if (licenseInfo.product === Product.Free) {
      return {
        textField: {
          get: (onClick) => (
            <LearnMore onClick={onClick} body="You are using Kite Free – Upgrade now to Kite Pro"/>
          ),
          onClick: this.upgradeAction,
        },
        button: {
          text: "Upgrade",
          onClick: this.upgradeAction,
        },
      }
    } else {
      return {
        textField: {
          get: () => <span>You are using Kite Pro</span>,
          onClick: () => {},
        },
        button: {
          text: "Manage",
          onClick: this.settingsAction,
        },
      }
    }
  }

  computeOptInProRow(licenseInfo: LicenseInfo): IProRow | null {
    if (!licenseInfo) {
      return null
    }

    let ret: IProRow = {
      textField: {
        get: () => <span></span>,
        onClick: () => {},
      },
      button: {
        text: "",
        onClick: () => {},
      },
    }

    switch (licenseInfo.product) {
      case Product.Free:
        if (licenseInfo.trial_available_duration) {
          const { unit, value } = licenseInfo.trial_available_duration
          ret = {
            textField: {
              get: (onClick) => (
                <LearnMore onClick={onClick} body={`Try out Kite Pro free for ${value} ${pluralize(unit, value)}`}/>
              ),
              onClick: this.startTrialFromLearnMore,
              loginData: {
                goal: Goals.startTrial,
                // onSuccess is a stub here. Set at render time to include appropriate guards.
                onSuccess: () => {},
              },
            },
            button: {
              text: "Start Trial",
              onClick: this.startTrialFromButton,
              loginData: {
                goal: Goals.startTrial,
                // onSuccess is a stub here. Set at render time to include appropriate guards.
                onSuccess: () => {},
              },
            },
          }
        } else {
          ret = {
            textField: {
              get: (onClick) => (
                <LearnMore onClick={onClick} body="You are using Kite Free – Upgrade now to Kite Pro"/>
              ),
              onClick: this.upgradeAction,
              loginData: {
                goal: Goals.upgrade,
                onSuccess: () => {},
              },
            },
            button: {
              text: "Upgrade",
              onClick: this.upgradeAction,
              loginData: {
                goal: Goals.upgrade,
                // onSuccess is a stub here. Set at render time to include appropriate guards.
                onSuccess: () => {},
              },
            },
          }
        }
        break
      case Product.Pro:
        if (licenseInfo.plan === Plan.Trial) {
          ret = {
            textField: {
              get: (onClick) => (
                <LearnMore onClick={onClick} body={`You have ${licenseInfo.days_remaining} days left in your Kite Pro trial`}/>
              ),
              onClick: this.helpAction,
            },
            button: {
              text: "Upgrade",
              onClick: this.upgradeAction,
              loginData: {
                goal: Goals.upgrade,
                // onSuccess is a stub here. Set at render time to include appropriate guards.
                onSuccess: () => {},
              },
            },
          }
        } else {
          ret = {
            textField: {
              get: () => <span>You are using Kite Pro</span>,
              onClick: () => {},
            },
            button: {
              text: "Manage",
              onClick: this.settingsAction,
            },
          }
        }
        break
    }

    return ret
  }

  logOut = () => {
    this.props.logOut()
  }

  logIn = () => {
    this.setState({ redirectToLogin: true })
  }

  staleActionGuard = (oldProRow: IProRow, selector: (p: IProRow) => () => void, action: () => void) : (() => void) => {
    const { fetchLicenseInfo, conversionCohort } = this.props
    return () => {
      fetchLicenseInfo().then((newLicenseInfo) => {
      // dispatch the passed action iff it hasn't changed due to an updated license.
      // otherwise, React will update the component.
        let newProRow = this.computeProRow(newLicenseInfo, conversionCohort)
        if (oldProRow === null || newProRow === null)
          return

        if (selector(oldProRow) !== selector(newProRow))
          return

        action()
      })
    }
  }

  // getOnClick adds appropriate stale-guards and returns the wrapped action.
  getOnClick = (row: IProRow | null, selector: (r: IProRow) => IProRowElement) : (() => void) => {
    let onClick = () => {}

    if (row) {
      const onClickSelector = (r: IProRow) => selector(r).onClick
      const guardedAction = this.staleActionGuard(row, onClickSelector, onClickSelector(row))

      const loginData = selector(row).loginData
      if (loginData) {
        // Prevent opening the modal if the action is stale, and prevent modal from executing a stale action.
        loginData.onSuccess = guardedAction
        onClick = this.staleActionGuard(row, onClickSelector, () => this.props.requireLogin(loginData))
      } else {
        onClick = guardedAction
      }
    }

    return onClick
  }

  componentDidUpdate(prevProps: AccountBannerProps) {
    if (this.props.kiteServerAvailable !== prevProps.kiteServerAvailable) {
      this.props.fetchLicenseInfo()
    }
  }

  render() {
    const { loading, redirectToLogin } = this.state

    if (loading) {
      return (
        <div className={[styles.container, styles.loading].join(" ")}>
          <Spinner theme={null} text="Fetching account info..."/>
        </div>
      )
    }

    if (redirectToLogin) {
      return <Redirect to="/login"/>
    }

    const { email, licenseInfo, conversionCohort } = this.props

    const proRow = this.computeProRow(licenseInfo, conversionCohort)
    const onProButtonClick = this.getOnClick(proRow, (r: IProRow) => r.button)
    const onLearnMoreClick = this.getOnClick(proRow, (r: IProRow) => r.textField)

    return (
      <div className={styles.container}>
        <Avatar email={email} useGravatar={this.state.gravatarReachable}/>
        <TextField
          className={styles.userRow}
          body={ email ? email : "Not logged in" }
        />
        <Button
          className={styles.userRow}
          onClick={email ? this.logOut : this.logIn}
          body={ email ? "Log out" : "Login or Create Account" }
        />
        {
          proRow &&
            <TextField
              className={[styles.proRow, styles.shiftDown, styles.smallFont].join(" ")}
              body={proRow.textField.get(onLearnMoreClick)}
            />
        }
        {
          proRow &&
            <Button
              className={styles.proRow}
              onClick={onProButtonClick}
              body={proRow.button.text}
            />
        }
      </div>
    )
  }
}

function Avatar(props: { email: string, useGravatar: boolean }) {
  const classes = [styles.userRow, styles.gravatar].join(" ")
  if (props.useGravatar) {
    return (
      <Gravatar
        className={classes}
        email={ props.email ? props.email : "" }
        protocol="https://"
      />
    )
  }
  return (
    <div className={classes}>
      <div className={styles.kiteIcon}/>
    </div>
  )
}

interface IClassBody {
  body: string | React.ReactElement;
  className?: string
}

interface IButton extends IClassBody {
  onClick: () => void
}

function TextField({ className, body }: IClassBody) {
  return (
    <span className={[styles.text, className].join(" ")}>
      { body }
    </span>
  )
}
function Button({ className, body, onClick }: IButton) {
  return (
    <button
      className={[styles.button, className].join(" ")}
      onClick={onClick}
    >
      { body }
    </button>
  )
}

function LearnMore({ body, onClick }: IButton): React.ReactElement {
  return (
    <div>
      { body }
      {' '}
      (
      <button
        className={[styles.linkButton, styles.smallFont, styles.text].join(" ")}
        onClick={onClick}
      >
        Learn more
      </button>
      )
    </div>
  )
}

function mapDispatchToProps (dispatch: ThunkDispatch<any, {}, AnyAction>) {
  return {
    fetchLicenseInfo: () => dispatch(fetchLicenseInfo()),
    startTrial: (src: CTASource) => dispatch(startTrial(src)),
    requireLogin: (d: ILoginModalData) => dispatch(requireLogin(d)),
    logOut: () => dispatch(logOut()),
  }
}

function mapStateToProps (state: any) {
  return {
    licenseInfo: state.license.licenseInfo,
    conversionCohort: state.license.conversionCohort,
    email: state.account.user.email,
    kiteServerAvailable: state.settings.kiteServerAvailable,
  }
}

export default connect(mapStateToProps, mapDispatchToProps)(AccountBanner)
