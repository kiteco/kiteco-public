import React, { MouseEvent } from 'react'
import { connect } from 'react-redux'
import { AnyAction } from 'redux'
import { ThunkDispatch } from 'redux-thunk'

import * as settings from '../../actions/settings'
import {WindowMode as WindowModeEnum} from '../../utils/settings'

import '../../assets/sidebar.css'
import styles from './index.module.css'

interface WindowModeProps {
  settings: any
  setWindowMode: (mode: any) => Promise<any>
}

interface WindowModeState {
  open: boolean
  hovering: boolean
}

class WindowMode extends React.Component<WindowModeProps, WindowModeState> {
  constructor(props: WindowModeProps) {
    super(props)
    this.state = {
      open: false,
      hovering: false,
    }
  }

  get windowModeString() {
    if (this.props.settings.windowMode === WindowModeEnum.NORMAL) {
      return 'Currently in normal window mode.'
    }
    if (this.props.settings.windowMode === WindowModeEnum.FOCUS_ON_DOCS) {
      return 'Currently focusing when new documentation is rendered. '
        + 'This experimental feature may not work properly on Windows and '
        + 'Linux systems.'
    }
    if (this.props.settings.windowMode === WindowModeEnum.ALWAYS_ON_TOP) {
      return 'Currently always on top.'
    }
  }

  get windowModeClassName() {
    if (this.props.settings.windowMode === WindowModeEnum.NORMAL) {
      return styles['sidebar__icon__window-mode--normal']
    }
    if (this.props.settings.windowMode === WindowModeEnum.FOCUS_ON_DOCS) {
      return styles['sidebar__icon__window-mode--focus-on-docs']
    }
    if (this.props.settings.windowMode === WindowModeEnum.ALWAYS_ON_TOP) {
      return styles['sidebar__icon__window-mode--always-on-top']
    }
  }

  showTooltip = (e: MouseEvent) => {
    this.setState({hovering: true})
  }

  hideTooltip = (e: MouseEvent) => {
    this.setState({hovering: false})
  }

  toggle = (e: MouseEvent) => {
    e.preventDefault()
    this.setState({open: !this.state.open})
  }

  setWindowMode = (mode: any) => {
    this.props.setWindowMode(mode)
    this.setState({open: false})
  }

  render() {
    return (
      <div className={styles['window-mode']}>
        <div
          className={
            [
              styles['sidebar__icon__window-mode'],
              this.windowModeClassName,
            ].join(' ')
          }
          onMouseEnter={this.showTooltip}
          onMouseLeave={this.hideTooltip}
          onClick={this.toggle}
        >
        </div>
        {this.state.open && <div className={styles.help__modal}>
          <div
            className={
              [
                styles.help__option,
                styles['help__normal-window'],
              ].join(' ')
            }
            onClick={() => this.setWindowMode(WindowModeEnum.NORMAL)}
          >
            Normal
          </div>
          <div
            className={
              [
                styles.help__option,
                styles['help__focus-on-docs'],
              ].join(' ')
            }
            onClick={() => this.setWindowMode(WindowModeEnum.FOCUS_ON_DOCS)}
          >
            Focus on docs
          </div>
          <div
            className={
              [
                styles.help__option,
                styles['help__always-on-top'],
              ].join(' ')
            }
            onClick={() => this.setWindowMode(WindowModeEnum.ALWAYS_ON_TOP)}
          >
            Always on top
          </div>
        </div>}
        {this.state.open &&
          <div
            className={styles.help__closer}
            onClick={this.toggle}
          />
        }
        {this.state.hovering && !this.state.open &&
          <div
            className={
              [
                styles.sidebar__tooltip,
                styles['sidebar__tooltip--thin'],
              ].join(' ')
            }
          >
            <div className={styles.sidebar__tooltip__title}>
              Change Kite's Window Mode
            </div>
            <p className={styles.sidebar__tooltip__paragraph}>
              {this.windowModeString}
            </p>
          </div>
        }
      </div>
    )
  }
}

const mapStateToProps = (state: any) => ({
  settings: state.settings,
})

const mapDispatchToProps = (dispatch: ThunkDispatch<any, {}, AnyAction>) => ({
  setWindowMode: (mode: any) => dispatch(settings.setWindowMode(mode)),
})

export default connect(mapStateToProps, mapDispatchToProps)(WindowMode)
