import {
  ChatCompletionRequest,
  ChatCompletionResponse,
  LLMProvider,
  LLMProviderInterface,
  ProviderConfig,
  StreamingChunk
} from '../types'
import { v4 as uuidv4 } from 'uuid'

export class OpenAIProvider implements LLMProviderInterface {
  provider: LLMProvider.OPENAI
  private apiKey: string
  private baseUrl: string
  private defaultModel: string
  private availableModels: string[]

  constructor(config: ProviderConfig) {
    this.provider = LLMProvider.OPENAI
    this.apiKey = config.apiKey
    this.baseUrl = config.baseUrl || 'https://api.openai.com'
    this.defaultModel = config.defaultModel || 'gpt-4-turbo'
    this.availableModels = config.models || [
      'gpt-4-turbo',
      'gpt-4-turbo-preview',
      'gpt-4',
      'gpt-4-32k',
      'gpt-3.5-turbo',
      'gpt-3.5-turbo-16k'
    ]
  }

  async getModels(): Promise<string[]> {
    try {
      const response = await fetch(`${this.baseUrl}/v1/models`, {
        headers: {
          Authorization: `Bearer ${this.apiKey}`
        }
      })

      if (!response.ok) {
        throw new Error(`Failed to fetch models: ${response.statusText}`)
      }

      const data = await response.json()
      // Filter for chat models only
      const chatModels = data.data
        .filter(
          (model: any) =>
            model.id.includes('gpt') &&
            !model.id.includes('instruct') &&
            !model.id.includes('embedding')
        )
        .map((model: any) => model.id)

      // Add the predefined models if they're not in the list
      const allModels = [...new Set([...this.availableModels, ...chatModels])]
      return allModels
    } catch (error) {
      console.error('Error fetching OpenAI models:', error)
      // Fall back to the predefined models list if API call fails
      return this.availableModels
    }
  }

  async sendMessage(request: ChatCompletionRequest): Promise<ChatCompletionResponse> {
    try {
      const model = request.model || this.defaultModel
      const temperature = request.temperature !== undefined ? request.temperature : 0.7
      const maxTokens = request.maxTokens || 2048

      const response = await fetch(`${this.baseUrl}/v1/chat/completions`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${this.apiKey}`
        },
        body: JSON.stringify({
          model,
          messages: request.messages,
          temperature,
          max_tokens: maxTokens
        })
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(`OpenAI API error: ${errorData.error?.message || response.statusText}`)
      }

      const data = await response.json()

      return {
        id: data.id,
        object: data.object,
        created: data.created,
        model: data.model,
        message: data.choices[0].message,
        usage: data.usage
      }
    } catch (error) {
      console.error('Error in OpenAI provider:', error)
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
      const maxTokens = request.maxTokens || 2048

      const response = await fetch(`${this.baseUrl}/v1/chat/completions`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${this.apiKey}`
        },
        body: JSON.stringify({
          model,
          messages: request.messages,
          temperature,
          max_tokens: maxTokens,
          stream: true
        })
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(`OpenAI API error: ${errorData.error?.message || response.statusText}`)
      }

      const reader = response.body?.getReader()
      if (!reader) {
        throw new Error('Response body cannot be read as stream')
      }

      const decoder = new TextDecoder()
      let fullContent = ''
      let responseId = ''
      let modelName = model

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        const text = decoder.decode(value)
        const lines = text
          .split('\n')
          .filter((line) => line.trim() !== '' && line.trim() !== 'data: [DONE]')

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const data = JSON.parse(line.slice(6))

              // If this is the first chunk, get the response ID
              if (!responseId && data.id) {
                responseId = data.id
                modelName = data.model || model
              }

              // Process delta content
              if (data.choices && data.choices[0].delta) {
                const delta = data.choices[0].delta

                if (delta.content) {
                  fullContent += delta.content

                  const chunk: StreamingChunk = {
                    id: responseId || uuidv4(),
                    object: 'chat.completion.chunk',
                    created: data.created || Date.now(),
                    model: modelName,
                    delta: {
                      role: delta.role || 'assistant',
                      content: delta.content
                    },
                    finishReason: data.choices[0].finish_reason || null
                  }

                  onChunk(chunk)
                } else if (data.choices[0].finish_reason) {
                  // Send the final chunk with finish reason
                  const finalChunk: StreamingChunk = {
                    id: responseId || uuidv4(),
                    object: 'chat.completion.chunk',
                    created: data.created || Date.now(),
                    model: modelName,
                    delta: {},
                    finishReason: data.choices[0].finish_reason
                  }

                  onChunk(finalChunk)
                }
              }
            } catch (e) {
              console.error('Error parsing OpenAI stream chunk:', e)
            }
          }
        }
      }

      // Construct the complete response
      const fullResponse: ChatCompletionResponse = {
        id: responseId || uuidv4(),
        object: 'chat.completion',
        created: Date.now(),
        model: modelName,
        message: {
          role: 'assistant',
          content: fullContent
        }
      }

      onComplete(fullResponse)
    } catch (error) {
      console.error('Error in OpenAI streaming:', error)
      onError(error as Error)
    }
  }
}
