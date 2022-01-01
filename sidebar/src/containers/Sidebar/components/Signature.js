import React from 'react'

import {
  StructuredSignature,
  CodeBlock,
} from '../../../components/Code'

class Signature extends React.Component {

  render() {
    let signatureBlock = null
    if (this.props.structuredSignature && this.props.repr) {
      signatureBlock = (
        <StructuredSignature
          repr={this.props.repr}
          signature={this.props.structuredSignature}
        />
      )
    } else if (this.props.signature) {
      // if we don't have a structuredSignature
      signatureBlock = (
        <CodeBlock
          code={this.props.signature}
        />
      )
    } else {
      signatureBlock = null
    }
    return (
      <div className={`${this.props.className} docs__signature`}>
        {this.props.title && <h2>Signature</h2>}
        {signatureBlock}
      </div>
    )
  }
}

export default Signature
