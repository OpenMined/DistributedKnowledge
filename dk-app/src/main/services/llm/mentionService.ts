import { CommandProcessResult } from '@shared/commandTypes'
import logger from '@shared/logging'
import { httpRequest, getApiBaseUrl } from '../../utils/http'

/**
 * Regex to match user mentions in the format @Username
 */
const USER_MENTION_REGEX = /@([a-zA-Z0-9_]+)/g

/**
 * Processes a potential user mention and sends the message to mentioned users
 * @param prompt The user input to process
 * @param userId The user ID for context
 * @returns Result with the response message
 */
export async function processMentions(
  prompt: string,
  userId: string
): Promise<{ payload: string }> {
  // Start detailed workflow tracking
  logger.info(`[MENTIONS_WORKFLOW] STARTING mention processing for user ${userId}`)
  logger.info(`[MENTIONS_WORKFLOW] Input message: "${prompt}"`)

  // If input is empty, return a default message
  if (!prompt) {
    logger.info(`[MENTIONS_WORKFLOW] Empty prompt detected, returning early`)
    return { payload: 'No message content to process.' }
  }

  // Start performance timing
  const start = Date.now()
  logger.debug(`Processing potential user mentions: "${prompt}" from user: ${userId}`)

  try {
    // Extract all user mentions
    logger.info(`[MENTIONS_WORKFLOW] Extracting mentions using regex: ${USER_MENTION_REGEX}`)
    const mentions = Array.from(prompt.matchAll(USER_MENTION_REGEX))
    logger.info(
      `[MENTIONS_WORKFLOW] Raw regex matches: ${JSON.stringify(mentions.map((m) => m[0]))}`
    )

    let usernames = mentions.map((match) => match[1])
    logger.info(
      `[MENTIONS_WORKFLOW] Extracted usernames (without @ symbol): ${JSON.stringify(usernames)}`
    )

    // Get current user's name for filtering (assuming userId is the username in this context)
    const currentUsername = userId
    logger.info(`[MENTIONS_WORKFLOW] Current username for filtering: ${currentUsername}`)

    // Filter out the current user from the mentions
    const originalCount = usernames.length
    usernames = usernames.filter((username) => username !== currentUsername)

    if (originalCount !== usernames.length) {
      logger.info(
        `[MENTIONS_WORKFLOW] Filtered out current user, removed ${originalCount - usernames.length} mention(s)`
      )
    }

    // If no mentions found (or only current user was mentioned), return early
    if (usernames.length === 0) {
      logger.info(`[MENTIONS_WORKFLOW] No valid mentions found after filtering, returning early`)
      return { payload: 'No other users were mentioned in your message.' }
    }

    logger.debug(
      `Found ${usernames.length} user mentions (excluding current user): ${usernames.join(', ')}`
    )
    logger.info(
      `[MENTIONS_WORKFLOW] Found ${usernames.length} valid user mentions: ${JSON.stringify(usernames)}`
    )

    // Get the API base URL
    const apiBaseUrl = getApiBaseUrl()
    logger.info(`[MENTIONS_WORKFLOW] API base URL: ${apiBaseUrl || 'NOT CONFIGURED'}`)

    if (!apiBaseUrl) {
      logger.error('API base URL not configured')
      logger.info(`[MENTIONS_WORKFLOW] FAILED - API base URL not configured`)
      return { payload: 'Cannot send messages: API URL not configured.' }
    }

    // Send the message using the /remote/message endpoint
    try {
      const endpoint = `${apiBaseUrl}/remote/message`
      logger.info(`[MENTIONS_WORKFLOW] Sending HTTP request to endpoint: ${endpoint}`)

      // Prepare the request body with the filtered usernames (current user already removed)
      const requestBody = {
        question: prompt,
        peers: usernames // Already filtered to exclude current user
      }
      logger.info(`[MENTIONS_WORKFLOW] Request body: ${JSON.stringify(requestBody)}`)

      // Send the HTTP request
      logger.info(`[MENTIONS_WORKFLOW] Sending POST request...`)
      const response = await httpRequest(endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(requestBody)
      })

      logger.info(`[MENTIONS_WORKFLOW] Response received - Status: ${response.status}`)
      logger.info(`[MENTIONS_WORKFLOW] Response data: ${JSON.stringify(response.data)}`)
      logger.debug(`Remote message response status: ${response.status}`, response.data)

      // Format the response with the mentions
      const mentionString = usernames.map((name) => `@${name}`).join(', ')
      const successResponse = `Message sent to ${mentionString}. They will be notified about this conversation.`

      const elapsed = Date.now() - start
      logger.info(`[MENTIONS_WORKFLOW] SUCCESS - Processing completed in ${elapsed}ms`)
      logger.debug(
        `mentions:processed - Mentioned users: ${usernames.join(', ')}, completed in ${elapsed}ms`
      )
      return { payload: successResponse }
    } catch (httpError) {
      const errorMsg =
        httpError instanceof Error ? httpError.message : String(httpError) || 'Unknown HTTP error'
      logger.error(`Failed to send remote message: ${errorMsg}`)
      logger.info(`[MENTIONS_WORKFLOW] HTTP REQUEST FAILED: ${errorMsg}`)

      // Check if the error has a response property (like axios errors)
      const errorObj = httpError as any
      if (errorObj && errorObj.response) {
        logger.info(`[MENTIONS_WORKFLOW] Error response status: ${errorObj.response.status}`)
        logger.info(
          `[MENTIONS_WORKFLOW] Error response data: ${JSON.stringify(errorObj.response.data || {})}`
        )
      }

      throw new Error(`Failed to send message to mentioned users: ${errorMsg}`)
    }
  } catch (err) {
    const error = err as Error
    logger.error(`Processing user mentions failed: ${error.message} (for prompt: ${prompt})`)
    logger.error('Error details:', error.stack || error.message)
    logger.info(`[MENTIONS_WORKFLOW] ERROR - Processing failed: ${error.message}`)
    logger.info(`[MENTIONS_WORKFLOW] Error stack: ${error.stack || 'No stack trace available'}`)
    return {
      payload: `Error processing mentions: ${error.message}`
    }
  }
}
