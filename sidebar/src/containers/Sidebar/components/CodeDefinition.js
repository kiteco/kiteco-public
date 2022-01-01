import React from 'react'
import { LinedBlock } from '../../../components/Code'

class CodeDefinition extends React.Component {

  constructor(props) {
    super(props)
    this.state = {
      showDefinition: false,
    }
  }

  toggleDefinition = () => {
    this.setState({
      showDefinition: !this.state.showDefinition,
    });
  }

  render() {
    //TODO: check why backend sends back double spaced
    let code = this.props.definition.code.split("\n\n").join("\n");
    const longDef = code.split("\n").length > 25;
    if (longDef && !this.state.showDefinition) {
      code = code.split("\n").slice(0, 25).join("\n");
    }
    return (
      <div className={`${this.props.className} docs__code-definition`}>
        <h2>Definition of {this.props.full_name}</h2>
        <div className="usage">
          <div className="usage-title">
            <div className="usage-filename">{this.props.definition.path}</div>
          </div>
          <LinedBlock
            key={this.props.definition.identifier + "-" + (this.state.showDefinition ? "all" : "collapsed")}
            code={code}
            startNum={this.props.definition.line_num}
            language={this.props.language}
            numberLines={true}
          />
          { longDef &&
            <button
              className="docs-toggle-definition"
              onClick={this.toggleDefinition}
            >
              {this.state.showDefinition ? "Collapse" : "Show all"}
            </button>
          }
        </div>
      </div>
    )
  }
}

export default CodeDefinition
