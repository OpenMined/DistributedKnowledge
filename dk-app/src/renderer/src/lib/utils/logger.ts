/**
 * Browser-friendly logger implementation for the renderer process
 * This avoids importing Node.js specific modules like winston
 */

// Log levels
export enum LogLevel {
  ERROR = 'error',
  WARN = 'warn',
  INFO = 'info',
  DEBUG = 'debug'
}

// Default log level
const DEFAULT_LOG_LEVEL = LogLevel.INFO

// Logger interface
interface Logger {
  error(message: string, ...meta: any[]): void
  warn(message: string, ...meta: any[]): void
  info(message: string, ...meta: any[]): void
  debug(message: string, ...meta: any[]): void
}

// Simple logger implementation for browser
class BrowserLogger implements Logger {
  private level: LogLevel
  private context: string

  constructor(context: string = 'app', level: LogLevel = DEFAULT_LOG_LEVEL) {
    this.level = level
    this.context = context
  }

  private shouldLog(level: LogLevel): boolean {
    const levels = {
      [LogLevel.ERROR]: 0,
      [LogLevel.WARN]: 1,
      [LogLevel.INFO]: 2,
      [LogLevel.DEBUG]: 3
    }

    return levels[level] <= levels[this.level]
  }

  private formatMessage(level: LogLevel, message: string, ...meta: any[]): string {
    let formatted = `[${new Date().toISOString()}] [${level.toUpperCase()}] [${this.context}] ${message}`

    if (meta && meta.length > 0) {
      try {
        formatted += ` ${meta.map((m) => (typeof m === 'object' ? JSON.stringify(m) : m)).join(' ')}`
      } catch (e) {
        formatted += ' [Error formatting meta data]'
      }
    }

    return formatted
  }

  error(message: string, ...meta: any[]): void {
    if (this.shouldLog(LogLevel.ERROR)) {
      console.error(this.formatMessage(LogLevel.ERROR, message, ...meta))
    }
  }

  warn(message: string, ...meta: any[]): void {
    if (this.shouldLog(LogLevel.WARN)) {
      console.warn(this.formatMessage(LogLevel.WARN, message, ...meta))
    }
  }

  info(message: string, ...meta: any[]): void {
    if (this.shouldLog(LogLevel.INFO)) {
      console.info(this.formatMessage(LogLevel.INFO, message, ...meta))
    }
  }

  debug(message: string, ...meta: any[]): void {
    if (this.shouldLog(LogLevel.DEBUG)) {
      console.debug(this.formatMessage(LogLevel.DEBUG, message, ...meta))
    }
  }
}

// Create a default logger instance
const defaultLogger = new BrowserLogger()

// Create a service-specific logger
export function createLogger(context: string): Logger {
  return new BrowserLogger(context)
}

// Export the default logger as the main interface
export default defaultLogger
