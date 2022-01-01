'use strict'

const child_process = require('child_process')

module.exports = {
  //from kite-installer code
  spawnPromise(cmd, cmdArgs, cmdOptions, rejectionType) {
    const args = [cmd]

    if (cmdArgs) {
      typeof cmdArgs === 'string'
        ? rejectionType = cmdArgs
        : args.push(cmdArgs)
    }

    if (cmdOptions) {
      typeof cmdOptions === 'string'
        ? rejectionType = cmdOptions
        : args.push(cmdOptions)
    }

    return new Promise((resolve, reject) => {
      const proc = child_process.spawn(...args)
      let stdout = ''
      let stderr = ''

      proc.stdout.on('data', data => stdout += data)
      proc.stderr.on('data', data => stdout += data)

      proc.on('close', code => {
        code
          ? reject({ type: rejectionType, data: stderr })
          : resolve(stdout)
      })

      proc.on('error', _ => {
        reject({ type: rejectionType, data: stderr })
      })
    })
  },
}