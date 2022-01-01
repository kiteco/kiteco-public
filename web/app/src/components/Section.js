import React from 'react'

const Section = ({
  children,
  className="",
  contentClassName="",
}) => <div className={`
  homepage__section
  ${className}`
}>
  <div className={`
    homepage__section__content
    ${contentClassName}
  `}>
    { children }
  </div>
</div>

export default Section
