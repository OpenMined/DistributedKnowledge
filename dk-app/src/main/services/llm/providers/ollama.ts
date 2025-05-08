import {
  ChatCompletionRequest,
  ChatCompletionResponse,
  LLMProvider,
  LLMProviderInterface,
  ProviderConfig,
  StreamingChunk
} from '../types'
import { v4 as uuidv4 } from 'uuid'

export class OllamaProvider implements LLMProviderInterface {
  provider: LLMProvider.OLLAMA
  private baseUrl: string
  private defaultModel: string
  private availableModels: string[]

  constructor(config: ProviderConfig) {
    this.provider = LLMProvider.OLLAMA
    this.baseUrl = config.baseUrl || 'http://localhost:11434'
    this.defaultModel = config.defaultModel || 'llama3'
    this.availableModels = config.models || []
  }

  async getModels(): Promise<string[]> {
    try {
      const response = await fetch(`${this.baseUrl}/api/tags`)
      if (!response.ok) {
        throw new Error(`Failed to fetch Ollama models: ${response.statusText}`)
      }

      const data = await response.json()
      if (!data.models) {
        return this.availableModels.length
          ? this.availableModels
          : ['llama3', 'llama3:8b', 'llama3:70b', 'mistral', 'mixtral']
      }

      const modelNames = data.models.map((model: any) => model.name)
      return modelNames
    } catch (error) {
      console.error('Error fetching Ollama models:', error)
      return this.availableModels.length
        ? this.availableModels
        : ['llama3', 'llama3:8b', 'llama3:70b', 'mistral', 'mixtral']
    }
  }

  async sendMessage(request: ChatCompletionRequest): Promise<ChatCompletionResponse> {
    try {
      const model = request.model || this.defaultModel
      const temperature = request.temperature !== undefined ? request.temperature : 0.7

      // Convert standard messages to Ollama format
      const messages = request.messages.map((msg) => ({
        role: msg.role === 'system' ? 'system' : msg.role === 'assistant' ? 'assistant' : 'user',
        content: msg.content
      }))

      const response = await fetch(`${this.baseUrl}/api/chat`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          model,
          messages,
          stream: false,
          options: {
            temperature
          }
        })
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(`Ollama API error: ${errorData.error || response.statusText}`)
      }

      const data = await response.json()

      return {
        id: uuidv4(),
        object: 'chat.completion',
        created: Date.now(),
        model,
        message: {
          role: 'assistant',
          content: data.message.content
        },
        usage: {
          promptTokens: data.prompt_eval_count || 0,
          completionTokens: data.eval_count || 0,
          totalTokens: (data.prompt_eval_count || 0) + (data.eval_count || 0)
        }
      }
    } catch (error) {
      console.error('Error in Ollama provider:', error)
      throw error
    }
  }

  async streamMessage(
    request: ChatCompletionRequest,
    onChunk: (chunk: StreamingChunk) => void,
    onComplete: (fullResponse: ChatCompletionResponse) => void,
    onError: (error: Error) => void
  ): Promise<void> {
    try {
      const model = request.model || this.defaultModel
      const temperature = request.temperature !== undefined ? request.temperature : 0.7

      // Convert standard messages to Ollama format
      const messages = request.messages.map((msg) => ({
        role: msg.role === 'system' ? 'system' : msg.role === 'assistant' ? 'assistant' : 'user',
        content: msg.content
      }))

      const response = await fetch(`${this.baseUrl}/api/chat`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          model,
          messages,
          stream: true,
          options: {
            temperature
          }
        })
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(`Ollama API error: ${errorData.error || response.statusText}`)
      }

      const reader = response.body?.getReader()
      if (!reader) {
        throw new Error('Response body cannot be read as stream')
      }

      const decoder = new TextDecoder()
      let fullContent = ''
      const responseId = uuidv4()
      const startTime = Date.now()
      let promptTokens = 0
      let completionTokens = 0

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        const text = decoder.decode(value)
        const lines = text.split('\n').filter((line) => line.trim() !== '')

        for (const line of lines) {
          try {
            const data = JSON.parse(line)

            if (data.message && data.message.content) {
              const contentDelta = data.message.content
              fullContent = contentDelta // Ollama replaces the whole content each time

              const chunk: StreamingChunk = {
                id: responseId,
                object: 'chat.completion.chunk',
                created: startTime,
                model,
                delta: {
                  role: 'assistant',
                  content: contentDelta
                },
                finishReason: data.done ? 'stop' : null
              }

              // Update token counts
              if (data.prompt_eval_count) promptTokens = data.prompt_eval_count
              if (data.eval_count) completionTokens = data.eval_count

              onChunk(chunk)

              // Check if this is the final chunk
              if (data.done) {
                const finalChunk: StreamingChunk = {
                  id: responseId,
                  object: 'chat.completion.chunk',
                  created: startTime,
                  model,
                  delta: {},
                  finishReason: 'stop'
                }

                onChunk(finalChunk)
              }
            }
          } catch (e) {
            console.error('Error parsing Ollama stream chunk:', e)
          }
        }
      }

      // Send complete response
      const fullResponse: ChatCompletionResponse = {
        id: responseId,
        object: 'chat.completion',
        created: startTime,
        model,
        message: {
          role: 'assistant',
          content: fullContent
        },
        usage: {
          promptTokens,
          completionTokens,
          totalTokens: promptTokens + completionTokens
        }
      }

      onComplete(fullResponse)
    } catch (error) {
      console.error('Error in Ollama streaming:', error)
      onError(error as Error)
    }
  }
}
