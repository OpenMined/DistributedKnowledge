import {
  ChatCompletionRequest,
  ChatCompletionResponse,
  LLMProvider,
  LLMProviderInterface,
  ProviderConfig,
  StreamingChunk
} from '../types'
import {
  formatMessagesForProvider,
  setupMCPConfig,
  processLLMResponse,
  processToolCalls
} from '../utils'
import { v4 as uuidv4 } from 'uuid'

export class AnthropicProvider implements LLMProviderInterface {
  provider: LLMProvider.ANTHROPIC
  private apiKey: string
  private baseUrl: string
  private defaultModel: string
  private availableModels: string[]

  // MCP related properties
  private mcpClient: any
  private mcpTools: any[] = []
  private mcpInitialized: boolean = false

  constructor(config: ProviderConfig) {
    this.provider = LLMProvider.ANTHROPIC
    this.apiKey = config.apiKey
    this.baseUrl = config.baseUrl || 'https://api.anthropic.com'
    this.defaultModel = config.defaultModel || 'claude-3-7-sonnet-latest'
    this.availableModels = config.models || [
      'claude-3-7-sonnet-latest',
      'claude-3-5-haiku-latest',
      'claude-3-5-sonnet-latest'
    ]

    // Initialize MCP asynchronously
    this.initMCPConfig().catch((err) => {
      console.error('Failed to initialize MCP configuration:', err)
    })
  }

  /**
   * Initialize MCP configuration - loads the client and tools once
   */
  private async initMCPConfig(): Promise<void> {
    try {
      const mcpConfig = await setupMCPConfig('anthropic')
      this.mcpClient = mcpConfig.client
      this.mcpTools = mcpConfig.tools
      this.mcpInitialized = true
      console.log(
        `Anthropic provider successfully initialized MCP with ${this.mcpTools.length} tools`
      )
    } catch (error) {
      console.error('Error initializing MCP config:', error)
      this.mcpInitialized = false
      this.mcpTools = []
    }
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

  /**
   * Stream messages with tool call support
   * This implementation carefully handles tool calls and ensures proper validation
   */
  async streamMessage(
    request: ChatCompletionRequest,
    onChunk: (chunk: StreamingChunk) => void,
    onComplete: (fullResponse: ChatCompletionResponse) => void,
    onError: (error: Error) => void,
    requestId?: string
  ): Promise<void> {
    // Generate a conversation ID for tracking this request
    const conversationId = requestId || uuidv4()
    console.log(`[${conversationId}] Starting stream message request`)

    try {
      // Initialize basic parameters
      const model = request.model || this.defaultModel
      const temperature = request.temperature !== undefined ? request.temperature : 0.7
      const maxTokens = request.maxTokens || 1024

      // Ensure MCP is initialized
      if (!this.mcpInitialized) {
        console.log(`[${conversationId}] MCP not initialized yet, initializing now...`)
        await this.initMCPConfig()

        // Check if initialization was successful
        if (!this.mcpInitialized) {
          console.warn(`[${conversationId}] MCP initialization failed, proceeding without tools`)
        }
      }

      // Format messages for Anthropic provider
      const formattedMessages = formatMessagesForProvider(request.messages, 'anthropic')
      console.log(
        `[${conversationId}] Formatted ${formattedMessages.length} messages for Anthropic`
      )

      // Make the initial request with tools if available
      const response = await this.makeStreamingRequest({
        model,
        messages: formattedMessages,
        temperature,
        maxTokens,
        conversationId,
        onChunk,
        useTools: true
      })

      console.log(
        `[${conversationId}] Initial response received with ${response.toolCalls.length} tool calls`
      )

      // Process the response
      if (response.toolCalls.length > 0) {
        console.log(`[${conversationId}] Processing tool calls from initial response`)
        await this.processToolCalls({
          toolCalls: response.toolCalls,
          content: response.content,
          responseId: response.responseId,
          modelName: response.modelName,
          request,
          model,
          temperature,
          maxTokens,
          conversationId,
          onChunk,
          onComplete
        })
      } else {
        // No tool calls, complete with initial response
        console.log(`[${conversationId}] No tool calls, completing with initial response`)
        const fullResponse: ChatCompletionResponse = {
          id: response.responseId,
          object: 'chat.completion',
          created: Date.now(),
          model: response.modelName,
          message: {
            role: 'assistant',
            content: response.content
          },
          usage: response.usage
        }

        onComplete(fullResponse)
      }
    } catch (error) {
      console.error(`[${conversationId}] Error in Anthropic streaming:`, error)
      // Log more details about the error
      if (error instanceof Error) {
        console.error(`[${conversationId}] Error name: ${error.name}`)
        console.error(`[${conversationId}] Error message: ${error.message}`)
        console.error(`[${conversationId}] Error stack: ${error.stack}`)
      } else {
        console.error(`[${conversationId}] Non-Error object thrown:`, error)
      }
      onError(error instanceof Error ? error : new Error(String(error)))
    }
  }

  /**
   * Make a streaming request to the Anthropic API
   * @private
   */
  private async makeStreamingRequest(params: {
    model: string
    messages: any[]
    temperature: number
    maxTokens: number
    conversationId: string
    onChunk: (chunk: StreamingChunk) => void
    useTools: boolean
  }): Promise<{
    responseId: string
    modelName: string
    content: string
    toolCalls: any[]
    usage: {
      promptTokens: number
      completionTokens: number
      totalTokens: number
    }
  }> {
    const { model, messages, temperature, maxTokens, conversationId, onChunk, useTools } = params

    // Make the request with appropriate parameters
    const requestBody: any = {
      model,
      messages,
      max_tokens: maxTokens,
      temperature,
      stream: true
    }

    // Only include tools if they are available and requested
    if (useTools && this.mcpTools.length > 0) {
      requestBody.tools = this.mcpTools
    }

    console.log(`[${conversationId}] Making request with ${useTools ? 'tools' : 'no tools'}`)

    const response = await fetch(`${this.baseUrl}/v1/messages`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'x-api-key': this.apiKey,
        'anthropic-version': '2023-06-01'
      },
      body: JSON.stringify(requestBody)
    })

    if (!response.ok) {
      const errorData = await response.json()
      throw new Error(`Anthropic API error: ${errorData.error?.message || response.statusText}`)
    }

    const reader = response.body?.getReader()
    if (!reader) {
      throw new Error('Response body cannot be read as stream')
    }

    // Process stream data
    const decoder = new TextDecoder()
    let responseContent = ''
    let responseId = ''
    const modelName = model
    let toolCalls: any[] = []
    let inputTokens = 0
    let outputTokens = 0

    try {
      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        const text = decoder.decode(value)
        const lines = text.split('\n').filter((line) => line.trim() !== '')

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const data = JSON.parse(line.slice(6))

              // Set response ID from first event that has it
              if (!responseId && data.message?.id) {
                responseId = data.message.id
                console.log(`[${conversationId}] Got response ID: ${responseId}`)
              }

              // Handle content streaming
              if (data.type === 'content_block_delta' && data.delta?.text) {
                responseContent += data.delta.text

                const chunk: StreamingChunk = {
                  id: responseId || uuidv4(),
                  object: 'chat.completion.chunk',
                  created: Date.now(),
                  model: modelName,
                  delta: {
                    role: 'assistant',
                    content: data.delta.text
                  },
                  finishReason: null
                }

                onChunk(chunk)
              }
              // Handle tool calls
              else if (data.type === 'tool_use') {
                console.log(`[${conversationId}] Received tool use event:`, data)
                const toolCall = {
                  id: data.id || uuidv4(),
                  type: 'function',
                  function: {
                    name: data.name,
                    arguments: typeof data.input === 'object' ? JSON.stringify(data.input) : '{}'
                  }
                }
                toolCalls.push(toolCall)
              }
              // Handle message completion
              else if (data.type === 'message_stop') {
                // Update token usage if available
                if (data.usage) {
                  inputTokens = data.usage.input_tokens || 0
                  outputTokens = data.usage.output_tokens || 0
                }

                // Final chunk with finish reason
                const finalChunk: StreamingChunk = {
                  id: responseId || uuidv4(),
                  object: 'chat.completion.chunk',
                  created: Date.now(),
                  model: modelName,
                  delta: {},
                  finishReason: 'stop'
                }

                onChunk(finalChunk)
              }
              // Handle message metadata
              else if (data.type === 'message_start' && data.message) {
                if (data.message.id && !responseId) {
                  responseId = data.message.id
                }

                // If the response has tool calls in the message object, extract them
                if (data.message.tool_calls && Array.isArray(data.message.tool_calls)) {
                  for (const tc of data.message.tool_calls) {
                    toolCalls.push({
                      id: tc.id || uuidv4(),
                      type: 'function',
                      function: {
                        name: tc.name,
                        arguments: typeof tc.input === 'object' ? JSON.stringify(tc.input) : '{}'
                      }
                    })
                  }
                  console.log(
                    `[${conversationId}] Extracted ${toolCalls.length} tool calls from message_start`
                  )
                }
              }
            } catch (e) {
              console.error(`[${conversationId}] Error parsing Anthropic stream chunk:`, e)
            }
          }
        }
      }
    } finally {
      reader.releaseLock()
    }

    console.log(
      `[${conversationId}] Stream complete: content length=${responseContent.length}, tool calls=${toolCalls.length}`
    )

    // Create usage stats
    const usage = {
      promptTokens: inputTokens,
      completionTokens: outputTokens,
      totalTokens: inputTokens + outputTokens
    }

    return {
      responseId: responseId || uuidv4(),
      modelName,
      content: responseContent,
      toolCalls,
      usage
    }
  }

  /**
   * Process tool calls and handle the follow-up request
   * @private
   */
  private async processToolCalls(params: {
    toolCalls: any[]
    content: string
    responseId: string
    modelName: string
    request: ChatCompletionRequest
    model: string
    temperature: number
    maxTokens: number
    conversationId: string
    onChunk: (chunk: StreamingChunk) => void
    onComplete: (fullResponse: ChatCompletionResponse) => void
  }): Promise<void> {
    const {
      toolCalls,
      content,
      responseId,
      modelName,
      request,
      model,
      temperature,
      maxTokens,
      conversationId,
      onChunk,
      onComplete
    } = params

    try {
      console.log(`[${conversationId}] Processing ${toolCalls.length} validated tool calls`)

      // Process tool calls and get tool responses
      console.log(
        `[${conversationId}] Starting tool calls processing with client:`,
        this.mcpClient ? 'MCP client initialized' : 'MCP client missing'
      )

      const toolResponses = await processToolCalls(toolCalls, this.mcpClient, 'anthropic')

      console.log(`[${conversationId}] Got ${toolResponses.length} tool responses`)

      // Create updated message array with tool calls and responses
      const assistantMessage = {
        role: 'assistant' as const,
        content: content || '',
        tool_calls: toolCalls
      }

      const updatedMessages = [...request.messages, assistantMessage, ...toolResponses]
      const formattedFollowUpMessages = formatMessagesForProvider(updatedMessages, 'anthropic')

      // Make the follow-up request
      const followUpResponse = await this.makeStreamingRequest({
        model,
        messages: formattedFollowUpMessages,
        temperature,
        maxTokens,
        conversationId,
        onChunk,
        useTools: false // Don't use tools for follow-up to avoid loops
      })

      // Construct the final response
      const fullResponse: ChatCompletionResponse = {
        id: followUpResponse.responseId,
        object: 'chat.completion',
        created: Date.now(),
        model: followUpResponse.modelName,
        message: {
          role: 'assistant',
          content: followUpResponse.content
        },
        usage: followUpResponse.usage
      }

      // Complete the request
      onComplete(fullResponse)
    } catch (error) {
      console.error(`[${conversationId}] Error processing tool calls:`, error)

      // Return partial response on error
      const errorResponse: ChatCompletionResponse = {
        id: responseId,
        object: 'chat.completion',
        created: Date.now(),
        model: modelName,
        message: {
          role: 'assistant',
          content: `${content || ''}\n\n[Error processing tool calls: ${(error as Error).message}]`
        }
      }

      onComplete(errorResponse)
    }
  }
}
