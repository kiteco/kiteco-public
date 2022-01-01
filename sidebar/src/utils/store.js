import { createStore, applyMiddleware, compose } from 'redux'
import thunkMiddleware from 'redux-thunk'
import { composeWithDevTools } from 'redux-devtools-extension'
import { routerMiddleware } from 'react-router-redux'
import createHistory from 'history/createMemoryHistory'

import { activefile, autosearch, related_code } from './sockets'
import { getNotificationMiddleware } from '../store/license'
import reducer from '../reducers'

// create history object
const history = createHistory()
const routerReduxMiddleware = routerMiddleware(history)
// only include redux devtools chrome extension with dev builds
const envCompose = process.env.NODE_ENV !== "production" ? composeWithDevTools : compose
// router middleware
const store = createStore(
  reducer,
  envCompose(applyMiddleware(
    thunkMiddleware,
    routerReduxMiddleware,
    activefile.middleware,
    autosearch.middleware,
    related_code.middleware,
    getNotificationMiddleware(),
  ))
)

export {
  store,
  history,
}
