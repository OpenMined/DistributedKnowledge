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
  // For packaged app, use same base directory as config
  if (process.type === 'browser') {
    const { app } = require('electron');
    return path.join(app.getPath('userData'), 'logs');
  }
  // Fallback for non-electron environment
  return path.join(process.cwd(), 'logs');
}

const logDir = getLogDirectory();

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
  logger.add(
    new transports.Console({
      format: consoleFormat
    })
  )
}

export default logger

export function createServiceLogger(serviceName: string) {
  return logger.child({ service: serviceName })
}
