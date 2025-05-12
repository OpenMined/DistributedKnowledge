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
      return
    }

    if (this.initializing) {
      // If initialization is already in progress, wait for it to complete
      return this.initPromise
    }

    this.initializing = true
    this.initPromise = this.initializeInternal()
    return this.initPromise
  }

  private async initializeInternal(): Promise<void> {
    try {
      logger.info('Initializing MCP client')
      this.mcpClient = await setupMCPConfig()
      this.initialized = true
      logger.info('MCP client initialized successfully')
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
      logger.warn(`Cannot get tools for provider ${provider}: MCP client is not initialized`)
      return []
    }

    try {
      logger.info(`Setting up MCP tools for provider: ${provider}`)
      const tools = await setupMCPTools(this.mcpClient, provider)
      // Cache the tools
      this.toolsByProvider[provider] = tools
      logger.info(`Successfully loaded ${tools.length} tools for provider ${provider}`)
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
      logger.info(`Cleared tools cache for provider: ${provider}`)
    } else {
      this.toolsByProvider = {}
      logger.info('Cleared all tools cache')
    }
  }
}

// Export singleton instance
export const mcpService = MCPService.getInstance()