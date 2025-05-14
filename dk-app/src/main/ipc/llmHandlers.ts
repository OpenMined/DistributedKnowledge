import { ipcMain } from 'electron'
import { LLMService } from '@services/llm'
import { httpRequest, getApiBaseUrl } from '../utils/http'
import { URL } from 'url'
import * as LLMTypes from '@shared/llmTypes'
// Alias types for easier use
type LLMProvider = LLMTypes.LLMProvider
type ChatCompletionRequest = LLMTypes.ChatCompletionRequest
type StreamingChunk = LLMTypes.StreamingChunk
type ChatCompletionResponse = LLMTypes.ChatCompletionResponse
type ProviderConfig = LLMTypes.ProviderConfig
// LLMConfig is imported from SharedTypes
import { appConfig, getLLMConfig, saveLLMConfig } from '@services/config'
import logger from '@shared/logging'
import { LLMChannels } from '@shared/channels'
import * as SharedTypes from '@shared/types'
type AIMessage = SharedTypes.AIMessage
import {
  getAIChatHistory,
  saveAIChatHistory,
  clearAIChatHistory
} from '@services/llm/aiChatService'
import { wrapIpcHandler } from '@utils/ipcErrorHandler'
import { AppError, ErrorType } from '@shared/errors'
// Import slash command and mention processing services
import { processSlashCommand } from '../services/llm/slashCommandService'
import { processMentions } from '../services/llm/mentionService'

// Create a singleton instance of the LLM service
const llmService = new LLMService()

// Initialize LLM service with config
function initLLMService(): void {
  // Get LLM config from main config.json (will initialize with defaults if needed)
  const config = getLLMConfig()
  logger.debug('Using LLM config from app config.json:', config)

  logger.debug('Initializing LLM providers from config:', config)

  // Initialize providers from config
  Object.entries(config.providers).forEach(([providerName, providerConfig]) => {
    try {
      if (providerConfig) {
        // Special cases:
        // 1. Ollama doesn't require an API key
        // 2. Don't initialize Anthropic if its API key is set to "ollama" (invalid placeholder)
        const typedConfig = providerConfig as ProviderConfig
        if (
          providerName === 'ollama' ||
          (providerName !== 'anthropic' && typedConfig.apiKey) ||
          (providerName === 'anthropic' && typedConfig.apiKey && typedConfig.apiKey !== 'ollama')
        ) {
          logger.debug(`Initializing provider ${providerName} with config:`, typedConfig)
          llmService.initProvider(providerName as LLMProvider, typedConfig)
        } else {
          logger.debug(`Skipping provider ${providerName} due to missing or invalid API key`)
        }
      }
    } catch (err) {
      logger.error(`Failed to initialize provider ${providerName}:`, err)
    }
  })

  // Set active provider if it's been initialized
  if (llmService.getAvailableProviders().includes(config.activeProvider as LLMProvider)) {
    logger.debug(`Setting active provider to ${config.activeProvider}`)
    llmService.setActiveProvider(config.activeProvider as LLMProvider)
  } else if (llmService.getAvailableProviders().length > 0) {
    // If the configured active provider isn't available, use the first available one
    const firstProvider = llmService.getAvailableProviders()[0]
    logger.debug(
      `Configured active provider not available. Using first available: ${firstProvider}`
    )
    llmService.setActiveProvider(firstProvider)

    // Update the config
    config.activeProvider = firstProvider
    saveLLMConfig(config)
  }

  logger.debug('Available providers after initialization:', llmService.getAvailableProviders())
  logger.debug('Active provider after initialization:', llmService.getActiveProvider())
}

// Register IPC handlers
export function registerLLMHandlers(): void {
  // Initialize the service
  initLLMService()

  // Get available providers
  ipcMain.handle(LLMChannels.GetProviders, async () => {
    return llmService.getAvailableProviders()
  })

  // Get active provider
  ipcMain.handle(LLMChannels.GetActiveProvider, async () => {
    return llmService.getActiveProvider()
  })

  // Set active provider
  ipcMain.handle(LLMChannels.SetActiveProvider, async (_, provider: LLMProvider) => {
    // Update config
    const config = getLLMConfig()
    config.activeProvider = provider

    // Set active provider
    llmService.setActiveProvider(provider)

    // Reinitialize provider with existing config
    if (config.providers[provider]) {
      logger.debug(`Reinitializing provider ${provider} after setting as active`)
      llmService.initProvider(provider, config.providers[provider])
    }

    // Save config
    saveLLMConfig(config)

    return true
  })

  // Get available models for the active provider
  ipcMain.handle(LLMChannels.GetModels, async () => {
    return await llmService.getModels()
  })

  // Get available models for a specific provider
  ipcMain.handle(LLMChannels.GetModelsForProvider, async (_, provider: LLMProvider) => {
    return await llmService.getModelsForProvider(provider)
  })

  // Send message to LLM
  ipcMain.handle(LLMChannels.SendMessage, async (_, request: ChatCompletionRequest) => {
    return await llmService.sendMessage(request)
  })

  // Process slash command - new handler for slash commands
  ipcMain.handle(
    LLMChannels.ProcessCommand,
    wrapIpcHandler(async (_, request: { prompt: string; userId: string }) => {
      const { prompt, userId } = request
      try {
        return await processSlashCommand(prompt, userId)
      } catch (error) {
        logger.error('Error processing slash command:', error)
        return {
          passthrough: false,
          payload: `Error processing command: ${error instanceof Error ? error.message : 'Unknown error'}`
        }
      }
    }, ErrorType.COMMAND_PROCESSOR)
  )

  // Process user mentions - handler for user mentions like @User
  ipcMain.handle(
    LLMChannels.ProcessMentions,
    wrapIpcHandler(async (_, request: { prompt: string; userId: string }) => {
      const { prompt, userId } = request
      logger.info(`[MENTIONS_IPC] RECEIVED mention processing request from renderer`)
      logger.info(`[MENTIONS_IPC] Prompt: "${prompt}", UserId: "${userId}"`)

      try {
        logger.info(`[MENTIONS_IPC] Calling processMentions function...`)
        const startTime = Date.now()
        const result = await processMentions(prompt, userId)
        const elapsedTime = Date.now() - startTime

        logger.info(`[MENTIONS_IPC] processMentions completed in ${elapsedTime}ms`)
        logger.info(
          `[MENTIONS_IPC] Result type: ${typeof result}, Value: ${JSON.stringify(result)}`
        )

        // Ensure we always return a proper formatted result
        if (!result) {
          logger.info(`[MENTIONS_IPC] Result is null or undefined, using default success message`)
          return { payload: 'Mention processed successfully' }
        } else if (typeof result === 'string') {
          logger.info(`[MENTIONS_IPC] Result is string: "${result}"`)
          return { payload: result }
        } else if (typeof result === 'object') {
          if (result.payload !== undefined) {
            logger.info(`[MENTIONS_IPC] Result has payload property: "${result.payload}"`)
            return { payload: result.payload }
          } else {
            logger.info(
              `[MENTIONS_IPC] Result is object without payload: ${JSON.stringify(result)}`
            )
            // Try to find any string property we can use
            const resultObj = result as Record<string, any>
            for (const key in resultObj) {
              if (typeof resultObj[key] === 'string') {
                logger.info(
                  `[MENTIONS_IPC] Using property "${key}" as payload: "${resultObj[key]}"`
                )
                return { payload: resultObj[key] }
              }
            }
            logger.info(
              `[MENTIONS_IPC] No suitable string property found, using default success message`
            )
            return { payload: 'Mention processed successfully' }
          }
        }

        logger.info(`[MENTIONS_IPC] Using default fallback success message`)
        // Default fallback
        return { payload: 'Mention processed successfully' }
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : 'Unknown error'
        logger.error(`[MENTIONS_IPC] ERROR: ${errorMessage}`)
        logger.error(
          `[MENTIONS_IPC] Error stack: ${error instanceof Error ? error.stack : 'No stack trace'}`
        )

        return {
          payload: `Error processing mentions: ${errorMessage}`
        }
      } finally {
        logger.info(`[MENTIONS_IPC] Request processing complete`)
      }
    }, ErrorType.COMMAND_PROCESSOR)
  )

  // Get available commands - for command autocompletion
  ipcMain.handle(
    LLMChannels.GetCommands,
    wrapIpcHandler(async () => {
      // Get from command registry directly
      // Prevent errors by providing fallback commands
      try {
        // Import directly from the module
        const { commandRegistry } = await import('../services/llm/commandRegistry')
        return commandRegistry.getAll().map((cmd) => ({
          name: cmd.name,
          summary: cmd.summary
        }))
      } catch (error) {
        logger.error('Failed to import command registry:', error)
        // Return basic commands as a fallback
        return [
          { name: 'help', summary: 'List available slash commands' },
          { name: 'clear', summary: 'Clear the chat history' },
          { name: 'version', summary: 'Show application version' },
          { name: 'echo', summary: 'Echo a message back' },
          { name: 'rag', summary: 'Search documents and reply with the input text' }
        ]
      }
    }, ErrorType.DATA_LOAD)
  )

  // Stream message from LLM
  ipcMain.on(
    LLMChannels.StreamMessage,
    async (event, requestId: string, request: ChatCompletionRequest) => {
      try {
        await llmService.streamMessage(
          request,
          // On chunk handler
          (chunk: StreamingChunk) => {
            event.sender.send(LLMChannels.StreamChunk, requestId, chunk)
          },
          // On complete handler
          (response: ChatCompletionResponse) => {
            event.sender.send(LLMChannels.StreamComplete, requestId, response)
          },
          // On error handler
          (error: Error) => {
            event.sender.send(LLMChannels.StreamError, requestId, error.message)
          },
          // Pass the requestId to streamMessage for logging
          requestId
        )
      } catch (error) {
        event.sender.send(LLMChannels.StreamError, requestId, (error as Error).message)
      }
    }
  )

  // Get LLM configuration
  ipcMain.handle(LLMChannels.GetConfig, async () => {
    return getLLMConfig()
  })

  // Update provider configuration
  ipcMain.handle(
    LLMChannels.UpdateProviderConfig,
    async (_, provider: LLMProvider, config: Partial<ProviderConfig>) => {
      try {
        // Log incoming config
        logger.debug(`Updating provider ${provider} with config:`, JSON.stringify(config))

        // Update config
        const fullConfig: SharedTypes.LLMConfig = getLLMConfig()

        // Log existing config before change
        logger.debug(
          `Existing config for provider ${provider}:`,
          JSON.stringify(fullConfig.providers[provider])
        )

        // Give priority to the incoming config values for defaultModel
        // This ensures user's model selection is preserved
        const defaultModel =
          config.defaultModel ||
          fullConfig.providers[provider]?.defaultModel ||
          (provider === 'openai' ? 'gpt-4o' : '')

        // Keep track of both existing and new models
        const existingModels = fullConfig.providers[provider]?.models || []
        const newModels = config.models || []
        const combinedModels = [...new Set([...existingModels, ...newModels])]

        // Make sure the selected model is always in the models list
        if (defaultModel && !combinedModels.includes(defaultModel)) {
          combinedModels.push(defaultModel)
          logger.debug(`Added default model ${defaultModel} to models list`)
        }

        fullConfig.providers[provider] = {
          ...(fullConfig.providers[provider] || {}),
          ...config,
          // Ensure these required fields are set for ProviderConfig
          apiKey: (fullConfig.providers[provider]?.apiKey || config.apiKey || '') as string,
          defaultModel: defaultModel as string,
          models: combinedModels as string[]
        } as ProviderConfig

        // Log the final provider config that will be saved
        logger.debug(
          `Final config for provider ${provider}:`,
          JSON.stringify(fullConfig.providers[provider])
        )

        saveLLMConfig(fullConfig)

        // Reinitialize the provider
        if (fullConfig.providers[provider]?.apiKey) {
          logger.debug(`Reinitializing provider ${provider}`)
          llmService.initProvider(provider, fullConfig.providers[provider]!)
        }

        return true
      } catch (error) {
        logger.error('Error updating provider config:', error)
        return false
      }
    }
  )

  // AI Chat History handlers
  ipcMain.handle(
    LLMChannels.GetAIChatHistory,
    wrapIpcHandler(async () => {
      return await getAIChatHistory()
    }, ErrorType.DATA_LOAD)
  )

  ipcMain.handle(
    LLMChannels.SaveAIChatHistory,
    wrapIpcHandler(async (_, messages: AIMessage[]) => {
      const success = await saveAIChatHistory(messages)
      if (!success) {
        throw new AppError('Failed to save AI chat history', ErrorType.DATA_SAVE)
      }
      return success
    }, ErrorType.DATA_SAVE)
  )

  ipcMain.handle(
    LLMChannels.ClearAIChatHistory,
    wrapIpcHandler(async () => {
      const success = await clearAIChatHistory()
      if (!success) {
        throw new AppError('Failed to clear AI chat history', ErrorType.DATA_SAVE)
      }
      return success
    }, ErrorType.DATA_SAVE)
  )

  // Fetch answers from the server - this allows bypassing CSP restrictions
  ipcMain.handle(
    LLMChannels.FetchAnswers,
    wrapIpcHandler(async (_, request: { query: string }) => {
      const { query } = request
      logger.info(`[ANSWERS_IPC] Received request to fetch answers for query: "${query}"`)

      try {
        // Get API base URL from configuration
        const apiBaseUrl = getApiBaseUrl()

        if (!apiBaseUrl) {
          logger.error('[ANSWERS_IPC] API base URL not configured')
          throw new Error('API base URL not configured')
        }

        // Create endpoint URL
        const endpoint = `${apiBaseUrl}/answers`
        logger.info(`[ANSWERS_IPC] Endpoint URL: ${endpoint}`)

        // Send the request using POST method with JSON body
        logger.info(`[ANSWERS_IPC] Sending POST request with query: "${query}"`)
        const response = await httpRequest(endpoint, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Accept: 'application/json'
          },
          body: JSON.stringify({ query })
        })

        logger.info(`[ANSWERS_IPC] Response received - Status: ${response.status}`)
        logger.info(`[ANSWERS_IPC] Response data: ${JSON.stringify(response.data)}`)

        return response.data
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : String(error)
        logger.error(`[ANSWERS_IPC] Failed to fetch answers: ${errorMessage}`)

        if (error instanceof Error && error.stack) {
          logger.error(`[ANSWERS_IPC] Error stack: ${error.stack}`)
        }

        throw new AppError(`Failed to fetch answers: ${errorMessage}`, ErrorType.NETWORK)
      }
    }, ErrorType.NETWORK)
  )
}
