import { ipcMain } from 'electron'
import fs from 'fs'
import path from 'path'
import { MCPChannels } from '../../shared/channels'
import { getAppPaths } from '../utils'
import { createServiceLogger } from '../../shared/logging'

// Create a specific logger for MCP handlers
const logger = createServiceLogger('mcpHandlers')

/**
 * Get the path to the MCP config file
 */
function getMCPConfigFilePath(): string {
  const appPaths = getAppPaths()
  // MCP config is stored at the base path (not configDir)
  const mcpConfigPath = path.join(appPaths.basePath || '', 'mcpconfig.json')
  logger.debug(`Using MCP config path: ${mcpConfigPath}`)
  return mcpConfigPath
}

/**
 * Load MCP configuration from mcpconfig.json
 */
function loadMCPConfig() {
  const configPath = getMCPConfigFilePath()
  logger.debug(`Loading MCP configuration from ${configPath}`)

  try {
    // Check if the config file exists
    if (!fs.existsSync(configPath)) {
      logger.debug(`MCP config file not found at ${configPath}`)

      // Return default configuration with empty servers
      return {
        mcpServers: {}
      }
    }

    // Read and parse config file
    const configFile = fs.readFileSync(configPath, 'utf8')
    const configData = JSON.parse(configFile)

    logger.debug(`Loaded MCP configuration from ${configPath}`)
    return configData
  } catch (error) {
    logger.error(`Failed to load MCP config file ${configPath}:`, error)

    // Return default configuration on error
    return {
      mcpServers: {}
    }
  }
}

/**
 * Save MCP configuration to mcpconfig.json
 */
function saveMCPConfig(config: any): boolean {
  try {
    // Ensure the configuration has the correct structure
    if (!config || typeof config !== 'object' || !config.mcpServers) {
      logger.error('Invalid MCP configuration format')
      return false
    }

    const configPath = getMCPConfigFilePath()

    // Ensure the directory exists
    const configDir = path.dirname(configPath)
    if (!fs.existsSync(configDir)) {
      fs.mkdirSync(configDir, { recursive: true })
    }

    // Write the config to disk
    fs.writeFileSync(configPath, JSON.stringify(config, null, 2), 'utf8')
    logger.debug(`MCP Configuration saved to ${configPath}`)

    return true
  } catch (error) {
    logger.error('Failed to save MCP configuration:', error)
    return false
  }
}

/**
 * Register IPC handlers for MCP configuration
 */
export function registerMCPHandlers(): void {
  // Log handler initialization
  logger.debug('Initializing MCP configuration handlers')

  // Get MCP configuration
  ipcMain.handle(MCPChannels.GetConfig, () => {
    try {
      return loadMCPConfig()
    } catch (error) {
      logger.error('Failed to get MCP configuration:', error)
      return { mcpServers: {} }
    }
  })

  // Save MCP configuration
  ipcMain.handle(MCPChannels.SaveConfig, (_, config) => {
    try {
      return saveMCPConfig(config)
    } catch (error) {
      logger.error('Failed to save MCP configuration:', error)
      return false
    }
  })
}
