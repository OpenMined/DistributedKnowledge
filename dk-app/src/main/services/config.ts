import { join, isAbsolute, dirname } from 'path'
import { readFileSync, existsSync, writeFileSync, mkdirSync } from 'fs'
import { app } from 'electron'
import * as SharedTypes from '../../shared/types'
// Alias types for easier use
type AppConfig = SharedTypes.AppConfig
type DKConfig = SharedTypes.DKConfig
type OnboardingStatus = SharedTypes.OnboardingStatus
type LLMConfig = SharedTypes.LLMConfig
import { createServiceLogger } from '../../shared/logging'
// Import LLMConfig and defaultLLMConfig for initialization
import { defaultLLMConfig } from './llm/config'
import { LLMProvider } from '@shared/llmTypes'
import { homedir } from 'os'
import { getAppPaths } from '../utils'

// Create a specific logger for config service
const logger = createServiceLogger('configService')

/**
 * Get the path to the config file
 * Uses a platform-independent location based on Electron's userData directory
 */
export function getConfigFilePath(): string {
  // Check for CONFIG_FILE environment variable first
  const customConfigPath = process.env.CONFIG_FILE || null

  if (customConfigPath) {
    // Use custom path if provided
    return isAbsolute(customConfigPath) ? customConfigPath : join(process.cwd(), customConfigPath)
  }

  // Use platform-independent location from our utility function
  const appPaths = getAppPaths()
  return appPaths.configFile
}

// Onboarding status tracking - default values will be updated in loadConfig()
const onboardingStatus: OnboardingStatus = {
  isFirstRun: false,
  currentStep: 0,
  totalSteps: 5, // Total number of steps in the wizard
  completed: false
}

// Default configuration - THIS SHOULD NEVER BE SAVED UNTIL ONBOARDING COMPLETES
export const appConfig: AppConfig = {
  serverURL: 'http://localhost:3000',
  userID: '', // Empty by default to prevent accidental saving
  private_key: '',
  public_key: '',
  database: {
    path: join(app.getPath('userData'), 'chat.db')
  },
  llm: defaultLLMConfig
}

// Validate config data structure
function validateConfig(config: any): string | null {
  if (!config) return 'Config is empty or invalid JSON'

  if (typeof config.serverURL !== 'string')
    return 'Invalid or missing serverURL field (string expected)'

  if (typeof config.userID !== 'string') return 'Invalid or missing userID field (string expected)'

  if (config.private_key !== undefined && typeof config.private_key !== 'string')
    return 'Invalid private_key field (string expected)'

  if (config.public_key !== undefined && typeof config.public_key !== 'string')
    return 'Invalid public_key field (string expected)'

  // Validate syftbox_config if present
  if (config.syftbox_config !== undefined && typeof config.syftbox_config !== 'string') {
    return 'Invalid syftbox_config field (string expected)'
  }

  // Validate dk_config if present
  if (config.dk_config !== undefined) {
    if (typeof config.dk_config !== 'object' || config.dk_config === null) {
      return 'Invalid dk_config field (object expected)'
    }

    if (typeof config.dk_config.dk !== 'string') {
      return 'Invalid or missing dk_config.dk field (string expected)'
    }

    if (typeof config.dk_config.project_path !== 'string') {
      return 'Invalid or missing dk_config.project_path field (string expected)'
    }

    if (typeof config.dk_config.http_port !== 'string') {
      return 'Invalid or missing dk_config.http_port field (string expected)'
    }
  }

  // Validate dk_api if present
  if (config.dk_api !== undefined && typeof config.dk_api !== 'string') {
    return 'Invalid dk_api field (string expected)'
  }

  // Validate database config if present
  if (config.database !== undefined) {
    if (typeof config.database !== 'object' || config.database === null) {
      return 'Invalid database field (object expected)'
    }

    if (typeof config.database.path !== 'string' || !config.database.path) {
      return 'Invalid or missing database.path field (non-empty string expected)'
    }
  }

  // Validate LLM config if present
  if (config.llm !== undefined) {
    // Check activeProvider
    if (typeof config.llm.activeProvider !== 'string') {
      return 'Invalid llm.activeProvider field (string expected)'
    }

    // Check providers object
    if (typeof config.llm.providers !== 'object' || config.llm.providers === null) {
      return 'Invalid llm.providers field (object expected)'
    }

    // Validate each provider
    for (const [providerName, providerConfig] of Object.entries(config.llm.providers)) {
      if (typeof providerConfig !== 'object' || providerConfig === null) {
        return `Invalid provider config for ${providerName} (object expected)`
      }

      // Only validate apiKey if it's not Ollama (which doesn't require an API key)
      if (providerName !== 'ollama' && typeof (providerConfig as any).apiKey !== 'string') {
        return `Invalid or missing apiKey for provider ${providerName} (string expected)`
      }

      if (typeof (providerConfig as any).defaultModel !== 'string') {
        return `Invalid or missing defaultModel for provider ${providerName} (string expected)`
      }

      if (!Array.isArray((providerConfig as any).models)) {
        return `Invalid models for provider ${providerName} (array expected)`
      }
    }
  }

  return null // Valid configuration
}

// Load configuration from config.json
export function loadConfig(): void {
  // Get the config file path
  const configPath = getConfigFilePath()

  try {
    // Check if the config file exists
    if (!existsSync(configPath)) {
      // If it doesn't exist, use default configuration but DON'T CREATE an empty config file
      logger.warn(`Config file not found at ${configPath}`)
      logger.debug('Using default configuration:', { config: appConfig })

      // Explicitly set first run status when config doesn't exist - this is CRITICAL
      onboardingStatus.isFirstRun = true
      onboardingStatus.completed = false

      logger.info(
        `Config file doesn't exist - setting first run status: isFirstRun=${onboardingStatus.isFirstRun}, Completed=${onboardingStatus.completed}`
      )
      return
    }

    // Read and parse config file
    const configFile = readFileSync(configPath, 'utf8')
    const configData = JSON.parse(configFile)

    // Validate config structure
    const validationError = validateConfig(configData)
    if (validationError) {
      logger.error(`Invalid configuration: ${validationError}`)

      // Check if we're using a custom config file specified via env var
      const usingCustomConfig =
        process.env.CONFIG_FILE !== undefined && process.env.CONFIG_FILE !== null

      if (usingCustomConfig) {
        app.exit(1) // Exit with error if custom config was specified but is invalid
      } else {
        logger.debug('Using default configuration:', { config: appConfig })
        return
      }
    }

    // Apply configuration
    Object.assign(appConfig, configData)

    // If we have dk_config, set the dk_api based on that config
    if (appConfig.dk_config) {
      appConfig.dk_api = `http://localhost:${appConfig.dk_config.http_port}`
      logger.debug(`Setting dk_api to ${appConfig.dk_api} based on dk_config.http_port`)
    }

    logger.info(`Loaded configuration from ${configPath}`)
    logger.debug('Configuration details:', { config: appConfig })

    // If syftbox_config path is specified, log it
    if (appConfig.syftbox_config) {
      logger.debug(`SyftBox configuration path set to: ${appConfig.syftbox_config}`)
    }

    // Config file exists, set onboarding status accordingly
    onboardingStatus.isFirstRun = false
    onboardingStatus.completed = true

    logger.info(
      `Config file exists - setting first run status: isFirstRun=${onboardingStatus.isFirstRun}, Completed=${onboardingStatus.completed}`
    )
  } catch (error) {
    logger.error(`Failed to load config file ${configPath}:`, error)

    // Check if we're using a custom config file specified via env var
    const usingCustomConfig =
      process.env.CONFIG_FILE !== undefined && process.env.CONFIG_FILE !== null

    if (usingCustomConfig) {
      app.exit(1) // Exit with error if custom config was specified but failed to load
    } else {
      logger.debug('Using default configuration:', { config: appConfig })
    }

    // Error reading config file, set onboarding status for first run
    onboardingStatus.isFirstRun = true
    onboardingStatus.completed = false

    logger.info(
      `Error reading config - setting first run status: isFirstRun=${onboardingStatus.isFirstRun}, Completed=${onboardingStatus.completed}`
    )
  }
}

/**
 * Get current onboarding status
 */
export function getOnboardingStatus(): OnboardingStatus {
  return { ...onboardingStatus }
}

/**
 * Set onboarding first run status
 */
export function setOnboardingFirstRun(isFirstRun: boolean): void {
  onboardingStatus.isFirstRun = isFirstRun
  logger.info(`Manually setting onboarding first run status to: ${isFirstRun}`)
}

/**
 * Set current onboarding step
 */
export function setOnboardingStep(step: number): boolean {
  if (step >= 0 && step <= onboardingStatus.totalSteps) {
    onboardingStatus.currentStep = step
    logger.debug(`Updated onboarding step to ${step}`)
    return true
  }
  return false
}

/**
 * Mark onboarding as complete
 */
export function completeOnboarding(): void {
  onboardingStatus.completed = true
  onboardingStatus.currentStep = onboardingStatus.totalSteps
  logger.info('Onboarding marked as complete')
}

/**
 * Save configuration to disk
 */
export function saveConfig(config: Partial<AppConfig>): boolean {
  try {
    // First, check if this is an explicit save from the onboarding process
    const isFromOnboarding = config.userID && config.serverURL && config.userID !== 'default-user'

    // If not from onboarding, and config.json doesn't exist yet, prevent saving
    if (!isFromOnboarding && !existsSync(getConfigFilePath())) {
      logger.warn('Prevented saving default configuration before onboarding completion')
      return false
    }

    // Merge the new config with the existing config
    Object.assign(appConfig, config)

    // Extra validation to NEVER save configurations with default or empty values
    if (!appConfig.userID || appConfig.userID === 'default-user' || appConfig.userID === '') {
      logger.warn('Cannot save config: Invalid or default userID detected')
      return false
    }

    if (!appConfig.serverURL) {
      logger.warn('Cannot save config: Missing serverURL')
      return false
    }

    // Get config file path
    const configPath = getConfigFilePath()

    // Ensure the directory exists
    const configDir = dirname(configPath)
    if (!existsSync(configDir)) {
      mkdirSync(configDir, { recursive: true })
    }

    logger.info(`Saving configuration to ${configPath} with userID=${appConfig.userID}`)

    // Write the config to disk
    writeFileSync(configPath, JSON.stringify(appConfig, null, 2), 'utf8')
    logger.info(`Configuration saved to ${configPath}`)

    // Update onboarding status since a valid config now exists
    onboardingStatus.isFirstRun = false
    onboardingStatus.completed = true
    logger.info('Config saved - updating onboarding status: isFirstRun=false, completed=true')

    return true
  } catch (error) {
    logger.error('Failed to save configuration:', error)
    return false
  }
}

/**
 * Save LLM configuration to the main config file
 */
export function saveLLMConfig(config: LLMConfig): boolean {
  logger.debug('Saving LLM configuration to main config file')
  return saveConfig({ llm: config })
}

/**
 * Get the current LLM configuration or initialize with defaults if needed
 */
export function getLLMConfig(): LLMConfig {
  if (!appConfig.llm) {
    logger.info('LLM config not found in app config, initializing with defaults')
    appConfig.llm = defaultLLMConfig
  }
  // Convert the string-based activeProvider to enum type
  const convertedConfig = {
    ...appConfig.llm,
    activeProvider: appConfig.llm.activeProvider as LLMProvider
  }

  return convertedConfig
}
