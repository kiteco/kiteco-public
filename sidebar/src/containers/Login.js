import React from 'react'
import { connect } from 'react-redux'
import { push } from 'react-router-redux'

import * as account from '../actions/account'
import * as settings from '../actions/settings'
import { getVersion } from '../actions/system'
import * as plugins from '../actions/plugins'
import { forceCheckOnline } from '../actions/system'
import { notify, dismiss } from '../store/notification'
import { fetchConversionCohort, fetchCohortTimeoutMS } from '../store/license'

import { track } from '../utils/analytics'
import { timeoutAfter } from '../utils/fetch'
import Spinner from '../components/Spinner'
import { AccountStates, Errors, getAccountState } from '../store/account'

import {
  getDefaultPlugins,
  getRunningInstallDisablePlugins,
  installMultiple,
} from '../utils/plugins'

import '../assets/login.css'

const CANNOT_DETECT_RUN = [ "neovim" ]

const stages = Object.freeze({
  email: "email",
  password: "password",
})

class Login extends React.Component {
  state = {
    email: "",
    password: "",
    error: "",
    message: "",
    stage: stages.email,
    loading: true,
    installAllEditors: true,
    emailRequired: undefined,
  }

  componentDidMount() {
    const { status, getUser, getServer, getPlugins, alreadySetup, getEmailRequired } = this.props

    if (status !== "logged-out")  {
      getUser()
    }
    getServer()

    getEmailRequired().then(emailRequired => this.setState({ emailRequired }))

    // send event when the login screen is first shown
    if (this.state.emailStepShown !== true) {
      this.setState({ emailStepShown: true })
      track({
        event: "onboarding_email_step_shown",
      })
    }

    if (!alreadySetup) {
      getPlugins()
        .then(() => {
          const plugins = this.props.plugins || []
          const numRunning = plugins.reduce((all, p) => all + p.running ? 1 : 0, 0)
          const numInstalled = plugins.reduce((all, p) => (all + ((p.editors.length > 0) ? 1 : 0)), 0)
          const numRunDetectable = plugins.reduce((all, p) => all + (!CANNOT_DETECT_RUN.includes(p.id) ? 1 : 0), 0)
          const numRunDetectableInstalled = plugins.reduce((all, p) => all + ((!CANNOT_DETECT_RUN.includes(p.id) && p.editors.length > 0) ? 1 : 0), 0)

          track({
            event: "login_screen_shown",
            props: {
              numRunning,
              numRunDetectable,
              numInstalled,
              numRunDetectableInstalled,
            },
          })
        })
    }

    const { defaultEmail, location } = this.props

    const queryEmail = location && location.state && location.state.queryEmail
    if (queryEmail) {
      this.setState({ email: queryEmail })
    } else {
      defaultEmail().then(({ data }) => {
        if (typeof(data) !== 'undefined') {
          this.setState({
            email: data.email || "",
            message: data.message || "",
          })
        }
      })
    }

    const activateLicense = location && location.state && location.state.activateLicense
    if (activateLicense) {
      this.setState({
        activateLicense: true,
        message: "Sign in to activate your license",
      })
    }

    this.setState({ loading: false })
  }


  /* User action handlers */

  onSubmitEmail = async (e) => {
    e.preventDefault()
    // don't want to double submit
    if (this.state.loading)
      return

    this.setState({ loading: true, error: "" })

    track({
      event: "onboarding_email_step_advanced",
      props: {
        email_provided: this.state.email !== "",
      },
    })

    let { ok, state } = await this.getAccountState()
    // Errors reported already by getAccountState.
    if (ok) {
      switch (state) {
        case AccountStates.IsNew:
          ok = await this.passwordlessAccountFlow()
          if (ok) {
            await this.epilogue()
          }
          break
        case AccountStates.ExistsNoPassword:
          ok = await this.requestPasswordReset()
          if (ok) {
            this.setState({ stage: stages.password }, this.focusPasswordField)
          }
          break
        case AccountStates.ExistsWithPassword:
          this.setState(
            {
              stage: stages.password,
              message: `Enter your password for ${this.state.email} to continue.`,
            },
            this.focusPasswordField
          )
          break
      }
    }
    this.setState({ loading: false })
  }

  onSubmitPassword = async (e) => {
    e && e.preventDefault()
    // Prevent double submit
    if (this.state.loading)
      return

    this.setState({ loading: true, error: "" })

    const ok = await this.logInFlow()
    if (ok) {
      await this.epilogue()
    }
    this.setState({ loading: false })
  }

  onContinueWithoutEmail = (e) => {
    e && e.preventDefault()
    track({
      event: "onboarding_email_step_advanced",
      props: {
        email_provided: false,
      },
    })
    this.epilogue()
  }

  onRequestPasswordReset = async e => {
    e && e.preventDefault()
    this.setState({ loading: true, error: "" })
    await this.requestPasswordReset()
    this.setState({ loading: false })
  }

  onChangeEmail = e => {
    e && e.preventDefault()
    this.setState({
      stage: stages.email,
      message: "",
      error: "",
      password: "",
      loading: false,
    })
  }


  /* Utility calls with error and message handling */

  getAccountState = async () => {
    const { email } = this.state
    const { checkEmail, forceCheckOnline } = this.props
    const res = await getAccountState(email, checkEmail, forceCheckOnline)
    if (res.error) {
      switch (res.error) {
        case Errors.NotOnlineError:
          this.setState({ error: "Please connect to the internet to continue" })
          break
        case Errors.NoEmailError:
          this.setState({ error: "Email required" })
          break
        case Errors.InvalidEmailError:
          this.setState({ error: "Invalid email address" })
          break
        default:
          // Some other error upstream error exists.
          this.setState({ error: res.error })
          break
      }
      return { ok: false, state: null }
    }
    return { ok: true, state: res.state }
  }

  requestPasswordReset = async () => {
    const { email } = this.state
    const { requestPasswordReset } = this.props
    const { success } = await requestPasswordReset({ email })
    if (!success) {
      this.setState({ error: "An error occurred requesting password reset – please try again" })
      return false
    }
    this.setState({ message: `We've sent an email to ${email} to set your password. Once you've done so, login with it here.`, error: "" })
    return true
  }


  /* Account Flows */

  passwordlessAccountFlow = async () => {
    const { createPasswordlessAccount } = this.props
    const { email } = this.state
    // create a passwordless account and continue to kite setup
    const { success, error } = await createPasswordlessAccount({ email, ignore_channel: true })
    if (!success) {
      this.setState({ error })
      return false
    }
    return true
  }

  logInFlow = async () => {
    const { email, password } = this.state
    const { logIn } = this.props
    let { type, error } = await logIn({ email, password })

    if (type === account.LOG_IN_FAILED) {
      if (error === account.NO_PASSWORD) {
        await this.requestPasswordReset()
      } else {
        this.setState({ error })
      }
      return false
    }
    return true
  }

  // What happens after authentication, where to go next
  epilogue = async (e) => {
    e && e.preventDefault()

    const { installAllEditors } = this.state
    const { setupCompleted, push, getVersion, getShowChooseEngine } = this.props
    if (!setupCompleted) {
      const { success, data } = await getVersion()
      track({
        event: "onboarding_identity_first_established",
        props: {
          client_version: success ? data : "failed_version_fetch",
        },
      })

      const { setSetupNotCompleted, setCurrentSetupStage, push, fetchConversionCohort } = this.props
      // standard set up flow
      const chooseEngine = await getShowChooseEngine()
      if (installAllEditors) {
        this.setState({ error: "", loading: true })
        await Promise.all([
          timeoutAfter(fetchConversionCohort, fetchCohortTimeoutMS),
          this.oneClickInstallProcess(),
        ])

        if (chooseEngine) {
          push("/choose-engine#restart-plugins")
        } else {
          push("/setup#restart-plugins")
        }
      } else {
        await setCurrentSetupStage()
        await setSetupNotCompleted()

        // always show normal setup flow here because they selected it
        if (chooseEngine) {
          push("/choose-engine")
        } else {
          push("/setup")
        }
      }
    } else {
      push("/login-redirector")
    }
  }

  oneClickInstallProcess = async () => {
    const {
      plugins,
      system,
      forceCheckOnline,
      installPlugin: install,
      notify,
    } = this.props
    // install plugins
    const toInstall = getDefaultPlugins({ plugins, system })
    const toNotifyInstallDisabled = getRunningInstallDisablePlugins({ plugins, system })
    const { results, installed, filtered, successes, errors } = await installMultiple({
      forceCheckOnline,
      toInstall,
      install,
    })
    track({
      event: "streamlined_onboarding_selected",
      props: {
        num_editors_rendered: results.length,
        num_plugins_installed: installed.length,
        plugins_installed: installed,
      },
    })
    if (!successes.every(s => s)) {
      let error = errors.pop()
      this.setState({ error: error.title || error })
      // TODO: surface these errors to the user somehow. Notification?
    }
    // send notifications for plugins that need to restart
    results.forEach(({ success, data }) => {
      if (success) {
        const { requires_restart, running, id } = data
        if (requires_restart === true && running === true) {
          notify({
            id: `plugins--${id}`,
            component: 'plugins',
            payload: {
              ...data,
            },
          })
        }
        dismiss('noplugins')
      }
    })
    // send notifications for plugins that could not be installed
    filtered.forEach(({ id, name }) => {
      notify({
        id: `plugins--${id}`,
        component: 'remote-plugin-offline',
        payload: { name, id },
      })
    })
    Object.values(toNotifyInstallDisabled).forEach(({ id, name }) => {
      notify({
        id: `plugins--${id}`,
        name,
        component: "running-plugin-install-failure",
        payload: { name, id },
      })
    })
  }


  /* MISC */

  getOnChangeForField = field => event => {
    const value = event.target.value
    this.setState({ [field]: value })

    if (this.state.stage === stages.email) {
      this.setState({ message: "" })
    }
  }

  focusPasswordField = () => this._password && this._password.focus();
  setInstallAllEditors = status => this.setState({ installAllEditors: status })

  render() {
    const { setupCompleted, className } = this.props

    if (typeof(setupCompleted) === 'undefined' || setupCompleted === 'notset') {
      return (
        <div className={`login login--${className} login--loading`}>
          <Spinner text="Checking your setup…"/>
        </div>
      )
    }

    const { loading, message, error, stage } = this.state
    const { activateLicense, installAllEditors } = this.state

    return (
      <form
        onSubmit={ stage === stages.email ? this.onSubmitEmail : this.onSubmitPassword }
        className={`login login--${className} ${loading ? "login--loading" : ""}`}
      >
        { loading &&
          <Spinner text="One moment please…"/>
        }
        { message &&
          <div className="login__row">
            <center>
              { message }
            </center>
          </div>
        }
        { error &&
          <div className="login__error">
            { error }
          </div>
        }
        <div className="login__row login__start">
          { stage === stages.email &&
            <input
              name="email"
              type="email"
              className="login__input"
              placeholder="Email"
              value={this.state.email}
              onChange={this.getOnChangeForField("email")}
            />
          }
        </div>
        <div className="login__row">
          { stage === stages.password &&
            <input
              name="password"
              type="password"
              className="login__input"
              placeholder="Password"
              value={this.state.password}
              onChange={this.getOnChangeForField("password")}
              ref={input => this._password = input}
            />
          }
        </div>
        { !setupCompleted &&
          <div>
            <Checkbox
              setTo={installAllEditors}
              setOn={() => this.setInstallAllEditors(true)}
              setOff={() => this.setInstallAllEditors(false)}
            >
              Install Kite for all supported editors
            </Checkbox>
            <Checkbox
              setTo={!installAllEditors}
              setOn={() => this.setInstallAllEditors(false)}
              setOff={() => this.setInstallAllEditors(true)}
            >
              Let me choose which editor plugins to install
            </Checkbox>
          </div>
        }
        <button
          className="setup__button login__submit"
          type="submit"
        >
          Continue
        </button>
        <div className={`setup__links-container ${stage === stages.email ? "setup__links-container--centered" : ""}`}>
          { stage === stages.password &&
              <Link
                onClick={this.onChangeEmail}
                text="Change email"
              />
          }
          { stage === stages.password &&
              <Link
                onClick={this.onRequestPasswordReset}
                text="Forgot password?"
              />
          }
          { /* the 'Continue without email' link may not appear for up to
               a second because an external request is needed to fetch
               country data to determine if an email is required for the
               user's country
            */
            stage === stages.email &&
            (activateLicense || (this.state.emailRequired !== undefined && !this.state.emailRequired)) &&
            <Link
              onClick={activateLicense ? this.epilogue : this.onContinueWithoutEmail}
              text={activateLicense ? "Cancel" : "Continue without email" }
            />
          }
        </div>
      </form>
    )
  }
}

function Link({ text, onClick }) {
  return (
    <a
      href="#"
      className="home__link__settings"
      onClick={onClick}
    >
      { text }
    </a>
  )
}

function Checkbox({ setTo, setOn, setOff, children }) {
  return (
    <div className="login__checkbox">
      <button
        className={`login__check ${ setTo ? "login__check--checked" : ""}`}
        type="button"
        onClick={ setTo ? setOff : setOn }
      >
        <div className="login__check__inner"/>
      </button>
      <div className="login__column">
        { children }
      </div>
    </div>
  )
}

const mapStateToProps = state => ({
  user: state.account.user,
  status: state.account.status,
  webapp: state.settings.webapp,
  system: state.system,
  plugins: state.plugins,
  setupCompleted: state.settings.setupCompleted,
  metricsId: state.account.metricsId,
  installId: state.account.installId,
})

const mapDispatchToProps = dispatch => ({
  fetchConversionCohort: () => dispatch(fetchConversionCohort()),
  getEmailRequired: () => dispatch(settings.getEmailRequired()),
  getUser: () => dispatch(account.getUser()),
  getVersion: () => dispatch(getVersion()),
  defaultEmail: () => dispatch(account.defaultEmail()),
  logIn: credentials => dispatch(account.logIn(credentials)),
  logOut: () => dispatch(account.logOut()),
  createPasswordlessAccount: credentials => dispatch(account.createPasswordlessAccount(credentials)),
  getServer: () => dispatch(settings.getServer()),
  requestPasswordReset: email => dispatch(account.requestPasswordReset(email)),
  forceCheckOnline: () => dispatch(forceCheckOnline()),
  notify: params => dispatch(notify(params)),
  dismiss: id => dispatch(dismiss(id)),
  push: params => dispatch(push(params)),
  getPlugins: () => dispatch(plugins.getPlugins()),
  installPlugin: id => dispatch(plugins.installPlugin(id)),
  setHaveShownWelcome: () => dispatch(settings.setHaveShownWelcome()),
  getHaveShownWelcome: () => dispatch(settings.getHaveShownWelcome()),
  setSetupCompleted: () => dispatch(settings.setSetupCompleted()),
  setSetupNotCompleted: () => dispatch(settings.setSetupNotCompleted()),
  setCurrentSetupStage: stage => dispatch(settings.setCurrentSetupStage(stage)),
  checkEmail: email => dispatch(account.checkNewEmail(email)),
  setAutoInstallPluginsEnabled: enabled => dispatch(settings.setAutoInstallPluginsEnabled(enabled)),
  getShowChooseEngine: () => dispatch(settings.getShowChooseEngine()),
})

export default connect(mapStateToProps, mapDispatchToProps)(Login)
