import React from 'react'
import { LinedBlock } from '../../../components/Code'

const formatFilePath = path => {
  // TODO: replace this with an os or separator check...
  if (path.startsWith("/windows/")) {
    path = path.replace("/windows/", "")
    let paths = path.split("/")
    paths = [`${paths.shift()}:`, ...paths]
    path = paths.join("\\")
  }
  return path
}

const LocalCodeUsages = ({ identifier, usages, className, language }) =>
  <div className={`${className} docs__local-code-usages`}>
    <h2>Examples of {identifier} in your code</h2>
    <div>
      { usages &&
        usages.length &&
        usages.map((usage, index) =>
          <div className="usage" key={usage.hash}>
            <div className="usage-title">
              <div className="usage-filename">
                {formatFilePath(usage.path)}
              </div>
            </div>
            <LinedBlock
              startNum={usage.line_num}
              code={usage.code}
              language={language}
              numberLines={true}
            />
          </div>
      )}
      { !usages &&
        <div>No examples were found in your codebase.</div>
      }
    </div>
  </div>

export default LocalCodeUsages
