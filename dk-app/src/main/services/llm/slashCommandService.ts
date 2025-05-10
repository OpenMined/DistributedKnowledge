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

  try {
    // Split command name and parameters
    let [cmdName, ...paramParts] = prompt.slice(1).split(/\s+/)
    cmdName = cmdName.toLowerCase()
    const paramStr = paramParts.join(' ')

    // Try to load the command registry
    let result
    try {
      // Try dynamic import of the command registry
      const { commandRegistry } = await import('./commandRegistry')
      const cmd = commandRegistry.get(cmdName)

      if (cmd) {
        logger.info({ cmdName, userId }, 'Executing slash command from registry')

        // Create command context
        const ctx = createCommandContext(userId)

        // Parse parameters if schema is provided
        const params = cmd.paramsSchema
          ? cmd.paramsSchema.parse(paramStr || '')
          : ((paramStr || '') as unknown)

        // Execute the command
        result = await cmd.handler(params, ctx)
      } else if (basicCommands[cmdName]) {
        logger.info({ cmdName, userId }, 'Executing basic slash command')
        result = await basicCommands[cmdName](paramStr)
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

      if (basicCommands[cmdName]) {
        logger.info({ cmdName, userId }, 'Executing basic slash command (fallback)')
        result = await basicCommands[cmdName](paramStr)
      } else {
        logger.warn({ cmdName, userId }, 'Unknown slash command')
        return {
          passthrough: false,
          payload: `Unknown command: /${cmdName}.`
        }
      }
    }

    logger.info({ cmd: cmdName, ms: Date.now() - start }, 'cmd:success')
    return { passthrough: false, payload: result }
  } catch (err) {
    const error = err as Error
    logger.error({ err: error, prompt }, 'Slash command failed')
    return {
      passthrough: false,
      payload: `Error executing command: ${error.message}`
    }
  }
}
