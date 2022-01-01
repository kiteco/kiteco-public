import React from 'react'

import './assets/file-tree.css'

const FileTree = ({
  root,
  separator,
  defaultOpen,
}) =>
  <div className="file-tree">
    <Folder
      {...root}
      separator={separator}
      defaultOpen={defaultOpen}
    />
  </div>

class Folder extends React.Component {
  constructor(props) {
    super(props)
    const { defaultOpen } = props
    this.state = {
      open: defaultOpen || false,
    }
  }

  toggle = () => {
    this.setState(state => ({
      ...state,
      open: !state.open,
    }))
  }

  render() {
    const { files, folders, name, separator } = this.props
    const { open } = this.state
    return <div className="file-tree__folder">
      <div
        className="file-tree__header"
        onClick={ this.toggle }
      >
        <div className={`
          file-tree__folder-indicator
          file-tree__folder-indicator--${open ? "open" : "closed"}
        `}/>
        <div className="file-tree__name file-tree__name--folder">
          <PathBreak path={name} separator={separator}/>
          { separator }
        </div>
      </div>
      { open &&
        <div className="file-tree__body">
          { folders && Object.keys(folders).map(folder =>
            <Folder
              key={folders[folder].name}
              { ...folders[folder] }
              separator={separator}
            />
          )}
          { files && files.map(file =>
            <File key={file.name} { ...file }/>
          )}
          { files && folders &&
            files.length === 0 &&
            Object.keys(folders).length === 0 &&
            <p className="file-tree__empty">
              No files or folders
            </p>
          }
        </div>
      }
    </div>
  }
}

const File = ({
  name,
  created_at,
  hashed_content,
  updated_at,
}) =>
  <div className="file-tree__file">
    <div className={`
      file-tree__name
      ${name === ".kiteignore"
        ? "file-tree__name--kiteignore"
        : "file-tree__name--file"
      }
    `}>
      { name }
    </div>
  </div>

export const PathBreak = ({ path, separator }) =>
  <span>
    { path.split(separator)
        .filter((f, i) => f || !i)
        .map((f, i) =>
          i
          ? <span key={i}><wbr/>{ separator }{ f }</span>
          : <span key={i}>{ f }</span>
    )}
  </span>

export default FileTree
