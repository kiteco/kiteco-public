import React from 'react'
import { connect } from "react-redux"
import { ThunkDispatch } from "redux-thunk"
import { AnyAction } from "redux"
import { track } from '../utils/analytics'
import '../assets/setup.css'
import localIcon from '../assets/ic_kite_local.svg'
import cloudIcon from '../assets/ic_kite_cloud.svg'
import chevronIcon from '../assets/ic_chevron.svg'
import styles from '../assets/choose-engine.module.css'
import { push } from 'react-router-redux'
import * as settings from "../actions/settings"

interface ChooseEngineProps {
  location: Location
  email: string
  push: (nextPage: string) => void
  setupStage: string
  setSelectedEngine: (engineID: string) => void
}

interface ChooseEngineState {
  engine: string
  showDetails: boolean
  email: string
  python_devs?: boolean
  localFirst: boolean
}

class ChooseEngine extends React.Component<ChooseEngineProps, ChooseEngineState> {
  constructor(props: ChooseEngineProps) {
    super(props)
    this.state = {
      engine: "local",
      showDetails: false,
      email: props.email,
      python_devs: undefined,
      localFirst: getLocalFirst(),
    }
    const { localFirst } = this.state
    let order = localFirst ? "local, cloud" : "cloud, local"
    track({ event: "onboarding_test_choose_engine_shown", props: { order: order }})
  }

  onEmailChange = (e: any) => {
    e.preventDefault()
    this.setState({
      email: e.target.value,
    })
  }

  onPythonDevsChange = (e: any) => {
    e.preventDefault()
    this.setState({
      python_devs: e.target.value,
    })
  }

  back = () => {
    this.setState({
      showDetails: false,
    })
  }

  continue = (engine: string) => {
    const { push, setupStage, setSelectedEngine } = this.props
    const { showDetails, email, python_devs } = this.state
    const nextPage = setupStage && setupStage !== "" ? `/setup${setupStage}` : "/setup"

    if (engine === "local") {
      track({ event: "onboarding_test_choose_engine_advanced", props: { selection: engine }})

      setSelectedEngine(engine)
      this.setState({
        engine: engine,
      })
      push(nextPage)
    } else if (showDetails) {
      // continue with the regular onboarding flow
      track({ event: "onboarding_test_cloud_info_advanced", props: undefined })

      setSelectedEngine(engine)
      push(nextPage)
    } else {
      // cloud was chosen, now show details page
      track({ event: "onboarding_test_choose_engine_advanced", props: { selection: engine }})
      track({ event: "onboarding_test_cloud_info_shown", props: undefined })

      setSelectedEngine(engine)
      this.setState({
        showDetails: true,
        engine: engine,
      })
    }
  }

  render() {
    const { engine, showDetails, localFirst } = this.state

    return <div className={`setup-main ${styles.chooseEngineMain}`}>
      <div className="setup__header--invisible"/>
      <div className={`setup ${styles.setup__engine}`}>
        {!showDetails && <div>
          <h2 className="setup__title">Choose your engine</h2>
          <p className="setup__text">Please select where Kite's completions are computed.</p>
          { localFirst ? <div>
            <LocalButton onClick={() => this.continue("local")}/>
            <CloudButton onClick={() => this.continue("cloud")}/>
          </div>
            :
            <div>
              <CloudButton onClick={() => this.continue("cloud")}/>
              <LocalButton onClick={() => this.continue("local")}/>
            </div>
          }
        </div>}

        {showDetails && engine === "cloud" && <div>
          <h2 className="setup__title">Kite Cloud is currently in development</h2>
          <p className="setup__text">Kite Cloud is currently in development by our engineering team. Once we are
            live you will automatically get access to smarter completions with lower memory and CPU usage.</p>
          <p className="setup__text">In the meantime Kite will return completions from our local engine.</p>
        </div>}

        {showDetails && <div>
          <button
            className="setup__button showup__animation"
            onClick={() => this.continue("cloud")}
          >
            Continue
          </button>

          <div className={styles.setupEngineCenter}>
            <a href="#" className={styles.setupEngineBack} onClick={this.back}>Back</a>
          </div>
        </div>}
      </div>
    </div>
  }
}

interface ButtonProps {
  icon: string
  label: string
  info: string[]
  onClick: () => void
}
function EngineButton(Props: ButtonProps) {
  return (
    <div className={styles.setupEngineSection}>
      <button
        className={styles.setupEngineButton}
        onClick={Props.onClick}
      >

        <div className={styles.setupEngineHeader}>
          <img alt="" className={styles.chooseEngineEmoji} src={Props.icon}/>
          <div className={styles.setupEngineChoice}>
            <h3 className={styles.setupEngineLabel}>{Props.label}</h3>
            <img alt="" className={styles.chevronEmoji} src={chevronIcon}/>
          </div>
        </div>

        <ul className={`${styles.setupEngineList} setup__text`}>
          {Props.info.map((infoItem) => {
            return <li>{infoItem}</li>
          })}
        </ul>
      </button>
    </div>
  )
}

function LocalButton(props: { onClick: () => void }) {
  const info = [
    "Code never leaves your computer",
    "Smaller models that fit on your computer",
    "Good results, reasonable memory and CPU usage",
  ]

  return (
    <EngineButton
      icon={localIcon}
      label={'Local processing'}
      info={info}
      onClick={props.onClick}
    />
  )
}

function CloudButton(props: { onClick: () => void }) {
  const info = [
    "Sends your code to Kite's powerful servers where it remains private",
    "40% smarter results, lower memory and CPU usage",
  ]

  return (
    <EngineButton
      icon={cloudIcon}
      label={'Cloud processing'}
      info={info}
      onClick={props.onClick}
    />
  )
}

function getLocalFirst () {
  return Math.random() >= 0.5
}

const mapStateToProps = (state: any, ownProps: ChooseEngineProps) => ({
  ...ownProps,
  email: state.account.user.email,
  setupStage: ownProps.location.hash,
})

const mapDispatchToProps = (dispatch: ThunkDispatch<any, {}, AnyAction>) => ({
  push: (params: any) => dispatch(push(params)),
  setSelectedEngine: (engineID: string) => dispatch(settings.setSelectedEngine(engineID)),
})

export default connect(mapStateToProps, mapDispatchToProps)(ChooseEngine)
