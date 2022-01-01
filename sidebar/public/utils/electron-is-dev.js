'use strict'

/**
 * Replaces and slightly modifies the originally included npm package
 * Why? It was causing an issue with 'build_electron.sh force' on local
 * machines due to its testing for 'electron' in the execpath
 */

const notKiteElectronPath = () => {
  return !/[\\/]osx[\\/]electron[\\/]Kite[.]app[\\/]/.test(process.execPath) &&
    !/[\\/]electron[\\/]/.test(process.execPath)
}

const getFromEnv = () => {
  if ('ELECTRON_IS_DEV' in process.env) {
    return parseInt(process.env.ELECTRON_IS_DEV, 10) === 1
  }

  return false
}

module.exports = {
  isDev: getFromEnv() || process.defaultApp || /[\\/]electron-prebuilt[\\/]/.test(process.execPath) || notKiteElectronPath(),
  isLocalhostDev: process.env.REACT_APP_ENV && process.env.REACT_APP_ENV === 'development'
}