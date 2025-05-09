import { ipcMain } from 'electron'
import { LLMService } from '@services/llm'
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
          logger.warn(`Skipping provider ${providerName} due to missing or invalid API key`)
        }
      }
    } catch (err) {
      logger.error(`Failed to initialize provider ${providerName}:`, err)
    }
  })

  // Set active provider if it's been initialized
  if (llmService.getAvailableProviders().includes(config.activeProvider as LLMProvider)) {
    logger.info(`Setting active provider to ${config.activeProvider}`)
    llmService.setActiveProvider(config.activeProvider as LLMProvider)
  } else if (llmService.getAvailableProviders().length > 0) {
    // If the configured active provider isn't available, use the first available one
    const firstProvider = llmService.getAvailableProviders()[0]
    logger.warn(`Configured active provider not available. Using first available: ${firstProvider}`)
    llmService.setActiveProvider(firstProvider)

    // Update the config
    config.activeProvider = firstProvider
    saveLLMConfig(config)
  }

  logger.info('Available providers after initialization:', llmService.getAvailableProviders())
  logger.info('Active provider after initialization:', llmService.getActiveProvider())
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
    llmService.setActiveProvider(provider)

    // Update config
    const config = getLLMConfig()
    config.activeProvider = provider
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
          }
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
        // Update config
        const fullConfig: SharedTypes.LLMConfig = getLLMConfig()
        fullConfig.providers[provider] = {
          ...(fullConfig.providers[provider] || {}),
          ...config,
          // Ensure these required fields are set for ProviderConfig
          apiKey: (fullConfig.providers[provider]?.apiKey || config.apiKey || '') as string,
          defaultModel: (fullConfig.providers[provider]?.defaultModel ||
            config.defaultModel ||
            '') as string,
          models: (fullConfig.providers[provider]?.models || config.models || []) as string[]
        } as ProviderConfig
        saveLLMConfig(fullConfig)

        // Reinitialize the provider
        if (fullConfig.providers[provider]?.apiKey) {
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
}
