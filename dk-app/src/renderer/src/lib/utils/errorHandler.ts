// Import the entire module to avoid export naming issues
import * as ErrorUtils from '@shared/errors'
// Alias the types for easier use in this file
type ApiResponse<T = unknown> = ErrorUtils.ApiResponse<T>
type ErrorResponse = ErrorUtils.ErrorResponse
type SuccessResponse<T = unknown> = ErrorUtils.SuccessResponse<T>
const { AppError, ErrorType, isErrorResponse } = ErrorUtils
// Import browser-friendly logger
import logger from './logger'

// Interface for toast service
interface ToastService {
  show: (message: string, options?: { type?: string; duration?: number }) => void
}

/**
 * Handles API responses and shows errors in UI if needed
 *
 * @param response API response from IPC call
 * @param toast Toast service for showing error notifications
 * @param errorMessage Optional custom error message to show
 * @returns The data from the response if successful, or throws an AppError if failed
 */
export function handleApiResponse<T>(
  response: ApiResponse<T>,
  toast?: ToastService | null,
  errorMessage?: string
): T {
  if (isErrorResponse(response)) {
    // Show error message via toast if available
    if (toast) {
      toast.show(errorMessage || response.error.message || 'An error occurred', { type: 'error' })
    }

    // Throw AppError for handling in the catch block
    throw new AppError(
      response.error.message,
      response.error.type,
      response.error.code,
      response.error.details
    )
  }

  // Return the data if successful
  return response.data
}

/**
 * Wraps an async function with error handling that shows a toast on error
 *
 * @param fn Async function to wrap
 * @param toast Toast service for showing error notifications
 * @param errorMessage Optional custom error message
 * @returns A wrapped function that handles errors
 */
export function withErrorHandling<T extends (...args: any[]) => Promise<any>>(
  fn: T,
  toast?: ToastService | null,
  errorMessage?: string
): (...args: Parameters<T>) => Promise<ReturnType<T>> {
  return async (...args: Parameters<T>): Promise<ReturnType<T>> => {
    try {
      const result = await fn(...args)

      // If result is an API response, handle it
      if (result && typeof result === 'object' && 'success' in result) {
        return handleApiResponse(result, toast, errorMessage) as ReturnType<T>
      }

      return result
    } catch (error) {
      // Show error via toast if available
      if (toast) {
        const message = error instanceof Error ? error.message : errorMessage || 'An error occurred'

        toast.show(message, { type: 'error' })
      }

      // Re-throw as AppError
      throw AppError.from(error)
    }
  }
}

/**
 * Wrapper for IPC calls with consistent error handling
 *
 * @param ipcCall Function that calls IPC
 * @param toast Toast service
 * @param errorMessage Custom error message
 * @returns Result of the IPC call with error handling
 */
export async function safeIpcCall<T>(
  ipcCall: () => Promise<ApiResponse<T> | T>,
  toast?: ToastService | null,
  errorMessage?: string
): Promise<T> {
  try {
    const result = await ipcCall()

    // If result is an API response, handle it
    if (result && typeof result === 'object' && 'success' in result) {
      return handleApiResponse(result as ApiResponse<T>, toast, errorMessage)
    }

    // Otherwise assume it's already the data type we want
    return result as T
  } catch (error) {
    // Show error via toast if available
    if (toast) {
      const message = error instanceof Error ? error.message : errorMessage || 'An error occurred'

      toast.show(message, { type: 'error' })
    }

    // Re-throw the error
    throw AppError.from(error, ErrorType.IPC)
  }
}
