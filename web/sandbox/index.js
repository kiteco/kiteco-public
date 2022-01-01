'use strict'

import React from 'react'
import { render } from 'react-dom'
import { createStore, applyMiddleware, compose } from 'redux'
import thunkMiddleware from 'redux-thunk'
import { Provider } from 'react-redux'

import reducer from './reducers'

import ScriptingSandbox from './sandbox/ScriptingSandbox'

import { LOAD_CACHED_COMPLETIONS } from './reducers/cached-completions'
// we assume it's here, because we can't import conditionally
import CACHED_COMPLETIONS from './assets/completions.json'

const THEME = window.kiteTheme || 'kite-dark'

if( !/Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent) ) {
  // some hack to make the UI/UX work on WP Mobile..
  const middleware = applyMiddleware(thunkMiddleware)

  const store = createStore(
    reducer,
    compose(middleware)
  )

  // initialize store
  store.dispatch({
    type: LOAD_CACHED_COMPLETIONS,
    completions: CACHED_COMPLETIONS,
  })

  render(
    <Provider store={store}>
      <ScriptingSandbox theme={THEME} />
    </Provider>,
    document.getElementById('kite-web-sandbox')
  )
 }
