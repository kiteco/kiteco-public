import React from 'react'

import './assets/code-block.css'

const CodeBlock = ({ code }) => (
  <pre className="code-block code-theme">
    <code>
      {code}
    </code>
  </pre>
)

const sanitizeSignature = (signature) => {
  return signature.substr(signature.lastIndexOf('.') + 1)
}

const StructuredCodeBlock = ({ code }) => {
  //adding a whitespace to the end of the comma
  //makes the text wrapping much cleaner
  const formatToken = (token) => {
    if(token === ',') return ', '
    if(token.endsWith('(')) return sanitizeSignature(token) + '\u200B'
    if(token === ')') return '\u200B)'
    return token
  }
  return {
    render() {
      return (
        <pre className="code-block code-block__structured code-theme">
          <code>
          {code.map((token, index) => {
            return token && <span
              className={"token " + token.tokenType + (token.token === ',' ? ' comma' : '')}
              key={"token-" + index}>
              {formatToken(token.token)}
            </span>
          })}
          </code>
        </pre>
      )
    },
  }
}

const noArgs = (signature) => {
  return !signature.vararg && !signature.kwarg && (!signature.parameters || signature.parameters.length === 0)
}

const EmptyStructuredSignature = ({ repr }) => {
  return (
    <tbody>
      <tr>
        <td className="line-code">
          <span
            className="token text"
            data-token-type="text"
          >
            {repr + "()"}
          </span>
        </td>
      </tr>
    </tbody>
  )
}

const ArgsStructuredSignature = ({ signature, repr }) => {
  return (
    <tbody>
      {signature.signature_parameters && signature.signature_parameters.map((param, i) => {
        return <tr key={"arg-" + i + "-" + param.name}>
          <td className="line-code structured-signature__arg">
            <span
              className="token argument"
              data-token-type="argument"
            >
              {param.name}
              <span className="token__default__value">
                {
                  param.default_value ?
                    "=" + param.default_value : ""
                }
                {
                  i < signature.signature_parameters.length - 1 
                    || signature.vararg
                    || (signature.keyword_only_parameters && signature.keyword_only_parameters.length > 0)
                    || signature.kwarg 
                    ? "," 
                    : ""
                }
              </span>
            </span>
          </td>
        </tr>
      })}
      {signature.vararg && <tr>
        <td className="line-code structured-signature__arg">
          <span
            className="token argument"
            data-token-type="argument"
          >
            {"*" + signature.vararg.name}
            {
              (signature.keyword_only_parameters && signature.keyword_only_parameters.length > 0)
                || signature.kwarg 
                ? "," 
                : ""
            }
          </span>
        </td>
      </tr>}
      {!signature.vararg && signature.keyword_only_parameters && signature.keyword_only_parameters.length > 0 && <tr>
        <td className="line-code structured-signature__arg">
          <span
            className="token argument"
            data-token-type="argument"
          >
            "*,"
          </span>
        </td>
      </tr>}
      {signature.keyword_only_parameters && signature.keyword_only_parameters.map((param, i) => {
        return <tr key={"arg-" + i + "-" + param.name}>
          <td className="line-code structured-signature__arg">
            <span
              className="token argument"
              data-token-type="argument"
            >
              {param.name}
              <span className="token__default__value">
                {
                  param.default_value ?
                    "=" + param.default_value : ""
                }
                {
                  i < signature.keyword_only_parameters.length - 1 || signature.kwarg ?
                    "," : ""
                }
              </span>
            </span>
          </td>
        </tr>
      })}
      {signature.kwarg && <tr>
        <td className="line-code structured-signature__arg">
          <span
            className="token argument"
            data-token-type="argument"
          >
            {"**" + signature.kwarg.name}
          </span>
        </td>
      </tr>}
    </tbody>
  )
}

const StructuredSignature = ({ signature, repr }) => {
  return (
    <pre className="code-block code-block__structured code-theme">
      <code>
        <table className="structured-signature__table">
          {noArgs(signature) ?
            <EmptyStructuredSignature
              repr={sanitizeSignature(repr)} /> :
            <ArgsStructuredSignature
              repr={sanitizeSignature(repr)}
              signature={signature}/>}
        </table>
      </code>
    </pre>
  )
}

export { CodeBlock, StructuredCodeBlock, StructuredSignature }
export { default as LinedBlock } from './LinedBlock'
