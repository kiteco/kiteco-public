import React from 'react'

import { StructuredCodeBlock } from '../../../components/Code'

const PopularPatterns = ({ structured_patterns, full_name, className}) =>
  <div className={`${className} docs__popular-patterns`}>
    <h2>How others used this</h2>
    {structured_patterns.map((invocation, index) =>
      <StructuredCodeBlock
        key={"inv-" + index + full_name}
        code={invocation.signature}
      />
    )}
  </div>

export default PopularPatterns
