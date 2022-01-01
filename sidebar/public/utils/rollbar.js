'use strict'
const Rollbar = require('rollbar')
const {type, release} = require('os')
const isDev = require('./electron-is-dev')

const rollbar = new Rollbar({
  accessToken: 'XXXXXXX',
  payload: {
    environment: isDev ? 'development' : 'production',
    source: 'copilot',
    copilot_version: process.env.npm_package_version,
    os: type() + ' ' + release(),
  },
});

const sendRollbarError = (payload) => {
  if(!isDev) {
    rollbar.error(payload)
  } else {
    console.log("Dev Error:: ", payload)
  }
}

module.exports = {
  sendRollbarError
}