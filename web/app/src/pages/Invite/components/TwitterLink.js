import React from 'react'

const TwitterLink = ({ className, message, children, onClick }) =>
  <a
    className={className}
    target="_blank"
    href={`https://twitter.com/home?status=${encodeURIComponent(message)}`}
    onClick={onClick || (() => {})}
    rel="noopener noreferrer"
  >
    { children }
  </a>

export default TwitterLink
