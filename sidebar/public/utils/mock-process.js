'use strict'

/**
 * Dummy implementation for unsupported OSs.
 */

module.exports = {
  isKiteRunning() {
    return new Promise((resolve, reject) => {
      resolve({isRunning:false})
    })
  },

  startKite() {
    return new Promise((resolve, reject) => {
      resolve('MOCK')
    })
  },

  stopKite() {
    return new Promise((resolve, reject) => {
      resolve('MOCK')
    })
  },
}