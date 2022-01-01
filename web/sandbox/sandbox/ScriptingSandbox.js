import React from 'react'
import { connect } from 'react-redux'

import FileProvider from './FileProvider'
import SandboxTabs from './SandboxTabs'
import Scripter from './Scripter'
import EditorSandbox from './EditorSandbox'
import SandboxDisclaimer from './SandboxDisclaimer'

const LATENCY_TEXT = "Kite's completions are taking longer than we'd like to appear because of your internet connection. When using Kite with your editor they will be instant."
const FILE_TOO_LARGE_TEXT = "We only support providing Kite's remote sandbox completions for buffers fewer than 2000 characters at this time."

class ScriptingSandbox extends React.Component {

  render() {
    return <FileProvider render={provider => (
      <React.Fragment>
          {provider.filenames.length > 1 &&
            <SandboxTabs
              filenames={provider.filenames}
              selectedFile={provider.activeFilename}
              changeFileFactory={provider.changeFileFactory}
              editorId={provider.editorId}
            />
          }
          <Scripter
            activeScriptName={provider.activeFilename}
            filenames={provider.filenames}
            editorId={provider.editorId}
            loop={true}
          >
            <EditorSandbox
              activeFile={provider.activeFilename}
              editorId={provider.editorId}
              setIdCb={provider.setEditorId}
              theme={this.props.theme}
            />
          </Scripter>
          {this.props.fileTooLargeMap[provider.editorId+provider.activeFilename] &&
            <SandboxDisclaimer
              theme={this.props.theme}
              text={FILE_TOO_LARGE_TEXT}
            />
          }
          {this.props.completionsHaveLatency &&
            !this.props.fileTooLargeMap[provider.editorId+provider.activeFilename] &&
            <SandboxDisclaimer
              theme={this.props.theme}
              text={LATENCY_TEXT}
            />
          }
        </React.Fragment>
    )}/>
  }
}

const mapStateToProps = (state, props) => ({
  completionsHaveLatency: state.sandboxCompletions.completionsHaveLatency,
  fileTooLargeMap: state.sandboxCompletions.fileTooLargeMap,
})

export default connect(mapStateToProps, null)(ScriptingSandbox)