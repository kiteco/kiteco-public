import React from 'react'
import { connect } from 'react-redux'
import { AnyAction } from 'redux'
import { ThunkDispatch } from 'redux-thunk'

import { Input, InputStatus } from '../Input'
import * as settings from '../../actions/settings'

interface Props {
  className: string
  getMaxFileSize: () => Promise<{ data: string, success: boolean}>
  setMaxFileSize: (size: string) => Promise<{ }>
}

interface State {
  size: string
  status: InputStatus
}

class MaxFileSizeInput extends React.Component<Props, State, { inputRef: any }> {
  constructor(props: Props) {
    super(props)
    this.state = {
      size: "1024",
      status: InputStatus.None,
    }
  }

  componentDidMount = () => {
    this.props.getMaxFileSize().then(response => {
      this.setState({
        size: response.data,
        status: InputStatus.Available,
      })
    })
  }

  onSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    (e.currentTarget.firstElementChild as HTMLInputElement).blur()
  }

  onFocus = (_: React.FocusEvent<HTMLInputElement>) => {
    this.setState({
      status: InputStatus.Edit,
    })
  }

  onChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    e.preventDefault()
    const size = e.target.value
    this.setState({
      size,
      status: InputStatus.Edit,
    })
  }

  onBlur = (_: React.FocusEvent<HTMLInputElement>) => {
    if (isNaN(parseInt(this.state.size))) {
      this.setState({
        status: InputStatus.Unavailable,
      })
    } else {
      this.props.setMaxFileSize(this.state.size).finally(this.updateStatus)
    }
  }

  updateStatus = () => {
    if (!this.state.size) {
      this.setState({
        status: InputStatus.None,
      })
    } else {
      this.setState({
        status: InputStatus.Loading,
      })
      this.props.getMaxFileSize().then((res) => {
        if (res.success) {
          this.setState({
            status: InputStatus.Available,
            size: res.data,
          })
        } else {
          this.setState({
            status: InputStatus.Unavailable,
            size: this.state.size,
          })
        }
      })
    }
  }

  render = () => {
    return (
      <Input
        className={this.props.className}
        placeholder=""
        value={this.state.size}
        status={this.state.status}
        onSubmit={this.onSubmit}
        onFocus={this.onFocus}
        onChange={this.onChange}
        onBlur={this.onBlur}
        updateStatus={this.updateStatus}
      >
      </Input>
    )
  }
}

function mapDispatchToProps(dispatch: ThunkDispatch<any, {}, AnyAction>) {
  return {
    getMaxFileSize: () => dispatch(settings.getMaxFileSize()),
    setMaxFileSize: (size: string) => dispatch(settings.setMaxFileSize(size)),
  }
}

export default connect(null, mapDispatchToProps)(MaxFileSizeInput)
