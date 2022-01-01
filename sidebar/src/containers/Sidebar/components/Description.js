import React from 'react'
import ReactDOM from 'react-dom';

// Extremely hacky and dangerous. I cry everytime I look at this.
class Description extends React.Component {
  toolTipTimeOut = null
  currentDetails = null
  toolTipMinPauseTime = 150

  NO_DOCS_CONTENT = '<body><p>No documentation available</p></body>'

  handleMouseMove = (event) => {
    let target;
    if (event.target.matches('a.internal_link')) {
      target = event.target;
    } else if (event.target.parentNode.matches('a.internal_link')) {
      target = event.target.parentNode;
    }
    if (target) {
      event.preventDefault();
      const identifier = target.getAttribute('data-identifier');
      if (identifier) {
        this.showToolTip({
          kind: "docs",
          data: { identifier },
          bounds: target.getBoundingClientRect(),
        })
      }
    } else {
      this.hideToolTip()
    }
  }

  showToolTip = (details) => {
    if (this.timeout) {
      clearTimeout(this.timeout)
    }
    this.timeout = setTimeout(() => {
      this.currentDetails = details
      this.props.showToolTip(this.currentDetails)
    }, this.toolTipMinPauseTime)
  }

  hideToolTip = () => {
    clearTimeout(this.timeout)
    if (this.currentDetails) {
      this.props.hideToolTip(this.currentDetails)
      this.currentDetails = null
    }
  }

  // sets internal links and handles them when necessary
  handleClick = (event) => {
    this.hideToolTip()
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

  //Idea is to take <pre><div>line_1</div>...<div>line_n</div></pre> to
  //<span>line_1</span><br/>...<span>line_n</span><br/> for formatting purposes
  transformRawDocs() {
    let pre = ReactDOM.findDOMNode(this).querySelector('pre.raw-docs')
    if(pre) {
      let descriptionEl = ReactDOM.findDOMNode(this).querySelector('div.docs__description')
      let line = Array.from(
        ReactDOM.findDOMNode(this).querySelectorAll('pre.raw-docs > .raw-docs-line')
      )
      .reduce((acc, val, i, arr) => {
        acc += `<span>${val.innerHTML}</span>`
        if(i < arr.length - 1) acc += "<br/>"
        pre.removeChild(val)
        return acc
      }, "")
      descriptionEl.innerHTML = line
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
        link.href = `/docs/${identifier}`
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
      }
    })
  }

  componentDidMount() {
    // only run if documentation_html
    if(this.props.description_html && this.props.description_html !== this.NO_DOCS_CONTENT) {
      this.transformRawDocs()
      this.markupLinks()
    }
  }

  componentDidUpdate() {
    // only run if documentation_html
    if(this.props.description_html && this.props.description_html !== this.NO_DOCS_CONTENT) {
      this.transformRawDocs()
      this.markupLinks()
    }
  }

  render() {
    return (
      <div className={`${this.props.className} docs__description-container`} id="description">
        {this.props.description_html && this.props.description_html !== this.NO_DOCS_CONTENT
          ? (<React.Fragment><h2>Description</h2>
            <div
              className="docs__description"
              dangerouslySetInnerHTML={{__html: this.props.description_html}}
              onClick={this.handleClick}
              onMouseMove={this.handleMouseMove}
            /></React.Fragment>)
          : (<React.Fragment><h2>Documentation</h2>
            <div className="docs__description--docstring">
              <code>{this.props.description_text}</code>
            </div></React.Fragment>)
        }
        
      </div>
    )
  }
}

export default Description
