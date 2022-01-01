import React from 'react'
import { DocsItem } from '../../util/listItems'

const OftenUsedWith = ({ items }) => {
  return (
  <section className='often-used-with'>
    <h3>
      Often used with
    </h3>
    <div>
      <ul className='code'>
        {items.map((item, i) => <DocsItem key={i} link={item.link} code={item.code}/>)}
      </ul>
    </div>
    <div className='hint'>
      Kite uses machine learning to draw connections between modules. <a href=''>Tell me more</a>
    </div>
  </section>
)
}

export default OftenUsedWith
