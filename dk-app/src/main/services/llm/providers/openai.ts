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

    // Ensure we use the provided defaultModel if specified
    this.defaultModel = config.defaultModel || 'gpt-4o'
    console.log(`OpenAIProvider constructor - using defaultModel: ${this.defaultModel}`)

    // Make sure our currently selected model is always in the available models list
    const defaultModels = ['gpt-4.1-nano', 'gpt-4.1-mini', 'gpt-4.1', 'gpt-4o', 'gpt-4o-mini']

    // Ensure our default model is in the models list
    if (
      this.defaultModel &&
      !defaultModels.includes(this.defaultModel) &&
      !config.models?.includes(this.defaultModel)
    ) {
      defaultModels.push(this.defaultModel)
    }

    this.availableModels = config.models || defaultModels

    // Double check our models list includes the default model
    if (this.defaultModel && !this.availableModels.includes(this.defaultModel)) {
      this.availableModels.push(this.defaultModel)
    }
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

      // Make sure our default model is always included
      if (this.defaultModel && !allModels.includes(this.defaultModel)) {
        allModels.push(this.defaultModel)
        console.log(`Added default model ${this.defaultModel} to models list`)
      }

      // Log the models we're returning
      console.log(`OpenAI models: ${allModels.join(', ')}`)

      return allModels
    } catch (error) {
      console.error('Error fetching OpenAI models:', error)
      // Fall back to the predefined models list if API call fails
      const fallbackModels = [...this.availableModels]

      // Make sure our default model is always included
      if (this.defaultModel && !fallbackModels.includes(this.defaultModel)) {
        fallbackModels.push(this.defaultModel)
      }

      return fallbackModels
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
