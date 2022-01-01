import React from 'react'
import ReactMarkdown from 'react-markdown'
import { Link } from 'react-router-dom'

import { LinedBlock } from '../../../components/Code'

import Navigation from '../../../components/Navigation'

import FileViewer from './FileViewer'
import Output from './Output'

import '../assets/code-example.css'

class CodeExample extends React.Component {

  renderChunks = (chunk, index) => {
    switch(chunk.type) {
      case "code":
        return <LinedBlock
          key={`${this.props.id}-${index}`}
          numberLines={false}
          code={chunk.content.code}
          references={chunk.content.references}
          language={this.props.language}
          highlightedIdentifier={this.props.full_name}
          />
      case "output":
        return <Output
          key={`${this.props.id}-${index}`}
          chunk={chunk}
          />
      default:
        return null;
    }
  }

  render() {
    return (
      <div className="examples__code-example">
        <div className="examples__code-example__title__wrapper">
          {this.props.standaloneExample ?
            <div
              className="examples__code-example__title"
            >
              <ReactMarkdown source={this.props.title}/>
            </div>
            :
            <Link
              to={`/examples/${this.props.language}/${this.props.id}`}
              className="examples__code-example__title"
            >
              <ReactMarkdown source={this.props.title}/>
            </Link>
          }

          {this.props.standaloneExample && <Navigation/>}
        </div>
        <div className="examples__code-example__prelude">
          {this.props.prelude.map(this.renderChunks)}
        </div>
        <div className="examples__code-example__main">
          {this.props.code.map(this.renderChunks)}
        </div>
        <div className="examples__code-example__postlude">
          {this.props.postlude.map(this.renderChunks)}
        </div>
        { this.props.inputFiles &&
          this.props.inputFiles.length > 0 &&
          <div className="examples__code-example__input-files">
            <h5 className="examples__code-example__input-files__title">
              Files used in this example
            </h5>
            {this.props.inputFiles.map((inputFile, index) =>
              <FileViewer
                key={inputFile.name + '-' + index}
                data={inputFile}
              />
            )}
        </div>}
      </div>
    )
  }
}

export default CodeExample
