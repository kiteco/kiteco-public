import React from 'react'
import { connect } from 'react-redux'
import * as actions from '../actions/scripts'
import { getMetricsDisabled } from '../actions/settings'
import { SCRIPTS } from '../utils/scripts'

class ThirdPartyScripts extends React.Component {
  componentDidMount() {
    this.props.getMetricsDisabled().then(() => {
      if(navigator.onLine && this.props.networkConnected) {
        SCRIPTS.forEach(script => {
          if(this.canUseScript(script)) {
            this.props.loadScript(script.name)
          }
        })
      }
    })
  }

  componentDidUpdate() {
    if(navigator.onLine && this.props.networkConnected) {
      const { scriptDict } = this.props.scripts
      SCRIPTS.forEach(script => {
        if(scriptDict[script.name] && !scriptDict[script.name].loaded && this.canUseScript(script)) {
          this.props.loadScript(script.name)
        }
      })
    }
  }

  canUseScript(script) {
    return !script.needsMetricsEnabled || !this.props.metricsDisabled
  }

  render() {
    return null
  }
}

const mapStateToProps = (state, ownProps) => ({
  scripts: state.scripts,
  networkConnected: state.system.networkConnected,
  metricsDisabled: state.settings.metricsDisabled,
})

const mapDispatchToProps = dispatch => ({
  loadScript: (name) => dispatch(actions.loadScript(name)),
  getMetricsDisabled: () => dispatch(getMetricsDisabled()),
})

export default connect(mapStateToProps, mapDispatchToProps)(ThirdPartyScripts)