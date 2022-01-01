import React from 'react'
import { connect } from 'react-redux'
import { AnyAction } from 'redux'
import { ThunkDispatch } from 'redux-thunk'

import AriaModal from 'react-aria-modal'
import { deactivate, ModalName, ModalNames } from '../../store/modals'
import * as account from '../../actions/account'
import { forceCheckOnline } from '../../actions/system'
import { AccountStates, Errors, getAccountState } from '../../store/account'
import styles from './index.module.css'

interface IProps {
  activeModal: ModalName,
  createPasswordlessAccount: (c: { email: string, ignore_channel: boolean }) => Promise<any>,
  checkNewEmail: (email: string) => Promise<any>,
  deactivate: () => void,
  defaultEmail: () => Promise<any>,
  forceCheckOnline: () => Promise<any>,
  getUser: () => Promise<any>,
  goal: string,
  logIn: (credentials: { email: string, password: string }) => Promise<any>,
  onSuccess: () => any,
  requestPasswordReset: (obj: { email: string }) => Promise<any>,
  user: { email?: string },
}

interface IState {
  email: string,
  error: string,
  loading: boolean,
  message: string,
  password: string,
  stage: stage,
}

export enum Goals {
  init = "",
  upgrade = "to upgrade to Kite Pro",
  startTrial = "to start your Kite Pro trial",
}

export type Goal = Goals.upgrade | Goals.startTrial | Goals.init

enum Stages {
  email = "email",
  password = "password",
}

type stage = Stages.email | Stages.password

const initialState = Object.freeze({
  email: "",
  password: "",
  error: "",
  message: "",
  loading: false,
  stage: Stages.email,
})

class LoginModal extends React.Component<IProps, IState> {
  private _inputRef = React.createRef<HTMLInputElement>();

  state = initialState;

  componentDidMount() {
    const { user, defaultEmail } = this.props
    if (user.email) {
      this.setState({ email: user.email }, this.onSubmitEmail)
    } else {
      defaultEmail()
        .then(({ data: { email, message }}) => {
          this.setState({ email, message })
        })
    }
  }

  /* User action handlers */

  onSubmitEmail = async (e?: React.SyntheticEvent) => {
    e && e.preventDefault()
    if (this.state.loading)
      return

    this.setState({ loading: true, error: "" })
    const { email } = this.state
    let { ok, state } = await this.getAccountState(email)
    // Errors reported already by getAccountState.
    if (ok) {
      switch (state) {
        case AccountStates.IsNew:
          ok = await this.createPasswordlessAccount()
          if (ok) {
            this.onSuccess()
          }
          break
        case AccountStates.ExistsNoPassword:
          ok = await this.requestPasswordReset()
          if (ok) {
            this.setState({ stage: Stages.password }, this.focusInputField)
          }
          break
        case AccountStates.ExistsWithPassword:
          this.setState(
            {
              stage: Stages.password,
              message: `Enter your password for ${this.state.email} to continue.`,
            },
            this.focusInputField
          )
          break
      }
    }
    this.setState({ loading: false })
  }

  onSubmitPassword = async (e: React.SyntheticEvent) => {
    e.preventDefault()
    if (this.state.loading)
      return

    this.setState({ loading: true, error: "" })
    const ok = await this.logIn()
    if (ok) {
      this.onSuccess()
    }
    this.setState({ loading: false })
  }

  onChangeEmail = () => {
    this.setState(initialState)
  }

  onEmailChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    this.setState({ email: e.target.value })
  }

  onPasswordChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    this.setState({ password: e.target.value })
  }

  /* Utility calls with error message handling */

  getAccountState = async (email: string): Promise<{ ok: boolean, state?: string | null }> => {
    const { checkNewEmail, forceCheckOnline } = this.props
    const res = await getAccountState(email, checkNewEmail, forceCheckOnline)
    const error = (res as { error: string }).error
    if (error) {
      switch (error) {
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
          this.setState({ error })
          break
      }
      return { ok: false, state: null }
    }
    return { ok: true, state: (res as { state: string }).state }
  }

  requestPasswordReset = async () => {
    const { email } = this.state
    const { requestPasswordReset } = this.props
    const { success } = await requestPasswordReset({ email })
    if (!success) {
      this.setState({ error: "An error occurred requesting password reset – please try again" })
      return false
    }
    this.setState({ message: `We've sent an email to ${email} for you to set your password first. After you set your password, login here.`, error: "" })
    return true
  }

  onSuccess = () => {
    const { deactivate, onSuccess } = this.props
    this.setState(initialState)
    onSuccess()
    deactivate()
  }


  /* Account Flows */

  createPasswordlessAccount = async () => {
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

  logIn = async () => {
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

  focusInputField = () => this._inputRef.current && this._inputRef.current.focus();

  render() {
    const { activeModal, deactivate, goal } = this.props
    if (activeModal !== ModalNames.LoginModal)
      return null

    const { stage, email, password, error, message } = this.state

    return (
      <AriaModal
        titleId="LoginModal"
        onExit={deactivate}
        dialogClass={styles.dialog}
        verticallyCenter={true}
        underlayStyle={{
          backdropFilter: 'blur(2px)',
          cursor: 'auto',
        }}
      >
        <div className={styles.container}>
          { stage === Stages.email &&
            <EmailFormWithRef
              error={error}
              fieldValue={email}
              message={message}
              goal={goal}
              onSubmit={this.onSubmitEmail}
              onChange={this.onEmailChange}
              placeholder={"Enter your email"}
              ref={this._inputRef}
            />
          }
          {
            stage === Stages.password &&
            <div className={styles.grid}>
              <PasswordFormWithRef
                email={email}
                error={error}
                fieldValue={password}
                goal={goal}
                message={message}
                onSubmit={this.onSubmitPassword}
                onChange={this.onPasswordChange}
                placeholder={`Password for ${email}`}
                ref={this._inputRef}
              />
              <div className={styles.spaceBetween}>
                <Link
                  text="Change email"
                  onClick={this.onChangeEmail}
                />
                <Link
                  text="Reset your password"
                  onClick={this.requestPasswordReset}
                />
              </div>
            </div>
          }
        </div>
      </AriaModal>
    )
  }
}

function Link({ text, onClick }: { text: string, onClick: () => any }) {
  return (
    <a
      href="#"
      className={styles.link}
      onClick={onClick}
    >
      { text }
    </a>
  )
}

interface IForm {
  error: string
  goal: string
  message: string
  onSubmit: (e: React.SyntheticEvent) => Promise<void>
  onChange: (e: React.ChangeEvent<HTMLInputElement>) => void
  placeholder: string
  fieldValue: string
}

interface IPasswordForm extends IForm {
  email: string
}

const EmailFormWithRef = React.forwardRef(EmailForm)
function EmailForm({ goal, fieldValue, error, message, onSubmit, onChange }: IForm, ref?: React.Ref<HTMLInputElement>): React.ReactElement {
  return (
    <form className={styles.form} onSubmit={onSubmit}>
      <div className={[styles.left, styles.title].join(" ")}>
        { "Create an account " + goal }
      </div>
      {(error || message) && <Message error={error} message={message} />}
      <input
        className={[styles.input, styles.fillWidth].join(" ")}
        name="email"
        type="email"
        placeholder="Enter your email"
        value={fieldValue}
        onChange={onChange}
        ref={ref}
      />
      <button className={[styles.button, styles.fillWidth].join(" ")}>
        Create Account
      </button>
    </form>
  )
}

const PasswordFormWithRef = React.forwardRef(PasswordForm)
function PasswordForm({ email, fieldValue, error, goal, message, onChange, onSubmit } : IPasswordForm, ref?: React.Ref<HTMLInputElement>) {
  return (
    <form onSubmit={onSubmit}>
      <div className={[styles.left, styles.title].join(" ")}>
        { "Log in to your account " + goal }
      </div>
      <Message error={error} message={message} />
      <input
        className={[styles.input, styles.fillWidth].join(" ")}
        name="password"
        type="password"
        placeholder={`Password for ${email}`}
        value={fieldValue}
        onChange={onChange}
        ref={ref}
      />
      <button className={[styles.button, styles.fillWidth].join(" ")}>
        Login
      </button>
    </form>
  )
}

// Message either shows an error, a message, or acts as a spacer (order of importance).
function Message({ error, message }: { error: string, message: string }) {
  const appliedStyles = [styles.left, styles.message]
  let displayText = ""

  if (!error && !message) {
    appliedStyles.push(styles.hidden)
    displayText = "invisible placeholder"
  } else if (error) {
    appliedStyles.push(styles.errorColor)
    displayText = error
  } else {
    appliedStyles.push(styles.messageColor)
    displayText = message
  }

  return (
    <div className={appliedStyles.join(" ")}>
      { displayText }
    </div>
  )
}

function mapStateToProps(state: any) {
  return {
    activeModal: state.modals.active,
    goal: state.modals.loginModalData.goal,
    onSuccess: state.modals.loginModalData.onSuccess,
    user: state.account.user,
  }
}

function mapDispatchToProps(dispatch: ThunkDispatch<any, {}, AnyAction>) {
  return {
    checkNewEmail: (email: string) => dispatch(account.checkNewEmail(email)),
    createPasswordlessAccount: (c: { email: string, ignore_channel: boolean }) => dispatch(account.createPasswordlessAccount(c)),
    deactivate: () => dispatch(deactivate()),
    defaultEmail: () => dispatch(account.defaultEmail()),
    forceCheckOnline: () => dispatch(forceCheckOnline()),
    getUser: () => dispatch(account.getUser()),
    logIn: (credentials: { email: string, password: string }) => dispatch(account.logIn(credentials)),
    requestPasswordReset: (obj: { email: string }) => dispatch(account.requestPasswordReset(obj)),
  }
}

export default connect(mapStateToProps, mapDispatchToProps)(LoginModal)
