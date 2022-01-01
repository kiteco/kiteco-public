import React, { PureComponent } from 'react'
import { Link } from 'react-router-dom';
import './index.css'

class Button extends PureComponent {
  render() {
    const Linkobject = (this.props.link.indexOf('http://') === 0 || this.props.link.indexOf('https://') === 0)
      ? 'a'
      : Link;

    return (
      this.props.link ?
        <Linkobject
          to={this.props.link}
          href={this.props.link}
          className={this.props.className ? `btn ${this.props.className}` : 'btn'}
        >
          {this.props.children}
        </Linkobject> :
        this.props.children
    )
  }
}

export default Button
