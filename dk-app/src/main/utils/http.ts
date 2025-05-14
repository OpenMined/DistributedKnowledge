import { createServiceLogger } from '../../shared/logging'
import * as https from 'https'
import * as http from 'http'
import { URL } from 'url'
import { appConfig } from '../services/config'

const logger = createServiceLogger('httpUtil')

/**
 * Makes an HTTP/HTTPS request to the specified URL
 * @param url The URL to request
 * @param options Optional HTTP request options
 * @returns Promise with the response body
 */
export async function httpRequest(
  url: string,
  options: {
    method?: string
    headers?: Record<string, string>
    body?: string | Buffer
    timeout?: number
  } = {}
): Promise<{ data: any; status: number }> {
  // Default timeout of 10 seconds
  const timeout = options.timeout || 10000
  const method = options.method || 'GET'

  // Generate unique request ID for tracking
  const requestId = Math.random().toString(36).substring(2, 12)
  logger.debug(`[HTTP_REQUEST:${requestId}] Making ${method} request to ${url}`)

  // For @mention requests, add extra logging
  const isMentionRequest = url.includes('/remote/message')
  if (isMentionRequest) {
    logger.info(`[MENTIONS_HTTP:${requestId}] Starting HTTP request to send remote message`)
    logger.info(`[MENTIONS_HTTP:${requestId}] URL: ${url}`)
    logger.info(`[MENTIONS_HTTP:${requestId}] Method: ${method}`)
    logger.info(`[MENTIONS_HTTP:${requestId}] Headers: ${JSON.stringify(options.headers || {})}`)
    logger.info(`[MENTIONS_HTTP:${requestId}] Body: ${options.body || 'empty'}`)
    logger.info(`[MENTIONS_HTTP:${requestId}] Timeout: ${timeout}ms`)
  }

  return new Promise((resolve, reject) => {
    try {
      const parsedUrl = new URL(url)
      const isHttps = parsedUrl.protocol === 'https:'

      const requestOptions = {
        hostname: parsedUrl.hostname,
        port: parsedUrl.port || (isHttps ? 443 : 80),
        path: `${parsedUrl.pathname}${parsedUrl.search}`,
        method,
        headers: {
          'Content-Type': 'application/json',
          ...options.headers
        },
        timeout
      }

      if (isMentionRequest) {
        logger.info(`[MENTIONS_HTTP:${requestId}] Parsed URL details:`)
        logger.info(`[MENTIONS_HTTP:${requestId}] - Hostname: ${parsedUrl.hostname}`)
        logger.info(`[MENTIONS_HTTP:${requestId}] - Port: ${requestOptions.port}`)
        logger.info(`[MENTIONS_HTTP:${requestId}] - Path: ${requestOptions.path}`)
        logger.info(`[MENTIONS_HTTP:${requestId}] - Protocol: ${isHttps ? 'HTTPS' : 'HTTP'}`)
      }

      const requestStartTime = Date.now()
      if (isMentionRequest) {
        logger.info(`[MENTIONS_HTTP:${requestId}] Creating request at ${new Date().toISOString()}`)
      }

      const clientRequest = (isHttps ? https : http).request(requestOptions, (res) => {
        if (isMentionRequest) {
          logger.info(`[MENTIONS_HTTP:${requestId}] Response received - Status: ${res.statusCode}`)
          logger.info(
            `[MENTIONS_HTTP:${requestId}] Response headers: ${JSON.stringify(res.headers)}`
          )
        }

        let data = ''

        res.on('data', (chunk) => {
          data += chunk
          if (isMentionRequest) {
            logger.info(
              `[MENTIONS_HTTP:${requestId}] Received data chunk: ${chunk.toString().substring(0, 100)}${chunk.toString().length > 100 ? '...' : ''}`
            )
          }
        })

        res.on('end', () => {
          const requestDuration = Date.now() - requestStartTime
          if (isMentionRequest) {
            logger.info(`[MENTIONS_HTTP:${requestId}] Response complete after ${requestDuration}ms`)
            logger.info(`[MENTIONS_HTTP:${requestId}] Raw response: ${data}`)
          }

          try {
            // Special handling for DELETE operations that return 204 No Content
            if (method === 'DELETE' && res.statusCode === 204) {
              logger.debug(
                `[HTTP_REQUEST:${requestId}] DELETE request returned 204 No Content - success with empty body`
              )
              if (isMentionRequest) {
                logger.info(
                  `[MENTIONS_HTTP:${requestId}] DELETE request succeeded with 204 No Content`
                )
              }
              resolve({ data: { success: true }, status: 204 })
              return
            }

            let parsedData = data
            // Try to parse as JSON if the response is not empty
            if (data && data.trim()) {
              try {
                parsedData = JSON.parse(data)
                if (isMentionRequest) {
                  logger.info(
                    `[MENTIONS_HTTP:${requestId}] Successfully parsed JSON response: ${JSON.stringify(parsedData)}`
                  )
                }
              } catch (parseError) {
                const errorMessage =
                  parseError instanceof Error ? parseError.message : String(parseError)
                logger.warn(
                  `[HTTP_REQUEST:${requestId}] Failed to parse response as JSON: ${errorMessage}`
                )
                if (isMentionRequest) {
                  logger.info(
                    `[MENTIONS_HTTP:${requestId}] Failed to parse as JSON: ${errorMessage}`
                  )
                  logger.info(`[MENTIONS_HTTP:${requestId}] Using raw string response`)
                }
                // Keep the raw data if parsing fails
              }
            } else if (
              res.statusCode !== undefined &&
              res.statusCode >= 200 &&
              res.statusCode < 300
            ) {
              // For empty response with success status code, return a success object
              logger.debug(
                `[HTTP_REQUEST:${requestId}] Request returned success status (${res.statusCode}) with empty body`
              )
              if (isMentionRequest) {
                logger.info(
                  `[MENTIONS_HTTP:${requestId}] Empty response with success status ${res.statusCode}`
                )
                logger.info(`[MENTIONS_HTTP:${requestId}] Using default success object`)
              }
              // Use 'as any' to bypass the type error for now - we're handling a special case
              parsedData = { success: true } as any
            }

            // Ensure statusCode always has a value
            const statusCode = res.statusCode !== undefined ? res.statusCode : 200
            if (isMentionRequest) {
              logger.info(
                `[MENTIONS_HTTP:${requestId}] Request completed successfully in ${requestDuration}ms`
              )
              logger.info(`[MENTIONS_HTTP:${requestId}] Final status code: ${statusCode}`)
            }
            resolve({ data: parsedData, status: statusCode })
          } catch (error) {
            const errorMessage = error instanceof Error ? error.message : String(error)
            logger.error(`[HTTP_REQUEST:${requestId}] Error processing response: ${errorMessage}`)
            if (isMentionRequest) {
              logger.error(
                `[MENTIONS_HTTP:${requestId}] Error processing response: ${errorMessage}`
              )
              logger.error(
                `[MENTIONS_HTTP:${requestId}] Stack trace: ${error instanceof Error ? error.stack : 'No stack trace'}`
              )
            }
            reject(error)
          }
        })
      })

      clientRequest.on('error', (error) => {
        const errorMessage = error instanceof Error ? error.message : String(error)
        logger.error(`[HTTP_REQUEST:${requestId}] Request error: ${errorMessage}`)
        if (isMentionRequest) {
          logger.error(`[MENTIONS_HTTP:${requestId}] Network error: ${errorMessage}`)
          logger.error(
            `[MENTIONS_HTTP:${requestId}] Stack trace: ${error instanceof Error ? error.stack : 'No stack trace'}`
          )

          // Additional network diagnostics for mentions
          logger.info(
            `[MENTIONS_HTTP:${requestId}] Connection details - Hostname: ${requestOptions.hostname}, Port: ${requestOptions.port}`
          )

          // Check if it's a common connection error
          if (errorMessage.includes('ECONNREFUSED')) {
            logger.info(
              `[MENTIONS_HTTP:${requestId}] Connection refused - Check if the server is running on ${requestOptions.hostname}:${requestOptions.port}`
            )
          } else if (errorMessage.includes('ETIMEDOUT')) {
            logger.info(
              `[MENTIONS_HTTP:${requestId}] Connection timed out - Server might be slow to respond`
            )
          } else if (errorMessage.includes('ENOTFOUND')) {
            logger.info(
              `[MENTIONS_HTTP:${requestId}] Host not found - Check hostname configuration`
            )
          }
        }
        reject(error)
      })

      clientRequest.on('timeout', () => {
        clientRequest.destroy()
        const timeoutDuration = Date.now() - requestStartTime
        logger.error(
          `[HTTP_REQUEST:${requestId}] Request timed out after ${timeout}ms (actual duration: ${timeoutDuration}ms)`
        )
        if (isMentionRequest) {
          logger.error(`[MENTIONS_HTTP:${requestId}] Request timed out after ${timeout}ms`)
          logger.error(
            `[MENTIONS_HTTP:${requestId}] Actual duration before timeout: ${timeoutDuration}ms`
          )
        }
        reject(new Error(`Request timed out after ${timeout}ms`))
      })

      // Write body data if provided
      if (options.body) {
        clientRequest.write(options.body)
        if (isMentionRequest) {
          logger.info(`[MENTIONS_HTTP:${requestId}] Wrote request body data`)
        }
      }

      clientRequest.end()
      if (isMentionRequest) {
        logger.info(`[MENTIONS_HTTP:${requestId}] Request sent at ${new Date().toISOString()}`)
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error)
      logger.error(`Failed to make HTTP request: ${errorMessage}`)
      reject(error)
    }
  })
}

/**
 * Gets the base URL for the API from the configuration
 * @returns The API base URL or null if not configured
 */
export function getApiBaseUrl(): string | null {
  // Check if dk_api is configured
  if (appConfig.dk_api) {
    // Ensure the URL doesn't end with a slash
    return appConfig.dk_api.endsWith('/') ? appConfig.dk_api.slice(0, -1) : appConfig.dk_api
  }

  return null
}
