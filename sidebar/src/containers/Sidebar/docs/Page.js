import React from 'react'
import ReactDOM from 'react-dom'
import { connect } from 'react-redux'
import { push } from 'react-router-redux'
import { Shortcuts } from 'react-shortcuts'
// import Helmet from 'react-helmet'
import queryString from 'query-string'
// import { register_once } from '../../utils/analytics'

import { navigatorOs } from '../../../utils/navigator'
//import { ENTERPRISE } from '../../utils/enterprise'
import { metrics } from '../../../utils/metrics'

import * as actions from '../../../actions/docs'
import { showToolTip, hideToolTip } from '../../../actions/tooltips'
import { notify } from '../../../store/notification'
import { GET } from '../../../actions/fetch'

// import { CodeExample } from '../Examples/components'
import CodeExamples from '../components/CodeExamples'
import CodeDefinition from '../components/CodeDefinition'
import { DocsToolTip } from '../components/ToolTip'
import LocalCodeUsages from '../components/LocalCodeUsages'
import Description from '../components/Description'
import Title from '../components/Title'
import Signature from '../components/Signature'
import Kwargs from '../components/Kwargs'
import ReturnValue from '../components/ReturnValue'
import PopularPatterns from '../components/PopularPatterns'
import TopMembers from '../components/TopMembers'
import CopyDetector from '../../../components/CopyDetector'
import Navigation from '../../../components/Navigation'

const { ipcRenderer } = window.require('electron')

class Page extends React.Component {
  HIGHLIGHT_CLASS = "docs-content__section--highlight"
  NO_DOCS_CONTENT = '<body><p>No documentation available</p></body>'

  constructor(props) {
    super(props)
    this.state = {
      highlightedSection: null,
      hasLoadedMembers: false,
      hasNotSeenFocusOnDocs: null,
    }
  }

  componentDidMount() {
    this.loadDocs(this.props)

    // focus component for key navigation
    const domNode = ReactDOM.findDOMNode(this)
    if (domNode) {
      domNode.focus()
    }
  }

  componentDidUpdate(prevProps) {
    if (prevProps.identifier !== this.props.identifier ||
      prevProps.language !== this.props.language) {
      this.setState({
        showDefinition: false,
        highlightedSection: null,
        hasLoadedMembers: false,
      })
      this.loadDocs(this.props)
      ipcRenderer.send('docs-rendered')
    }

    if (prevProps.identifier !== this.props.identifier ||
      prevProps.language !== this.props.language ||
      prevProps.hash !== this.props.hash) {
      // focus component for key navigation
      const domNode = ReactDOM.findDOMNode(this)
      if (domNode) {
        domNode.focus()
      }
      this.scrollToTop()
    }
  }

  loadDocs(props) {
    metrics.incrementCounter('sidebar_new_docs_loaded')
    props = props || this.props
    const { language, identifier } = props
    this.props.fetchDocs(language, identifier)
      .then(this.scrollToTop)
  }

  loadMembers(props) {
    if (!this.state.hasLoadedMembers) {
      props = props || this.props
      const { language, identifier } = props
      this.props.fetchMembers(language, identifier)
        .then(() => {
          this.setState({
            hasLoadedMembers: true,
          })
        })
    }
  }

  moreMembersHandler = () => {
    this.loadMembers(this.props)
  }

  highlightSectionClassName = name => {
    if (name === this.state.highlightedSection) {
      return this.HIGHLIGHT_CLASS
    }
    return ""
  }

  scrollToTop = () => {
    if (this.contentRef) {
      this.contentRef.scrollTop = 0
    }
  }



  jumpTo = anchor => e => {
    const target = this.contentRef.querySelector(`#${anchor}`)
    //unhighlight element so can be clicked multiply with fun effect
    this.setState({
      highlightedSection: null,
    })
    //let current stack execution occur before setting up animation
    setTimeout(() => {
      if (target) {
        this.contentRef.scrollTop = target.offsetTop
        this.setState({
          highlightedSection: anchor,
        })
      }
    }, 0)
  }

  navigate = (action, e) => {
    switch (action) {
      case 'MOVE_DOWN':
        this.contentRef.scrollTop += 20
        return
      case 'MOVE_UP':
        this.contentRef.scrollTop -= 20
        return
      default:
        return
    }
  }

  render() {
    const { accountStatus, docs: { data, status }} = this.props
    const examplesPresent = data.report && data.report.examples && data.report.examples.length
    // docs can come in two forms
    const docsPresent = data.report && (data.report.description_html && data.report.description_html !== this.NO_DOCS_CONTENT || data.report.description_text)
    const fullname = data.value ? data.value.repr : ""

    let content = <div></div>
    if (status === "success" || status === "loading") {
      content =
        <div className="docs-page__content" ref={el => this.contentRef = el}>
          <div className="docs-container">
            <title>
              {`Kite Docs | ${this.props.language} | ${this.props.identifier}`}
            </title>
            <DocsToolTip
              get={this.props.get}
              language={this.props.language}
            />

            {this.props.docs.identifier &&
              <Title
                full_name={fullname}
                parent={data.symbol && data.symbol.parent}
                language={this.props.language}
                status={status}
                name={data.symbol && data.symbol.name}
                type={data.value && data.value.kind}
                typeRepr={data.value && data.value.type}
                typeId={data.value && data.value.type_id}
              />
            }

            {data &&
              <div className="docs-content">

                <div>

                  {data.value && data.value.kind === 'module' &&
                    data.value.details.module.members &&
                    <TopMembers
                      className={this.highlightSectionClassName("top-members")}
                      kind="module"
                      language={this.props.language}
                      full_name={fullname}
                      moreMembersHandler={this.moreMembersHandler}
                      membersHaveLoaded={this.state.hasLoadedMembers}
                      {...data.value.details.module}
                    />
                  }

                  {data.value && data.value.kind === 'type' &&
                    data.value.details.type.members &&
                    <TopMembers
                      className={this.highlightSectionClassName("top-members")}
                      kind="type"
                      language={this.props.language}
                      full_name={fullname}
                      moreMembersHandler={this.moreMembersHandler}
                      membersHaveLoaded={this.state.hasLoadedMembers}
                      {...data.value.details.type}
                    />
                  }

                  {data.value && data.value.details.function &&
                    <CopyDetector
                      full_name={fullname}
                      origin="docs-signature"
                      accountStatus={accountStatus}
                      os={this.props.os}>
                      <Signature
                        className={this.highlightSectionClassName("signature")}
                        title={true}
                        repr={data.value.repr}
                        structuredSignature={data.value.details.function}
                      />
                    </CopyDetector>
                  }

                  {data.value &&
                    data.value.details.function &&
                    data.value.details.function.language_details &&
                    data.value.details.function.language_details.python &&
                    data.value.details.function.language_details.python.kwarg &&
                    data.value.details.function.language_details.python.kwarg_parameters &&
                    data.value.details.function.language_details.python.kwarg_parameters.length > 0 &&
                    <CopyDetector
                      full_name={fullname}
                      origin="docs-kwargs"
                      accountStatus={accountStatus}
                      os={this.props.os}>
                      <Kwargs
                        full_name={fullname}
                        className={this.highlightSectionClassName("kwargs")}
                        kwarg={data.value.details.function.language_details.python.kwarg}
                        kwargParameters={data.value.details.function.language_details.python.kwarg_parameters}
                      />
                    </CopyDetector>
                  }

                  {data.value &&
                    data.value.details.function &&
                    data.value.details.function.return_value &&
                    <CopyDetector
                      full_name={fullname}
                      origin="docs-kwargs"
                      accountStatus={accountStatus}
                      os={this.props.os}>
                      <ReturnValue
                        returnValues={data.value.details.function.return_value}
                      />
                    </CopyDetector>
                  }

                  {data.value && data.value.details.function &&
                    data.value.details.function.structured_patterns &&
                    <CopyDetector
                      full_name={fullname}
                      origin="docs-patterns"
                      accountStatus={accountStatus}
                      os={this.props.os}>
                      <PopularPatterns
                        className={this.highlightSectionClassName("popular-patterns")}
                        full_name={fullname}
                        structured_patterns={data.value.details.function.structured_patterns}
                      />
                    </CopyDetector>
                  }

                  {docsPresent &&
                    <Description
                      className={this.highlightSectionClassName("description")}
                      description_html={data.report.description_html}
                      description_text={data.report.description_text}
                      push={this.props.push}
                      showToolTip={this.props.showToolTip}
                      hideToolTip={this.props.hideToolTip}
                    />
                  }

                </div>

                <div>
                  {examplesPresent &&
                    <div className={`
                    docs__code-examples
                    ${this.highlightSectionClassName("examples")}
                  `} id="examples">
                      <h2>How to</h2>
                      <CodeExamples
                        full_name={this.props.docs.identifier}
                        language={this.props.language}
                        examples={data.report.examples.slice(0, 2)}
                      />
                    </div>
                  }

                  {data.local_usages &&
                    <LocalCodeUsages
                      className={this.highlightSectionClassName("usages")}
                      language={this.props.language}
                      identifier={fullname}
                      {...data.local_usages}
                    />
                  }

                  {data.definition &&
                    <CopyDetector
                      full_name={fullname}
                      origin="docs-definition"
                      accountStatus={accountStatus}
                      os={this.props.os}>
                      <CodeDefinition
                        className={this.highlightSectionClassName("definition")}
                        definition={data.definition}
                        full_name={fullname}
                        language={this.props.language}
                      />
                    </CopyDetector>
                  }

                </div>
              </div>
            }
          </div>
        </div>
    } else if (status === "failed") {
      const failedIdentifier = this.props.identifier.replace('python;', '')
      content =
        <div className="docs-container failed">
          <div className="docs__failed__wrapper">
            {failedIdentifier.length > 20 ?
              <div className="docs__failed__title">
                <h2 className="docs__failed__title__label">Docs not available</h2>
              </div>
              :
              <div className="docs__failed__title">
                <h2 className="docs__failed__title__label">Docs not available for</h2>
                <h2 className="docs__failed__title__name">{this.props.identifier}</h2>
              </div>
            }
            <Navigation />
          </div>
        </div>
    }
    return (
      <Shortcuts
        name='Page'
        alwaysFireHandler={true}
        handler={this.navigate}
        className="docs-page__wrapper"
      >
        { content }
      </Shortcuts>
    )
  }
}

const mapDispatchToProps = dispatch => ({
  get: params => dispatch(GET(params)),
  push: params => dispatch(push(params)),
  showToolTip: params => dispatch(showToolTip(params)),
  hideToolTip: params => dispatch(hideToolTip(params)),
  fetchDocs: (language, identifier) => dispatch(actions.fetchDocs(language, identifier)),
  fetchMembers: (language, identifier) => dispatch(actions.fetchMembers(language, identifier)),
  notify: params => dispatch(notify(params)),
})

const mapStateToProps = (state, ownProps) => ({
  docs: state.docs,
  docsExamples: state.docsExamples,
  language: ownProps.language || "python",
  identifier: ownProps.identifier || ownProps.match.params.id,
  hash: (ownProps.location.hash || "").substring(1),
  features: state.account.plan.features || {},
  accountStatus: state.account.status,
  os: ownProps.os || navigatorOs(),
  source: queryString.parse(ownProps.location.search).source,
  alwaysOnTop: state.settings.alwaysOnTop,
})

export default connect(mapStateToProps, mapDispatchToProps)(Page)
