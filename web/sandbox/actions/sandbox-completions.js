import { POST } from './fetch'
import { sandboxCompletionsPath } from '../utils/urls'
import { createJson } from '../utils/fetch'
import { LOG_RESPONSE_TIME } from '../utils/fetch-middleware';

export const REGISTER_COMPLETIONS = 'register completions'
export const registerCompletions = (completions, editorId, filename) => ({
  type: REGISTER_COMPLETIONS,
  completions,
  editorId,
  filename,
})

export const LOAD_COMPLETIONS_FAILED = 'load completions failed'
export const loadCompletionsFailed = (editorId, filename) => ({
  type: LOAD_COMPLETIONS_FAILED,
  editorId,
  filename,
})

export const COMPLETIONS_FILE_TOO_LARGE = 'completions file too large'
export const completionsFileTooLarge = (editorId, filename) => ({
  type: COMPLETIONS_FILE_TOO_LARGE,
  editorId,
  filename,
})

export const CLEAR_COMPLETIONS = 'clear completions'
export const clearCompletions = (editorId, filename) => ({
  type: CLEAR_COMPLETIONS,
  editorId,
  filename,
})

export const COMPLETIONS_FILE_CLEARED = 'completions file cleared'
export const completionsFileCleared = (editorId, filename) => ({
  type: COMPLETIONS_FILE_CLEARED,
  editorId,
  filename,
})

export const LOG_COMPLETIONS_RESPONSE_TIME = 'log completions response time'
export const logCompletionsResponseTime = (times) => ({
  type: LOG_COMPLETIONS_RESPONSE_TIME,
  times,
})

export const SKIP_COMPLETIONS = 'skip completions'
export const skipCompletions = (skip, editorId, filename) => dispatch => dispatch({
  type: SKIP_COMPLETIONS,
  skip,
  editorId,
  filename,
})

export const MAX_CHARS = 2000
export const fetchCompletions = (text="", cursor_bytes=0, id="", filename="", timeout=0) => dispatch => {
  if(text.length > MAX_CHARS) {
    dispatch(completionsFileTooLarge(id, filename))
    return Promise.resolve(COMPLETIONS_FILE_TOO_LARGE)
  }
  const opts = {
    url: sandboxCompletionsPath(id),
    // uncommenting the below allows sandbox completions to function in localhost development
   /*  urlPrefix: "https://staging.kite.com", */
    options: createJson({ text, cursor_bytes, filename, id }),
    cursorPos: cursor_bytes,
    middleware: { [LOG_RESPONSE_TIME]: {} }
  }
  if(timeout && typeof timeout === "number") {
    opts.timeout = timeout
  }
  return dispatch(POST(opts))
    .then(({ success, data, middleware, error }) => {
      if(middleware && middleware[LOG_RESPONSE_TIME]) {
        dispatch(logCompletionsResponseTime(middleware[LOG_RESPONSE_TIME]))
      }
      if(success && data.completions) {
        dispatch(registerCompletions(data.completions, id, filename))
        return data.completions
      } else {
        dispatch(loadCompletionsFailed(id, filename))
        return LOAD_COMPLETIONS_FAILED
      }
    })
}