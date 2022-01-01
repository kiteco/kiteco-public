'use strict';

const { readFileSync, appendFileSync, readdirSync, unlinkSync } = require('fs')
const { resolve, join } = require('path')

const distPath = resolve(__dirname, '..', 'dist')

const jsFileRegex = /^([\w-]+)\.js/

const filenames = readdirSync(distPath)

const nameArrs = filenames.filter(filename => jsFileRegex.test(filename))
                        .map(filename => jsFileRegex.exec(filename))

nameArrs.forEach(nameArr => {
  const bundlePath = join(distPath, nameArr[0])
  let toAppend = ''
  const digitRegex = new RegExp(`^\\d\\.${nameArr[0]}$`)
  const digitFilenames = filenames.filter(filename => digitRegex.test(filename))
  digitFilenames.forEach(filename => {
    const filepath = join(distPath, filename)
    const contents = readFileSync(filepath)
    toAppend = `${toAppend}\n${contents}`
    unlinkSync(filepath)
  })
  appendFileSync(bundlePath, toAppend)
})

console.log('Successfully concatenated js files!')
process.exit(0)
