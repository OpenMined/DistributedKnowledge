import { createLogger, format, transports } from 'winston'
import path from 'path'
import fs from 'fs'

const LOG_LEVELS = {
  error: 0,
  warn: 1,
  info: 2,
  http: 3,
  verbose: 4,
  debug: 5,
  silly: 6
}

const isDevelopment = process.env.NODE_ENV !== 'production'

// Determine logs directory location
function getLogDirectory() {
  // For packaged app, use the centralized logging directory
  if (process.type === 'browser') {
    try {
      // Avoid circular dependency by direct electron app usage
      const { app } = require('electron')
      const userData = app.getPath('userData')
      const configDir = path.join(userData, 'config')
      const basePath = path.dirname(configDir)
      return path.join(basePath, 'logs')
    } catch (error) {
      console.error('Failed to get logs directory:', error)
    }

    // Fallback to old behavior if central config fails
    const { app } = require('electron')
    return path.join(app.getPath('userData'), 'logs')
  }
  // Fallback for non-electron environment
  return path.join(process.cwd(), 'logs')
}

const logDir = getLogDirectory()

// Create logs directory if it doesn't exist
if (!fs.existsSync(logDir)) {
  fs.mkdirSync(logDir, { recursive: true })
}

const logFormat = format.combine(
  format.timestamp({ format: 'YYYY-MM-DD HH:mm:ss' }),
  format.errors({ stack: true }),
  format.splat(),
  format.json()
)

const consoleFormat = format.combine(
  format.colorize(),
  format.timestamp({ format: 'YYYY-MM-DD HH:mm:ss' }),
  format.printf(({ timestamp, level, message, ...metadata }) => {
    let metaStr = ''
    if (Object.keys(metadata).length > 0 && metadata.stack !== undefined) {
      metaStr = `\n${metadata.stack}`
    } else if (Object.keys(metadata).length > 0) {
      metaStr = JSON.stringify(metadata, null, 2)
    }
    return `[${timestamp}] ${level}: ${message}${metaStr}`
  })
)

const logger = createLogger({
  level: 'info',
  levels: LOG_LEVELS,
  format: logFormat,
  defaultMeta: { service: 'dk' },
  transports: [
    new transports.File({
      filename: path.join(logDir, 'error.log'),
      level: 'error'
    }),
    new transports.File({
      filename: path.join(logDir, 'combined.log'),
      maxsize: 5242880, // 5MB
      maxFiles: 5
    })
  ]
})

// Add console transport in development mode
if (isDevelopment) {
  const consoleTransport = new transports.Console({
    format: consoleFormat,
    handleExceptions: true
  })

  // Add error handling for EPIPE and other write errors
  consoleTransport.on('error', (error: any) => {
    // Silently handle EPIPE errors that occur when the console stream is closed
    if (error.code === 'EPIPE') {
      return
    }
    // Log other errors to stderr directly
    process.stderr.write(`Winston console transport error: ${error.message}\n`)
  })

  logger.add(consoleTransport)
}

export default logger

export function createServiceLogger(serviceName: string) {
  return logger.child({ service: serviceName })
}
