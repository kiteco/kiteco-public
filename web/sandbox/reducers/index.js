import { combineReducers } from 'redux'
import sandboxCompletions from './sandbox-completions'
import cachedCompletions from './cached-completions'
import sandboxEditors from './sandbox-editors'

const reducer = combineReducers({
  sandboxCompletions,
  cachedCompletions,
  sandboxEditors,
})

export default reducer