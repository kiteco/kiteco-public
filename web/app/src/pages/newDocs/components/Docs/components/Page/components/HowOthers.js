import React from 'react'
import { StructuredCodeBlock } from '../../util/Code'

const HowOthers = ({ patterns }) => {
  return (
    <section className='how-others-used-this'>
      <h3>
        How others used this
      </h3>
      <div>
        <pre>
          {patterns.map((pattern, i) => <div key={i} className="how-others-pattern"><StructuredCodeBlock code={pattern} /></div>)}
        </pre>
      </div>
      <div className='hint'>
        Kite uses machine learning to show you common signatures. {/* Put back in when we have better link... <a href='https://help.kite.com/article/56-python' target='_blank'>Tell me more</a> */}
      </div>
    </section>
  )
}

export default HowOthers
