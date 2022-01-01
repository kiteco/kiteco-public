import React from 'react'
import { Link } from 'react-router-dom'
import { connect } from 'react-redux'
import { goBack } from 'react-router-redux'
import { withRouter } from 'react-router'

import '../assets/nav.css'
import TopBar from '../components/TopBar'
import { actionHandled } from '../actions/kite-protocol'

import { ENTERPRISE } from '../utils/enterprise'

const navigationElements = [
  {
    path: '/settings',
    regex: /^\/settings(\/(?!(plugins|logs)).*)?$/g,
    label: 'Home',
  },
  {
    path: '/settings/plugins',
    regex: /^\/settings\/plugins(\/.*)?$/g,
    label: 'Plugins',
  },
]

const Nav = ({
  path,
  goBack,
  history,
  actionHandled,
  os }) => {
  // TODO: Make behavior consistent
  // Whether or not your history is consistent should not
  // depend on how many pages you've visited
  const closeSettings = (e) => {
    //erase any memory of kite:// filename params
    actionHandled()
    const { index, entries } = history
    const shouldGoBack = (index > 0) &&
      (entries[index - 1].pathname.startsWith('/docs/') ||
        entries[index - 1].pathname.startsWith('/examples/'))
    if (shouldGoBack) {
      e.preventDefault()
      goBack()
    }
  }

  const isPath = (el, path) => {
    return path.match(el.regex) !== null
  }

  return (
    <div className={`
      nav-container
      ${ENTERPRISE ? "nav-container--enterprise" : ""}
    `}>
      { os !== 'windows' && <TopBar />}
      <div className="nav__title">
        <Link to="/" onClick={closeSettings}>
          <div className="nav__title__back"></div>
        </Link>
        <h4 className="nav__title__h4">Local Settings</h4>
      </div>
      <div className="nav">
        {navigationElements.map((el) => {
          return <Link
            key={el.label}
            className={"navigation-element " + (isPath(el, path) ? "navigation-element--state-active" : "")}
            to={el.path}>
            {el.label}
          </Link>
        })}
      </div>
    </div>
  )
}


const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
  os: state.system.os,
})

const mapDispatchToProps = dispatch => ({
  goBack: params => dispatch(goBack()),
  actionHandled: () => dispatch(actionHandled()),
})

export default withRouter(connect(mapStateToProps, mapDispatchToProps)(Nav))
