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
  }

  /**
   * Set the active provider
   */
  public setActiveProvider(provider: LLMProvider): void {
    if (!this.providers.has(provider)) {
      throw new Error(`Provider ${provider} not initialized`)
    }
    this.activeProvider = provider
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
    onError: (error: Error) => void
  ): Promise<void> {
    const provider = this.getProviderInstance()
    return provider.streamMessage(request, onChunk, onComplete, onError)
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
