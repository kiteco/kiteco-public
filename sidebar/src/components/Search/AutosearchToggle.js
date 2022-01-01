import React from 'react'
import { connect } from 'react-redux'

import { enableAutosearch, disableAutosearch } from '../../actions/search'

class AutosearchToggle extends React.Component {

  constructor(props) {
    super(props)
    this.state = {
      tooltipEnabled: false,
      shouldDisplayTooltip: false,
    }
  }

  componentDidUpdate(props) {
    const { enabled: prevEnabled } = props
    const { enabled: nextEnabled } = this.props
    if ( nextEnabled && !prevEnabled && this.state.shouldDisplayTooltip ) {
      this.displayAutosearchTooltip()
    }
    if(!this.state.shouldDisplayTooltip) {
      this.setState({ shouldDisplayTooltip: true })
    }
  }

  handleMouseEnter = () => {
    this.setState({
      tooltipEnabled: true,
    })
  }

  handleMouseLeave = () => {
    this.setState({
      tooltipEnabled: false,
    })
  }

  displayAutosearchTooltip = () => {
    this.setState({
      tooltipEnabled: true,
    }, () => {
      setTimeout(() => this.setState({
        tooltipEnabled: false,
      }), 4000)
    })
  }

  render() {
    const { enable, disable, enabled } = this.props
    const { tooltipEnabled } = this.state

    return <div>
      <div className={"search__auto-search__button search__auto-search__button--enabled-" + enabled}
        onClick={ enabled ? disable : enable }
        onMouseEnter={ this.handleMouseEnter }
        onMouseLeave={ this.handleMouseLeave }>
        { enabled ? 'Docs are' : 'Click for docs'}<br />
        { enabled ? 'following cursor' : 'to follow cursor'}
      </div>
      { tooltipEnabled && <div className="search__auto-search__modal">
        <div className="search__auto-search__modal__arrow"/>
        <h4 className="search__auto-search__modal_title">{ enabled ? "Kite Python docs are following your cursor" : "Kite Python docs are not followng your cursor" }</h4>
        <p className="search__auto-search__modal_description">
          { enabled ?
            "Kite is following your typing and showing relevant Python docs. Click to disable. Set the default in settings. Kite only provides docs for Python." :
            "Click to allow Kite to follow your cursor and automatically search for identifiers while you code. Set the default in settings. Kite only provides docs for Python."
          }
        </p>
      </div>
      }
    </div>
  }
}

const mapDispatchToProps = dispatch => ({
  enable: () => dispatch(enableAutosearch()),
  disable: () => dispatch(disableAutosearch()),
})

const mapStateToProps = (state, ownProps) => ({
  enabled: state.search.autosearchEnabled,
})

export default connect(mapStateToProps, mapDispatchToProps)(AutosearchToggle)
