import React from 'react'
import { connect } from 'react-redux'

import { registerEditorFiles, activeFileChanged } from '../actions/sandbox-editors'
import { clearCompletions } from '../actions/sandbox-completions'

import {
  TAB_NUMERICS,
  FILENAMES,
} from './constants'

class FileProvider extends React.Component {
  constructor(props) {
    super(props)

    this.setEditorId = (id) => {
      this.setState(state => ({
        editorId: id,
      }))
      this.props.registerEditorFiles(id, FILENAMES)
    }

    this.changeFileFactory = filename => () => {
      this.props.clearCompletions(this.state.editorId, this.state.activeFilename)
      this.props.activeFileChanged(this.state.editorId)
      this.setState({ activeFilename: filename })
    }

    this.state = {
      filenames: FILENAMES,
      activeFilename: TAB_NUMERICS,
      setEditorId: this.setEditorId,
      editorId: "",
      changeFileFactory: this.changeFileFactory,
    }
  }

  render() {
    return this.props.render(this.state)
  }
}

const mapDispatchToProps = dispatch => ({
  registerEditorFiles: (id, filenames) => dispatch(registerEditorFiles(id, filenames)),
  clearCompletions: (id, filename) => dispatch(clearCompletions(id, filename)),
  activeFileChanged: (id) => dispatch(activeFileChanged(id)),
})

export default connect(null, mapDispatchToProps)(FileProvider)