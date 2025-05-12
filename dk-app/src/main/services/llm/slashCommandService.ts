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
      logger.debug({ role, textLength: text.length }, 'Message appended from command')
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
  answer: async (params: string) => {
    try {
      logger.info('Executing /answer basic command with params:', params)

      try {
        // Try direct import first
        const documentServiceModule = await import('../documentService')
        logger.info('Successfully imported documentService in basic commands')

        // Search the RAG server with the input text as query, limit to 3 results
        logger.info('Calling documentService.searchDocuments in basicCommands')
        const searchResults = await documentServiceModule.documentService.searchDocuments(params, 3)
        logger.info('Basic command - Got search results:', JSON.stringify(searchResults))

        // Format the response with the search results and the original text
        let response = `Replied: ${params || 'No message provided'}`

        if (searchResults && searchResults.documents && searchResults.documents.length > 0) {
          logger.info(`Basic command - Found ${searchResults.documents.length} documents`)

          // Add special format marker for document results that can be parsed by the client
          response += `\n\n[DOCUMENT_RESULTS_START]`

          // Add document results in a JSON format that can be parsed on the client
          const documentResults = searchResults.documents.map((doc, index) => {
            return {
              id: index + 1,
              title: doc.title || doc.filename || `Document ${index + 1}`,
              content: doc.content || 'No content available'
            };
          });

          // Add the JSON string to the response
          response += JSON.stringify(documentResults);
          response += `[DOCUMENT_RESULTS_END]`
        } else {
          logger.info('Basic command - No documents found in search results')
          response += `\n\nNo related documents found.`
        }

        logger.info('Basic command - Returning response from /answer')
        return response
      } catch (importError) {
        logger.error('Error importing documentService in basic commands:', importError)
        return `Replied: ${params || 'No message provided'}\n\n(Document service import error: ${importError.message || 'Unknown error'})`
      }
    } catch (error) {
      // If anything fails, fall back to a basic reply
      logger.error('Error executing answer command fallback:', error)
      return `Replied: ${params || 'No message provided'}\n\n(Error: ${error.message || 'Unknown error'})`
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
  logger.info(`Processing slash command: "${prompt}" from user: ${userId}`)

  try {
    // Split command name and parameters
    let [cmdName, ...paramParts] = prompt.slice(1).split(/\s+/)
    cmdName = cmdName.toLowerCase()
    const paramStr = paramParts.join(' ')

    logger.info(`Parsed command name: "${cmdName}", parameters: "${paramStr}"`)

    // Try to load the command registry
    let result
    try {
      // Try dynamic import of the command registry
      logger.info('Attempting to import command registry')
      const { commandRegistry } = await import('./commandRegistry')
      logger.info('Command registry import successful, looking for command:', cmdName)

      const cmd = commandRegistry.get(cmdName)
      logger.info('Command registry lookup result:', cmd ? 'Command found' : 'Command not found')

      if (cmd) {
        logger.info({ cmdName, userId }, 'Executing slash command from registry')

        // Create command context
        const ctx = createCommandContext(userId)
        logger.info('Created command context for user:', userId)

        // Parse parameters if schema is provided
        let params
        if (cmd.paramsSchema) {
          logger.info('Parsing parameters with schema')
          params = cmd.paramsSchema.parse(paramStr || '')
          logger.info('Parameters parsed successfully:', params)
        } else {
          logger.info('No schema provided, using raw parameter string')
          params = paramStr || ''
        }

        // Execute the command
        logger.info(`Executing command handler for "${cmdName}"`)
        result = await cmd.handler(params, ctx)

        // Check if this is a special LLM request response
        if (result && typeof result === 'object' && result.type === 'llm_request') {
          logger.info('Received LLM request from command')

          // Return a special object that tells the client to initiate an LLM request
          return {
            passthrough: false,
            payload: result.displayResponse,
            llmRequest: {
              type: 'llm_request',
              messages: result.messages
            }
          }
        }

        logger.info(`Command handler execution complete, result length: ${result ? (typeof result === 'string' ? result.length : JSON.stringify(result).length) : 0}`)
      } else if (basicCommands[cmdName]) {
        logger.info({ cmdName, userId }, 'Executing basic slash command')
        logger.info(`Calling basic command for "${cmdName}" with parameters "${paramStr}"`)
        result = await basicCommands[cmdName](paramStr)
        logger.info(`Basic command execution complete, result length: ${result ? result.length : 0}`)
      } else {
        logger.warn({ cmdName, userId }, 'Unknown slash command')
        return {
          passthrough: false,
          payload: `Unknown command: /${cmdName}.`
        }
      }
    } catch (importError) {
      // Fallback to basic commands
      logger.warn({ error: importError }, 'Failed to import command registry, using basic commands')
      logger.error('Import error details:', importError)

      if (basicCommands[cmdName]) {
        logger.info({ cmdName, userId }, 'Executing basic slash command (fallback)')
        logger.info(`Calling fallback basic command for "${cmdName}" with parameters "${paramStr}"`)
        try {
          result = await basicCommands[cmdName](paramStr)
          logger.info(`Fallback basic command execution complete, result length: ${result ? result.length : 0}`)
        } catch (basicCommandError) {
          logger.error('Basic command execution error:', basicCommandError)
          return {
            passthrough: false,
            payload: `Error executing basic command: ${basicCommandError.message || 'Unknown error'}`
          }
        }
      } else {
        logger.warn({ cmdName, userId }, 'Unknown slash command')
        return {
          passthrough: false,
          payload: `Unknown command: /${cmdName}.`
        }
      }
    }

    logger.info({ cmd: cmdName, ms: Date.now() - start }, 'cmd:success')
    logger.info(`Command successful, returning result: "${result && result.substring(0, 50)}${result && result.length > 50 ? '...' : ''}"`)
    return { passthrough: false, payload: result }
  } catch (err) {
    const error = err as Error
    logger.error({ err: error, prompt }, 'Slash command failed')
    logger.error('Error details:', error.stack || error.message)
    return {
      passthrough: false,
      payload: `Error executing command: ${error.message}`
    }
  }
}
