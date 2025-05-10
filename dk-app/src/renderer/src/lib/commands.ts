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
  { name: 'echo', description: 'Echo a message back' }
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

        // Call server-side command processor
        const result = await window.api.llm.processCommand({
          prompt: commandText,
          userId
        })

        // Check if result is valid
        if (result && typeof result.payload === 'string') {
          return result.payload
        } else {
          throw new Error('Invalid response from server')
        }
      } catch (serverError) {
        logger.error('Server command execution failed:', serverError)
        // Fall back to client-side implementation if available
      }
    }

    // Client-side fallback implementation
    switch (commandName) {
      case 'clear':
        return 'Chat history cleared.'

      case 'echo':
        return `Echo: ${args || 'No message provided'}`

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
