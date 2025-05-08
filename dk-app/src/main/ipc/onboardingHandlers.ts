import { ipcMain, app } from 'electron'
import { join, isAbsolute, dirname } from 'path'
import { Channels } from '../../shared/constants'
import {
  getOnboardingStatus,
  setOnboardingStep,
  completeOnboarding,
  saveConfig,
  getConfigFilePath,
  setOnboardingFirstRun
} from '../services/config'
import { generateKeys } from '../services/client'
import { loadOrCreateKeys, getAppPaths } from '../utils'
import { AppConfig, OnboardingConfig } from '../../shared/types'
import logger from '../../shared/logging'
import { LLMProvider } from '@shared/llmTypes'
import { homedir } from 'os'
import { mkdirSync, existsSync, writeFileSync } from 'fs'

/**
 * Register IPC handlers for onboarding functionality
 */
export function registerOnboardingHandlers(): void {
  // Log handler initialization
  logger.info('Initializing onboarding handlers')

  // Get onboarding status
  ipcMain.handle(Channels.GetOnboardingStatus, () => {
    try {
      logger.info('HANDLER: GetOnboardingStatus called')
      const status = getOnboardingStatus()

      // Check if config file exists - this is the most critical check
      const configPath = getConfigFilePath()
      logger.info(`HANDLER: Checking config file at path: ${configPath}`)

      // Explicitly check existence of config file without depending on other logic
      const configExists = existsSync(configPath)
      logger.info(`HANDLER: Config file exists: ${configExists}`)

      // If config file exists, ensure onboarding is not shown
      if (configExists) {
        status.isFirstRun = false
        status.completed = true
        logger.info(
          'HANDLER: Config file found, setting onboarding status to isFirstRun=false, completed=true'
        )
      } else {
        // Config doesn't exist - force onboarding
        status.isFirstRun = true
        status.completed = false

        // Persist these values for consistency across the app
        setOnboardingFirstRun(true)

        logger.info(
          'HANDLER: *** CONFIG FILE NOT FOUND *** Forcing onboarding with isFirstRun=true, completed=false'
        )

        // Log an extra diagnostic line
        logger.info('HANDLER: App should show onboarding wizard since configExists=false')
      }

      // Construct the response with explicit configExists flag
      const response = {
        success: true,
        status,
        configExists // Critical flag for renderer
      }

      logger.info(`HANDLER: Sending response to renderer: ${JSON.stringify(response)}`)

      return response
    } catch (error) {
      logger.error('HANDLER: Failed to get onboarding status:', error)

      // Still include configExists=false in error responses
      return {
        success: false,
        error: 'Failed to get onboarding status',
        configExists: false // Assuming error means config doesn't exist
      }
    }
  })

  // Set onboarding step
  ipcMain.handle(Channels.SetOnboardingStep, (_, step: number) => {
    try {
      const updated = setOnboardingStep(step)
      return {
        success: updated,
        currentStep: updated ? step : getOnboardingStatus().currentStep
      }
    } catch (error) {
      logger.error('Failed to set onboarding step:', error)
      return {
        success: false,
        error: 'Failed to set onboarding step'
      }
    }
  })

  // Complete onboarding
  ipcMain.handle(Channels.CompleteOnboarding, () => {
    try {
      logger.info('Completing onboarding process...')
      completeOnboarding()

      // Start background processes when onboarding is completed
      try {
        // Dynamically import all necessary modules
        Promise.all([
          import('../index').then((module) => module.startExternalProcesses),
          import('../services/appService').then((module) => module.loadSyftboxConfig),
          import('../services/trackerService').then((module) => module.trackerService),
          import('../services/documentService').then((module) => module.documentService)
        ])
          .then(([startExternalProcesses, loadSyftboxConfig, trackerService, documentService]) => {
            logger.info('Onboarding completed - reinitializing services')

            // First reload SyftBox configuration and verify it worked
            const configLoaded = loadSyftboxConfig()
            logger.info(
              `SyftBox configuration ${configLoaded ? 'successfully loaded' : 'failed to load'}`
            )

            // Only start processes if config was loaded successfully
            if (configLoaded) {
              // Then start external processes
              startExternalProcesses()
              logger.info('External processes started')
            } else {
              logger.error('External processes not started due to missing SyftBox configuration')
            }

            // Stop existing services before restarting
            logger.info('Stopping background services before restart')

            // Stop tracker service if running
            if (trackerService.stopTrackerScan) {
              trackerService.stopTrackerScan()
            }

            // Stop document service if running
            if (documentService.stopDocumentDataFetch) {
              documentService.stopDocumentDataFetch()
            }

            // Increased delay to ensure clean restart and full config loading
            setTimeout(() => {
              if (configLoaded) {
                logger.info('Starting background services with new configuration')

                // Start tracker service
                if (trackerService.startTrackerScan) {
                  trackerService.startTrackerScan()
                  logger.info('Tracker service started successfully with new configuration')
                } else {
                  logger.error('Unable to start tracker service - method not found')
                }

                // Start document data service
                if (documentService.startDocumentDataFetch) {
                  documentService.startDocumentDataFetch()
                  logger.info('Document data service started successfully with new configuration')
                } else {
                  logger.error('Unable to start document data service - method not found')
                }
              } else {
                logger.error('Cannot start background services: SyftBox configuration not loaded')
              }
            }, 2000)
          })
          .catch((err) => {
            logger.error('Failed to import and initialize services:', err)
          })
      } catch (processError) {
        logger.error('Error starting external processes after onboarding:', processError)
      }

      return {
        success: true
      }
    } catch (error) {
      logger.error('Failed to complete onboarding:', error)
      return {
        success: false,
        error: 'Failed to complete onboarding'
      }
    }
  })

  // Save onboarding config
  ipcMain.handle(Channels.SaveOnboardingConfig, (_, config: OnboardingConfig) => {
    try {
      // Validate minimal required fields and ensure userID is not the default
      if (
        !config.serverURL ||
        !config.userID ||
        config.userID === 'default-user' ||
        config.userID === ''
      ) {
        return {
          success: false,
          error: 'Valid Server URL and User ID are required'
        }
      }

      // Get platform-specific paths
      const appPaths = getAppPaths()
      const defaultSyftboxConfigPath = appPaths.syftboxConfig
      const syftboxDir = dirname(defaultSyftboxConfigPath)

      // Check if SyftBox is configured
      if (!existsSync(syftboxDir)) {
        logger.warn(
          `SyftBox config directory does not exist: ${syftboxDir}. SyftBox features will not work.`
        )
      } else if (!existsSync(defaultSyftboxConfigPath)) {
        logger.warn(
          `SyftBox config file does not exist: ${defaultSyftboxConfigPath}. SyftBox features will not work.`
        )
      } else {
        logger.info(`Found existing SyftBox config at: ${defaultSyftboxConfigPath}`)
      }

      // Get platform-specific paths
      const projectPath = appPaths.dataDir
      const dkPath = appPaths.dkBinary

      // Create project directories if they don't exist
      try {
        // Create config directory
        if (!existsSync(appPaths.configDir)) {
          mkdirSync(appPaths.configDir, { recursive: true })
          logger.info(`Created config directory: ${appPaths.configDir}`)
        }

        // Create project_path directory
        if (!existsSync(projectPath)) {
          mkdirSync(projectPath, { recursive: true })
          logger.info(`Created project data directory: ${projectPath}`)

          // Create model_config.json based on selected LLM provider
          try {
            const modelConfigPath = join(projectPath, 'model_config.json')
            let modelConfig: any = {
              parameters: {
                temperature: 0.7,
                max_tokens: 1000
              }
            }

            // Set provider-specific configuration
            if (config.llm?.activeProvider === 'ollama') {
              modelConfig.provider = 'ollama'
              modelConfig.model = config.llm.providers['ollama']?.defaultModel || 'gemma3:4b'
              modelConfig.base_url = 'http://localhost:11434/api/generate'
            } else if (config.llm?.activeProvider === 'anthropic') {
              modelConfig.provider = 'anthropic'
              modelConfig.api_key = config.llm.providers['anthropic']?.apiKey || ''
              modelConfig.model =
                config.llm.providers['anthropic']?.defaultModel || 'claude-3-opus-20240229'
            } else if (config.llm?.activeProvider === 'openai') {
              modelConfig.provider = 'openai'
              modelConfig.api_key = config.llm.providers['openai']?.apiKey || ''
              modelConfig.model = config.llm.providers['openai']?.defaultModel || 'gpt-4-turbo'
            } else if (config.llm?.activeProvider === 'gemini') {
              modelConfig.provider = 'gemini'
              modelConfig.api_key = config.llm.providers['gemini']?.apiKey || ''
              modelConfig.model = config.llm.providers['gemini']?.defaultModel || 'gemini-1.5-pro'
            }

            // Write the config file
            writeFileSync(modelConfigPath, JSON.stringify(modelConfig, null, 2))
            logger.info(
              `Created model_config.json with ${modelConfig.provider} provider configuration`
            )
          } catch (configError) {
            logger.error(`Failed to create model_config.json: ${configError}`)
          }
        }

        // Create directory for dk binary
        const dkDir = dirname(dkPath)
        if (!existsSync(dkDir)) {
          mkdirSync(dkDir, { recursive: true })
          logger.info(`Created DK binary directory: ${dkDir}`)
        }
      } catch (dirError) {
        logger.error(`Failed to create project directories: ${dirError}`)
      }

      // Convert OnboardingConfig to AppConfig for saving
      const appConfig: Partial<AppConfig> = {
        serverURL: config.serverURL,
        userID: config.userID,
        // Add default syftbox_config path (ensure it's an absolute path)
        syftbox_config: isAbsolute(defaultSyftboxConfigPath)
          ? defaultSyftboxConfigPath
          : join(process.cwd(), defaultSyftboxConfigPath),
        // Add default dk_config with proper values
        dk_config: {
          dk: dkPath,
          project_path: projectPath,
          http_port: '4232'
        }
      }

      // Add auth keys if provided
      if (config.private_key) {
        appConfig.private_key = config.private_key
      }

      if (config.public_key) {
        appConfig.public_key = config.public_key
      }

      // Add LLM config if provided
      if (config.llm) {
        // Ensure we correctly convert any string providers to LLMProvider enum
        const convertedLLMConfig = {
          activeProvider: config.llm.activeProvider as LLMProvider,
          providers: {} as {
            [key in LLMProvider]?: {
              apiKey: string
              baseUrl?: string
              defaultModel: string
              models: string[]
            }
          }
        }

        // Process each provider and ensure apiKey is required
        for (const [key, value] of Object.entries(config.llm.providers)) {
          if (value) {
            convertedLLMConfig.providers[key as LLMProvider] = {
              apiKey: value.apiKey || '', // Ensure apiKey is always a string
              baseUrl: value.baseUrl,
              defaultModel: value.defaultModel,
              models: value.models
            }
          }
        }

        appConfig.llm = convertedLLMConfig
      }

      // Save the config
      const saved = saveConfig(appConfig)
      return {
        success: saved
      }
    } catch (error) {
      logger.error('Failed to save onboarding config:', error)
      return {
        success: false,
        error: 'Failed to save onboarding config'
      }
    }
  })

  // Generate authentication keys
  ipcMain.handle(Channels.GenerateAuthKeys, async () => {
    try {
      // Define paths for key storage
      const keysDir = join(app.getPath('userData'), 'keys')
      const privateKeyPath = join(keysDir, 'private')
      const publicKeyPath = join(keysDir, 'public')

      // Generate keys and store them in files
      const keys = await generateKeys()

      // Create key directory and write the keys to files
      await loadOrCreateKeys(privateKeyPath, publicKeyPath)

      logger.info('Authentication keys generated successfully')

      return {
        success: true,
        keys: {
          private_key: privateKeyPath,
          public_key: publicKeyPath
        }
      }
    } catch (error) {
      logger.error('Failed to generate authentication keys:', error)
      return {
        success: false,
        error: 'Failed to generate authentication keys'
      }
    }
  })

  // Pull the nomic-embed-text model for Ollama
  ipcMain.handle(Channels.PullNomicEmbedModel, async () => {
    try {
      logger.info('Starting to pull nomic-embed-text model for Ollama')

      // First, log what we're about to do
      logger.info('Attempting to pull nomic-embed-text model - checking API first')

      // Try to ping Ollama API to confirm it's running
      const pingResponse = await fetch('http://localhost:11434/api/tags').catch((err) => {
        logger.error('Failed to connect to Ollama API:', err)
        throw new Error('Ollama is not running. Please start Ollama first.')
      })

      if (!pingResponse.ok) {
        logger.error(`Ollama API check failed with status: ${pingResponse.status}`)
        throw new Error('Ollama API returned an error. Please ensure Ollama is running correctly.')
      }

      // Attempt to pull the model using Ollama API
      logger.info('Initiating model pull request to Ollama API...')
      const response = await fetch('http://localhost:11434/api/pull', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          name: 'nomic-embed-text' // Remove ':latest' to be more compatible
        })
      })

      if (!response.ok) {
        const error = await response.text()
        logger.error(`Failed to pull nomic-embed-text model: ${error}`)
        return {
          success: false,
          error: `Failed to pull model: ${error}`
        }
      }

      logger.info('Successfully started pulling nomic-embed-text model')
      return {
        success: true,
        message: 'Started pulling nomic-embed-text model. This may take a few minutes to complete.'
      }
    } catch (error) {
      logger.error('Error pulling nomic-embed-text model:', error)
      const errorMessage = error instanceof Error ? error.message : String(error)
      return {
        success: false,
        error: `Failed to pull model: ${errorMessage}`
      }
    }
  })

  // Check if external services are installed
  ipcMain.handle(Channels.CheckExternalServices, async () => {
    try {
      // Check if Ollama is installed and if the required model is available
      let ollamaInstalled = false
      let nomicEmbedModelInstalled = false
      try {
        // Try to call the Ollama API to see if it's running
        const controller = new AbortController()
        const timeoutId = setTimeout(() => controller.abort(), 2000)

        const response = await fetch('http://localhost:11434/api/tags', {
          signal: controller.signal
        }).catch(() => null)

        clearTimeout(timeoutId)

        ollamaInstalled = response?.ok === true
        logger.info(
          `Ollama service check: ${ollamaInstalled ? 'Running' : 'Not Running/Not Installed'}`
        )

        // If Ollama is running, check if nomic-embed-text model is installed
        if (ollamaInstalled && response) {
          try {
            const data = await response.json()
            // Log the structure of the data for debugging
            logger.info(`Ollama API response structure: ${JSON.stringify(Object.keys(data))}`)

            // Be flexible about where the models might be in the response
            const modelsList = data.models || data.tags || data.objects || []

            // Log what we found
            logger.info(`Found models list with ${modelsList.length} entries`)

            if (modelsList.length > 0) {
              // Check for nomic-embed-text model in the list
              logger.info(
                `Available Ollama models: ${JSON.stringify(modelsList.map((m: any) => m.name || m))}`
              )

              // More flexible checking for the model name
              nomicEmbedModelInstalled = modelsList.some((model: any) => {
                // Get the model name as string
                const modelName = model.name || model.toString()

                // Log for debugging
                logger.info(`Checking model: ${modelName}`)

                // Check if it contains 'nomic-embed-text' in any form
                return (
                  modelName.toLowerCase().includes('nomic-embed-text') ||
                  (modelName.toLowerCase().includes('nomic') &&
                    modelName.toLowerCase().includes('embed'))
                )
              })

              // Always log the result
              logger.info(`nomic-embed-text model detection result: ${nomicEmbedModelInstalled}`)
              logger.info(
                `nomic-embed-text model check: ${nomicEmbedModelInstalled ? 'Installed' : 'Not Installed'}`
              )
            }
          } catch (modelError) {
            logger.warn('Failed to check nomic-embed-text model:', modelError)
          }
        }
      } catch (error) {
        logger.warn('Failed to check Ollama installation:', error)
      }

      // Check if Syftbox is installed by checking if config.json exists
      let syftboxInstalled = false
      try {
        const appPaths = getAppPaths()
        const syftboxConfigPath = appPaths.syftboxConfig

        // Check if the path exists
        const exists = existsSync(syftboxConfigPath)
        logger.info(
          `SyftBox config check: ${syftboxConfigPath} - ${exists ? 'Exists' : 'Not Found'}`
        )

        syftboxInstalled = exists

        if (syftboxInstalled) {
          logger.info('Syftbox installation verified: config.json found')
        } else {
          logger.warn('Syftbox installation not detected: config.json not found')
        }
      } catch (error) {
        logger.warn('Failed to check Syftbox installation:', error)
        logger.error('Error details:', error)
      }

      return {
        success: true,
        status: {
          ollama: ollamaInstalled,
          syftbox: syftboxInstalled,
          nomicEmbedModel: nomicEmbedModelInstalled
        }
      }
    } catch (error) {
      logger.error('Failed to check external services:', error)
      return {
        success: false,
        error: 'Failed to check external services',
        status: {
          ollama: false,
          syftbox: false,
          nomicEmbedModel: false
        }
      }
    }
  })
}
