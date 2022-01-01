import React from 'react'
import { connect } from 'react-redux'
import ErrorOverlay from '../components/ErrorOverlay'
import * as system from '../actions/system'
import * as account from '../actions/account'
import * as settings from '../actions/settings'

/**
 * KitedWaiter wraps children and waits to render them
 * until after kited signals that it is ready via polling
 * through getKitedReady.
 * After getting that signal, it stops polling
 */
class KitedWaiter extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      ready: this.props.kitedReady && typeof this.props.os !== 'undefined'
    }
  }

  UNSAFE_componentWillMount() {
    const {
      getKitedReady,
      getSystemInfo
    } = this.props
    getSystemInfo().then(() => {
      this.poller = setInterval(() => {
        getKitedReady()
      }, 300)
    })
  }

  componentDidUpdate() {
    const {
      kitedReady,
      getDefaultTheme,
      checkIfOnline,
      getUser,
    } = this.props
    if(kitedReady !== this.state.ready) {
      clearInterval(this.poller)
      // fetch env variables from kited since it's now ready
      Promise.all([
        getDefaultTheme(),
        getUser(),
        checkIfOnline(),
      ]).then((res) => {
        setTimeout(() => {
          this.setState({ ready: kitedReady })
        }, 200)
      })
    }
  }

  render() {
    if(!this.state.ready) {
      //render some overlay
      return <ErrorOverlay
        title="Initializing..."
        spinner={true}
      />
    }
    return this.props.children
  }
}

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
  kitedReady: state.system.kitedReady,
  os: state.system.os,
})

const mapDispatchToProps = dispatch => ({
  getKitedReady: () => dispatch(system.getKitedReady()),
  getUser: () => dispatch(account.getUser()),
  getDefaultTheme: () => dispatch(settings.getDefaultTheme()),
  checkIfOnline: () => dispatch(system.checkIfOnline()),
  getSystemInfo: () => dispatch(system.getSystemInfo()),
})

export default connect(mapStateToProps, mapDispatchToProps)(KitedWaiter)
