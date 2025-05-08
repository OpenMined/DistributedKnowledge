import {
  ChatCompletionRequest,
  ChatCompletionResponse,
  LLMProvider,
  LLMProviderInterface,
  ProviderConfig,
  StreamingChunk
} from '../types'
import { v4 as uuidv4 } from 'uuid'

export class GeminiProvider implements LLMProviderInterface {
  provider: LLMProvider.GEMINI
  private apiKey: string
  private baseUrl: string
  private defaultModel: string
  private availableModels: string[]

  constructor(config: ProviderConfig) {
    this.provider = LLMProvider.GEMINI
    this.apiKey = config.apiKey
    this.baseUrl = config.baseUrl || 'https://generativelanguage.googleapis.com'
    this.defaultModel = config.defaultModel || 'gemini-1.5-pro'
    this.availableModels = config.models || [
      'gemini-1.5-flash',
      'gemini-1.5-pro',
      'gemini-pro',
      'gemini-pro-vision'
    ]
  }

  async getModels(): Promise<string[]> {
    try {
      const response = await fetch(`${this.baseUrl}/v1beta/models?key=${this.apiKey}`)
      if (!response.ok) {
        throw new Error(`Failed to fetch Gemini models: ${response.statusText}`)
      }

      const data = await response.json()
      const geminiModels = data.models
        .filter((model: any) => model.name.includes('gemini'))
        .map((model: any) => model.name.split('/').pop())

      // Add the predefined models if they're not in the list
      const allModels = [...new Set([...this.availableModels, ...geminiModels])]
      return allModels
    } catch (error) {
      console.error('Error fetching Gemini models:', error)
      return this.availableModels
    }
  }

  async sendMessage(request: ChatCompletionRequest): Promise<ChatCompletionResponse> {
    try {
      const model = request.model || this.defaultModel
      const temperature = request.temperature !== undefined ? request.temperature : 0.7
      const maxTokens = request.maxTokens || 2048

      // Convert our standard messages format to Gemini's format
      const contents = this._convertMessagesToGeminiFormat(request.messages)

      const response = await fetch(
        `${this.baseUrl}/v1beta/models/${model}:generateContent?key=${this.apiKey}`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({
            contents,
            generationConfig: {
              temperature,
              maxOutputTokens: maxTokens,
              topP: 0.95,
              topK: 40
            }
          })
        }
      )

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(`Gemini API error: ${errorData.error?.message || response.statusText}`)
      }

      const data = await response.json()

      // Map Gemini response to our standard format
      let responseText = ''
      if (
        data.candidates &&
        data.candidates.length > 0 &&
        data.candidates[0].content &&
        data.candidates[0].content.parts
      ) {
        responseText = data.candidates[0].content.parts.map((part: any) => part.text || '').join('')
      }

      return {
        id: uuidv4(),
        object: 'chat.completion',
        created: Date.now(),
        model: model,
        message: {
          role: 'assistant',
          content: responseText
        },
        usage: {
          promptTokens: data.usageMetadata?.promptTokenCount || 0,
          completionTokens: data.usageMetadata?.candidatesTokenCount || 0,
          totalTokens:
            (data.usageMetadata?.promptTokenCount || 0) +
            (data.usageMetadata?.candidatesTokenCount || 0)
        }
      }
    } catch (error) {
      console.error('Error in Gemini provider:', error)
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

      // Convert our standard messages format to Gemini's format
      const contents = this._convertMessagesToGeminiFormat(request.messages)

      const response = await fetch(
        `${this.baseUrl}/v1beta/models/${model}:streamGenerateContent?key=${this.apiKey}`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({
            contents,
            generationConfig: {
              temperature,
              maxOutputTokens: maxTokens,
              topP: 0.95,
              topK: 40
            }
          })
        }
      )

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(`Gemini API error: ${errorData.error?.message || response.statusText}`)
      }

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
        const chunks = text.split('}\n{').map((chunk, i, arr) => {
          if (i === 0) return chunk + (arr.length > 1 ? '}' : '')
          if (i === arr.length - 1) return '{' + chunk
          return '{' + chunk + '}'
        })

        for (const chunk of chunks) {
          try {
            const data = JSON.parse(chunk)

            if (
              data.candidates &&
              data.candidates.length > 0 &&
              data.candidates[0].content &&
              data.candidates[0].content.parts
            ) {
              const textPart = data.candidates[0].content.parts
                .map((part: any) => part.text || '')
                .join('')

              if (textPart) {
                fullContent += textPart

                const streamChunk: StreamingChunk = {
                  id: responseId,
                  object: 'chat.completion.chunk',
                  created: startTime,
                  model,
                  delta: {
                    role: 'assistant',
                    content: textPart
                  },
                  finishReason: data.candidates[0].finishReason || null
                }

                onChunk(streamChunk)
              }

              // Check if this is the final chunk
              if (data.candidates[0].finishReason) {
                const finalChunk: StreamingChunk = {
                  id: responseId,
                  object: 'chat.completion.chunk',
                  created: startTime,
                  model,
                  delta: {},
                  finishReason: data.candidates[0].finishReason
                }

                onChunk(finalChunk)
              }
            }
          } catch (e) {
            console.error('Error parsing Gemini stream chunk:', e)
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
      console.error('Error in Gemini streaming:', error)
      onError(error as Error)
    }
  }

  // Helper method to convert standard messages format to Gemini's format
  private _convertMessagesToGeminiFormat(messages: Array<{ role: string; content: string }>) {
    return messages.map((message) => {
      // Gemini uses 'user' and 'model' roles
      const role = message.role === 'assistant' ? 'model' : 'user'
      // System messages in Gemini need to be prepended to the first user message
      if (message.role === 'system') {
        return {
          role: 'user',
          parts: [{ text: `System: ${message.content}` }]
        }
      }
      return {
        role,
        parts: [{ text: message.content }]
      }
    })
  }
}
