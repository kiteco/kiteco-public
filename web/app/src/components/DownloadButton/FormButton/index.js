import React, { PureComponent } from 'react'
import './assets/formElements.css'
import PropTypes from 'prop-types'
import { Link } from 'react-router-dom'

export class FormButton extends PureComponent {
  static propTypes = {
    modifiers: PropTypes.arrayOf(PropTypes.string),
    url: PropTypes.string,
  }

  static defaultProps = {
    modifiers: [],
  }

  render (){
    const {className, disabled, type, children, value, modifiers, url, newTab, ...rest} = this.props
    const Linkobject = url ? () => ((url.indexOf('http://') === 0 || url.indexOf('https://') === 0)
      ?
      <a
        {...rest}
        className={"form__button__link"}
        href={url}
        target={newTab ? '_blank' : '_self'}
        rel="noopener noreferrer"
      > </a>
      :
      <Link
        {...rest}
        className={"form__button__link"}
        to={url}
        target={newTab ? '_blank' : '_self'}
        rel="noopener noreferrer"
      />) : null
    return (
      <div
        className={
          `form__button ${className ? className : ''}
          ${disabled ? 'form__button--disabled' : ''}
          ${modifiers.map(modifier => ` form__button--${modifier}`)}
        `}
      >
        {children || value}
        {
          url ?
          <Linkobject/>
          :
          <input
            {...rest}
            type={type || 'button'}
            value=""
            className="form__button__input"
          />
        }
      </div>
    )
  }
}
