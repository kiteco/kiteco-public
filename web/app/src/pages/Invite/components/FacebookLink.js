import React from 'react'

const FacebookLink = ({ className, link, children, onClick }) =>
  <a
    className={className}
    target="_blank"
    href={`https://www.facebook.com/sharer/sharer.php?u=${link}`}
    onClick={onClick || (() => {})}
  >
    { children }
  </a>

export default FacebookLink
