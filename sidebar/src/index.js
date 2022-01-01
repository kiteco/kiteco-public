import 'babel-polyfill'
import React from 'react';
import { render } from 'react-dom'
import { Provider } from 'react-redux'
import { Route } from 'react-router-dom'
import {
  addErrorHandling,
} from './utils/browser-window'

import {
  ConnectedRouter as Router
} from 'react-router-redux'

import App from './containers/App'
import { store, history } from './utils/store'

import './assets/variables.css'
import './assets/colors.css'
import './assets/index.css'
const {ipcRenderer, remote} = window.require("electron")

const currentWindow = remote.getCurrentWindow();
const osName = currentWindow.kite_os;
if (osName) {
  document.body.classList.add(osName);
}

/** Uncaught Error handling */
addErrorHandling(store, ipcRenderer)

render(
  <Provider store={store}>
    <Router history={history}>
      <Route path="/" component={App} />
    </Router>
  </Provider>,
  document.getElementById('root')
)
