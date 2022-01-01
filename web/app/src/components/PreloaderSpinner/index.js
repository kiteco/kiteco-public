import React, { PureComponent } from 'react';

import './PreloaderSpinner.css';

export default class PreloaderSpinner extends PureComponent {
  static defaultProps = {
    containerSize: 30,
    containerClass: '',
  }

  render() {
    const { containerSize, containerClass } = this.props;
    const size = typeof containerSize !== 'number' || containerSize < 30 ? 30 : containerSize;
    const wrapClass = typeof containerClass !== 'string' ? '' : containerClass;
    return (
      <div
        className={`preloader ${wrapClass}`}
        style={{ width: `${size}px`, height: `${size}px` }}
      >
        <div className='preloader__spinner'>
          <div/>
          <div/>
          <div/>
          <div/>
          <div/>
          <div/>
          <div/>
          <div/>
          <div/>
          <div/>
          <div/>
          <div/>
        </div>
      </div>
    );
  }
}
