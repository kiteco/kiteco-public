import React from 'react'

class ScrollToTop extends React.Component {
  componentDidMount() {
    setTimeout(() => {
      window.scrollTo(0, 0)
    },0)
  }

  componentDidUpdate(prevProps) {
    if (prevProps.pageID !== this.props.pageID) {
      setTimeout(() => {
        window.scrollTo(0, 0)
      },0)
    }
  }

  render() {
    return null
  }
}

export default ScrollToTop
