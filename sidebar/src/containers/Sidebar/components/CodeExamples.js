import React from 'react'
import { connect } from 'react-redux'
import * as actions from '../../../actions/examples'
import CopyDetector from '../../../components/CopyDetector'
import { CodeExample } from '../../Examples/components'
import { arrayEquals } from '../../../utils/functional'

class CodeExamples extends React.Component {
  componentDidMount() {
    this.loadDocsExamples(this.props)
  }

  componentDidUpdate(prevProps) {
    if (!arrayEquals(prevProps.exampleIds, this.props.exampleIds, v => v.id) ||
      prevProps.language !== this.props.language) {
      this.loadDocsExamples(this.props)
    }
  }

  loadDocsExamples(props) {
    props = props || this.props
    const { language } = props
    this.props.fetchExamples(language, props.exampleIds.map(e => e.id))
  }

  render() {
    const { accountStatus, examples, exampleIds } = this.props
    return <div>
      { accountStatus === 'loading' && <span className="spinner"></span> }
      { exampleIds && exampleIds.length && exampleIds.map(e => {
        const example = examples.data[e.id];
        return example && <CopyDetector
          full_name={this.props.full_name}
          origin={`docs-example-${example.id}`}
          key={`copy-example-${example.id}`}
          accountStatus={accountStatus}
          os={this.props.os}>
          <CodeExample
            full_name={this.props.full_name}
            language={this.props.language}
            key={`${example.id}`}
            {...example}
            />
        </CopyDetector>
      }) }
    </div>
  }
}

const mapDispatchToProps = dispatch => ({
  fetchExamples: (language, identifiers) => dispatch(actions.fetchExamples(language, identifiers)),
})

const mapStateToProps = (state, ownProps) => ({
  examples: state.examples,
  full_name: ownProps.full_name,
  language: ownProps.language,
  exampleIds: ownProps.examples,
  accountStatus: state.account.status,
})

export default connect(mapStateToProps, mapDispatchToProps)(CodeExamples)
