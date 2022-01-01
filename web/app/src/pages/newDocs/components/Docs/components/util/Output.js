import React from 'react'
import {connect} from 'react-redux'

const DirectoryTable = ({ entries, thClass }) =>
  <table className="examples__output__content__dir-table">
    <thead>
      <tr>
        {Object.keys(entries[0]).map((columnHeader, index) =>
          <th className={thClass} key={"header-" + index}>{columnHeader}</th>
        )}
      </tr>
    </thead>
    <tbody>
      {entries.map((file, index) => {
        return <tr key={"file-" + index}>
          {Object.keys(file).map((attribute, j) =>
            <td key={"attr-"+ j}>{file[attribute]}</td>
          )}
        </tr>;
      })}
    </tbody>
  </table>

const Output = ({chunk, style}) => ({
  createDirectoryTree(entries) {
    if (!entries.name.startsWith("kite.py")) {
      let children = "";
      if (entries.listing.length > 0) {
        children = entries.listing.map(listing => {
          let result = this.createDirectoryTree(listing);
          if (result) {
            result = result.split("\n");
            result.pop();
            result = "\t" + result.join("\n\t") + "\n";
          }
          return result;
        }).join("");
      }
      return entries.name + "\n" + children;
    } else {
      return "";
    }
  },

  render() {
    let output = null;
    const { font } = style
    switch(chunk.output_type) {
      case "plaintext":
        output = chunk.content.value;
        break;
      case "directory_listing_tree":
        const text = this.createDirectoryTree(chunk.content.entries)
        output = <div className="examples__output__content__dir-tree">
            { text  }
          </div>
        break;
      case "directory_listing_table":
        const thClass = font.name === 'Inconsolata' || font.name === 'Input'
          ? 'bold-font'
          : ''
        output = <DirectoryTable thClass={thClass} entries={chunk.content.entries} />
        break;
      case "image":
        output = <img
          className="examples__output__image"
          alt={chunk.content.caption}
          src={"data:" + chunk.content.encoding + ';base64,' + chunk.content.data}
        />
        break;
      case "file":
        output = decodeURIComponent(escape(window.atob(chunk.content.data)));
        break;
      default:
        return null;
    }
    return <div className="example-output">
      <h3 className="output-header">
      { chunk.output_type === 'directory_listing_tree' || chunk.output_type === 'directory_listing_table'
        ? chunk.content.caption
        : 'Output'
      }
    </h3>
      <div className="output-wrapper">
        <div className="output">
        {chunk.content.caption && 
          (chunk.output_type !== 'directory_listing_tree' && chunk.output_type !== 'directory_listing_table') &&
          <div className="output__caption">
            {chunk.content.caption}
          </div>
        }
          {output}
        </div>
      </div>
    </div>
  }
})

const mapStateToProps = (state, ownProps) => ({
  ...ownProps,
  style: state.stylePopup
})

export default connect(mapStateToProps)(Output)