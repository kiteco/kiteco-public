import React from 'react'
import { Action } from 'redux'
import { ThunkDispatch } from 'redux-thunk'
import { connect } from 'react-redux'

import * as system from '../actions/system'
import {
  fetchLicenseInfo,
  getConversionCohort,
  getAllFeaturesPro,
  getPaywallCompletionsRemaining,
} from '../store/license'

interface PollerProps {
  checkIfOnline: () => void
  // Refreshing pulls from the backend, which would be a lot of traffic
  // We only need to check for license info changes when the cohort
  // changes due to a push message (eg paywall invalidates trials)
  // Other flows call kite://refresh-licenses so don't refresh here
  fetchLicenseInfoWithoutRefresh: () => void
  getAllFeaturesPro: () => void
  getConversionCohort: () => void
  getPaywallCompletionsRemaining: () => void
}

class StatePoller extends React.Component<PollerProps, {}> {
  private intervalIDs: number[]

  constructor(props: PollerProps) {
    super(props)
    this.intervalIDs = []
  }

  // Any additions here should be also be added to kited's
  // ignoreForLogging func to avoid spamming client.log
  componentDidMount() {
    this.poll(this.props.checkIfOnline, 3000)
    this.poll(this.props.fetchLicenseInfoWithoutRefresh, 3000)
    this.poll(this.props.getConversionCohort, 3000)
    this.poll(this.props.getPaywallCompletionsRemaining, 1000)
    this.poll(this.props.getAllFeaturesPro, 3000)
  }

  poll(fn: () => void, interval: number) {
    this.intervalIDs.push(window.setInterval(fn, interval))
  }

  componentWillUnmount() {
    this.intervalIDs.forEach((_, value) => clearInterval(value))
  }

  render() {
    return null
  }
}

const mapDispatchToProps = (dispatch: ThunkDispatch<object, any, Action>) => ({
  checkIfOnline: () => dispatch(system.checkIfOnline()),
  fetchLicenseInfoWithoutRefresh: () => dispatch(fetchLicenseInfo({ refresh: false })),
  getAllFeaturesPro: () => dispatch(getAllFeaturesPro()),
  getConversionCohort: () => dispatch(getConversionCohort()),
  getPaywallCompletionsRemaining: () => dispatch(getPaywallCompletionsRemaining()),
})

export default connect(null, mapDispatchToProps)(StatePoller)
