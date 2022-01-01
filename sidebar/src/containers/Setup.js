import React from 'react'
import { connect } from 'react-redux'
import { push } from 'react-router-redux'

import { track } from '../utils/analytics'

import * as account from '../actions/account'
import * as plugins from '../actions/plugins'
import * as settings from '../actions/settings'
import { disableAutosearch } from '../actions/search'
import * as system from '../actions/system'
import '../assets/setup.css'

import Error from './Error'

import SetupPlugins from '../components/Setup/Plugins'
import SetupRestartPlugins from '../components/Setup/RestartPlugins'
import Done from '../components/Setup/Done'
import {
  getConversionCohort,
  getAllFeaturesPro,
} from '../store/license'

class Setup extends React.Component {

  componentDidMount() {
    track({ event: "onboarding_setup_mounting" })
    this.props.getSystemInfo()
    const start = (new Date()).getTime() / 1000
    this.props.getPlugins().then(() => {
      const end = (new Date()).getTime() / 1000
      track({
        event: "onboarding_plugins_initial_loaded",
        props: {
          time: end - start,
        },
      })
    })
    this.props.getAutoInstallPluginsEnabled()
    this.props.getIconVisible()
    this.props.setAutosearchDefaultOff()
    this.props.getLastSetupStage().then((stage) => {
      if (stage) {
        this.props.push({
          ...this.props.location,
          hash: stage,
        })
      }
    })
  }

  stages = [
    "#restart-plugins",
    "#done",
  ]

  advance = (payload) => {
    const i = this.stages.indexOf(this.props.stage)
    if (i !== -1) {
      track({ event: `onboarding_${ this.props.stage.substring(1) }_step_advanced` })
    } else {
      track({ event: "onboarding_plugins_step_advanced" })
    }
    const next = Math.min(i + 1, this.stages.length - 1)
    if (i !== next) {
      this.props.setCurrentSetupStage(this.stages[next])
      this.props.push({
        ...this.props.location,
        hash: this.stages[next],
        state: payload,
      })
    }
  }

  setSetupCompleted = async () => {
    track({ event: "onboarding_completion_attempted" })
    const { setSetupCompleted, getConversionCohort, getAllFeaturesPro } = this.props
    const { success } = await setSetupCompleted()
    // Seed store with cohort and related info
    await Promise.all([
      getConversionCohort(),
      getAllFeaturesPro(),
    ])

    if (success) {
      track({ event: "onboarding_completion_succeeded" })
      this.props.push('/')
    }
  }

  render() {
    let stage = null
    switch (this.props.stage) {
      case "#restart-plugins":
        stage = <SetupRestartPlugins
          advance={this.advance}
          plugins={this.props.plugins}
          location={this.props.location}
        />
        break
      case "#done":
        stage = <Done
          advance={this.advance}
          setSetupCompleted={this.setSetupCompleted}
        />
        break
      default:
        stage = <SetupPlugins
          advance={this.advance}
        />
    }
    return (
      <div className={`setup-main ${this.props.shouldBlur ? 'main--blur' : ''}`}>
        <div className="setup__header--invisible"/>
        <Error/>
        <div className="setup">
          { stage }
        </div>
      </div>
    )
  }
}

const mapStateToProps = (state, props) => ({
  stage: props.location.hash,
  account: state.account,
  plugins: state.plugins,
  settings: state.settings,
  system: state.system,
  ...props,
})

const mapDispatchToProps = dispatch => ({
  push: loc => dispatch(push(loc)),
  getUser: () => dispatch(account.getUser()),
  getPlugins: () => dispatch(plugins.getPlugins()),
  getAutoInstallPluginsEnabled: () => dispatch(settings.getAutoInstallPluginsEnabled()),
  getIconVisible: () => dispatch(settings.getIconVisible()),
  setIconInvisible: () => dispatch(settings.setIconInvisible()),
  setAutosearchDefaultOff: () => {
    dispatch(settings.setAutosearchDefaultOff())
    //so as to get initial autosearch state to match initial autosearch default
    dispatch(disableAutosearch())
  },
  getConversionCohort: () => dispatch(getConversionCohort()),
  setSetupCompleted: () => dispatch(settings.setSetupCompleted()),
  getSystemInfo: () => dispatch(system.getSystemInfo()),
  forceCheckOnline: () => dispatch(system.forceCheckOnline()),
  setCurrentSetupStage: (stage) => dispatch(settings.setCurrentSetupStage(stage)),
  getLastSetupStage: () => dispatch(settings.getLastSetupStage()),
  getAllFeaturesPro: () => dispatch(getAllFeaturesPro()),
})

export default connect(mapStateToProps, mapDispatchToProps)(Setup)
