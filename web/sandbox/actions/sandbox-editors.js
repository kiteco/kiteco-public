import { COMPLETIONS_FILE_CLEARED, MAX_CHARS } from "./sandbox-completions";

export const SET_BUFFER = 'set sandbox buffer'
export const setBuffer = (buffer, editorId, filename) => dispatch => {
  if(buffer.length < MAX_CHARS) {
    dispatch({
      type: COMPLETIONS_FILE_CLEARED,
      editorId,
      filename,
    })
  }
  return dispatch({
    type: SET_BUFFER,
    buffer,
    editorId,
    filename,
  })
}

export const SET_CURSOR = 'set cursor'
export const setCursor = (pos, editorId) => dispatch => dispatch({
  type: SET_CURSOR,
  pos,
  editorId,
})

export const CURSOR_SET = 'cursor set'
export const cursorSet = (editorId) => dispatch => dispatch({
  type: CURSOR_SET,
  editorId,
})

export const SET_CURSOR_POS = 'set cursor pos'
export const setCursorPos = (editorId, pos) => dispatch => dispatch({
  type: SET_CURSOR_POS,
  editorId,
  pos
})

export const SET_CURSOR_BLINK = 'set cursor blink'
export const setCursorBlink = (blink, editorId) => dispatch => dispatch({
  type: SET_CURSOR_BLINK,
  editorId,
  blink,
})

export const CONCAT_TO_BUFFER = 'concat to sandbox buffer'
export const concatToBuffer = (str, editorId, filename) => dispatch => dispatch({
  type: CONCAT_TO_BUFFER,
  str,
  editorId,
  filename,
})

export const TYPE_KEY = 'type sandbox buffer key'
export const typeKey = (key, editorId, filename) => dispatch => dispatch({
  type: TYPE_KEY,
  key,
  editorId,
  filename,
})

export const SET_COMPLETION_IDX = 'set completion idx'
export const setCompletionIdx = (idx, editorId) => dispatch => dispatch({
  type: SET_COMPLETION_IDX,
  idx,
  editorId,
})

export const INCREMENT_COMPLETION_IDX = 'increment completion idx'
export const incrementCompletionIdx = (editorId) => dispatch => dispatch({
  type: INCREMENT_COMPLETION_IDX,
  editorId,
})

export const SELECT_COMPLETION = 'select completion'
export const selectCompletion = (editorId) => dispatch => dispatch({
  type: SELECT_COMPLETION,
  editorId,
})

export const COMPLETION_SELECTED = 'completion selected'
export const completionSelected = (editorId) => dispatch => dispatch({
  type: COMPLETION_SELECTED,
  editorId,
})

export const COMPLETION_MOVED = 'completion moved'
export const completionMoved = (editorId) => dispatch => dispatch({
  type: COMPLETION_MOVED,
  editorId,
})

export const CLOSE_COMPLETIONS = 'close completions'
export const closeCompletions = (editorId) => dispatch => dispatch({
  type: CLOSE_COMPLETIONS,
  editorId,
})

export const COMPLETIONS_CLOSED = 'completions closed'
export const completionsClosed = (editorId) => dispatch => dispatch({
  type: COMPLETIONS_CLOSED,
  editorId,
})

export const RESET_COMPLETION_UI = 'reset completion ui'
export const resetCompletionUI = (editorId) => dispatch => dispatch({
  type: RESET_COMPLETION_UI,
  editorId,
})

export const FOCUS_EDITOR = 'focus editor'
export const focusEditor = (editorId) => dispatch => dispatch({
  type: FOCUS_EDITOR,
  editorId,
})

export const EDITOR_FOCUSED = 'editor focused'
export const editorFocused = (editorId) => dispatch => dispatch({
  type: EDITOR_FOCUSED,
  editorId,
})

export const REGISTER_EDITOR_FILES = 'register editor files'
export const registerEditorFiles = (editorId, filenames) => dispatch => dispatch({
  type: REGISTER_EDITOR_FILES,
  editorId,
  filenames,
})

export const ACTIVE_FILE_CHANGED = 'active file changed'
export const activeFileChanged = (editorId) => dispatch => dispatch({
  type: ACTIVE_FILE_CHANGED,
  editorId,
})

export const REGISTER_EDITOR = 'register editor'
export const registerEditor = (editorId) => dispatch => dispatch({
  type: REGISTER_EDITOR,
  editorId,
})

export const DELETE_EDITOR = 'delete editor'
export const deleteEditor = (editorId) => dispatch => dispatch({
  type: DELETE_EDITOR,
  editorId,
})

export const SHOW_CAPTION = 'show caption'
export const showCaption = (editorId, caption, completionCaptionPadding) => dispatch => dispatch({
  type: SHOW_CAPTION,
  editorId,
  caption,
  completionCaptionPadding,
})

export const HIDE_CAPTION = 'hide caption'
export const hideCaption = (editorId) => dispatch => dispatch({
  type: HIDE_CAPTION,
  editorId,
})

export const SHOW_OVERLAY = 'show overlay'
export const showOverlay = (editorId) => dispatch => dispatch({
  type: SHOW_OVERLAY,
  editorId
})

export const HIDE_OVERLAY = 'hide overlay'
export const hideOverlay = (editorId) => dispatch => dispatch({
  type: HIDE_OVERLAY,
  editorId,
})

export const SHOW_HOVER_OVERLAY = 'show hover overlay'
export const showHoverOverlay = (editorId) => dispatch => dispatch({
  type: SHOW_HOVER_OVERLAY,
  editorId,
})

export const HIDE_HOVER_OVERLAY = 'hide hover overlay'
export const hideHoverOverlay = (editorId) => dispatch => dispatch({
  type: HIDE_HOVER_OVERLAY,
  editorId,
})

export const SET_COMPLETIONS_BOX_DIMENSIONS = 'set completion box dimensions'
export const setCompletionsBoxDimensions = (editorId, dimensions={}) => dispatch => dispatch({
  type: SET_COMPLETIONS_BOX_DIMENSIONS,
  editorId,
  dimensions,
})

export const SHOW_CURSOR_CAPTION = 'show cursor caption'
export const showCursorCaption = (editorId, caption="", placement="", marginTop=0, marginLeft=0, afterClass='', moves=0) => dispatch => dispatch({
  type: SHOW_CURSOR_CAPTION,
  editorId,
  caption,
  placement,
  marginTop,
  marginLeft,
  afterClass,
  moves,
})

export const HIDE_CURSOR_CAPTION = 'hide cursor caption'
export const hideCursorCaption = (editorId) => dispatch => dispatch({
  type: HIDE_CURSOR_CAPTION,
  editorId,
})