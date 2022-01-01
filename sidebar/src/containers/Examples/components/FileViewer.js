import React from 'react'

import '../assets/file-viewer.css'

const typeToClass = (type) => {
  switch (type) {
    case "text/csv":
    case "text/plain":
      return "file-text";
    case "image/png":
    case "image/jpg":
    case "image/gif":
      return "file-image";
    default:
      return "file-default";
  }
}

const renderFile = (data) => {
  switch (data.mime_type) {
    case "image/png":
    case "image/jpg":
    case "image/gif":
      return <img
        alt={data.name}
        src={'data:' + data.mime_type + ';base64,' + data.contents_base64}
      />
    case "text/csv":
    case "text/plain":
    case "":
      return null;
    default:
      return <pre className="examples__file-viewer__content--file">
        <code>
          { window.atob(data.contents_base64) }
        </code>
      </pre>
  }
}

class FileViewer extends React.Component {
  constructor(props) {
    super(props)
    this.state = {
      show: false,
    }
  }

  toggle = () => {
    this.setState({
      show: !this.state.show,
    })
  }

  render() {
    return (
      <div
        className={`
          examples__file-viewer
          ${typeToClass(this.props.data.mime_type)}
          ${this.state.show ? "examples__file-viewer--show" :
          "examples__file-viewer--hide"}
        `}
      >
        <button
          className="examples__file-viewer__name"
          onClick={this.toggle}
        >
          <div className="examples__file-viewer__icon"/>
          <span>{ this.props.data.name }</span>
        </button>
        { this.state.show &&
          <div className="examples__file-viewer__content">
            { renderFile(this.props.data) }
          </div>
        }
      </div>
    )
  }
}
export default FileViewer
