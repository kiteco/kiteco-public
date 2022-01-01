import React from 'react'
import { connect } from 'react-redux'
import { AnyAction } from 'redux'
import { ThunkDispatch } from 'redux-thunk'

import { Input, InputStatus } from '../Input'
import * as settings from '../../actions/settings'

import styles from './index.module.css'

interface Props {
  className: string
  getKiteServerURL: () => Promise<{ data: string }>
  setKiteServerURL: (url: string) => Promise<{}>
  getKiteServerStatus: () => Promise<{ data: { available: boolean, ping: number } }>
}

interface State {
  url: string
  status: InputStatus
  ping: number
}

class KiteServerInput extends React.Component<Props, State, { inputRef: any }> {
  constructor(props: Props) {
    super(props)
    this.state = {
      url: '',
      status: InputStatus.None,
      ping: 0,
    }
  }

  componentDidMount = () => {
    this.props.getKiteServerURL().then(response => {
      this.setState({ url: response.data })
      this.updateStatus()
    })
  }

  onSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    (e.currentTarget.firstElementChild as HTMLInputElement).blur()
  }

  onURLFocus = (e: React.FocusEvent<HTMLInputElement>) => {
    this.setState({
      status: InputStatus.Edit,
    })
  }

  onURLChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    e.preventDefault()
    const url = e.target.value
    this.setState({
      url: url,
      status: InputStatus.Edit,
      ping: 0,
    })
  }

  onURLBlur = (e: React.FocusEvent<HTMLInputElement>) => {
    this.setState({
      status: InputStatus.Loading,
    })
    this.props.setKiteServerURL(this.state.url).then(() => {
      this.updateStatus()
    }).catch(_ => this.setState({
      status: InputStatus.Unavailable,
      ping: 0,
    }))
  }

  updateStatus = () => {
    if (!this.state.url) {
      // Trigger the Redux action to update state in downstream components.
      this.props.getKiteServerStatus()
      this.setState({
        status: InputStatus.None,
        ping: 0,
      })
      return
    }
    this.props.getKiteServerStatus().then((res) => {
      if (res.data.available) {
        this.setState({
          status: InputStatus.Available,
          ping: res.data.ping,
        })
      } else {
        this.setState({
          status: InputStatus.Unavailable,
          ping: 0,
        })
      }
    })
  }

  render = () => {
    return (
      <div>
        <Input
          className={this.props.className}
          placeholder="host:port"
          value={this.state.url}
          status={this.state.status}
          onSubmit={this.onSubmit}
          onFocus={this.onURLFocus}
          onChange={this.onURLChange}
          onBlur={this.onURLBlur}
          updateStatus={this.updateStatus}
        >
        </Input>
        {this.state.ping > 0 &&
          <p className={styles.msg + ' ' + styles.ping}>{"Ping: " + this.state.ping + "ms"}</p>
        }
        {this.state.status === InputStatus.Unavailable &&
          <p className={styles.msg + ' ' + styles.error}>Unable to connect to this server. Please ensure that you are connected to the internet and that you have the correct server address.</p>
        }
      </div>
    )
  }
}

function mapDispatchToProps(dispatch: ThunkDispatch<any, {}, AnyAction>) {
  return {
    getKiteServerURL: () => dispatch(settings.getKiteServerURL()),
    setKiteServerURL: (url: string) => dispatch(settings.setKiteServerURL(url)),
    getKiteServerStatus: () => dispatch(settings.getKiteServerStatus()),
  }
}

export default connect(null, mapDispatchToProps)(KiteServerInput)
