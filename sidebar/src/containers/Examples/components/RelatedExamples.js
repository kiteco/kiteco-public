import React from 'react'
import { Link } from 'react-router-dom'

import '../assets/related-examples.css'

const RelatedExamples = ({ examples, language }) =>
  <ul className="related-examples">
    { examples.map((example, i) =>
      <li className="related-examples__li" key={i}>
        <Link
          className="related-examples__link"
          to={`/examples/${language}/${example.id}`}
        >
          { example.title }
        </Link>
      </li>
    )}
  </ul>

export default RelatedExamples
