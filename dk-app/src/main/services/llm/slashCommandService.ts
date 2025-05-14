import { CommandCtx, CommandProcessResult } from '@shared/commandTypes'
import logger from '@shared/logging'

/**
 * Creates a command context for slash command execution
 */
export function createCommandContext(userId: string): CommandCtx {
  return {
    userId,
    now: new Date(),
    appendMessage: (role, text) => {
      // This will be replaced with actual implementation when needed
      logger.debug(`Message appended from command - role: ${role}, length: ${text.length}`)
    }
  }
}

/**
 * Basic command handlers for fallback
 */
const basicCommands = {
  clear: async () => {
    return 'Chat history cleared.'
  },
  echo: async (params: string) => {
    return `Echo: ${params || 'No message provided'}`
  },
  rag: async (params: string) => {
    try {
      logger.debug('Executing /rag basic command with params:', params)

      try {
        // Try direct import first
        const documentServiceModule = await import('../documentService')
        logger.debug('Successfully imported documentService in basic commands')

        // Search the RAG server with the input text as query, limit to 3 results
        logger.debug('Calling documentService.searchDocuments in basicCommands')
        const searchResults = await documentServiceModule.documentService.searchDocuments(params, 3)
        logger.debug('Basic command - Got search results:', JSON.stringify(searchResults))

        // Format the response with the search results and the original text
        let response = `Replied: ${params || 'No message provided'}`

        if (searchResults && searchResults.documents && searchResults.documents.length > 0) {
          logger.debug(`Basic command - Found ${searchResults.documents.length} documents`)

          // Add special format marker for document results that can be parsed by the client
          response += `\n\n[DOCUMENT_RESULTS_START]`

          // Add document results in a JSON format that can be parsed on the client
          const documentResults = searchResults.documents.map((doc, index) => {
            return {
              id: index + 1,
              title: doc.file || `Document ${index + 1}`,
              content: doc.content || 'No content available'
            }
          })

          // Add the JSON string to the response
          response += JSON.stringify(documentResults)
          response += `[DOCUMENT_RESULTS_END]`
        } else {
          logger.debug('Basic command - No documents found in search results')
          response += `\n\nNo related documents found.`
        }

        logger.debug('Basic command - Returning response from /rag')
        return response
      } catch (importError) {
        logger.error(
          `Error importing documentService in basic commands: ${importError instanceof Error ? importError.message : String(importError)}`
        )
        return `Replied: ${params || 'No message provided'}\n\n(Document service import error: ${importError instanceof Error ? importError.message : String(importError) || 'Unknown error'})`
      }
    } catch (error) {
      // If anything fails, fall back to a basic reply
      logger.error(
        `Error executing rag command fallback: ${error instanceof Error ? error.message : String(error)}`
      )
      return `Replied: ${params || 'No message provided'}\n\n(Error: ${error instanceof Error ? error.message : String(error) || 'Unknown error'})`
    }
  }
}

/**
 * Processes a potential slash command
 * @param prompt The user input to process
 * @param userId The user ID for context
 * @returns Result with passthrough flag and payload
 */
export async function processSlashCommand(
  prompt: string,
  userId: string
): Promise<CommandProcessResult> {
  // If not a slash command, just pass through
  if (!prompt?.startsWith('/')) {
    return { passthrough: true, payload: prompt }
  }

  // Start performance timing
  const start = Date.now()
  logger.debug(`Processing slash command: "${prompt}" from user: ${userId}`)

  try {
    // Split command name and parameters
    let [cmdName, ...paramParts] = prompt.slice(1).split(/\s+/)
    cmdName = cmdName.toLowerCase()
    const paramStr = paramParts.join(' ')

    logger.debug(`Parsed command name: "${cmdName}", parameters: "${paramStr}"`)

    // Try to load the command registry
    let result
    try {
      // Try dynamic import of the command registry
      logger.debug('Attempting to import command registry')
      const { commandRegistry } = await import('./commandRegistry')
      logger.debug('Command registry import successful, looking for command:', cmdName)

      const cmd = commandRegistry.get(cmdName)
      logger.debug('Command registry lookup result:', cmd ? 'Command found' : 'Command not found')

      if (cmd) {
        logger.debug(`Executing slash command from registry: ${cmdName} for user: ${userId}`)

        // Create command context
        const ctx = createCommandContext(userId)
        logger.debug('Created command context for user:', userId)

        // Parse parameters if schema is provided
        let params
        if (cmd.paramsSchema) {
          logger.debug('Parsing parameters with schema')
          params = cmd.paramsSchema.parse(paramStr || '')
          logger.debug('Parameters parsed successfully:', params)
        } else {
          logger.debug('No schema provided, using raw parameter string')
          params = paramStr || ''
        }

        // Execute the command
        logger.debug(`Executing command handler for "${cmdName}"`)
        result = await cmd.handler(params, ctx)

        // Check if this is a special LLM request response
        // Cast result to any to handle the dynamic properties
        const resultAsAny = result as any
        if (result && typeof result === 'object' && resultAsAny.type === 'llm_request') {
          logger.debug('Received LLM request from command')

          // Return a special object that tells the client to initiate an LLM request
          return {
            passthrough: false,
            payload: resultAsAny.displayResponse,
            llmRequest: {
              type: 'llm_request',
              messages: resultAsAny.messages
            }
          }
        }

        logger.debug(
          `Command handler execution complete, result length: ${result ? (typeof result === 'string' ? result.length : JSON.stringify(result).length) : 0}`
        )
      } else if (
        cmdName in basicCommands &&
        typeof basicCommands[cmdName as keyof typeof basicCommands] === 'function'
      ) {
        logger.debug(`Executing basic slash command: ${cmdName} for user: ${userId}`)
        logger.debug(`Calling basic command for "${cmdName}" with parameters "${paramStr}"`)
        result = await (basicCommands[cmdName as keyof typeof basicCommands] as Function)(paramStr)
        logger.debug(
          `Basic command execution complete, result length: ${result ? result.length : 0}`
        )
      } else {
        logger.debug(`Unknown slash command: ${cmdName} for user: ${userId}`)
        return {
          passthrough: false,
          payload: `Unknown command: /${cmdName}.`
        }
      }
    } catch (importError) {
      // Fallback to basic commands
      logger.debug(
        `Failed to import command registry, using basic commands: ${importError instanceof Error ? importError.message : String(importError)}`
      )
      logger.error('Import error details:', importError)

      if (
        cmdName in basicCommands &&
        typeof basicCommands[cmdName as keyof typeof basicCommands] === 'function'
      ) {
        logger.debug(`Executing basic slash command (fallback): ${cmdName} for user: ${userId}`)
        logger.debug(
          `Calling fallback basic command for "${cmdName}" with parameters "${paramStr}"`
        )
        try {
          result = await (basicCommands[cmdName as keyof typeof basicCommands] as Function)(
            paramStr
          )
          logger.debug(
            `Fallback basic command execution complete, result length: ${result ? result.length : 0}`
          )
        } catch (basicCommandError) {
          const error = basicCommandError as Error
          logger.error('Basic command execution error:', error)
          return {
            passthrough: false,
            payload: `Error executing basic command: ${error.message || 'Unknown error'}`
          }
        }
      } else {
        logger.debug(`Unknown slash command: ${cmdName} for user: ${userId}`)
        return {
          passthrough: false,
          payload: `Unknown command: /${cmdName}.`
        }
      }
    }

    logger.debug(`cmd:success - Command ${cmdName} completed in ${Date.now() - start}ms`)
    logger.debug(
      `Command successful, returning result: "${typeof result === 'string' ? result.substring(0, 50) + (result.length > 50 ? '...' : '') : result}"`
    )
    return { passthrough: false, payload: result?.toString() || '' }
  } catch (err) {
    const error = err as Error
    logger.error(`Slash command failed: ${error.message}, for prompt: ${prompt}`)
    logger.error('Error details:', error.stack || error.message)
    return {
      passthrough: false,
      payload: `Error executing command: ${error.message}`
    }
  }
}
