import React from 'react'
//multiline clamping
import Shiitake from 'shiitake'

import { Link } from 'react-router-dom'
import { connect } from 'react-redux'

import ToolTip, { ToolTipTrigger } from '../../ToolTip'
import { normalizeValueReportFromSymbol } from '../../../utils/value-report'
import { symbolReportPath } from '../../../utils/urls'

const Synopsis = ({ synopsis }) => {
    return (
      <Shiitake
        lines={3}
        overflowNode={<span> ...</span>}
        throttleRate={250}>
        {synopsis}
      </Shiitake>
    )
}

class DocsToolTipComponent extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      loaded: false,
      data: { value: {}, report: {} },
      tooltipData: {},
      show: false,
      hasShown: false,
    }
  }

  componentDidUpdate() {
    if(this.state.hasShown && this.props.show && this.state.loaded) {
      setTimeout(() => {
        this.setState({
          show: true,
        })
      }, this.RENDER_DELAY)
    } else if(this.state.show && !this.props.show) {
      this.setState({ show: false })
    }
  }

  //for rendering the ellipsis properly in Shiitake
  RENDER_DELAY = 300
  LOAD_RENDER_DELAY = 300

  update = data => {
    if (data.identifier !== this.state.tooltipData.identifier) {
      this.setState({
        loaded: false,
        tooltipData: data,
        show: false,
        hasShown: false,
      })
      this.fetchDocs({
        language: this.props.language,
        identifier: data.identifier,
      })
    }
  }

  fetchDocs = ({language, identifier}) => {
    //TODO: REMOVE THIS HACK AFTER API IS FIXED
    if (!identifier.startsWith(`${language};`)){
      identifier = `${language};${identifier}`
    }
    return this.props.get({
      url: symbolReportPath(identifier)
    })
      .then(({ success, data }) => {
        if (success && data) {
          this.setState({
            loaded: true,
            data: normalizeValueReportFromSymbol(data),
          })
          setTimeout(() => {
            this.setState({
               show: true,
               hasShown: true,
            })
          }, this.RENDER_DELAY)
        }
      })
  }

  render() {
    return (
      <div className={this.state.show ? "" : "tooltip--unrendered"}>
        <ToolTip
          assignedKind="docs"
          assignedShow={this.state.loaded}
          update={this.update}
        >
          <div className="tooltip__row">
            { this.state.data.value.repr &&
              <div className="tooltip__title">
                <span>{this.state.data.value.repr}</span>
              </div>
            }
            { this.state.data.value.kind &&
              this.state.data.value.kind !== "unknown" &&
              <div className="tooltip__kind">
                {this.state.data.kind}
              </div>
            }
          </div>
          { this.state.data.value.synopsis &&
            <Synopsis synopsis={this.state.data.value.synopsis} />
          }
        </ToolTip>
      </div>
    )
  }
}

export class DocsToolTipTrigger extends React.Component {
  render() {
    return(
      <ToolTipTrigger
        className={this.props.className}
        kind="docs"
        data={{identifier: this.props.identifier}}
      >
        <Link
          className={`${this.props.className} tooltip-link`}
          to={`/docs/${this.props.identifier}`}
        >
          {this.props.children}
        </Link>
      </ToolTipTrigger>
    )
  }
}

export const DocsToolTip = connect((state, ownProps) => ({
  ...state.tooltips,
  ...ownProps,
}))(DocsToolTipComponent)
