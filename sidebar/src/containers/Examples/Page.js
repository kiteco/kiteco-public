import React from 'react'
import ReactDOM from 'react-dom'
import { connect } from 'react-redux'
import { Shortcuts } from 'react-shortcuts'
// import Helmet from 'react-helmet'

import * as actions from '../../actions/examples'
import { GET } from '../../actions/fetch'

import { CodeExample, RelatedExamples } from './components'
import { DocsToolTip } from '../Sidebar/components/ToolTip'

import './assets/examples.css'

class Page extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      data: {},
      status: "loading",
    }
  }

  componentDidMount() {
    this.props.fetchExamples(this.props.language, this.props.id);
    // focus component for key navigation
    const domNode = ReactDOM.findDOMNode(this)
    if(domNode) {
      domNode.focus()
    }
  }

  componentDidUpdate(prevProps) {
    if (this.props.id !== prevProps.id ||
      this.props.language !== prevProps.language) {
      this.props.fetchExamples(this.props.language, this.props.id);
      // focus component for key navigation
      const domNode = ReactDOM.findDOMNode(this)
      if(domNode) {
        domNode.focus()
      }
    }
  }

  navigate = (action, e) => {
    switch(action) {
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
    const { examples, id, language } = this.props;
    const example = examples.data[id];

    return <Shortcuts 
      name='Page'
      alwaysFireHandler={true}
      handler={this.navigate}
      className="examples-wrapper"
    >
      <DocsToolTip
        get={this.props.get}
        language={language}
      />
      <div className="examples" ref={el => this.contentRef = el}>
        <title>
          {`Kite Examples | ${language} | ${examples.data.title || "" }`}
        </title>
        { examples.status === "success" && example &&
          <div className="examples__section examples__card">
            <CodeExample
              language={language}
              standaloneExample={true}
              { ...example }
            />
          </div>
        }
        { examples.status === "success" &&
          example && example.related &&
          <div className="examples__section">
            <h2>Related Examples</h2>
            <RelatedExamples
              examples={example.related}
              language={language}
            />
          </div>
        }
        { examples.status === "loading" &&
          <div className="spinner"/>
        }
        { examples.status === "failed" &&
          <div className="examples__section">
            We couldn't load the requested Code Example
          </div>
        }
      </div>
      <div className="examples__bottom-buffer">

      </div>
    </Shortcuts>
  }
}

const mapStateToProps = (state, ownProps) => ({
  examples: state.examples,
  id: ownProps.match.params.id,
  language: ownProps.match.params.language,
})

const mapDispatchToProps = dispatch => ({
  get: params => dispatch(GET(params)),
  fetchExamples: (language, id) => dispatch(actions.fetchExamples(language, [id])),
})

export default connect(mapStateToProps, mapDispatchToProps)(Page)
