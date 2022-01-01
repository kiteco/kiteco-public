import React from 'react'
import ErrorOverlay from '../components/ErrorOverlay'

const { ipcRenderer } = window.require("electron")

export default class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props)
    this.state = { hasError: this.props.alreadyError ? true : false }
  }

  componentDidCatch(error, info) {
    this.setState({ hasError: true })
    //send error info over ipc to be logged to Rollbar
    ipcRenderer.send('error-boundary', { name: error.name, message: error.message, info })
    console.log('Error Boundary error: ', error.name, error.message)
    console.log('Error Boundary info: ', info)
  }

  render() {
    if(this.state.hasError) {
      return <ErrorOverlay 
        title="Huh... weird"
        subtitle="Something unexpected occurred. We'll investigate what happened"
        handler={this.props.handler}
        btnText="Reload the Copilot"
      />
    }
    return this.props.children
  }
}