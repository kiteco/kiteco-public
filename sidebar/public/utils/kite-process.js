'use strict'

const { platform } = require('os')

function getSupport() {
  switch(platform()) {
    case 'darwin': return require('./osx-process.js')
    case 'win32': return require('./windows-process.js')
    case 'linux': return require('./linux-process.js')
    default: return require('./mock-process.js')
  }
}

const system = getSupport()

const kiteProcess = {
  launchKite() {
    return this.stopKite()
      .then(() => this.startKite())
  },

  isKiteRunning() {
    return system.isKiteRunning()
  },

  startKite() {
    return system.startKite()
  },

  stopKite() {
    return system.stopKite()
  },
}

module.exports = {
  kiteProcess,
}