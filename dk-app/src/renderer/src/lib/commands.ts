import { writable } from 'svelte/store'
import logger from '@lib/utils/logger'

// Store to control command popup visibility
export const commandPopupVisible = writable(false)

// Store for selected command index
export const selectedCommandIndex = writable(0)

// Command type definition
export interface Command {
  name: string
  description: string
  serverSide?: boolean
}

// Basic commands - client side by default, mark server-side ones explicitly
export const commands: Command[] = [
  { name: 'clear', description: 'Clear chat history' },
  { name: 'echo', description: 'Echo a message back' },
  { name: 'answer', description: 'Search documents and reply with the input text', serverSide: true }
]

// Keep a cache of server-side commands for command popup
let serverCommands: Command[] = []

// Load commands from server if available
export async function initializeCommands(): Promise<void> {
  try {
    // Try to get commands from server
    if (window.api?.llm?.getCommands) {
      const result = await window.api.llm.getCommands()

      if (result && Array.isArray(result)) {
        serverCommands = result.map((cmd) => ({
          name: cmd.name,
          description: cmd.summary || 'No description',
          serverSide: true
        }))

        // Merge with client-side commands, prioritizing server implementations
        const serverCommandNames = new Set(serverCommands.map((cmd) => cmd.name))

        // Filter out client commands that have server implementations
        const uniqueClientCommands = commands.filter((cmd) => !serverCommandNames.has(cmd.name))

        // Update commands array with combined set
        commands.splice(0, commands.length, ...uniqueClientCommands, ...serverCommands)

        logger.debug('Loaded server commands:', serverCommands)
      }
    }
  } catch (error) {
    logger.warn('Failed to load server commands:', error)
    // Keep using client-side commands as fallback
  }
}

// Execute a command and return the result
export async function executeCommand(commandText: string): Promise<string> {
  try {
    logger.debug('Executing command:', commandText)

    // Parse the command
    const parts = commandText.trim().split(/\s+/)
    const commandName = parts[0].substring(1).toLowerCase() // Remove slash
    const args = parts.slice(1).join(' ')

    // Find the command in our registry
    const command = commands.find((cmd) => cmd.name === commandName)

    // If command not found
    if (!command) {
      return `Unknown command: /${commandName}.`
    }

    // If it's a server-side command, pass to server
    if (command.serverSide && window.api?.llm?.processCommand) {
      try {
        // Get a random user ID - in a real app this would be the actual user ID
        const userId = crypto.randomUUID()
        logger.debug(`Executing server-side command: ${commandName}, serverSide=${command.serverSide}`)

        // Call server-side command processor
        logger.debug(`Calling processCommand with "${commandText}" for user ${userId}`)
        const result = await window.api.llm.processCommand({
          prompt: commandText,
          userId
        })

        logger.debug(`Server response received:`, result)

        // Check if result is valid - the server response might be nested
        if (result) {
          // Check for LLM request special format
          if (result.llmRequest && result.llmRequest.type === 'llm_request') {
            logger.debug('Received LLM request from server, returning special response')

            // Return a special response that includes both displayable content and LLM request data
            return {
              type: 'llm_request',
              displayText: result.payload,
              messages: result.llmRequest.messages
            };
          }
          // Standard string payload response
          else if (typeof result.payload === 'string') {
            logger.debug(`Server returned valid direct payload with length: ${result.payload.length}`)
            logger.debug(`First 100 chars: ${result.payload.substring(0, 100)}`)

            return result.payload;
          }
          // Handle nested response structure (result.data.payload)
          else if (result.success === true && result.data && typeof result.data.payload === 'string') {
            logger.debug(`Server returned valid nested payload with length: ${result.data.payload.length}`)
            logger.debug(`First 100 chars: ${result.data.payload.substring(0, 100)}`)

            return result.data.payload;
          }
          // Handle other valid response formats
          else if (result.payload && typeof result.payload.payload === 'string') {
            logger.debug(`Server returned valid double-nested payload`)
            return result.payload.payload;
          }
        }

        // If we get here, the response format is not recognized
        logger.error(`Invalid server response format:`, result)
        throw new Error('Invalid response format from server')
      } catch (serverError) {
        logger.error('Server command execution failed:', serverError)
        logger.error('Error details:', serverError)
        // Fall back to client-side implementation if available
      }
    } else {
      logger.debug(`Not executing as server-side command: ${commandName}, serverSide=${command?.serverSide}, processCommand available=${!!window.api?.llm?.processCommand}`)
    }

    // Client-side fallback implementation
    switch (commandName) {
      case 'clear':
        return 'Chat history cleared.'

      case 'echo':
        return `Echo: ${args || 'No message provided'}`

      case 'answer':
        // In the client-side implementation, we don't have direct access to the document service
        // so we'll just use a simple response but mark it as being processed server-side
        return `Replied: ${args || 'No message provided'}\n\n(Processing document search on server...)`

      default:
        return `Command /${commandName} could not be executed.`
    }
  } catch (error) {
    logger.error('Error executing command:', error)
    return `Error: ${error instanceof Error ? error.message : 'Unknown error'}`
  }
}

// Show command popup
export function showCommandPopup() {
  commandPopupVisible.set(true)
  selectedCommandIndex.set(0)
  logger.debug('Command popup visibility set to true')
}

// Hide command popup
export function hideCommandPopup() {
  commandPopupVisible.set(false)
  logger.debug('Command popup visibility set to false')
}

// Filter commands by prefix
export function filterCommands(prefix: string): Command[] {
  const query = prefix.toLowerCase().substring(1) // Remove slash
  return commands.filter((cmd) => cmd.name.startsWith(query))
}
