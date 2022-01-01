'use strict';
const path = require('path');
const os = require('os')
const winston = require('winston');

const logFormat = winston.format.printf(({ level, message, label, timestamp }) => {
  return `${timestamp} [${label}] ${level}: ${message}`;
});

const logFilePath = os.platform() !== 'win32'
  ? path.join(os.homedir(), '.kite', 'logs', 'copilot.log')
  : path.join(os.homedir(), 'AppData', 'Local', 'Kite', 'logs', 'copilot.log')

const logger = winston.createLogger({
  level: 'info',
  exitOnError: false,
  format: winston.format.combine(
    winston.format.label({ label: 'copilot' }),
    winston.format.timestamp(),
    logFormat
  ),
  transports: [
    new winston.transports.File({
      filename: logFilePath,
      level: 'info',
      maxsize: 1000000,
      maxFiles: 3,
      maxRetries: 5,
      handleExceptions: true,
    })
  ]
});

module.exports = {logger, logFilePath}