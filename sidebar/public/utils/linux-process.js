'use strict'

const path = require('path')
const fs = require('fs')
const os = require('os')
const {spawn} = require('child_process')
const {spawnPromise} = require('./process-utils')

const KITED_PATH = /kited/
const ELECTRON = /\/electron\//

const memInstallPath = () => {
    // first, try to launch via $HOME/.local/share/kite/kited, as this is a wrapper which handles restarts
    let homePath = path.join(os.homedir(), ".local", "share", "kite", "kited")
    if (fs.existsSync(homePath)) {
        return homePath
    }

    // then, try to launch via /opt/kite/kited
    let globalPath = "/opt/kite/kited"
    if (fs.existsSync(globalPath)) {
        return globalPath
    }

    // return the path to kited based on __dirname, e.g like $prefix/linux-unpacked/resources/app.asar/build/utils/kited
    let dir = __dirname;
    while (dir.length > 0) {
        if (path.basename(dir) === "linux-unpacked") {
            return path.join(path.dirname(dir), "kited")
        }

        dir = path.dirname(dir)
    }
    return "kited"
}

const installPath = memInstallPath()

module.exports = {
    //resolves with a pid
    isKiteRunning() {
        return spawnPromise('/bin/ps', ['-axo', 'pid,command'], {encoding: 'utf8',}, 'ps_error')
            .then(stdout => {
                const procs = stdout.split('\n')
                const kiteprocs = procs.filter(s => KITED_PATH.test(s) && !ELECTRON.test(s)) //filter out dev electron builds
                if (kiteprocs.length > 0) {
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
        let env = Object.create(process.env)
        env.KITE_SKIP_ONBOARDING = '1'
        let kited = spawn(installPath, ['--sidebar-restart'], {stdio: "ignore", env: env, detached: true})
        kited.unref();
        return Promise.resolve()
    },

    stopKite() {
        return this.isKiteRunning()
            .then((res) => {
                if (res.running) {
                    //even if multiple Kite processes, for now, just assume 0-index
                    const pid = res.processes[0].trim().split(" ")[0]
                    return spawnPromise('kill', [pid], 'kill_error')
                } else {
                    return Promise.resolve()
                }
            })
    },
}