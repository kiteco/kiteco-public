import React from 'react'
import {connect} from 'react-redux'

import * as actions from '../actions/logs'

import '../assets/common.css'
import '../assets/logs.css'

const electron = window.require('electron')

class Logs extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      uploadingLogs: false,
      uploadError: false,
      logsCopied: false,
      capturing: false,
      captureError: false,
      profilesCopied: false,
    }
  }

  componentDidUpdate() {
    console.log('updated logs', this.props.logs)
  }

  get logsGeneratedAt() {
    return this.props.logs.logsGeneratedAt !== null
      ? new Date(this.props.logs.logsGeneratedAt)
      : null
  }

  get profilesCapturedAt() {
    return this.props.logs.profilesCapturedAt !== null
      ? new Date(this.props.logs.profilesCapturedAt)
      : null
  }

  copyLogsUrl = e => {
    e.preventDefault()
    navigator.clipboard.writeText(this.props.logs.logsUrl).then(() => {
      this.setState({logsCopied: true})
      setTimeout(() => {
        this.setState({logsCopied: false})
      }, 5000)
    })
  }

  copyProfilesUrl = e => {
    e.preventDefault()
    navigator.clipboard.writeText(this.props.logs.profilesUrl).then(() => {
      this.setState({profilesCopied: true})
      setTimeout(() => {
        this.setState({profilesCopied: false})
      }, 5000)
    })
  }

  uploadLogs = e => {
    e.preventDefault()
    this.setState({uploadingLogs: true})

    this.props.uploadLogs().then(() => {
      this.setState({
        uploadingLogs: false,
        uploadError: false,
      })
    }).catch(() => {
      this.setState({
        uploadingLogs: false,
        uploadError: true,
      })
    })
  }

  capture = e => {
    e.preventDefault()
    this.setState({capturing: true})

    this.props.capture().then(() => {
      this.setState({
        capturing: false,
        captureError: false,
      })
    }).catch(() => {
      this.setState({
        capturing: false,
        captureError: true,
      })
    })
  }

  openGitHub = e => {
    e.preventDefault()
    electron.shell.openExternal('https://github.com/kiteco/plugins/issues')
  }

  render() {
    return (
      <div className="main__sub">
        <div className="logs">
          <div className="logs__section">
            <h2 className="section__title">
              Debug Issues
            </h2>
            <p>
              If you are experiencing issues with Kite, you can generate logs to
              help us debug. Your logs will be stored at a secure URL that can
              only be accessed by Kite.
            </p>
            <div className="section__disclaimer">
              <strong>IMPORTANT:</strong> To help us debug effectively, please
              generate your logs <i>while you are experiencing the problems.</i>
              <br/>
              <br/>
              Also, please ensure you are connected to the internet when
              generating your logs.
            </div>
            <p>
              After you have generated your logs, visit our&nbsp;
              <a
                className="logs__link"
                onClick={this.openGitHub}
              >
                GitHub repository
              </a>
              &nbsp;to report an issue.
            </p>
          </div>
          <div className="logs__section">
            <h2 className="section__title">
              Application Logs
            </h2>
            <p>
              You can automatically generate logs for us to refer to when you
              have a bug to report.
            </p>
            <div className="logs__cta-row">
              <button
                className={'logs__cta-row__cta cta-button '
                  + (this.state.uploadingLogs ? 'cta-button--disabled' : '')}
                onClick={e => this.uploadLogs(e)}
              >
                {!this.state.uploadingLogs
                  ? 'Generate'
                  : 'Generating...'}
              </button>
              <div className="logs__cta-row__description">
                {this.state.uploadError
                  ? (
                    <span>
                      There was an error uploading your logs. Please try again.
                    </span>
                  )
                  : (this.props.logs.logsUrl === null
                    ? 'You have no logs generated yet'
                    : (
                      <span>
                        Logs generated at&nbsp;
                        {this.logsGeneratedAt.toLocaleString()}.
                        ({!this.state.logsCopied
                          ? (
                            <a
                              className="logs__link"
                              onClick={this.copyLogsUrl}>
                              Copy Secure Link
                            </a>
                          )
                          : (
                            <a className="logs__link logs__link--disabled">
                              Copied to clipboard
                            </a>
                          )
                        })
                      </span>
                    )
                  )
                }
              </div>
            </div>
          </div>
          <div className="logs__section">
            <h2 className="section__title">
              Resource Usage
            </h2>
            <p>
              If you are experiencing performance issues, you can also generate
              CPU and memory profiles.
            </p>
            <div className="logs__cta-row">
              <button
                className={'logs__cta-row__cta cta-button '
                  + (this.state.capturing ? 'cta-button--disabled' : '')}
                onClick={e => this.capture(e)}
              >
                {!this.state.capturing
                  ? 'Capture'
                  : 'Capturing - This may take a few minutes...'}
              </button>
              <div className="logs__cta-row__description">
                {this.state.captureError
                  ? (
                    <span>
                      There was an error capturing your resource usage profiles.
                      Please try again.
                    </span>
                  )
                  : (this.props.logs.profilesUrl === null
                    ? 'You have no profiles generated yet'
                    : (
                      <span>
                        Profiles generated at&nbsp;
                        {this.profilesCapturedAt.toLocaleString()}.
                        ({!this.state.profilesCopied
                          ? (
                            <a
                              className="logs__link"
                              onClick={this.copyProfilesUrl}
                            >
                              Copy Secure Link
                            </a>
                          )
                          : (
                            <a className="logs__link logs__link--disabled">
                              Copied to clipboard
                            </a>
                          )
                        })
                      </span>
                    )
                  )
                }
              </div>
            </div>
          </div>
        </div>
      </div>
    )
  }
}

const mapStateToProps = state => ({
  logs: state.logs,
})

const mapDispatchToProps = dispatch => ({
  uploadLogs: () => dispatch(actions.uploadLogs()),
  capture: () => dispatch(actions.capture()),
})

export default connect(mapStateToProps, mapDispatchToProps)(Logs)
