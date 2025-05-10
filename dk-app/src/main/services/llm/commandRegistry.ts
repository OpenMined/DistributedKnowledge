import { z } from 'zod'
import { SlashCommandMeta, CommandRegistry } from '@shared/commandTypes'
import logger from '@shared/logging'

// Create the command registry
class SlashCommandRegistryImpl implements CommandRegistry {
  private commands = new Map<string, SlashCommandMeta>()

  register(command: SlashCommandMeta): void {
    if (this.commands.has(command.name)) {
      logger.warn({ cmd: command.name }, 'Command already exists, overwriting')
    }
    this.commands.set(command.name, command)
    logger.info({ cmd: command.name }, 'Registered slash command')
  }

  get(name: string): SlashCommandMeta | undefined {
    return this.commands.get(name)
  }

  getAll(): SlashCommandMeta[] {
    return Array.from(this.commands.values())
  }
}

// Singleton instance
export const commandRegistry = new SlashCommandRegistryImpl()

// Register the core commands

// Help command
commandRegistry.register({
  name: 'help',
  summary: 'List available slash commands',
  handler: async (_, ctx) => {
    const cmds = commandRegistry
      .getAll()
      .map((c) => `• **/${c.name}** — ${c.summary}`)
      .join('\n')
    return `Here are the available commands:\n${cmds}`
  }
})

// Clear command to clear the chat history
commandRegistry.register({
  name: 'clear',
  summary: 'Clear the chat history',
  handler: async (_, ctx) => {
    // The actual clearing will be handled by the client
    // We just return a message to be shown
    return 'Chat history cleared.'
  }
})

// Version command to display app version
commandRegistry.register({
  name: 'version',
  summary: 'Show application version',
  handler: async (_, ctx) => {
    // In a real implementation, this would get the actual version
    // For now, just return a placeholder
    return `Distributed Knowledge App v1.0.0`
  }
})

// Example command with parameters using zod
commandRegistry.register({
  name: 'echo',
  summary: 'Echo a message back',
  paramsSchema: z.string().min(1, 'Please provide a message to echo'),
  handler: async (params, ctx) => {
    return `Echo: ${params}`
  }
})
