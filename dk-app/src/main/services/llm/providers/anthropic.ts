import {
  ChatCompletionRequest,
  ChatCompletionResponse,
  LLMProvider,
  LLMProviderInterface,
  ProviderConfig,
  StreamingChunk
} from '../types'
import { v4 as uuidv4 } from 'uuid'

export class AnthropicProvider implements LLMProviderInterface {
  provider: LLMProvider.ANTHROPIC
  private apiKey: string
  private baseUrl: string
  private defaultModel: string
  private availableModels: string[]

  constructor(config: ProviderConfig) {
    this.provider = LLMProvider.ANTHROPIC
    this.apiKey = config.apiKey
    this.baseUrl = config.baseUrl || 'https://api.anthropic.com'
    this.defaultModel = config.defaultModel || 'claude-3-opus-20240229'
    this.availableModels = config.models || [
      'claude-3-opus-20240229',
      'claude-3-sonnet-20240229',
      'claude-3-haiku-20240307',
      'claude-2.1',
      'claude-2.0',
      'claude-instant-1.2'
    ]
  }

  async getModels(): Promise<string[]> {
    return this.availableModels
  }

  async sendMessage(request: ChatCompletionRequest): Promise<ChatCompletionResponse> {
    try {
      // Prepare the Anthropic API request
      const model = request.model || this.defaultModel
      const temperature = request.temperature !== undefined ? request.temperature : 0.7
      const maxTokens = request.maxTokens || 1024

      // Transform messages to Anthropic format
      const messages = request.messages.map((msg) => {
        return {
          role: msg.role === 'assistant' ? 'assistant' : msg.role === 'system' ? 'system' : 'user',
          content: msg.content
        }
      })

      const response = await fetch(`${this.baseUrl}/v1/messages`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'x-api-key': this.apiKey,
          'anthropic-version': '2023-06-01'
        },
        body: JSON.stringify({
          model,
          messages,
          max_tokens: maxTokens,
          temperature
        })
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(`Anthropic API error: ${errorData.error?.message || response.statusText}`)
      }

      const data = await response.json()

      // Transform Anthropic response to our standard format
      return {
        id: data.id,
        object: 'chat.completion',
        created: Date.now(),
        model: data.model,
        message: {
          role: 'assistant',
          content: data.content[0].text
        },
        usage: {
          promptTokens: data.usage.input_tokens,
          completionTokens: data.usage.output_tokens,
          totalTokens: data.usage.input_tokens + data.usage.output_tokens
        }
      }
    } catch (error) {
      console.error('Error in Anthropic provider:', error)
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
      // Prepare the Anthropic API request for streaming
      const model = request.model || this.defaultModel
      const temperature = request.temperature !== undefined ? request.temperature : 0.7
      const maxTokens = request.maxTokens || 1024

      // Transform messages to Anthropic format
      const messages = request.messages.map((msg) => {
        return {
          role: msg.role === 'assistant' ? 'assistant' : msg.role === 'system' ? 'system' : 'user',
          content: msg.content
        }
      })

      const response = await fetch(`${this.baseUrl}/v1/messages`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'x-api-key': this.apiKey,
          'anthropic-version': '2023-06-01'
        },
        body: JSON.stringify({
          model,
          messages,
          max_tokens: maxTokens,
          temperature,
          stream: true
        })
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(`Anthropic API error: ${errorData.error?.message || response.statusText}`)
      }

      // Process the streaming response
      const reader = response.body?.getReader()
      if (!reader) {
        throw new Error('Response body cannot be read as stream')
      }

      const decoder = new TextDecoder()
      let fullContent = ''
      const responseId = uuidv4()
      const startTime = Date.now()

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        const text = decoder.decode(value)
        const lines = text.split('\n').filter((line) => line.trim() !== '')

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            const data = JSON.parse(line.slice(6))

            if (data.type === 'content_block_delta') {
              fullContent += data.delta.text

              // Send chunk to caller
              const chunk: StreamingChunk = {
                id: responseId,
                object: 'chat.completion.chunk',
                created: startTime,
                model,
                delta: {
                  role: 'assistant',
                  content: data.delta.text
                },
                finishReason: null
              }

              onChunk(chunk)
            } else if (data.type === 'message_stop') {
              // Final chunk with finish reason
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
        }
      }

      onComplete(fullResponse)
    } catch (error) {
      console.error('Error in Anthropic streaming:', error)
      onError(error as Error)
    }
  }
}
