import React from 'react'
import { connect } from 'react-redux'
import { goBack, goForward } from 'react-router-redux'
import { withRouter } from 'react-router'

import './navigation.css'

class Navigation extends React.Component {
  render() {
    const { index, length, entries } = this.props.history
    const showGoBack = (index > 0) &&
          (entries[index - 1].pathname.startsWith('/docs/') ||
          entries[index - 1].pathname.startsWith('/examples/')),

          showGoForward = (index < length - 1) &&
          (entries[index + 1].pathname.startsWith('/docs/') ||
          entries[index + 1].pathname.startsWith('/examples/'))

    return (
      <div className="navigation__history">
        <div className={"navigation__history__back " + (showGoBack ? 'navigation__history__button--enabled' : '')} onClick={this.props.goBack}></div>
        <div className={"navigation__history__forward " + (showGoForward ? 'navigation__history__button--enabled' : '')} onClick={this.props.goForward}></div>
      </div>
    )
  }
}

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
})

const mapDispatchToProps = dispatch => ({
  goBack: params => dispatch(goBack()),
  goForward: params => dispatch(goForward()),
})

export default withRouter(connect(mapStateToProps, mapDispatchToProps)(Navigation))
