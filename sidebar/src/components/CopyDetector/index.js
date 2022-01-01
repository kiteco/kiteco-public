import React from 'react'
import ReactDOM from 'react-dom';
// import { track } from '../../utils/analytics'
import { Link } from 'react-router-dom'

import './assets/copy-detector.css'

class CopyDetector extends React.Component {

  constructor(props) {
    super(props)
    this.state = {
      copied: false,
    }
  }

  componentDidMount() {
    ReactDOM.findDOMNode(this).addEventListener('copy', this.contentHasBeenCopied);
  }

  componentWillUnmount() {
    ReactDOM.findDOMNode(this).removeEventListener('copy', this.contentHasBeenCopied);
  }

  contentHasBeenCopied = (event) => {
    /// const { full_name, origin, accountStatus } = this.props
    this.setState({
      copied: true,
    }, () => setTimeout(() => this.setState({ copied: false }), 3500))

    // track({
    //   event: "webapp: user copied code",
    //   props: {
    //     full_name: full_name,
    //     origin: origin,
    //     accountStatus: accountStatus,
    //   }
    // })
  }



  render() {
    const { accountStatus, children } = this.props
    const { copied } = this.state
    let renderChildren = null

    if (children) {
      if (typeof children === 'function') {
        renderChildren = children()
      } else {
        renderChildren = React.Children.only(children)
      }
    }

    return <div className="copy-detection-wrapper">
      {copied && accountStatus === 'logged-out' && <div className="copy-detection-wrapper__copied-overlay">
        <div className="copy-detection-wrapper__label">
          <div className="copy-detection-wrapper__label--title">
            Selection copied to clipboard!
          </div>
          Get this example and more<br/> inside your editor with Kite
          <br/>
          <Link className="copy-detection-wrapper__label--link" to="/">Learn more</Link>
        </div>
      </div>}
      {renderChildren}
    </div>
  }

}

export default CopyDetector
