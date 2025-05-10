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

export class OpenRouterProvider implements LLMProviderInterface {
  provider: LLMProvider.OPENROUTER
  private apiKey: string
  private baseUrl: string
  private defaultModel: string
  private availableModels: string[]

  // MCP related properties
  private mcpClient: any
  private mcpTools: any[] = []
  private mcpInitialized: boolean = false

  constructor(config: ProviderConfig) {
    this.provider = LLMProvider.OPENROUTER
    this.apiKey = config.apiKey
    this.baseUrl = config.baseUrl || 'https://openrouter.ai/api'

    // Ensure we use the provided defaultModel if specified
    this.defaultModel = config.defaultModel || 'anthropic/claude-3-opus'
    console.log(`OpenRouterProvider constructor - using defaultModel: ${this.defaultModel}`)

    // Make sure our currently selected model is always in the available models list
    const defaultModels = [
      'anthropic/claude-3-opus',
      'anthropic/claude-3-sonnet',
      'anthropic/claude-3-haiku',
      'openai/gpt-4o',
      'mistralai/mistral-large',
      'google/gemini-pro'
    ]

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
      const mcpConfig = await setupMCPConfig('openrouter', {
        headers: {
          'HTTP-Referer': 'https://distributedknowledge.org',
          'X-Title': 'Distributed Knowledge App'
        }
      })
      this.mcpClient = mcpConfig.client
      this.mcpTools = mcpConfig.tools
      this.mcpInitialized = true
      console.log(
        `OpenRouter provider successfully initialized MCP with ${this.mcpTools.length} tools`
      )
    } catch (error) {
      console.error('Error initializing MCP config:', error)
      this.mcpInitialized = false
      this.mcpTools = []
    }
  }

  async getModels(): Promise<string[]> {
    try {
      const response = await fetch(`${this.baseUrl}/v1/models`, {
        headers: {
          Authorization: `Bearer ${this.apiKey}`,
          'HTTP-Referer': 'https://distributedknowledge.org',
          'X-Title': 'Distributed Knowledge App'
        }
      })

      if (!response.ok) {
        throw new Error(`Failed to fetch models: ${response.statusText}`)
      }

      const data = await response.json()
      // Extract model IDs from OpenRouter response
      const fetchedModels = data.data.filter((model: any) => model.id).map((model: any) => model.id)

      // Add the predefined models if they're not in the list
      const allModels = [...new Set([...this.availableModels, ...fetchedModels])]

      // Make sure our default model is always included
      if (this.defaultModel && !allModels.includes(this.defaultModel)) {
        allModels.push(this.defaultModel)
        console.log(`Added default model ${this.defaultModel} to models list`)
      }

      // Log the models we're returning
      console.log(`OpenRouter models: ${allModels.join(', ')}`)

      return allModels
    } catch (error) {
      console.error('Error fetching OpenRouter models:', error)
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
          Authorization: `Bearer ${this.apiKey}`,
          'HTTP-Referer': 'https://distributedknowledge.org',
          'X-Title': 'Distributed Knowledge App'
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
        throw new Error(`OpenRouter API error: ${errorData.error?.message || response.statusText}`)
      }

      const data = await response.json()

      // Process the response with OpenAI-compatible format
      const processedResponse = processLLMResponse(data, 'openrouter')

      return {
        id: data.id,
        object: data.object,
        created: data.created,
        model: data.model,
        message: data.choices[0].message,
        usage: data.usage
      }
    } catch (error) {
      console.error('Error in OpenRouter provider:', error)
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

      // Format messages for OpenRouter provider (using OpenAI format)
      const formattedMessages = formatMessagesForProvider(request.messages, 'openai')
      console.log(
        `[${conversationId}] Formatted ${formattedMessages.length} messages for OpenRouter`
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
          }
        }

        onComplete(fullResponse)
      }
    } catch (error) {
      console.error(`[${conversationId}] Error in OpenRouter streaming:`, error)
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
   * Make a streaming request to the OpenRouter API
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
    toolChoice?: 'auto' | 'none'
  }): Promise<{
    responseId: string
    modelName: string
    content: string
    toolCalls: any[]
  }> {
    const {
      model,
      messages,
      temperature,
      maxTokens,
      conversationId,
      onChunk,
      useTools,
      toolChoice = 'auto'
    } = params

    // Make the request with appropriate parameters
    const requestBody: any = {
      model,
      messages,
      temperature,
      max_tokens: maxTokens,
      stream: true
    }

    // Only include tools if they are available and requested
    if (useTools && this.mcpTools.length > 0) {
      requestBody.tools = this.mcpTools
      requestBody.tool_choice = toolChoice
    }

    console.log(
      `[${conversationId}] Making request with ${useTools ? 'tools' : 'no tools'}, tool_choice: ${toolChoice}`
    )

    const response = await fetch(`${this.baseUrl}/v1/chat/completions`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${this.apiKey}`,
        'HTTP-Referer': 'https://distributedknowledge.org',
        'X-Title': 'Distributed Knowledge App'
      },
      body: JSON.stringify(requestBody)
    })

    if (!response.ok) {
      const errorData = await response.json()
      throw new Error(`OpenRouter API error: ${errorData.error?.message || response.statusText}`)
    }

    const reader = response.body?.getReader()
    if (!reader) {
      throw new Error('Response body cannot be read as stream')
    }

    // Process stream data
    const decoder = new TextDecoder()
    let responseContent = ''
    let responseId = ''
    let modelName = model
    let toolCalls: any[] = []

    try {
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
              // Extract the JSON string from the line
              const jsonStr = line.slice(6)

              // Add safeguard for malformed JSON
              let data
              try {
                data = JSON.parse(jsonStr)
              } catch (jsonError) {
                // Log the problematic JSON string with error details for diagnostics
                console.error(`[${conversationId}] Malformed JSON in stream: "${jsonStr}"`)
                console.error(`[${conversationId}] JSON parse error: ${jsonError.message}`)
                continue
              }

              // Get response ID from first chunk
              if (!responseId && data.id) {
                responseId = data.id
                modelName = data.model || model
              }

              // Process delta content
              if (data.choices && data.choices[0].delta) {
                const delta = data.choices[0].delta

                // Handle tool calls
                if (delta.tool_calls) {
                  // Process tool calls here
                  this.processToolCallChunk(delta.tool_calls, toolCalls, conversationId)
                  console.log(
                    `[${conversationId}] Processed tool call chunks, current count: ${toolCalls.length}`
                  )
                }

                // Handle content
                if (delta.content) {
                  responseContent += delta.content

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
                  // Send final chunk with finish reason
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
              console.error(`[${conversationId}] Error parsing OpenRouter stream chunk:`, e)
            }
          }
        }
      }
    } finally {
      reader.releaseLock()
    }

    // Validate tool calls before returning
    const validatedToolCalls = this.validateToolCalls(toolCalls, conversationId)

    console.log(
      `[${conversationId}] Stream complete: content length=${responseContent.length}, tool calls=${validatedToolCalls.length}`
    )

    return {
      responseId: responseId || uuidv4(),
      modelName,
      content: responseContent,
      toolCalls: validatedToolCalls
    }
  }

  /**
   * Process a tool call chunk from the streaming API
   * @private
   */
  private processToolCallChunk(
    toolCallDeltas: any[],
    toolCalls: any[],
    conversationId: string
  ): void {
    try {
      // Handle first tool call chunk - initialize the array
      if (!toolCalls.length) {
        for (const tc of toolCallDeltas) {
          const id = tc.id || uuidv4()

          // Check if function property exists
          if (!tc.function) {
            console.warn(
              `[${conversationId}] Tool call delta missing function property: ${JSON.stringify(tc)}`
            )
            continue
          }

          // Create tool call with safe defaults
          console.log(
            `[${conversationId}] Tool call function arguments:`,
            JSON.stringify(tc.function?.arguments)
          )

          // Create the tool call object
          toolCalls.push({
            id,
            type: tc.type || 'function',
            function: {
              // Important: Use a meaningful fallback name if empty
              name: tc.function?.name || `unknown_function_${id.substring(0, 8)}`,
              // Ensure arguments is valid JSON
              arguments: tc.function?.arguments || '{}'
            }
          })

          // Log what we've created
          console.log(
            `[${conversationId}] Created tool call:`,
            JSON.stringify({
              id,
              name: toolCalls[toolCalls.length - 1].function.name,
              arguments: toolCalls[toolCalls.length - 1].function.arguments
            })
          )

          console.log(
            `[${conversationId}] Initialized tool call: id=${id}, name=${toolCalls[toolCalls.length - 1].function.name}`
          )
        }
      } else {
        // Update existing tool calls with new chunks
        for (const tc of toolCallDeltas) {
          let existingTool = null

          if (tc.id) {
            existingTool = toolCalls.find((t) => t.id === tc.id)
          } else if (tc.index !== undefined && tc.index < toolCalls.length) {
            existingTool = toolCalls[tc.index]
          }

          if (existingTool) {
            // Check if the function property exists in the delta
            if (!tc.function) {
              console.warn(
                `[${conversationId}] Tool call delta missing function property for update: ${JSON.stringify(tc)}`
              )
              continue
            }

            // Update function name if provided and not empty
            if (tc.function?.name && tc.function.name.trim() !== '') {
              existingTool.function.name = tc.function.name
            }

            // Update function arguments
            if (tc.function?.arguments) {
              // Log the current state of arguments
              console.log(
                `[${conversationId}] Updating arguments: existing="${existingTool.function.arguments || ''}", new="${tc.function.arguments}"`
              )

              // Concatenate with proper handling for empty strings
              const currentArgs = existingTool.function.arguments || ''
              const newArgs = tc.function.arguments

              // Special handling for incomplete JSON
              let concatenatedArgs = currentArgs + newArgs

              // Check if concatenated args may form valid JSON
              try {
                // Check if we have seemingly complete JSON after concatenation
                if (
                  concatenatedArgs.trim().startsWith('{') &&
                  (concatenatedArgs.match(/{/g) || []).length ===
                    (concatenatedArgs.match(/}/g) || []).length
                ) {
                  // Try to parse it to validate
                  JSON.parse(concatenatedArgs)

                  // If we reach here, it's valid JSON - log success
                  console.log(
                    `[${conversationId}] Successfully concatenated arguments into valid JSON`
                  )
                }
              } catch (e) {
                // Log parsing failure but continue with concatenation
                console.log(
                  `[${conversationId}] Concatenated arguments not yet valid JSON: ${e.message}`
                )
              }

              // Always update with concatenated args - we'll validate at the end
              existingTool.function.arguments = concatenatedArgs

              // Log the final state
              console.log(
                `[${conversationId}] Updated arguments: "${existingTool.function.arguments}"`
              )
            }
          }
        }
      }
    } catch (e) {
      console.error(`[${conversationId}] Error processing tool call chunk:`, e)
    }
  }

  /**
   * Validate tool calls to ensure they meet the API requirements
   * @private
   */
  private validateToolCalls(toolCalls: any[], conversationId: string): any[] {
    return toolCalls
      .map((tc, index) => {
        // Skip tool calls with missing function property
        if (!tc.function) {
          console.warn(
            `[${conversationId}] Skipping tool call without function property: ${JSON.stringify(tc)}`
          )
          return null
        }

        // Ensure tool call has an ID
        if (!tc.id) {
          tc.id = uuidv4()
          console.log(`[${conversationId}] Generated missing tool call ID: ${tc.id}`)
        }

        // Ensure function name is valid
        if (!tc.function.name || tc.function.name.trim() === '') {
          tc.function.name = `unknown_function_${tc.id.substring(0, 8)}`
          console.log(
            `[${conversationId}] Applied fallback name for empty function name: ${tc.function.name}`
          )
        }

        // Validate function arguments
        if (typeof tc.function.arguments === 'string') {
          try {
            // Try to parse the arguments to validate JSON
            JSON.parse(tc.function.arguments)
            console.log(
              `[${conversationId}] Tool call arguments for ${tc.function.name} are valid JSON`
            )
          } catch (jsonError) {
            // If not valid JSON, apply advanced repairs
            console.warn(
              `[${conversationId}] Invalid JSON in tool call arguments for ${tc.function.name}: ${jsonError.message}`
            )

            try {
              // Get original args for detailed logging
              const originalArgs = tc.function.arguments

              // Advanced JSON repair algorithm
              let args = tc.function.arguments.trim()

              // If args don't start with { and don't end with }, try to extract JSON object
              if (!(args.startsWith('{') && args.endsWith('}'))) {
                const objectMatch = args.match(/{[^]*?}/)
                if (objectMatch && objectMatch[0]) {
                  args = objectMatch[0]
                  console.log(`[${conversationId}] Extracted JSON object from arguments: "${args}"`)
                }
              }

              // Step 1: Fix quotes - ensure property names are quoted
              const propertyNameFixed = args.replace(/(\\w+)(?=\\s*:)/g, '"$1"')
              if (propertyNameFixed !== args) {
                args = propertyNameFixed
                console.log(`[${conversationId}] Added missing quotes to property names`)
              }

              // Step 2: Fix single quotes replacing with double quotes
              const singleQuotesFixed = args
                .replace(/'([^']*)'(?=\\s*:)/g, '"$1"')
                .replace(/:\\s*'([^']*)'/g, ': "$1"')
              if (singleQuotesFixed !== args) {
                args = singleQuotesFixed
                console.log(`[${conversationId}] Replaced single quotes with double quotes`)
              }

              // Step 3: Fix trailing commas
              const trailingCommaFixed = args.replace(/,\\s*}/g, '}').replace(/,\\s*]/g, ']')
              if (trailingCommaFixed !== args) {
                args = trailingCommaFixed
                console.log(`[${conversationId}] Removed trailing commas`)
              }

              // Step 4: Ensure braces and brackets are balanced
              const openBraces = (args.match(/{/g) || []).length
              const closeBraces = (args.match(/}/g) || []).length
              if (openBraces > closeBraces) {
                args += '}'.repeat(openBraces - closeBraces)
                console.log(`[${conversationId}] Added ${openBraces - closeBraces} closing braces`)
              }

              const openBrackets = (args.match(/\[/g) || []).length
              const closeBrackets = (args.match(/\]/g) || []).length
              if (openBrackets > closeBrackets) {
                args += ']'.repeat(openBrackets - closeBrackets)
                console.log(
                  `[${conversationId}] Added ${openBrackets - closeBrackets} closing brackets`
                )
              }

              // Step 5: Check for unterminated strings
              if (args.includes('"') && (args.match(/"/g) || []).length % 2 !== 0) {
                // Find last unmatched quote
                const matchedQuotes = args.match(/(?:\\\\"|"[^"]*")/g) || []
                const totalQuotes = (args.match(/"/g) || []).length

                if (matchedQuotes.length * 2 < totalQuotes) {
                  args += '"'
                  console.log(`[${conversationId}] Added missing closing quote`)
                }
              }

              // Step 6: Fix control characters
              const controlCharsFixed = args.replace(/[\\u0000-\\u001F]+/g, ' ')
              if (controlCharsFixed !== args) {
                args = controlCharsFixed
                console.log(`[${conversationId}] Removed control characters`)
              }

              // Try parsing with all repairs
              try {
                JSON.parse(args)
                tc.function.arguments = args
                console.log(
                  `[${conversationId}] Successfully repaired JSON arguments for ${tc.function.name}`
                )
                console.log(`[${conversationId}] Original: ${originalArgs}`)
                console.log(`[${conversationId}] Repaired: ${args}`)
              } catch (secondError) {
                // Still failed, use empty object
                console.error(
                  `[${conversationId}] JSON repair attempts failed for ${tc.function.name}, using empty object`
                )
                tc.function.arguments = '{}'
              }
            } catch (repairError) {
              // Something went wrong in our repair logic
              console.error(
                `[${conversationId}] Error in JSON repair process: ${repairError.message}`
              )
              tc.function.arguments = '{}'
            }
          }
        } else if (!tc.function.arguments) {
          tc.function.arguments = '{}'
        }

        // Final validation - ensure what we have is actually valid JSON before returning
        if (typeof tc.function.arguments === 'string') {
          try {
            // One final parse to validate
            const parsed = JSON.parse(tc.function.arguments)

            // If it's an empty string after being parsed (happens with "{}" sometimes),
            // convert back to empty object literal
            if (parsed === '') {
              tc.function.arguments = '{}'
            }
          } catch (finalValidationError) {
            // Safety fallback - if we still have invalid JSON after all our efforts
            console.error(
              `[${conversationId}] Final validation failed - forcing to empty object: ${finalValidationError.message}`
            )
            tc.function.arguments = '{}'
          }
        }

        return tc
      })
      .filter(Boolean) // Remove nulls
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

      // Ensure MCP is properly initialized
      if (!this.mcpInitialized || !this.mcpClient) {
        console.log(`[${conversationId}] MCP not properly initialized, trying to reinitialize...`)
        await this.initMCPConfig()

        if (!this.mcpInitialized || !this.mcpClient) {
          console.error(
            `[${conversationId}] Failed to initialize MCP client, cannot process tool calls`
          )
          throw new Error('MCP client initialization failed, cannot process tool calls')
        }
      }

      // Validate and log tool calls before processing
      console.log(`[${conversationId}] Tool calls to process:`, JSON.stringify(toolCalls))

      const toolResponses = await processToolCalls(toolCalls, this.mcpClient, 'openai')

      console.log(`[${conversationId}] Got ${toolResponses.length} tool responses`)

      // Ensure all tool responses have valid tool_call_ids
      const validatedToolResponses = toolResponses.map((response) => {
        if (!response.tool_call_id && toolCalls.length > 0) {
          console.log(`[${conversationId}] Adding missing tool_call_id to tool response`)
          return {
            ...response,
            tool_call_id: toolCalls[0].id
          }
        }
        return response
      })

      // Create updated message array with tool calls and responses
      const assistantMessage = {
        role: 'assistant' as const,
        content: content || '',
        tool_calls: toolCalls
      }

      const updatedMessages = [...request.messages, assistantMessage, ...validatedToolResponses]
      const formattedFollowUpMessages = formatMessagesForProvider(updatedMessages, 'openai')

      // Log the follow-up messages
      console.log(
        `[${conversationId}] Follow-up messages prepared with ${formattedFollowUpMessages.length} messages`
      )

      // Make the follow-up request with toolChoice=none to ensure it generates text
      const followUpResponse = await this.makeStreamingRequest({
        model,
        messages: formattedFollowUpMessages,
        temperature,
        maxTokens,
        conversationId,
        onChunk,
        useTools: true,
        toolChoice: 'none' // Force no tool calling in follow-up
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
        }
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
