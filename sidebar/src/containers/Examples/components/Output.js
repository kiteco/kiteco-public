import React from 'react'

import '../assets/output.css'

const DirectoryTable = ({ entries }) =>
  <table className="examples__output__content__dir-table">
    <thead>
      <tr>
        {Object.keys(entries[0]).map((columnHeader, index) =>
          <th key={"header-" + index}>{columnHeader}</th>
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

const Output = ({chunk}) => ({

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
    switch(chunk.output_type) {
      case "plaintext":
        output = chunk.content.value;
        break;
      case "directory_listing_tree":
        output = <div className="examples__output__content__dir-tree">
            { this.createDirectoryTree(chunk.content.entries) }
          </div>
        break;
      case "directory_listing_table":
        output = <DirectoryTable entries={chunk.content.entries} />
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
    return <pre
      className="examples__output"
      >
      {chunk.content.caption &&
        <div className="examples__output__caption">
          {chunk.content.caption}
        </div>
      }
      <div className="examples__output__content">
        {output}
      </div>
    </pre>
  }
})

export default Output
