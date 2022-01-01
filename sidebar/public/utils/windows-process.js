'use strict'

const { execSync, spawn } = require('child_process')
const { spawnPromise } = require('./process-utils')
const { join } = require('path')

const KEY_BAT = `"${join(__dirname, 'read-key.bat')}"`
const FALLBACK_INSTALL_PATH = process.env.ProgramW6432
  ? join(process.env.ProgramW6432, 'Kite')
  : 'C:\\Program Files\\Kite'

const memInstallPath = () => {
  try {
    const registryPath = String(execSync(KEY_BAT)).trim()
    return () => {
      if (registryPath !== 'not found') return registryPath
      return FALLBACK_INSTALL_PATH
    }
  } catch (err) {
    console.error('error finding registry', err)
    return () => { return FALLBACK_INSTALL_PATH }
  }
}

const installPath = memInstallPath()
const KITE_EXE_PATH = join(installPath(), 'kited.exe')

module.exports = {
  isKiteRunning() {
    return spawnPromise('tasklist', 'tasklist_error')
      .then(stdout => {
        const procs = stdout.split('\n')
        const kiteprocs = procs.filter(proc => proc.indexOf("kited.exe") !== -1)
        if (kiteprocs.length > 0) {
          return {
            processes: kiteprocs,
            running: true
          }
        }
        return { running: false }
      })
  },

  startKite() {
    var env = Object.create(process.env)
    env.KITE_SKIP_ONBOARDING = '1'
    spawn(`${KITE_EXE_PATH}`, ['--sidebar-restart'], { detached: true, env: env })
    return Promise.resolve()
  },

  stopKite() {
    return this.isKiteRunning()
      .then((res) => {
        if (res.running) {
          return spawnPromise('taskkill', ['/im', 'kited.exe', '/f'], 'taskkill_err')
        } else {
          return Promise.resolve()
        }
      })
  },
}
