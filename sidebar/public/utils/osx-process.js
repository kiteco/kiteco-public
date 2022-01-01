'use strict'

const { spawnSync } = require('child_process')
const { spawnPromise } = require('./process-utils')

const KITED_PATH = /Kite\.app\/Contents\/MacOS\/(Kite\s|Kite$)/
const XCODE = /\/Xcode\//
const ELECTRON = /\/electron\//

const memInstallPath = () => {
  let installPaths = []
  return () => {
    if(installPaths.length === 0) {
      //compute paths
      installPaths = String(spawnSync('mdfind', [
        'kMDItemCFBundleIdentifier = "com.kite.Kite"',
      ]).stdout)
        .trim()
        .split('\n')
        .filter(path => !XCODE.test(path)) //filter out development paths
    }
    return installPaths[0]
  }
}

const installPath = memInstallPath()

module.exports = {
  //resolves with a pid
  isKiteRunning() {
    return spawnPromise('/bin/ps', [
      '-axo', 'pid,command',
    ], {
      encoding: 'utf8',
    }, 'ps_error')
      .then(stdout => {
        const procs = stdout.split('\n')
        const kiteprocs = procs.filter(s => KITED_PATH.test(s) 
                                            && !ELECTRON.test(s) //filter out dev electron builds
                                            && !s.includes('Kite.app/Contents/Resources')) //filter out production electron app
        if(kiteprocs.length > 0) {
          return {
            processes: kiteprocs,
            running: true,
          }
        } else {
          return {
            running: false,
          }
        }
      })
  },

  startKite() {
    return spawnPromise('open', [
      '-a', installPath(), '--args', '"--sidebar-restart"',
    ])
  },

  stopKite() {
    return this.isKiteRunning()
      .then((res) => {
        if(res.running) {
          //even if multiple Kite processes, for now, just assume 0-index
          const pid = res.processes[0].trim().split(" ")[0]
          return spawnPromise('/bin/kill', [pid], 'kill_error')
        } else {
          return Promise.resolve()
        }
      })
  },
}