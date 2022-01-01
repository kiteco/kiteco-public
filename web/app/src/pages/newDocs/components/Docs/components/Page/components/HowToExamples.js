import React from 'react'
import ReactMarkdown from "react-markdown";
import { connect } from 'react-redux'
import { Link } from "react-router-dom";

import * as actions from '../../../../../../../redux/actions/examples'
import { setDocsLoading } from '../../../../../../../redux/actions/loading'
import { arrayEquals } from '../../../../../../../utils/functional'

import CuratedExample from './CuratedExample'

const IndexLink = ({ link, text }) => {
  return (
    <li>
      <Link to={link}>
        <ReactMarkdown source={text} />
      </Link>
    </li>
  );
};

const MAX_EXAMPLES = 2
const MAX_ANSWERS = 5

class HowToExamples extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      showAllExamples: false,
      showAllAnswers: true,
    }
  }

  componentDidMount() {
    if (!this.props.exampleIds) {
      return
    }
    this.props.setDocsLoading(true)
    this.props.newFetchExamples(this.props.language, this.props.exampleIds.map(e => e.id))
      .then(() => {
        this.props.setDocsLoading(false)
      })
  }

  componentDidUpdate(prevProps) {
    if(this.props.exampleIds && !arrayEquals(prevProps.exampleIds, this.props.exampleIds, ex => ex.id)) {
      this.props.setDocsLoading(true)
      this.props.newFetchExamples(this.props.language, this.props.exampleIds.map(e => e.id))
        .then(() => {
          this.props.setDocsLoading(false)
        })
    }
  }

  toggleShowAllExamples = () => {
    this.setState({ showAllExamples: !this.state.showAllExamples })
  }
  toggleShowAllAnswers = () => {
    this.setState({ showAllAnswers: !this.state.showAllAnswers })
  }

  render() {
    const { examples, exampleIds, id, language, answers_links } = this.props
    if (!((exampleIds && exampleIds.length) || (answers_links && answers_links.length))) {
        return null;
    }

    const answersToShow = !this.state.showAllAnswers && answers_links
      ? answers_links.slice(0, MAX_ANSWERS - 1)
      : answers_links
    const examplesToShow = !this.state.showAllExamples && exampleIds
      ? exampleIds.slice(0, MAX_EXAMPLES - 1)
      : exampleIds

    return (
        <div>
        { exampleIds && exampleIds.length &&
          <section className="how-to-examples">
            <h3 key="h3">How to</h3>
            { examplesToShow.map(ex => {
              const example = examples.data[ex.id]
              return example && <CuratedExample
                key={ex.id}
                example={example}
                id={ex.id}
                full_name={id}
                language={language}
              />
            }) }
            { exampleIds.length > MAX_EXAMPLES &&
              <button key="show-more" onClick={this.toggleShowAllExamples} className='show-more'>
              {this.state.showAllExamples ? 'collapse' : 'more'} examples
              </button> }
          </section>
        }

        { answers_links && answers_links.length &&
          <section className="how-to-examples">
            <h3 key="h3">Python answers</h3>
            <div key="div"><ul className="how-to">{
              answersToShow.map((link, i) => (
                <IndexLink
                  key={i}
                  link={`/${language}/answers/${link.slug}`}
                  text={link.title}
                />
              ))}
            </ul></div>
            { answers_links.length > MAX_ANSWERS &&
              <button key="show-more" onClick={this.toggleShowAllAnswers} className='show-more'>
              {this.state.showAllAnswers ? 'collapse' : 'more'} answers
              </button> }
          </section>
        }
      </div>
    )
  }
}

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
  examples: state.examples,
})

const mapDispatchToProps = dispatch => ({
  newFetchExamples: (language, identifiers) => dispatch(actions.newFetchExamples(language, identifiers)),
  setDocsLoading: (isLoading) => dispatch(setDocsLoading(isLoading))
})

export default connect(mapStateToProps, mapDispatchToProps)(HowToExamples)
