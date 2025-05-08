import { IpcMainInvokeEvent } from 'electron'
import logger from '@shared/logging'
import * as ErrorUtils from '@shared/errors'

// Alias the types and functions for easier use
type ApiResponse<T = unknown> = ErrorUtils.ApiResponse<T>
const { AppError, ErrorType, createSuccessResponse, createErrorResponse } = ErrorUtils

/**
 * Wraps an IPC handler function with standardized error handling.
 * This ensures all IPC handlers return responses in a consistent format.
 *
 * @param handler The original handler function
 * @param errorType The default error type to use if an unknown error occurs
 * @returns A wrapped handler with error handling
 */
export function wrapIpcHandler<T>(
  handler: (event: IpcMainInvokeEvent, ...args: any[]) => Promise<T>,
  errorType: (typeof ErrorType)[keyof typeof ErrorType] = ErrorType.IPC
): (event: IpcMainInvokeEvent, ...args: any[]) => Promise<ApiResponse<T>> {
  return async (event: IpcMainInvokeEvent, ...args: any[]): Promise<ApiResponse<T>> => {
    try {
      // Call the original handler
      const result = await handler(event, ...args)

      // Return a standardized success response
      return createSuccessResponse(result)
    } catch (error) {
      // Log the error
      logger.error(`IPC handler error (${errorType}):`, error)

      // Return a standardized error response
      return createErrorResponse(error, errorType)
    }
  }
}

/**
 * Wrap multiple IPC handlers with error handling
 *
 * @param handlers Object mapping channel names to handler functions
 * @param errorTypes Object mapping channel names to error types
 * @returns Object with the same keys but wrapped handlers
 */
export function wrapIpcHandlers<T extends Record<string, any>>(
  handlers: T,
  errorTypes: Partial<Record<keyof T, (typeof ErrorType)[keyof typeof ErrorType]>> = {}
): T {
  const wrappedHandlers = { ...handlers }

  for (const [channel, handler] of Object.entries(handlers)) {
    if (typeof handler === 'function') {
      const errorType = (errorTypes as any)[channel] || ErrorType.IPC
      ;(wrappedHandlers as any)[channel] = wrapIpcHandler(handler, errorType)
    }
  }

  return wrappedHandlers
}

/**
 * Helper to try an operation that might fail and return a standardized response
 * Useful for non-IPC operations that still need standardized error handling
 *
 * @param operation The operation to try
 * @param errorType The default error type to use if an unknown error occurs
 * @returns A standardized API response
 */
export async function tryOperation<T>(
  operation: () => Promise<T>,
  errorType: (typeof ErrorType)[keyof typeof ErrorType] = ErrorType.UNKNOWN
): Promise<ApiResponse<T>> {
  try {
    const result = await operation()
    return createSuccessResponse(result)
  } catch (error) {
    logger.error(`Operation error (${errorType}):`, error)
    return createErrorResponse(error, errorType)
  }
}
