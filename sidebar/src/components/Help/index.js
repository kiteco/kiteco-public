import React from 'react'
import { connect } from 'react-redux'

import { Domains } from '../../utils/domains'
import { show } from '../../utils/doorbell'
import { forceCheckOnline } from '../../actions/system'
import { notify } from '../../store/notification'

import './help.css'

class Help extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      open: false,
    }
  }

  toggle = () => {
    //forceonlinecheck here
    this.props.forceCheckOnline().then(({ success, isOnline }) => {
      if (success && isOnline) {
        this.setState(state => ({ ...state, open: !state.open }))
      } else {
        this.setState({ open: false })
        this.props.notify({
          id: 'offline',
          component: 'offline',
          payload: {
            copy: 'Thanks for wanting to provide feedback!',
          },
        })
      }
    })
  }

  openDoorbell = () => {
    this.toggle()
    show()
  }

  render() {
    const { open } = this.state
    const { webapp } = this.props
    return <div className="help">
      <div onClick={this.toggle} className="help__icon"></div>
      { open &&
        <div className="help__modal">
          <a
            rel="noopener noreferrer"
            target="_blank"
            href={`https://${Domains.Help}`}
            className="help__option help__get-help"
          >
            Help
          </a>
          <div
            onClick={this.openDoorbell}
            className="help__option help__give-feedback"
          >
            Send feedback
          </div>
        </div>
      }
      { open &&
        <div
          onClick={this.toggle}
          className="help__closer"
        />
      }
    </div>
  }
}

const mapStateToProps = (state, ownProps) => ({
  webapp: state.settings.webapp,
  ...ownProps,
})

const mapDispatchToProps = dispatch => ({
  forceCheckOnline: () => dispatch(forceCheckOnline()),
  notify: params => dispatch(notify(params)),
})

export default connect(mapStateToProps, mapDispatchToProps)(Help)
