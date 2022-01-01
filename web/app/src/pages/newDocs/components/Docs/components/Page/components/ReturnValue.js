import React from 'react'

const ReturnValue = ({ returnValues }) => {
  return (
    <section className="return-values">
      <h3>
        Returns
      </h3>
      <div>
        <pre>
          <code className='with-syntax-highlighting code'>
          &#8627; {returnValues.map((val, i) => {
              return <span className='return-value' key={i}>
                <span className='literal'>{val.type}</span>
                {i < returnValues.length - 1 && <span className='punctuation small-text'> &#10072; </span>}
              </span>
            })}
          </code>
        </pre>
      </div>
    </section>
  )
}

export default ReturnValue