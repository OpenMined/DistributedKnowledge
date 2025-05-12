<script lang="ts">
  import { cn } from '@lib/utils'
  import { onMount, onDestroy, afterUpdate } from 'svelte'
  import {
    Bot,
    User,
    ArrowRight,
    LoaderCircle,
    Copy,
    CheckCircle,
    MoreVertical,
    Trash2,
    Server,
    FileText
  } from 'lucide-svelte'
  import { formatMessageTimestamp } from '@shared/utils'
  import * as SharedTypes from '@shared/types'
  type AIMessage = SharedTypes.AIMessage
  import * as LLMTypes from '@shared/llmTypes'
  // Alias types for easier use
  type ChatCompletionRequest = LLMTypes.ChatCompletionRequest
  type StreamingChunk = LLMTypes.StreamingChunk
  type ChatCompletionResponse = LLMTypes.ChatCompletionResponse
  type LLMProvider = LLMTypes.LLMProvider
  import logger from '@lib/utils/logger'
  import { safeIpcCall } from '@lib/utils/errorHandler'
  import { AppError, ErrorType } from '@shared/errors'

  // Import simple command system
  import SimpleCommandPopup from './ui/SimpleCommandPopup.svelte'
  import MCPServersModal from './MCPServersModal.svelte'
  import {
    commandPopupVisible,
    showCommandPopup,
    hideCommandPopup,
    executeCommand,
    filterCommands
  } from '@lib/commands'
  import { clsx } from 'clsx'

  // Reference to the chat container for auto-scrolling
  let chatContainer: HTMLElement

  // Initial welcome message
  let messages: AIMessage[] = [
    {
      id: crypto.randomUUID(),
      role: 'assistant',
      content: "Hello! I'm your AI assistant. How can I help you today?",
      timestamp: new Date()
    }
  ]

  let newMessageText = ''
  let isWaitingForResponse = false
  let activeProvider: LLMProvider | '' = ''
  let availableProviders: LLMProvider[] = []
  let activeModel: string = ''
  let availableModels: string[] = []
  let requestId = ''

  // Initialize LLM-related data
  async function initLLMData() {
    try {
      logger.debug('Initializing LLM data...')

      availableProviders = (await window.api.llm.getProviders()) || []
      logger.debug('Available providers:', availableProviders)

      activeProvider = (await window.api.llm.getActiveProvider()) || ''
      logger.debug('Active provider:', activeProvider)

      // Get models for the active provider
      if (activeProvider) {
        availableModels = (await window.api.llm.getModels()) || []
        logger.debug('Available models:', availableModels)

        // Get config to find the default model
        const config = await window.api.llm.getConfig()
        logger.debug('LLM config:', config)

        if (config?.providers?.[activeProvider]?.defaultModel) {
          activeModel = config.providers[activeProvider].defaultModel
          logger.debug('Using default model from config:', activeModel)
        } else if (availableModels.length > 0) {
          activeModel = availableModels[0]
          logger.debug('Using first available model:', activeModel)
        }
      } else {
        logger.warn('No active provider set')
      }
    } catch (error) {
      logger.error('Failed to initialize LLM data:', error)
    }
  }

  // Streaming message handlers
  function handleStreamChunk(
    event: Electron.IpcRendererEvent,
    streamRequestId: string,
    chunk: StreamingChunk
  ) {
    logger.debug(`Stream chunk received for requestId: ${streamRequestId}, current requestId: ${requestId}`)

    if (streamRequestId !== requestId) {
      logger.debug('Ignoring chunk for different requestId')
      return
    }

    logger.debug(`Processing stream chunk: ${JSON.stringify(chunk.delta)}`)

    // Find the placeholder message and update its content
    let updatedMessage = false;
    messages = messages.map((msg) => {
      if (msg.isLoading && msg.role === 'assistant') {
        updatedMessage = true;
        logger.debug(`Updating message ${msg.id} with chunk content`)
        return {
          ...msg,
          content: msg.content + (chunk.delta.content || ''),
          isLoading: true
        }
      }
      return msg
    })

    if (!updatedMessage) {
      logger.warn('No loading assistant message found to update with stream chunk')
    }
  }

  function handleStreamComplete(
    event: Electron.IpcRendererEvent,
    streamRequestId: string,
    response: ChatCompletionResponse
  ) {
    logger.debug(`Stream complete received for requestId: ${streamRequestId}, current requestId: ${requestId}`)

    if (streamRequestId !== requestId) {
      logger.debug('Ignoring completion for different requestId')
      return
    }

    logger.debug('Stream complete, updating message state')

    // Remove the loading state from the assistant message
    let updatedMessage = false;
    messages = messages.map((msg) => {
      if (msg.isLoading && msg.role === 'assistant') {
        updatedMessage = true;
        logger.debug(`Marking message ${msg.id} as complete`)
        return {
          ...msg,
          isLoading: false
        }
      }
      return msg
    })

    if (!updatedMessage) {
      logger.warn('No loading assistant message found to mark as complete')
    }

    isWaitingForResponse = false

    // Save to database via IPC
    logger.debug('Saving updated messages to history after stream completion')
    saveAIChatHistory(messages)
  }

  function handleStreamError(
    event: Electron.IpcRendererEvent,
    streamRequestId: string,
    error: string
  ) {
    logger.debug(`Stream error received for requestId: ${streamRequestId}, current requestId: ${requestId}`)

    if (streamRequestId !== requestId) {
      logger.debug('Ignoring error for different requestId')
      return
    }

    logger.error('Error streaming message:', error)

    // Update the error message
    let updatedMessage = false;
    messages = messages.map((msg) => {
      if (msg.isLoading && msg.role === 'assistant') {
        updatedMessage = true;
        logger.debug(`Updating message ${msg.id} with error: ${error}`)
        return {
          ...msg,
          content: `I'm sorry, I encountered an error: ${error}`,
          isLoading: false
        }
      }
      return msg
    })

    if (!updatedMessage) {
      logger.warn('No loading assistant message found to update with error')
    }

    isWaitingForResponse = false

    // Still save to database for error tracking
    logger.debug('Saving updated messages to history after stream error')
    saveAIChatHistory(messages)
  }

  // Save messages to database through main process
  async function saveAIChatHistory(chatHistory: AIMessage[]) {
    try {
      await safeIpcCall(
        () => window.api.llm.saveAIChatHistory(chatHistory),
        {
          show: (message, options) => window.api.toast.show(message, options)
        },
        'Failed to save chat history'
      )
    } catch (error) {
      // Error is already shown to user via toast from safeIpcCall
      logger.error('Failed to save chat history:', error)
    }
  }

  // Setup streaming message handlers
  function setupStreamHandlers() {
    logger.debug('Setting up stream handlers')
    if (!window.api?.llm?.onStreamChunk) {
      logger.error('Stream handlers not available on window.api.llm')
      return
    }

    // Clean up any existing handlers first to prevent duplicates
    cleanupStreamHandlers()

    // Setup new handlers
    window.api.llm.onStreamChunk(handleStreamChunk)
    window.api.llm.onStreamComplete(handleStreamComplete)
    window.api.llm.onStreamError(handleStreamError)

    logger.debug('Stream handlers set up successfully')
  }

  // Clean up event listeners
  function cleanupStreamHandlers() {
    logger.debug('Cleaning up stream handlers')
    if (window.api?.llm?.removeStreamListeners) {
      window.api.llm.removeStreamListeners(
        handleStreamChunk,
        handleStreamComplete,
        handleStreamError
      )
      logger.debug('Stream handlers cleaned up')
    } else {
      logger.warn('Cannot clean up stream handlers - removeStreamListeners not available')
    }
  }

  // References for command functionality
  let inputTextarea: HTMLTextAreaElement
  let userId = crypto.randomUUID() // For command context

  // Initialize commands function
  async function initCommands() {
    logger.debug('Initializing command system')
    try {
      // Use the command initializer to load server commands if available
      await import('@lib/commands').then((commands) => {
        if (commands.initializeCommands) {
          return commands.initializeCommands()
        }
      })
      logger.debug('Command system initialized')
    } catch (error) {
      logger.error('Failed to initialize commands:', error)
    }
  }

  onMount(async () => {
    try {
      // Load conversation history from main process using safe IPC call
      const savedMessages = await safeIpcCall(
        () => window.api.llm.getAIChatHistory(),
        {
          show: (message, options) => window.api.toast.show(message, options)
        },
        'Failed to load chat history'
      )

      if (savedMessages && savedMessages.length > 0) {
        messages = savedMessages
      }
    } catch (error) {
      // Error already shown to user via toast from safeIpcCall
      logger.error('Failed to load AI chat history:', error)

      // Use default welcome message if loading fails
      messages = [
        {
          id: crypto.randomUUID(),
          role: 'assistant',
          content: "Hello! I'm your AI assistant. How can I help you today?",
          timestamp: new Date()
        }
      ]
    }

    // Initialize LLM data
    try {
      await initLLMData()
    } catch (error) {
      logger.error('Failed to initialize LLM data:', error)
      window.api.toast.show('Failed to initialize AI service. Some features may be unavailable.', {
        type: 'warning'
      })
    }

    // Load slash commands
    try {
      await initCommands()
      logger.debug('Slash commands initialized')
    } catch (error) {
      logger.error('Failed to initialize slash commands:', error)
    }

    // Set up stream handlers
    setupStreamHandlers()
  })

  onDestroy(() => {
    cleanupStreamHandlers()
  })

  // Auto-scroll to bottom whenever messages change
  afterUpdate(() => {
    if (chatContainer) {
      chatContainer.scrollTo({
        top: chatContainer.scrollHeight,
        behavior: 'smooth'
      })
    }
  })

  // Function to send a message to the AI assistant
  async function sendMessage() {
    if (!newMessageText.trim() || isWaitingForResponse) return

    // Validation checks
    if (!activeProvider || !activeModel) {
      window.api.toast.show('No AI provider or model configured. Please check your LLM settings.', {
        type: 'error'
      })
      return
    }

    const userMessage: AIMessage = {
      id: crypto.randomUUID(),
      role: 'user',
      content: newMessageText.trim(),
      timestamp: new Date()
    }

    // Add user message to the chat
    messages = [...messages, userMessage]

    // Clear input field
    newMessageText = ''

    // Create a placeholder for the assistant's response
    const assistantPlaceholder: AIMessage = {
      id: crypto.randomUUID(),
      role: 'assistant',
      content: '',
      timestamp: new Date(),
      isLoading: true
    }

    messages = [...messages, assistantPlaceholder]
    isWaitingForResponse = true

    try {
      // Prepare the chat history to send to the LLM
      // Convert our UI messages to the format expected by the LLM API
      const chatHistory = messages
        .filter((msg) => msg.role === 'user' || (msg.role === 'assistant' && !msg.isLoading))
        .map((msg) => ({
          role: msg.role,
          content: msg.content
        }))

      // Generate a unique request ID for this conversation
      requestId = crypto.randomUUID()

      // Create a properly typed request
      const request: LLMTypes.ChatCompletionRequest = {
        messages: chatHistory,
        model: activeModel,
        stream: true
      }

      try {
        // Use the streaming API to get a response - this is a fire-and-forget operation
        // so we wrap it in try/catch but don't await it
        window.api.llm.streamMessage(requestId, request)

        // Save messages to database
        await saveAIChatHistory(messages)
      } catch (error) {
        // Handle streaming initiation error
        throw new AppError(
          error instanceof Error ? error.message : 'Failed to start AI message streaming',
          ErrorType.LLM_API
        )
      }
    } catch (error) {
      // Log the error
      logger.error('Error sending message to AI:', error)

      // Update the placeholder with an error message
      const errorMessage =
        error instanceof AppError
          ? error.message
          : error instanceof Error
            ? error.message
            : 'Unknown error occurred while processing your request'

      // Update the placeholder message with the error
      messages = messages.map((msg) => {
        if (msg.id === assistantPlaceholder.id) {
          return {
            ...msg,
            content: `I'm sorry, I encountered an error: ${errorMessage}`,
            isLoading: false
          }
        }
        return msg
      })

      isWaitingForResponse = false

      // Show error to user
      window.api.toast.show(`Error: ${errorMessage}`, { type: 'error' })

      // Save error state to database
      await saveAIChatHistory(messages)
    }
  }

  // Simple command handlers for the new approach
  function handleSimpleInputChange() {
    // Show command popup if input starts with /
    if (newMessageText.startsWith('/')) {
      showCommandPopup()
      logger.debug('Command mode activated. Text:', newMessageText)
    } else {
      hideCommandPopup()
    }
  }

  function handleSimpleKeyPress(event: KeyboardEvent) {
    // Handle Enter key to execute command or send message
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault()

      // Handle command if input starts with /
      if (newMessageText.trim().startsWith('/')) {
        handleSimpleCommandExecute()
      } else {
        sendMessage()
      }
      return
    }

    // Handle Tab key for command autocompletion
    if (event.key === 'Tab' && newMessageText.startsWith('/')) {
      event.preventDefault()

      // Import filterCommands directly from lib/commands
      const filteredCmds = filterCommands(newMessageText)

      // If there are suggestions available, use the first one
      if (filteredCmds.length > 0) {
        const suggestion = filteredCmds[0]
        newMessageText = `/${suggestion.name} `

        // Set cursor position after the command and space
        setTimeout(() => {
          if (inputTextarea) {
            inputTextarea.focus()
            inputTextarea.selectionStart = suggestion.name.length + 2 // +2 for / and space
            inputTextarea.selectionEnd = suggestion.name.length + 2
          }
        }, 0)
      }
    }
  }

  function handleSimpleCommandSelect(cmd: { name: string }) {
    newMessageText = `/${cmd.name} `

    if (inputTextarea) {
      inputTextarea.focus()

      // Set cursor position after the command and space
      setTimeout(() => {
        if (inputTextarea) {
          inputTextarea.selectionStart = cmd.name.length + 2 // +2 for / and space
          inputTextarea.selectionEnd = cmd.name.length + 2
        }
      }, 0)
    }
  }

  async function handleSimpleCommandExecute() {
    const commandText = newMessageText.trim()
    logger.debug('Executing command:', commandText)

    // Add user message (the command) to chat
    const userMessage: AIMessage = {
      id: crypto.randomUUID(),
      role: 'user',
      content: commandText,
      timestamp: new Date()
    }

    messages = [...messages, userMessage]

    // Clear input field
    newMessageText = ''
    hideCommandPopup()

    try {
      // Show a loading indicator for the command execution
      const placeholderMessage: AIMessage = {
        id: crypto.randomUUID(),
        role: 'assistant',
        content: 'Processing command...',
        timestamp: new Date(),
        isLoading: true
      }

      messages = [...messages, placeholderMessage]

      // Execute the command (could be local or server-side)
      logger.debug(`Calling executeCommand with "${commandText}"`)
      const result = await executeCommand(commandText)

      // Check if this is a special LLM request result
      if (result && typeof result === 'object' && result.type === 'llm_request') {
        logger.debug('Received LLM request from command execution')

        // Update the placeholder message with the display text
        messages = messages.map((msg) => {
          if (msg.id === placeholderMessage.id) {
            logger.debug(`Found placeholder message, updating with LLM request display text`)
            return {
              ...msg,
              content: result.displayText,
              isLoading: true  // Keep loading while we start LLM request
            }
          }
          return msg
        })

        // Save the current messages state
        await saveAIChatHistory(messages)

        // Set waiting state for UI
        isWaitingForResponse = true
        logger.debug('Set isWaitingForResponse to true for LLM request')

        try {
          // Check if we have access to the LLM API
          if (!window.api?.llm?.streamMessage) {
            logger.error('LLM streaming API not available')
            throw new Error('LLM streaming API not available')
          }

          // Check active provider and model
          if (!activeProvider || !activeModel) {
            logger.error('No LLM provider or model configured', { activeProvider, activeModel })
            throw new Error('No AI provider or model configured. Please check your LLM settings.')
          }

          // Prepare the LLM request
          const llmRequest: LLMTypes.ChatCompletionRequest = {
            messages: result.messages,
            model: activeModel,
            stream: true
          }

          // Log the request details for debugging
          logger.debug(`Preparing LLM request with model: ${activeModel}`, {
            messageCount: result.messages.length,
            firstMessageSample: result.messages[0]?.content?.substring(0, 50) + '...'
          })

          // Generate a unique request ID for this conversation
          requestId = crypto.randomUUID()
          logger.debug(`Generated new requestId: ${requestId}`)

          // The isWaitingForResponse flag is already set

          // Use the streaming API to get a response
          logger.debug('Starting LLM request with messages from command')
          window.api.llm.streamMessage(requestId, llmRequest)
          logger.debug('LLM stream request sent successfully')

          // The rest will be handled by the streaming handlers
          return;
        } catch (llmError) {
          // Update the placeholder with an error message if the LLM request fails
          logger.error('Error starting LLM request from command:', llmError)

          messages = messages.map((msg) => {
            if (msg.id === placeholderMessage.id) {
              const errorMessage = llmError instanceof Error ? llmError.message : 'Unknown error';
              logger.error(`Error details: ${errorMessage}`);
              logger.debug(`Updating placeholder message ${msg.id} with error`)
              return {
                ...msg,
                content: result.displayText + `\n\nError: Failed to start AI processing: ${errorMessage}\nPlease try again.`,
                isLoading: false
              }
            }
            return msg
          })

          // Reset waiting state
          isWaitingForResponse = false;
        }
      } else {
        // Standard text result
        logger.debug(`Command execution result received with length: ${result ? result.length : 0}`)
        logger.debug(`First 100 chars: ${result ? result.substring(0, 100) : 'no result'}`)

        if (result && typeof result === 'string' && result.includes('Related Documents:')) {
          logger.debug('Result contains document references')
        }

        // Replace placeholder with result
        logger.debug(`Updating placeholder message ${placeholderMessage.id} with result`)
        messages = messages.map((msg) => {
          if (msg.id === placeholderMessage.id) {
            logger.debug(`Found placeholder message, updating content from "${msg.content}" to result`)
            return {
              ...msg,
              content: result,
              isLoading: false
            }
          }
          return msg
        })
      }

      // Special case for /clear command
      if (commandText.startsWith('/clear')) {
        await clearChat()
      }
    } catch (error) {
      // Handle error in command execution
      logger.error('Command execution failed:', error)

      // Add error message or update placeholder
      const errorMsg = `Error executing command: ${error instanceof Error ? error.message : 'Unknown error'}`

      // Check if we already have a placeholder to update
      const hasPlaceholder = messages.some((msg) => msg.isLoading)

      if (hasPlaceholder) {
        // Update placeholder with error
        messages = messages.map((msg) => {
          if (msg.isLoading) {
            return {
              ...msg,
              content: errorMsg,
              isLoading: false
            }
          }
          return msg
        })
      } else {
        // Add new error message
        messages = [
          ...messages,
          {
            id: crypto.randomUUID(),
            role: 'assistant',
            content: errorMsg,
            timestamp: new Date()
          }
        ]
      }
    }

    // Save the conversation
    logger.debug(`Saving chat history with ${messages.length} messages`)
    // Log content of the last message
    if (messages.length > 0) {
      const lastMsg = messages[messages.length - 1]
      logger.debug(`Last message (id: ${lastMsg.id}, role: ${lastMsg.role}) content length: ${lastMsg.content.length}`)
      logger.debug(`Last message first 100 chars: "${lastMsg.content.substring(0, 100)}"`)
    }
    await saveAIChatHistory(messages)
  }

  // Function to copy text to clipboard
  function copyToClipboard(text: string, messageId: string) {
    // Create a temporary object to track copy state
    const messageCopyState = { ...copyState }

    // Set this message's copy state to true (showing checkmark)
    messageCopyState[messageId] = true
    copyState = messageCopyState

    // Copy to clipboard
    navigator.clipboard.writeText(text).catch((err) => {
      console.error('Failed to copy text: ', err)
      window.api.toast.show('Failed to copy text to clipboard', { type: 'error' })
    })

    // Reset copy state after 2 seconds
    setTimeout(() => {
      const resetState = { ...copyState }
      resetState[messageId] = false
      copyState = resetState
    }, 2000)
  }

  // Track copy button states (for showing copy/check icons)
  let copyState: Record<string, boolean> = {}

  // Track dropdown menu state
  let showDropdown = false
  let showMCPModal = false

  // Close dropdown when clicking outside
  function handleClickOutside(event: MouseEvent) {
    if (showDropdown) {
      showDropdown = false
    }
  }

  // Toggle dropdown menu visibility
  function toggleDropdown(event: MouseEvent) {
    event.stopPropagation() // Prevent event from bubbling up
    showDropdown = !showDropdown
  }

  // Handle document clicks to close dropdown
  onMount(() => {
    document.addEventListener('click', handleClickOutside)
  })

  onDestroy(() => {
    document.removeEventListener('click', handleClickOutside)
  })

  // Clear the chat history
  async function clearChat() {
    try {
      // Clear history via main process and get the welcome message back
      const success = await safeIpcCall(
        () => window.api.llm.clearAIChatHistory(),
        {
          show: (message, options) => window.api.toast.show(message, options)
        },
        'Failed to clear chat history'
      )

      if (success) {
        // Get the fresh history with just the welcome message
        const freshHistory = await safeIpcCall(() => window.api.llm.getAIChatHistory(), {
          show: (message, options) => window.api.toast.show(message, options)
        })
        messages = freshHistory
      } else {
        // Fallback if clearing fails (should not happen with the error handling)
        messages = [
          {
            id: crypto.randomUUID(),
            role: 'assistant',
            content: 'Chat history cleared. How can I help you today?',
            timestamp: new Date()
          }
        ]
        // Try to save the fallback message
        await saveAIChatHistory(messages)
      }
    } catch (error) {
      // Error already shown via toast from safeIpcCall
      logger.error('Failed to clear chat history:', error)

      // Still give user feedback by updating the UI, even if the server-side operation failed
      messages = [
        {
          id: crypto.randomUUID(),
          role: 'assistant',
          content:
            'I tried to clear the chat history but encountered an error. Let me know if you want to try again.',
          timestamp: new Date()
        }
      ]
    }
  }
</script>

<div class="flex flex-col h-full w-full">
  <div class="p-4 border-b border-border bg-background flex items-center justify-between">
    <div class="flex items-center gap-3">
      <div
        class="flex-shrink-0 w-8 h-8 bg-primary/10 rounded-full flex items-center justify-center text-sm font-medium text-primary"
      >
        <Bot size={18} />
      </div>
      <div>
        <h2 class="text-base font-semibold text-foreground">AI Assistant</h2>
        <p class="text-xs text-muted-foreground">
          {activeProvider && activeModel
            ? `Powered by ${activeProvider} - ${activeModel}`
            : 'No AI provider configured'}
        </p>
      </div>
    </div>
    <div class="relative">
      <button
        class="p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted/80 transition-colors"
        aria-label="Chat options"
        on:click={toggleDropdown}
      >
        <MoreVertical size={16} />
      </button>

      <!-- Dropdown menu -->
      {#if showDropdown}
        <div
          class="absolute right-0 z-10 w-40 rounded-md shadow-lg bg-popover border border-border"
          style="top: 2rem; right: 0;"
          on:click|stopPropagation
        >
          <div class="py-1">
            <button
              class="flex items-center gap-2 w-full px-4 py-2 text-sm hover:bg-muted/80 transition-colors"
              on:click={() => {
                showDropdown = false
                showMCPModal = true
              }}
            >
              <Server size={16} />
              MCP Servers
            </button>

            <hr class="my-1 border-border" />

            <button
              class="flex items-center gap-2 w-full px-4 py-2 text-sm text-destructive hover:bg-muted/80 transition-colors"
              on:click={() => {
                showDropdown = false
                clearChat()
              }}
            >
              <Trash2 size={16} />
              Clear History
            </button>
          </div>
        </div>
      {/if}
    </div>
  </div>

  <div
    bind:this={chatContainer}
    class="flex-1 p-4 overflow-y-auto flex flex-col gap-5 custom-scrollbar"
  >
    {#each messages as message (message.id)}
      <div class="flex flex-col gap-1">
        <div
          class={cn(
            'flex items-start gap-3 p-3 rounded-lg transition-colors group',
            message.role === 'assistant' ? 'hover:bg-muted/40' : 'hover:bg-muted/20'
          )}
        >
          <div
            class={cn(
              'flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium',
              message.role === 'assistant'
                ? 'bg-primary/10 text-primary'
                : 'bg-secondary text-secondary-foreground'
            )}
          >
            {#if message.role === 'assistant'}
              <Bot size={18} />
            {:else}
              <User size={18} />
            {/if}
          </div>
          <div class="flex-1 min-w-0">
            <div class="flex items-baseline justify-between">
              <div class="flex items-baseline gap-2">
                <span class="font-medium text-sm">
                  {message.role === 'assistant' ? 'AI Assistant' : 'You'}
                </span>
                <span class="text-xs text-muted-foreground">
                  {formatMessageTimestamp(message.timestamp)}
                </span>
              </div>

              <!-- Copy button for AI assistant messages only - visible on hover -->
              {#if message.role === 'assistant' && message.content && !message.isLoading}
                <button
                  class="text-muted-foreground hover:text-foreground transition-colors p-1 opacity-0 group-hover:opacity-100"
                  on:click={() => copyToClipboard(message.content, message.id)}
                  aria-label="Copy message"
                  title="Copy message"
                >
                  {#if copyState[message.id]}
                    <CheckCircle size={16} class="text-success" />
                  {:else}
                    <Copy size={16} />
                  {/if}
                </button>
              {/if}
            </div>

            {#if message.isLoading && message.content === ''}
              <div class="mt-1.5 flex items-center gap-2 text-muted-foreground">
                <LoaderCircle size={16} class="animate-spin" />
                <span class="text-sm">Thinking...</span>
              </div>
            {:else}
              <!-- Check if this is a document result message -->
              {#if message.content && message.content.includes('[DOCUMENT_RESULTS_START]') && message.content.includes('[DOCUMENT_RESULTS_END]')}
                {@const textParts = message.content.split('[DOCUMENT_RESULTS_START]')}
                {@const documentPart = message.content.substring(
                  message.content.indexOf('[DOCUMENT_RESULTS_START]') + '[DOCUMENT_RESULTS_START]'.length,
                  message.content.indexOf('[DOCUMENT_RESULTS_END]')
                )}
                {@const documents = (() => {
                  try {
                    return JSON.parse(documentPart);
                  } catch (e) {
                    console.error('Failed to parse document results', e);
                    return [];
                  }
                })()}

                <!-- Display the regular message content -->
                <div class="mt-1 text-sm text-foreground whitespace-pre-wrap">
                  {textParts[0]}
                </div>

                <!-- If we have document results, show document icon buttons -->
                {#if documents && documents.length > 0}
                  <div class="mt-3 flex flex-wrap gap-2">
                    <span class="text-xs text-muted-foreground mb-1 w-full">Related Documents:</span>
                    {#each documents as doc}
                      <button
                        class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-muted hover:bg-muted/80 transition-colors text-xs border border-border"
                        title={doc.title}
                        on:click={() => {
                          // Create a temporary modal or popup with document content
                          const docModal = document.createElement('div');
                          docModal.style.position = 'fixed';
                          docModal.style.top = '0';
                          docModal.style.left = '0';
                          docModal.style.width = '100%';
                          docModal.style.height = '100%';
                          docModal.style.backgroundColor = 'rgba(0,0,0,0.7)';
                          docModal.style.zIndex = '9999';
                          docModal.style.display = 'flex';
                          docModal.style.justifyContent = 'center';
                          docModal.style.alignItems = 'center';

                          const content = document.createElement('div');
                          content.style.maxWidth = '800px';
                          content.style.maxHeight = '80vh';
                          content.style.width = '80%';
                          content.style.backgroundColor = 'white';
                          content.style.borderRadius = '8px';
                          content.style.padding = '20px';
                          content.style.boxShadow = '0 4px 6px rgba(0,0,0,0.1)';
                          content.style.overflow = 'auto';

                          const header = document.createElement('div');
                          header.style.display = 'flex';
                          header.style.justifyContent = 'space-between';
                          header.style.alignItems = 'center';
                          header.style.marginBottom = '12px';
                          header.style.paddingBottom = '8px';
                          header.style.borderBottom = '1px solid #eee';

                          const title = document.createElement('h3');
                          title.textContent = doc.title;
                          title.style.fontSize = '18px';
                          title.style.fontWeight = 'bold';
                          title.style.margin = '0';

                          const closeBtn = document.createElement('button');
                          closeBtn.textContent = 'Close';
                          closeBtn.style.padding = '4px 8px';
                          closeBtn.style.borderRadius = '4px';
                          closeBtn.style.backgroundColor = '#eee';
                          closeBtn.style.border = 'none';
                          closeBtn.style.cursor = 'pointer';
                          closeBtn.onclick = () => document.body.removeChild(docModal);

                          header.appendChild(title);
                          header.appendChild(closeBtn);

                          const body = document.createElement('div');
                          body.style.whiteSpace = 'pre-wrap';
                          body.style.fontSize = '14px';
                          body.textContent = doc.content;

                          content.appendChild(header);
                          content.appendChild(body);
                          docModal.appendChild(content);

                          // Close on background click
                          docModal.onclick = (e) => {
                            if (e.target === docModal) {
                              document.body.removeChild(docModal);
                            }
                          };

                          document.body.appendChild(docModal);
                        }}
                      >
                        <FileText size={14} />
                        {doc.title.length > 20 ? doc.title.substring(0, 20) + '...' : doc.title}
                      </button>
                    {/each}
                  </div>
                {/if}
              {:else}
                <!-- Regular message -->
                <div class="mt-1 text-sm text-foreground whitespace-pre-wrap">
                  {message.content}
                  {#if message.isLoading}
                    <LoaderCircle
                      size={16}
                      class="inline-block animate-spin ml-1 align-text-bottom"
                    />
                  {/if}
                </div>
              {/if}
            {/if}
          </div>
        </div>
      </div>
    {/each}
  </div>

  <div class="flex items-start gap-2 p-4 border-t border-border bg-background">
    <div class="flex-1 relative">
      <textarea
        bind:this={inputTextarea}
        class="block w-full px-3.5 py-2.5 rounded-md border border-border bg-background text-foreground text-sm focus:outline-none focus:border-primary resize-none min-h-[40px] max-h-[120px]"
        placeholder="Message AI Assistant... (type / for commands)"
        bind:value={newMessageText}
        on:keydown={handleSimpleKeyPress}
        on:input={handleSimpleInputChange}
        disabled={isWaitingForResponse || !activeProvider}
        rows="1"
      ></textarea>

      <!-- Simple Command Popup -->
      <SimpleCommandPopup
        inputText={newMessageText}
        onSelectCommand={(cmd) => handleSimpleCommandSelect(cmd)}
      />
    </div>

    <button
      class={cn(
        'self-stretch px-4 rounded-md border-none font-medium cursor-pointer flex items-center justify-center gap-2 min-w-[80px]',
        isWaitingForResponse || !newMessageText.trim() || !activeProvider
          ? 'bg-muted text-muted-foreground cursor-not-allowed'
          : 'bg-primary text-primary-foreground hover:bg-primary/90'
      )}
      on:click={sendMessage}
      disabled={isWaitingForResponse || !newMessageText.trim() || !activeProvider}
    >
      {#if isWaitingForResponse}
        <LoaderCircle size={16} class="animate-spin" />
      {:else}
        <span>Send</span>
        <ArrowRight size={16} />
      {/if}
    </button>
  </div>
</div>

<!-- MCP Servers Modal -->
<MCPServersModal bind:showModal={showMCPModal} on:close={() => (showMCPModal = false)} />
