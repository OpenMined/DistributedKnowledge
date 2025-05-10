/**
 * Standard error types and error handling utilities for the application.
 * This ensures consistent error representation and handling between main and renderer processes.
 */

// Error types enum
export enum ErrorType {
  // Generic errors
  UNKNOWN = 'unknown',
  VALIDATION = 'validation',
  NOT_FOUND = 'not_found',
  PERMISSION_DENIED = 'permission_denied',

  // Network/server related errors
  NETWORK = 'network',
  SERVER = 'server',
  TIMEOUT = 'timeout',
  UNAUTHORIZED = 'unauthorized',

  // Configuration errors
  CONFIG = 'config',

  // LLM API errors
  LLM_API = 'llm_api',
  LLM_CONFIG = 'llm_config',

  // Data errors
  DATA_SAVE = 'data_save',
  DATA_LOAD = 'data_load',

  // Command processing errors
  COMMAND_PROCESSOR = 'command_processor',
  COMMAND_NOT_FOUND = 'command_not_found',
  COMMAND_PARAM_ERROR = 'command_param_error',

  // Application errors
  APPLICATION = 'application',
  IPC = 'ipc'
}

// Standard error response structure
export interface ErrorResponse {
  success: false
  error: {
    type: ErrorType
    message: string
    code?: string | number
    details?: Record<string, unknown>
  }
}

// Standard success response structure
export interface SuccessResponse<T = unknown> {
  success: true
  data: T
}

// Union type for all responses
export type ApiResponse<T = unknown> = SuccessResponse<T> | ErrorResponse

// App error class for consistent error handling
export class AppError extends Error {
  type: ErrorType
  code?: string | number
  details?: Record<string, unknown>

  constructor(
    message: string,
    type: ErrorType = ErrorType.UNKNOWN,
    code?: string | number,
    details?: Record<string, unknown>
  ) {
    super(message)
    this.name = 'AppError'
    this.type = type
    this.code = code
    this.details = details

    // Capture stack trace
    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, AppError)
    }
  }

  // Convert to standard error response
  toResponse(): ErrorResponse {
    return {
      success: false,
      error: {
        type: this.type,
        message: this.message,
        code: this.code,
        details: this.details
      }
    }
  }

  // Create from any error type
  static from(error: unknown, defaultType: ErrorType = ErrorType.UNKNOWN): AppError {
    if (error instanceof AppError) {
      return error
    }

    if (error instanceof Error) {
      return new AppError(error.message, defaultType)
    }

    return new AppError(
      typeof error === 'string' ? error : 'An unknown error occurred',
      defaultType
    )
  }
}

// Create success response
export function createSuccessResponse<T>(data: T): SuccessResponse<T> {
  return {
    success: true,
    data
  }
}

// Create error response from any error
export function createErrorResponse(
  error: unknown,
  defaultType: ErrorType = ErrorType.UNKNOWN
): ErrorResponse {
  return AppError.from(error, defaultType).toResponse()
}

// Error handling utility for async functions (can be used with Promise.catch)
export function handleError(
  error: unknown,
  defaultType: ErrorType = ErrorType.UNKNOWN
): ErrorResponse {
  return createErrorResponse(error, defaultType)
}

// Type guard to check if a response is successful
export function isSuccessResponse<T>(response: ApiResponse<T>): response is SuccessResponse<T> {
  return response.success === true
}

// Type guard to check if a response is an error
export function isErrorResponse(response: ApiResponse): response is ErrorResponse {
  return response.success === false
}
