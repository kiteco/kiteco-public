import React from 'react'
import { Link } from 'react-router-dom'

import { stripBackticks } from "./Titles";

const { Fragment } = React

const CodeArray = ({ code }) => {
  return code.map((chunk, i) => <Fragment key={i}>{chunk}<wbr /></Fragment>)
}

const DocsItem = ({ link, code, popularity, showType, type, onItemClick, title }) => {
  if (Array.isArray(code)) {
    code = <CodeArray code={code} />
  }
  return (
    <li onClick={onItemClick} data-popularity={popularity}>
      <Link
        className='code'
        to={link}><code
          title={title}
          className='code'
        >
          {code}
        </code></Link>
      &nbsp;
      {showType && type && <span className='type-badge' title={type}>{type[0]}</span>}
    </li>
  )
}

const HowToItem = ({ link, text, code, onItemClick, title }) => {
  code = stripBackticks(code);
  return (
    <li onClick={onItemClick}>
      {text && <span className='item-preface'>{text} </span>} <Link to={link}><code className='code' title={title}>{code}</code></Link>
    </li>
  )
}

export {
  DocsItem,
  HowToItem
}
