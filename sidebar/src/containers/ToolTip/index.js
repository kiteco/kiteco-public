import React from 'react'
import ReactDOM from 'react-dom'
import { connect } from 'react-redux'

import * as actions from '../../actions/tooltips'

import './assets/tooltip.css'

/**
 * ToolTip is the tooltip viewer. It should be placed
 * in a large parent container. It is designed to only pickup
 * tooltips that are assigned with a particular `kind`
 * with the prop, `assignedKind`.
 *
 * Note that the parent should be styled position: relative
 *
 * It is designed to be composed under another React Component.
 * See /src/components/Sidebar/ToolTip.js for an example.
 *
 * This wrapper component should assign assignedKind and
 * assignedShow. assignedKind determines the kind of tooltips
 * to be shown. This syncs up ToolTipTriggers
 * and ToolTip. assignedShow can prevent something from being shown.
 */
class ToolTip extends React.Component {
  // Save node so that we can get bounds on this node
  // to be able to calculate tooltip positions
  node = null

  DEFAULT_REM_WIDTH = 30
  FONT_OFFSET = parseInt(window.getComputedStyle(document.documentElement).fontSize, 10)
  HEIGHT_NOT_SET = 0

  state = {
    hidden: true,
    tooltipHeight: this.HEIGHT_NOT_SET,
    node: null,
  }

  componentDidMount() {
    this.setState({
      node: ReactDOM.findDOMNode(this)
    })
  }

  static getDerivedStateFromProps(props, state) {
    if(props.assignedShow && props.show) {
      const offsetHeight = state.node.offsetHeight
      if(state.tooltipHeight !== offsetHeight) {
        if(offsetHeight > 0) {
          return { tooltipHeight: offsetHeight }
        }
      }
    }
    return null
  }

  componentDidUpdate(prevProps) {
    if (prevProps.update && this.props.kind === this.props.assignedKind) {
      prevProps.update(this.props.data)
    }
  }

  arrowXPosition = () => {
    const { left } = this.props.bounds
    const { width, left: parentLeft } = this.state.node.parentNode.getBoundingClientRect()
    if ((left - parentLeft) > (width/2)) {
      return "right"
    } else {
      return "left"
    }
  }

  arrowYPosition = () => {
    const { top } = this.props.bounds
    if(top < window.innerHeight / 2) return "top"
    return "bottom"
  }

  calculatePosition = () => {
    const { bottom, left, right, top, height } = this.props.bounds
    const ownWidth = this.DEFAULT_REM_WIDTH *
                     this.FONT_OFFSET *
                     window.devicePixelRatio

    const parent = this.state.node.parentNode
    const {
      top: parentTop,
      left: parentLeft,
      right: parentRight,
      bottom: parentBottom,
    } = parent.getBoundingClientRect()
    const parentWidth = parentRight - parentLeft
    const positionObj = {}

    if(this.arrowYPosition() === "bottom") {
      const toolTipHeight = (this.state.tooltipHeight + height) +
                            this.FONT_OFFSET *
                            window.devicePixelRatio
      positionObj.top = bottom - parentTop - toolTipHeight
    } else {
      positionObj.top = bottom - parentTop
    }

    if (this.arrowXPosition() === "right") {
      const posRight = parentRight - right
      positionObj.right = posRight
      positionObj.width = posRight + ownWidth - parentWidth > 0 ?
          ownWidth - (posRight + ownWidth - parentWidth + parentLeft)
          : undefined
    } else {
      const posLeft = left - parentLeft
      positionObj.left = posLeft
      positionObj.width = posLeft + ownWidth > parentWidth ?
          ownWidth - (posLeft + ownWidth - parentWidth + (parentLeft * 2))
          : undefined
    }
    return positionObj
  }

  render() {
    if (this.props.assignedShow &&
      this.props.show &&
      this.props.kind === this.props.assignedKind
    ) {
      return (
        <div
          className={`tooltip tooltip--${this.arrowXPosition()} tooltip--${this.arrowYPosition()}`}
          style={this.calculatePosition()}
        >
          {this.props.children}
        </div>
      )
    } else {
      return <div className="tooltip--hidden"></div>
    }
  }
}

/*
 * ToolTipTrigger contains all the functionality necessary
 * to send out a tooltip to be received and displayed by ToolTip
 * This component is designed to be composed and to be assigned
 * the prop, assignedKind which should determine the which
 * ToolTips are shown.
 */
class ToolTipTriggerComponent extends React.Component {
  minPauseTime = 150
  timeout = null

  constructor(props) {
    super(props)
    this.state = {
      sentShowToolTip: false,
      currentBounds: null,
      sentData: {},
    }
  }

  componentWillUnmount() {
    clearTimeout(this.timeout)
  }

  handleMouseOver = () => {
    this.timeout = setTimeout(() => {
      this.show()
      this.setState({
        sentShowToolTip: true,
      })
    }, this.minPauseTime)
  }

  handleMouseOut = () => {
    clearTimeout(this.timeout)
    if (this.state.sentShowToolTip) {
      this.hide()
      this.setState({
        sentShowToolTip: false,
      })
    }
  }

  show = () => {
    const bounds = this.getBounds()
    this.setState({
      currentBounds: bounds,
    })
    const bundle = {
      kind: this.props.kind,
      data: this.props.data,
      bounds: bounds,
    }
    this.setState({
      sentData: bundle,
    })
    this.props.dispatch(actions.showToolTip(bundle))
  }

  hide = () => {
    this.props.dispatch(actions.hideToolTip(this.state.sentData))
  }

  getBounds = () => {
    return ReactDOM.findDOMNode(this).getBoundingClientRect()
  }

  render() {
    return (
      <span className="tooltip-span"
        onMouseOver={this.handleMouseOver}
        onMouseOut={this.handleMouseOut}
        onClick={this.handleMouseOut}
      >
        {this.props.children}
      </span>
    )
  }
}

export default connect((state, ownProps) => ({
  ...state.tooltips,
  ...ownProps,
}))(ToolTip)

export const ToolTipTrigger = connect((state, ownProps) => ({
  ...state.tooltips,
  ...ownProps,
}))(ToolTipTriggerComponent)
