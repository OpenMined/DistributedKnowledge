import { Client } from '@modelcontextprotocol/sdk/client/index.js'
import { setupMCPConfig, setupMCPTools } from './utils'
import chalk from 'chalk'
import { createServiceLogger } from '@shared/logging'

// Create logger for MCP service
const logger = createServiceLogger('mcpService')

class MCPService {
  private static instance: MCPService
  private mcpClient: Client | null = null
  private toolsByProvider: Record<string, any[]> = {}
  private initialized: boolean = false
  private initializing: boolean = false
  private initPromise: Promise<void> | null = null

  private constructor() {}

  public static getInstance(): MCPService {
    if (!MCPService.instance) {
      MCPService.instance = new MCPService()
    }
    return MCPService.instance
  }

  public async initialize(): Promise<void> {
    if (this.initialized) {
      return Promise.resolve()
    }

    if (this.initializing && this.initPromise) {
      // If initialization is already in progress, wait for it to complete
      return this.initPromise
    }

    this.initializing = true
    this.initPromise = this.initializeInternal()
    return this.initPromise
  }

  private async initializeInternal(): Promise<void> {
    try {
      logger.debug('Initializing MCP client')
      // Use 'openai' as default provider
      const defaultProvider = 'openai'
      const result = await setupMCPConfig(defaultProvider)
      // setupMCPConfig returns an object with client and tools properties
      if (result && result.client) {
        this.mcpClient = result.client
        // Cache the tools if they're available
        if (Array.isArray(result.tools) && result.tools.length > 0) {
          this.toolsByProvider[defaultProvider] = result.tools
        }
      } else {
        throw new Error('setupMCPConfig did not return a valid client instance')
      }
      this.initialized = true
      logger.debug('MCP client initialized successfully')
    } catch (error) {
      logger.error('Failed to initialize MCP client:', error)
      this.mcpClient = null
    } finally {
      this.initializing = false
    }
  }

  public async getClient(): Promise<Client | null> {
    if (!this.initialized) {
      await this.initialize()
    }
    return this.mcpClient
  }

  public async getToolsForProvider(provider: string): Promise<any[]> {
    if (!this.initialized) {
      await this.initialize()
    }

    // Return cached tools if available
    if (this.toolsByProvider[provider]) {
      return this.toolsByProvider[provider]
    }

    // If client isn't initialized, return empty array
    if (!this.mcpClient) {
      logger.debug(`Cannot get tools for provider ${provider}: MCP client is not initialized`)
      return []
    }

    try {
      logger.debug(`Setting up MCP tools for provider: ${provider}`)
      const tools = await setupMCPTools(this.mcpClient, provider)
      // Cache the tools
      this.toolsByProvider[provider] = tools
      logger.debug(`Successfully loaded ${tools.length} tools for provider ${provider}`)
      return tools
    } catch (error) {
      logger.error(`Failed to setup MCP tools for provider ${provider}:`, error)
      return []
    }
  }

  // Clear tools cache for a specific provider or all providers
  public clearToolsCache(provider?: string): void {
    if (provider) {
      delete this.toolsByProvider[provider]
      logger.debug(`Cleared tools cache for provider: ${provider}`)
    } else {
      this.toolsByProvider = {}
      logger.debug('Cleared all tools cache')
    }
  }
}

// Export singleton instance
export const mcpService = MCPService.getInstance()
