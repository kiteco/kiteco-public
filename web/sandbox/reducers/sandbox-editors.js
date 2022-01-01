//TODO(dane): may need to add capbilities for handling multiple / named
// editors and buffers

import * as actions from '../actions/sandbox-editors'

const defaultState = {
  editorMap: {}
}

const editorDefault = {
  bufferMap: {},
  completionIndex: -1,
  shouldSelectCompletion: false,
  shouldMoveCompletion: false,
  shouldCloseCompletions: false,
  shouldFocusEditor: false,
  showCaption: false,
  shouldShowOverlay: false,
  shouldShowHoverOverlay: false,
  caption: "",
  cursor: null,
  cursorPos: -1,
  cursorBlink: false,
  completionCaptionPadding: false,
  completionsBoxDimensions: {},
  completionsBoxHighlightLevel: 0,
  showCursorCaption: false,
  cursorCaption: "",
  cursorCaptionPlacement: "",
  cursorCaptionMarginTop: 0,
  cursorCaptionMarginLeft: 0,
  cursorCaptionAfterClass: '',
}

const setBuffer = (state, action) => {
  const { [action.editorId]: targetEditor, ...otherEditors } = state.editorMap
  if(targetEditor && targetEditor.bufferMap) {
    const { [action.filename]: targetBuffer, ...otherBuffers } = targetEditor.bufferMap
    return {
      editorMap: {
        ...otherEditors,
        [action.editorId]: {
          ...targetEditor,
          bufferMap: {
            ...otherBuffers,
            [action.filename]: action.buffer
          }
        }
      }
    }
  }
  return state
}

const setCursor = (state, action) => {
  const { [action.editorId]: targetEditor, ...otherEditors } = state.editorMap
  return {
    editorMap: {
      ...otherEditors,
      [action.editorId]: {
        ...targetEditor,
        completionIndex: targetEditor.completionIndex + 1,
        cursor: action.pos,
      }
    }
  }
}

const cursorSet = (state, action) => {
  const { [action.editorId]: targetEditor, ...otherEditors } = state.editorMap
  return {
    editorMap: {
      ...otherEditors,
      [action.editorId]: {
        ...targetEditor,
        cursor: null,
      }
    }
  }
}

const setCursorPos = (state, action) => {
  const { [action.editorId]: targetEditor, ...otherEditors } = state.editorMap
  return {
    editorMap: {
      ...otherEditors,
      [action.editorId]: {
        ...targetEditor,
        cursorPos: action.pos,
      }
    }
  }
}

const setCursorBlink = (state, action) => {
  const { [action.editorId]: targetEditor, ...otherEditors } = state.editorMap
  return {
    editorMap: {
      ...otherEditors,
      [action.editorId]: {
        ...targetEditor,
        cursorBlink: action.blink,
      }
    }
  }
}

const concatToBuffer = (state, action) => {
  const { [action.editorId]: targetEditor, ...otherEditors } = state.editorMap
  if(targetEditor && targetEditor.bufferMap) {
    const { [action.filename]: targetBuffer, ...otherBuffers } = targetEditor.bufferMap
    const newBuf = targetBuffer + action.str
  
    return {
      editorMap: {
        ...otherEditors,
        [action.editorId]: {
          ...targetEditor,
          bufferMap: {
            ...otherBuffers,
            [action.filename]: newBuf,
          },
        },
      },
    }
  }
  return state
}

const typeKey = (state, action) => {
  const { [action.editorId]: targetEditor, ...otherEditors } = state.editorMap
  if(targetEditor && targetEditor.bufferMap) {
    const { [action.filename]: targetBuffer, ...otherBuffers } = targetEditor.bufferMap
    const newBuf = targetBuffer + action.key
    return {
      editorMap: {
        ...otherEditors,
        [action.editorId]: {
          ...targetEditor,
          bufferMap: {
            ...otherBuffers,
            [action.filename]: newBuf,
          },
        },
      },
    }
  }
  return state
}

const setCompletionIdx = (state, action) => {
  const { [action.editorId]: targetEditor, ...otherEditors } = state.editorMap

  return {
    editorMap: {
      ...otherEditors,
      [action.editorId]: {
        ...targetEditor,
        completionIndex: action.idx
      }
    }
  }
}

const incrementCompletionIdx = (state, action) => {
  const { [action.editorId]: targetEditor, ...otherEditors } = state.editorMap
  return {
    editorMap: {
      ...otherEditors,
      [action.editorId]: {
        ...targetEditor,
        completionIndex: targetEditor.completionIndex + 1,
        shouldMoveCompletion: true,
      }
    }
  }
}

const selectCompletion = (state, action) => {
  const { [action.editorId]: targetEditor, ...otherEditors } = state.editorMap

  return {
    editorMap: {
      ...otherEditors,
      [action.editorId]: {
        ...targetEditor,
        shouldSelectCompletion: true,
      }
    }
  }
}

const completionSelected = (state, action) => {
  const { [action.editorId]: targetEditor, ...otherEditors } = state.editorMap

  return {
    editorMap: {
      ...otherEditors,
      [action.editorId]: {
        ...targetEditor,
        completionIndex: -1,
        shouldSelectCompletion: false,
      }
    }
  }
}

const completionMoved = (state, action) => {
  const { [action.editorId]: targetEditor, ...otherEditors } = state.editorMap

  return {
    editorMap: {
      ...otherEditors,
      [action.editorId]: {
        ...targetEditor,
        shouldMoveCompletion: false,
      }
    }
  }
}

const resetCompletionUI = (state, action) => {
  const { [action.editorId]: targetEditor, ...otherEditors } = state.editorMap

  return {
    editorMap: {
      ...otherEditors,
      [action.editorId]: {
        ...targetEditor,
        completionIndex: -1,
        shouldMoveCompletion: false,
        shouldSelectCompletion: false,
      }
    }
  }
}

const focusEditor = (state, action) => {
  const { [action.editorId]: targetEditor, ...otherEditors } = state.editorMap

  return {
    editorMap: {
      ...otherEditors,
      [action.editorId]: {
        ...targetEditor,
        shouldFocusEditor: true,
      }
    }
  }
}

const editorFocused = (state, action) => {
  const { [action.editorId]: targetEditor, ...otherEditors } = state.editorMap

  return {
    editorMap: {
      ...otherEditors,
      [action.editorId]: {
        ...targetEditor,
        shouldFocusEditor: false,
      }
    }
  }
}

const registerEditorFiles = (state, action) => {
  const { [action.editorId]: targetEditor={}, ...otherEditors } = state.editorMap

  // for each filename, will need to create a field that doesn't overwrite something
  // else that's present
  const initializedBuffers = action.filenames.reduce((bufferObj, filename) => {
    bufferObj[filename] = ""
    return bufferObj
  }, {})
  return {
    editorMap: {
      [action.editorId]: {
        ...targetEditor,
        bufferMap: {
          ...initializedBuffers,
          ...targetEditor.bufferMap,
        }
      },
      ...otherEditors
    } 
  }
}

const activeFileChanged = (state, action) => {
  const { [action.editorId]: targetEditor={}, ...otherEditors } = state.editorMap

  return {
    editorMap: {
      ...otherEditors,
      [action.editorId]: {
        ...editorDefault,
        bufferMap: targetEditor.bufferMap,
      }
    }
  }
}

const registerEditor = (state, action) => {
  const editorMap = {
    ...state.editorMap,
    [action.editorId]: { ...editorDefault }
  }
  return {
    editorMap
  }
}

const deleteEditor = (state, action) => {
  const { [action.editorId]: editorToDelete, ...editors } = state.editorMap
  return {
    editorMap: {
      ...editors
    }
  }
}

const showCaption = (state, action) => {
  const { [action.editorId]: targetEditor, ...otherEditors } = state.editorMap

  return {
    editorMap: {
      ...otherEditors,
      [action.editorId]: {
        ...targetEditor,
        showCaption: true,
        caption: action.caption,
        completionCaptionPadding: action.completionCaptionPadding || false,
      }
    }
  }
}

const hideCaption = (state, action) => {
  const { [action.editorId]: targetEditor, ...otherEditors } = state.editorMap

  return {
    editorMap: {
      ...otherEditors,
      [action.editorId]: {
        ...targetEditor,
        showCaption: false,
        caption: "",
        completionCaptionPadding: false,
      }
    }
  }
}

const showOverlay = (state, action) => {
  const { [action.editorId]: targetEditor, ...otherEditors } = state.editorMap

  return {
    editorMap: {
      ...otherEditors,
      [action.editorId]: {
        ...targetEditor,
        shouldShowOverlay: true,
      }
    }
  }
}

const hideOverlay = (state, action) => {
  const { [action.editorId]: targetEditor, ...otherEditors } = state.editorMap

  return {
    editorMap: {
      ...otherEditors,
      [action.editorId]: {
        ...targetEditor,
        shouldShowOverlay: false,
      }
    }
  }
}

const showHoverOverlay = (state, action) => {
  const { [action.editorId]: targetEditor, ...otherEditors } = state.editorMap

  return {
    editorMap: {
      ...otherEditors,
      [action.editorId]: {
        ...targetEditor,
        shouldShowHoverOverlay: true,
      }
    }
  }
}

const hideHoverOverlay = (state, action) => {
  const { [action.editorId]: targetEditor, ...otherEditors } = state.editorMap

  return {
    editorMap: {
      ...otherEditors,
      [action.editorId]: {
        ...targetEditor,
        shouldShowHoverOverlay: false,
      }
    }
  }
}

const closeCompletions = (state, action) => {
  const { [action.editorId]: targetEditor, ...otherEditors } = state.editorMap

  return {
    editorMap: {
      ...otherEditors,
      [action.editorId]: {
        ...targetEditor,
        shouldCloseCompletions: true,
      }
    }
  }
}

const completionsClosed = (state, action) => {
  const { [action.editorId]: targetEditor, ...otherEditors } = state.editorMap

  return {
    editorMap: {
      ...otherEditors,
      [action.editorId]: {
        ...targetEditor,
        shouldCloseCompletions: false,
      }
    }
  }
}

const setCompletionsBoxDimensions = (state, action) => {
  const { [action.editorId]: targetEditor, ...otherEditors } = state.editorMap

  return {
    editorMap: {
      ...otherEditors,
      [action.editorId]: {
        ...targetEditor,
        completionsBoxDimensions: { ...action.dimensions },
      }
    }
  }
}

const showCursorCaption = (state, action) => {
  const { [action.editorId]: targetEditor, ...otherEditors } = state.editorMap

  return {
    editorMap: {
      ...otherEditors,
      [action.editorId]: {
        ...targetEditor,
        showCursorCaption: true,
        cursorCaption: action.caption,
        cursorCaptionPlacement: action.placement,
        cursorCaptionMarginTop: action.marginTop,
        cursorCaptionMarginLeft: action.marginLeft,
        cursorCaptionAfterClass: action.afterClass,
        completionsBoxHighlightLevel: action.moves,
      }
    }
  }
}

const hideCursorCaption = (state, action) => {
  const { [action.editorId]: targetEditor, ...otherEditors } = state.editorMap

  return {
    editorMap: {
      ...otherEditors,
      [action.editorId]: {
        ...targetEditor,
        showCursorCaption: false,
        cursorCaption: '',
        cursorCaptionPlacement: '',
        ursorCaptionMarginTop: 0,
        cursorCaptionMarginLeft: 0,
        cursorCaptionAfterClass: '',
        completionsBoxHighlightLevel: 0
      }
    }
  }
}

const sandboxEditors = (state = defaultState, action) => {
  switch(action.type) {
    case actions.SET_BUFFER:
      return setBuffer(state, action)
    case actions.SET_CURSOR:
      return setCursor(state, action)
    case actions.CURSOR_SET:
      return cursorSet(state, action)
    case actions.SET_CURSOR_POS:
      return setCursorPos(state, action)
    case actions.SET_CURSOR_BLINK:
      return setCursorBlink(state, action)
    case actions.CONCAT_TO_BUFFER:
      return concatToBuffer(state, action)
    case actions.TYPE_KEY:
      return typeKey(state, action)
    case actions.SET_COMPLETION_IDX:
      return setCompletionIdx(state, action)
    case actions.INCREMENT_COMPLETION_IDX:
      return incrementCompletionIdx(state, action)
    case actions.SELECT_COMPLETION:
      return selectCompletion(state, action)
    case actions.COMPLETION_SELECTED:
      return completionSelected(state, action)
    case actions.COMPLETION_MOVED:
      return completionMoved(state, action)
    case actions.RESET_COMPLETION_UI:
      return resetCompletionUI(state, action)
    case actions.FOCUS_EDITOR:
      return focusEditor(state, action)
    case actions.EDITOR_FOCUSED:
      return editorFocused(state, action)
    case actions.REGISTER_EDITOR_FILES:
      return registerEditorFiles(state, action)
    case actions.ACTIVE_FILE_CHANGED:
      return activeFileChanged(state, action)
    case actions.registerEditor:
      return registerEditor(state, action)
    case actions.DELETE_EDITOR:
      return deleteEditor(state, action)
    case actions.SHOW_CAPTION:
      return showCaption(state, action)
    case actions.HIDE_CAPTION:
      return hideCaption(state, action)
    case actions.SHOW_OVERLAY:
      return showOverlay(state, action)
    case actions.HIDE_OVERLAY:
      return hideOverlay(state, action)
    case actions.HIDE_HOVER_OVERLAY:
      return hideHoverOverlay(state, action)
    case actions.SHOW_HOVER_OVERLAY:
      return showHoverOverlay(state, action)
    case actions.CLOSE_COMPLETIONS:
      return closeCompletions(state, action)
    case actions.COMPLETIONS_CLOSED:
      return completionsClosed(state, action)
    case actions.SET_COMPLETIONS_BOX_DIMENSIONS:
      return setCompletionsBoxDimensions(state, action)
    case actions.SHOW_CURSOR_CAPTION:
      return showCursorCaption(state, action)
    case actions.HIDE_CURSOR_CAPTION:
      return hideCursorCaption(state, action)
    default:
      return state
  }
}

export default sandboxEditors