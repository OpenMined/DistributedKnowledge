import {
  ChatCompletionRequest,
  ChatCompletionResponse,
  LLMProvider,
  LLMProviderInterface,
  ProviderConfig,
  StreamingChunk
} from './types'
import { AnthropicProvider, OpenAIProvider, GeminiProvider, OllamaProvider } from './providers'
import { appConfig } from '../config'

/**
 * LLM Service - provides a unified interface to various LLM providers
 */
export class LLMService {
  private providers: Map<LLMProvider, LLMProviderInterface>
  private activeProvider: LLMProvider

  constructor() {
    this.providers = new Map()
    this.activeProvider = LLMProvider.ANTHROPIC // Default provider

    // The initialization of providers is now done in the llmHandlers.ts file
    // The constructor no longer tries to auto-initialize providers from config
  }

  /**
   * Initialize a provider with configuration
   */
  public initProvider(provider: LLMProvider, config: ProviderConfig): void {
    console.log(`Initializing provider ${provider} with config:`, JSON.stringify(config))

    // Ensure we have a valid defaultModel
    if (!config.defaultModel) {
      console.warn(`No defaultModel in config for ${provider}, using fallback`)
      config.defaultModel = this.getDefaultModelForProvider(provider)
    }

    // Check if models array contains the defaultModel
    if (config.models && !config.models.includes(config.defaultModel)) {
      console.warn(`defaultModel ${config.defaultModel} not in models array for ${provider}`)
      // Add it to the models array
      config.models.push(config.defaultModel)
    }

    switch (provider) {
      case LLMProvider.ANTHROPIC:
        this.providers.set(provider, new AnthropicProvider(config))
        break
      case LLMProvider.OPENAI:
        this.providers.set(provider, new OpenAIProvider(config))
        break
      case LLMProvider.GEMINI:
        this.providers.set(provider, new GeminiProvider(config))
        break
      case LLMProvider.OLLAMA:
        this.providers.set(provider, new OllamaProvider(config))
        break
      default:
        throw new Error(`Unsupported provider: ${provider}`)
    }

    // Log the provider instance for debugging
    const providerInstance = this.providers.get(provider)
    console.log(
      `Provider ${provider} initialized with defaultModel: ${(providerInstance as any).defaultModel}`
    )
  }

  /**
   * Set the active provider
   */
  public setActiveProvider(provider: LLMProvider): void {
    // Initialize the provider with default config if not initialized
    if (!this.providers.has(provider)) {
      // Use default config from the defaultLLMConfig
      const defaultProviderConfig = {
        apiKey: '',
        defaultModel: this.getDefaultModelForProvider(provider),
        models: this.getDefaultModelsForProvider(provider)
      }
      this.initProvider(provider, defaultProviderConfig)
    }
    this.activeProvider = provider
  }

  /**
   * Get default model for a provider
   */
  private getDefaultModelForProvider(provider: LLMProvider): string {
    switch (provider) {
      case LLMProvider.ANTHROPIC:
        return 'claude-3-opus-20240229'
      case LLMProvider.OPENAI:
        return 'gpt-4o'
      case LLMProvider.GEMINI:
        return 'gemini-1.5-pro'
      case LLMProvider.OLLAMA:
        return 'gemma3:4b'
      default:
        return ''
    }
  }

  /**
   * Get default models for a provider
   */
  private getDefaultModelsForProvider(provider: LLMProvider): string[] {
    switch (provider) {
      case LLMProvider.ANTHROPIC:
        return ['claude-3-opus-20240229', 'claude-3-sonnet-20240229', 'claude-3-haiku-20240307']
      case LLMProvider.OPENAI:
        return ['gpt-4.1-nano', 'gpt-4.1-mini', 'gpt-4.1', 'gpt-4o', 'gpt-4o-mini']
      case LLMProvider.GEMINI:
        return ['gemini-1.5-pro', 'gemini-1.5-flash', 'gemini-pro']
      case LLMProvider.OLLAMA:
        return ['gemma3:4b', 'gemma:2b', 'qwen2.5:latest']
      default:
        return []
    }
  }

  /**
   * Get the current active provider
   */
  public getActiveProvider(): LLMProvider {
    return this.activeProvider
  }

  /**
   * Get a list of available providers (those that have been initialized)
   */
  public getAvailableProviders(): LLMProvider[] {
    return Array.from(this.providers.keys())
  }

  /**
   * Get all available models for the active provider
   */
  public async getModels(): Promise<string[]> {
    const provider = this.getProviderInstance()
    return provider.getModels()
  }

  /**
   * Get all available models for a specific provider
   */
  public async getModelsForProvider(provider: LLMProvider): Promise<string[]> {
    const providerInstance = this.providers.get(provider)
    if (!providerInstance) {
      throw new Error(`Provider ${provider} not initialized`)
    }
    return providerInstance.getModels()
  }

  /**
   * Send a message to the active provider
   */
  public async sendMessage(request: ChatCompletionRequest): Promise<ChatCompletionResponse> {
    const provider = this.getProviderInstance()
    return provider.sendMessage(request)
  }

  /**
   * Stream a message response from the active provider
   */
  public async streamMessage(
    request: ChatCompletionRequest,
    onChunk: (chunk: StreamingChunk) => void,
    onComplete: (fullResponse: ChatCompletionResponse) => void,
    onError: (error: Error) => void,
    requestId?: string
  ): Promise<void> {
    const provider = this.getProviderInstance()
    return provider.streamMessage(request, onChunk, onComplete, onError, requestId)
  }

  /**
   * Helper method to get the current provider instance
   */
  private getProviderInstance(): LLMProviderInterface {
    const provider = this.providers.get(this.activeProvider)
    if (!provider) {
      throw new Error(`Active provider ${this.activeProvider} not initialized`)
    }
    return provider
  }
}
