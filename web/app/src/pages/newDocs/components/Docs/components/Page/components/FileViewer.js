import React from 'react'
import { LinedBlock } from '../../util/Code'

const renderFile = (data, language) => {
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
      return <pre>
        <code>
          <LinedBlock
            code={window.atob(data.contents_base64)}
            numberLines={false}
            language={language}
          />
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
      <div>
        <button
          className="file-name code"
          onClick={this.toggle}
        >
          <span>{ this.props.data.name }</span>
        </button>
        { this.state.show &&
          <div className="file-output">
            { renderFile(this.props.data, this.props.language) }
          </div>
        }
      </div>
    )
  }
}

export default FileViewer
