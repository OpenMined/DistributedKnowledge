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

export class OllamaProvider implements LLMProviderInterface {
  provider: LLMProvider.OLLAMA
  private baseUrl: string
  private defaultModel: string
  private availableModels: string[]

  // MCP related properties
  private mcpClient: any
  private mcpTools: any[] = []
  private mcpInitialized: boolean = false

  constructor(config: ProviderConfig) {
    this.provider = LLMProvider.OLLAMA
    this.baseUrl = config.baseUrl || 'http://localhost:11434'
    this.defaultModel = config.defaultModel || 'llama3'
    this.availableModels = config.models || []

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
      const mcpConfig = await setupMCPConfig('ollama')
      this.mcpClient = mcpConfig.client
      this.mcpTools = mcpConfig.tools
      this.mcpInitialized = true
      console.log(`Ollama provider successfully initialized MCP with ${this.mcpTools.length} tools`)
    } catch (error) {
      console.error('Error initializing MCP config:', error)
      this.mcpInitialized = false
      this.mcpTools = []
    }
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
      const maxTokens = request.maxTokens || 2048

      // Ensure MCP is initialized
      if (!this.mcpInitialized) {
        console.log(`[${conversationId}] MCP not initialized yet, initializing now...`)
        await this.initMCPConfig()

        // Check if initialization was successful
        if (!this.mcpInitialized) {
          console.warn(`[${conversationId}] MCP initialization failed, proceeding without tools`)
        }
      }

      // Format messages for Ollama provider
      const formattedMessages = formatMessagesForProvider(request.messages, 'ollama')
      console.log(`[${conversationId}] Formatted ${formattedMessages.length} messages for Ollama`)

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
      console.error(`[${conversationId}] Error in Ollama streaming:`, error)
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
   * Make a streaming request to the Ollama API
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
      stream: true,
      options: {
        temperature
      }
    }

    // Only include tools if they are available and requested
    if (useTools && this.mcpTools.length > 0) {
      requestBody.tools = this.mcpTools
    }

    console.log(`[${conversationId}] Making request with ${useTools ? 'tools' : 'no tools'}`)

    const response = await fetch(`${this.baseUrl}/api/chat`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(requestBody)
    })

    if (!response.ok) {
      const errorData = await response.json()
      throw new Error(`Ollama API error: ${errorData.error || response.statusText}`)
    }

    const reader = response.body?.getReader()
    if (!reader) {
      throw new Error('Response body cannot be read as stream')
    }

    // Process stream data
    const decoder = new TextDecoder()
    let responseContent = ''
    const responseId = uuidv4()
    const modelName = model
    let toolCalls: any[] = []
    let promptTokens = 0
    let completionTokens = 0

    try {
      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        const text = decoder.decode(value)
        const lines = text.split('\n').filter((line) => line.trim() !== '')

        for (const line of lines) {
          try {
            const data = JSON.parse(line)

            if (data.message && data.message.content) {
              responseContent = data.message.content // Ollama replaces content with each chunk

              const chunk: StreamingChunk = {
                id: responseId,
                object: 'chat.completion.chunk',
                created: Date.now(),
                model: modelName,
                delta: {
                  role: 'assistant',
                  content: data.message.content
                },
                finishReason: data.done ? 'stop' : null
              }

              // Update token counts if available
              if (data.prompt_eval_count) promptTokens = data.prompt_eval_count
              if (data.eval_count) completionTokens = data.eval_count

              onChunk(chunk)

              // Check for tool calls in the message
              if (data.message.tool_calls) {
                toolCalls = this.processToolCallsFromMessage(
                  data.message.tool_calls,
                  conversationId
                )
              }

              // Check if this is the final chunk
              if (data.done) {
                const finalChunk: StreamingChunk = {
                  id: responseId,
                  object: 'chat.completion.chunk',
                  created: Date.now(),
                  model: modelName,
                  delta: {},
                  finishReason: 'stop'
                }

                onChunk(finalChunk)
              }
            }
          } catch (e) {
            console.error(`[${conversationId}] Error parsing Ollama stream chunk:`, e)
          }
        }
      }
    } finally {
      reader.releaseLock()
    }

    console.log(
      `[${conversationId}] Stream complete: content length=${responseContent.length}, tool calls=${toolCalls.length}`
    )

    // Create the final usage stats
    const usage = {
      promptTokens,
      completionTokens,
      totalTokens: promptTokens + completionTokens
    }

    return {
      responseId,
      modelName,
      content: responseContent,
      toolCalls,
      usage
    }
  }

  /**
   * Process tool calls from Ollama response
   * @private
   */
  private processToolCallsFromMessage(toolCalls: any[], conversationId: string): any[] {
    if (!Array.isArray(toolCalls)) {
      console.warn(`[${conversationId}] Tool calls is not an array:`, toolCalls)
      return []
    }

    return toolCalls.map((tc: any) => {
      // Ensure each tool call has an ID
      const id = tc.id || uuidv4()

      // Validate function properties
      if (!tc.function || !tc.function.name) {
        console.warn(`[${conversationId}] Invalid tool call object:`, tc)
        return {
          id,
          type: 'function',
          function: {
            name: `unknown_function_${id.substring(0, 8)}`,
            arguments: '{}'
          }
        }
      }

      // Ensure arguments is properly formatted
      let args = tc.function.arguments
      if (typeof args === 'object') {
        try {
          args = JSON.stringify(args)
        } catch (e) {
          console.error(`[${conversationId}] Error stringifying arguments:`, e)
          args = '{}'
        }
      } else if (typeof args !== 'string') {
        args = '{}'
      }

      return {
        id,
        type: 'function',
        function: {
          name: tc.function.name,
          arguments: args
        }
      }
    })
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
      console.log(`[${conversationId}] Processing ${toolCalls.length} tool calls`)

      // Process tool calls and get tool responses
      console.log(
        `[${conversationId}] Starting tool calls processing with client:`,
        this.mcpClient ? 'MCP client initialized' : 'MCP client missing'
      )

      const toolResponses = await processToolCalls(toolCalls, this.mcpClient, 'ollama')

      console.log(`[${conversationId}] Got ${toolResponses.length} tool responses`)

      // Create updated message array with tool calls and responses
      const assistantMessage = {
        role: 'assistant' as const,
        content: content || '',
        tool_calls: toolCalls
      }

      const updatedMessages = [...request.messages, assistantMessage, ...toolResponses]
      const formattedFollowUpMessages = formatMessagesForProvider(updatedMessages, 'ollama')

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
