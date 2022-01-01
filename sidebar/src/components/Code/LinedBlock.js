import React from 'react'
import { DocsToolTipTrigger } from '../../containers/Sidebar/components/ToolTip'

import { processCode } from '../../utils/code'

import './assets/lined-block.css'
import 'prismjs/components/prism-python.min.js'

const LinedBlock = ({
  code,
  startNum,
  language,
  numberLines,
  references,
  highlightedIdentifier,
}) =>
  <pre className="lined-block code-theme">
    { language === "python" &&
      <code className="language-python">
        <PythonCodeBlock
          lines={processCode(code, "python", references)}
          startNum={startNum}
          language={language}
          numberLines={numberLines}
          highlight={highlightedIdentifier}
        />
      </code>
    }
    { !language &&
      { code }
    }
  </pre>

const PythonCodeBlock = ({ lines, highlight, language, numberLines, startNum = 1 }) =>
  <table className="lined-block__table">
    <tbody>
      {lines.map((line, index) => {
        return <tr key={index}>
          {numberLines &&
            <td
              className="lined-block__td lined-block__line-number"
              data-line-number={startNum + index}
            >
            </td>
          }
          <td className="lined-block__td">
            {line.map((token, i) =>
              <Token
                key={i}
                token={token}
                language={language}
                highlight={highlight}
              />
            )}
          </td>
        </tr>;
      })}
    </tbody>
  </table>

const Token = ({ token, language, highlight }) => {
  if (token.ref) {
    return <DocsToolTipTrigger
      className="lined-block__reference"
      language={language}
      identifier={token.ref.fully_qualified}
    >
      <span
        className={`
          token
          ${token.type}
          ${token.ref.fully_qualified === highlight ?
            "lined-block__highlighted-identifier" : ""
          }
        `}
        data-token-type={token.type}
      >
        {token.content}
      </span>
    </DocsToolTipTrigger>
  } else {
    return <span
      className={`token ${token.type}`}
      data-token-type={token.type}>
      {token.content}
    </span>
  }
}

export default LinedBlock
