// Types and interfaces for slash command functionality
import { z, ZodSchema } from 'zod'

// Command context provided to handlers
export interface CommandCtx {
  userId: string
  now: Date
  appendMessage: (role: 'user' | 'assistant', text: string) => void
  // Can be extended with more adapters (db, http) later
}

// Slash command metadata with optional parameter schema
export interface SlashCommandMeta<TParams = unknown> {
  /** canonical command name without leading slash */
  name: string
  /** short description for the popup */
  summary: string
  /** optional parameter specification (free-form string after the first space) */
  paramsSchema?: ZodSchema<TParams>
  /** execute the command and return a string (markdown allowed) */
  handler: (params: TParams, ctx: CommandCtx) => Promise<string>
}

// Frontend-specific interface for command popup items
export interface CommandPopupItem {
  name: string // "/help"
  summary: string // "Show available commands"
  highlightRange: [number, number] // for bold formatting
}

// Command processing result for the middleware
export interface CommandProcessResult {
  passthrough: boolean
  payload: string
  llmRequest?: {
    type: string
    messages: any[]
  }
}

// Command registry interface
export interface CommandRegistry {
  register: (command: SlashCommandMeta) => void
  get: (name: string) => SlashCommandMeta | undefined
  getAll: () => SlashCommandMeta[]
}
