import React from 'react'
import ReactDOM from 'react-dom'
import { connect } from 'react-redux'
import { push } from 'connected-react-router'

import { queryIdToRoute } from '../../../../../../../utils/route-parsing'

const TRUNCATED_HEIGHT = 440

class Description extends React.Component {

  constructor(props) {
    super(props)
    this.descriptionText = React.createRef()
    this.descriptionTitle = React.createRef()
    const { defaultExpanded } = this.props
    this.state = {
      expanded: defaultExpanded || false,
      showToggleTextBtn: false,
    }
  }



  toggle = () => {
    this.setState({
      expanded: !this.state.expanded
    }, () => {
      if(!this.state.expanded) {
        this.descriptionTitle.current.scrollIntoView({
          behavior: 'smooth'
        })
      }
    })
  }

  computeToggleBtn = () => {
    let descriptionTextEl = this.descriptionText.current
    if(descriptionTextEl.clientHeight >= TRUNCATED_HEIGHT) {
      if(!this.state.showToggleTextBtn) {
        this.setState({ showToggleTextBtn: true })
      }
    } else {
      if(this.state.showToggleTextBtn) {
        this.setState({ showToggleTextBtn: false })
      }
    }
  }

  markupLinks() {
    let links =  [].slice.call(
      ReactDOM.findDOMNode(this).querySelectorAll('a.internal_link')
    );

    links.forEach((link) => {
      let path = link.getAttribute('href');
      if (path && path[0] === "#") {
        const identifier = path.substring(1);
        link.href = `/${this.props.language}/docs${queryIdToRoute(identifier)}`
        link.setAttribute('data-identifier', identifier);
      }
    })

    let externalLinks = [].slice.call(ReactDOM.findDOMNode(this).querySelectorAll('a.external_link'));

    externalLinks.forEach((link) => {
      let path = link.getAttribute('href');
      if (path && path[0] === "#") {
        path = path.substring(1);
        link.setAttribute('href', path);
        link.setAttribute('target', '_blank');
        link.setAttribute('rel', 'nofollow');
      }
    })
  }

  componentDidMount() {
    if(this.props.description_html) {
      this.computeToggleBtn()
      this.markupLinks()
    }
  }

  componentDidUpdate(prevProps) {
    if(this.props.description_html) {
      this.computeToggleBtn()
      this.markupLinks()
      if(prevProps.description_html !== this.props.description_html) {
        this.setState({
          expanded: false,
        })
      }
    }
  }

  // sets internal links and handles them when necessary
  handleClick = (event) => {
    if (event.isDefaultPrevented() || event.metaKey || event.ctrlKey) {
      return;
    }
    let el = null;

    if (event.target.className === 'internal_link') {
      el = event.target;
    } else if (event.target.parentNode.className === 'internal_link') {
      el = event.target.parentNode;
    }

    if (el !== null ) {
      event.preventDefault();
      let path = el.getAttribute('href');
      this.props.push(path);
    } else  {
      return;
    }
  }

  render() {
    const {expanded, showToggleTextBtn} = this.state
    const { docString, description_html } = this.props
    return (<section className='description'>
      <h3 ref={this.descriptionTitle}>
        {description_html && "Description"}
        {!description_html && "Documentation"}
      </h3>
      { description_html && <div
        className={`description-text ${expanded || !showToggleTextBtn ? '' : 'truncated-text'}`}
        ref={this.descriptionText}
        dangerouslySetInnerHTML={{__html: description_html}}
        onClick={this.handleClick}
      /> }
      {
        !description_html && docString && <div
          className={`description-text ${expanded || !showToggleTextBtn ? '' : 'truncated-text'}`}
          ref={this.descriptionText}
        >
          <code className="documentation-string">{docString}</code>
        </div>
      }
      { showToggleTextBtn && <button
        className={`toggle-truncated-text${expanded ? ' contract' : ''}`}
        onClick={this.toggle}></button>}
    </section>)
  }
}

const mapStateToProps = (state, ownProps) => ({

})

const mapDispatchToProps = dispatch => ({
  push: params => dispatch(push(params)),
})

export default connect(mapStateToProps, mapDispatchToProps)(Description)
