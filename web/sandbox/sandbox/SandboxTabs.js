import React from 'react'
import { connect } from 'react-redux'

import './sandbox-tabs.css'

const SandboxTab = ({ changeFile, name, selected, editorHasOverlay }) => {
  return (
    <div 
      className={`sandbox-tabs__tab${selected ? ' highlighted' : ''}${editorHasOverlay ? ' overlaid' : ''}`}
      onClick={selected ? null : changeFile}
    >
      { name }
    </div>
  )
}

const SandboxTabs = ({ filenames, selectedFile, changeFileFactory, editorHasOverlay }) => {
  return (
    <div className="sandbox-tabs">
      {
        filenames.map((name, i) => <SandboxTab
                                  key={i}
                                  name={name}
                                  changeFile={changeFileFactory(name)}
                                  selected={selectedFile === name}
                                  editorHasOverlay={editorHasOverlay}
                                />)
      }
    </div>
  )
}

const mapStateToProps = (state, props) => {
  const storeEditor = state.sandboxEditors.editorMap[props.editorId] || {}
  return {
    editorHasOverlay: storeEditor.shouldShowOverlay || storeEditor.shouldShowHoverOverlay,
  }
}

export default connect(mapStateToProps, null)(SandboxTabs)