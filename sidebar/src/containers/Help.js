import React from 'react'
import { Link } from 'react-router-dom'
import { connect } from 'react-redux'

import TopBar from '../components/TopBar'

import { getMRUEditor } from '../actions/plugins'
import { Domains } from '../utils/domains'

import '../assets/help.css'
import '../assets/sidebar.css'

class Help extends React.Component {

  componentDidMount() {
    const { getMRUEditor } = this.props
    getMRUEditor()
    this.getMRUEditorInterval = setInterval(getMRUEditor, 3000)
  }

  componentWillUnmount() {
    clearInterval(this.getMRUEditorInterval)
  }

  render() {
    const { shouldBlur, os, mruEditor } = this.props

    return (
      <div className={`main ${shouldBlur ? 'main--blur' : ''}`}>
        { os !== 'windows' && <TopBar/> }
        <div className="wrapper-page">

          <div className="help__title">
            <Link to="/">
              <div className="help__title__back"/>
            </Link>
            <h4 className="help__title__h4">Feedback and Help</h4>
          </div>

          <div className="help__body">
            <LinkedAnimBanner
              child={ <div className="help__banner__text">FAQ and Help Center</div> }
              className="help__banner__icon--question"
              href={`https://${Domains.Help}`}
            />
            <LinkedAnimBanner
              child={ <div className="help__banner__text">File a Github Issue</div> }
              className="help__banner__icon--github"
              href="https://github.com/kiteco/plugins/issues"
              delay={1}
            />
            <LinkedAnimBanner
              child={ <div className="help__banner__text">{ getRateText(mruEditor) }</div> }
              className="help__banner__icon--star"
              href={ getRateLink(mruEditor) }
              delay={2}
            />
            <LinkedAnimBanner
              child={ <div className="help__banner__text">Share Kite</div> }
              className="help__banner__icon--outlink"
              href={`https://${Domains.PrimaryHost}/invite`}
              delay={3}
            />
          </div>

        </div>
      </div>
    )
  }
}

const LinkedAnimBanner = ({ href, child, delay, className }) =>
  <a href={href} target="blank_">
    <AnimBanner child={child} delay={delay} className={className}/>
  </a>

const AnimBanner = ({ child, delay, className }) => {
  let delayClass = delay ? "showup__animation--delay" : ""
  if (delay >= 2)
    delayClass += `-${delay}`

  className = className ? className : ""
  return (
    <div className={`help__banner ${className} showup__animation ${delayClass}`}>
      {child}
    </div>
  )
}

function getRateLink(mruEditor) {
  let rateLink = "https://github.com/kiteco/plugins"

  switch (true) {
    case mruEditor.toLowerCase().includes("vscode"):
      rateLink = "https://marketplace.visualstudio.com/items?itemName=kiteco.kite&ssr=false#review-details"
      break
    case mruEditor.toLowerCase().includes("atom"):
      rateLink = "https://github.com/kiteco/atom-plugin"
      break
    case mruEditor.toLowerCase().includes("sublime"):
      rateLink = "https://github.com/kiteco/KiteSublime"
      break
    case mruEditor.toLowerCase().includes("vim"):
      rateLink = "https://github.com/kiteco/vim-plugin"
      break
    default:
      break
  }

  return rateLink
}

function getRateText(mruEditor) {
  if (mruEditor && mruEditor.toLowerCase().includes("vscode"))
    return "Rate Plugin"

  return "Star Plugin"
}

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
  os: state.system.os,
  mruEditor: state.pluginsInfo.mruEditor,
})

const mapDispatchToProps = dispatch => ({
  getMRUEditor: () => dispatch(getMRUEditor()),
})

export default connect(mapStateToProps, mapDispatchToProps)(Help)
