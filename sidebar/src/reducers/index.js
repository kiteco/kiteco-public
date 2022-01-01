import { combineReducers } from 'redux'
import { routerReducer } from 'react-router-redux'

import { plugins, pluginsInfo } from './plugins'
import errors from './errors'
import account from './account'
import settings from './settings'
import system from './system'
import { reducer as license } from '../store/license'
import { reducer as modals } from '../store/modals'
import docs from './docs'
import examples from './examples'
import tooltips from './tooltips'
import search from './search'
import polling from './polling'
import { reducer as notification } from '../store/notification'
import { reducer as related_code } from '../store/related-code/related-code'
import { reducer as remotecontent } from '../store/remotecontent'
import kiteProtocol from './kite-protocol'
import scripts from './scripts'
import activeFile from './active-file'
import logs from './logs'

const reducer = combineReducers({
  routing: routerReducer,
  plugins,
  pluginsInfo,
  errors,
  account,
  settings,
  system,
  license,
  modals,
  docs,
  examples,
  tooltips,
  search,
  polling,
  notification,
  kiteProtocol,
  scripts,
  activeFile,
  logs,
  related_code,
  remotecontent,
})

export default reducer
