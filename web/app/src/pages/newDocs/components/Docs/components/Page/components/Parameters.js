import React from 'react'
import { Parameter } from '../../util/Code'

const Parameters = ({ name, parameters }) => {
  return (
    <section className='parameters'>
      <h3>
        Signature
      </h3>
      <div>
        <pre>
          <code className='with-syntax-highlighting code'>
            {name}&nbsp;<span className='punctuation'>(</span>
            {parameters.map((param, i) => <Parameter key={i} param={param} includeComma={i !== parameters.length - 1} />)}
            <span className='punctuation'>)</span>
          </code>
        </pre>
      </div>
    </section>
  )
}

export default Parameters