<script lang="ts">
  import { createEventDispatcher, onMount, afterUpdate } from 'svelte'
  import { cn } from '../lib/utils'
  import { Reply, PaperclipIcon, Send, Bot, ArrowDown, MoreVertical, Copy } from 'lucide-svelte'
  import { formatMessageTimestamp } from '../../../shared/utils'

  // Event dispatcher
  const dispatch = createEventDispatcher()

  // Reference to the message container for auto-scrolling
  let messageContainer: HTMLElement

  // Track if user has scrolled up and if we should auto-scroll
  let showScrollButton = false
  let shouldAutoScroll = true

  // Define the message attachment interface
  export interface MessageAttachment {
    id: string
    type: 'image' | 'video' | 'audio' | 'document' | 'other'
    url: string
    filename: string
    size?: number
    mimeType?: string
    thumbnailUrl?: string
    width?: number
    height?: number
    duration?: number
  }

  // Define the enhanced message interface
  export interface EnhancedMessage {
    id: number | string
    sender: {
      id: number | string
      name: string
      avatar: string
      online?: boolean
    }
    text: string
    timestamp: string | Date
    messageType?: 'text' | 'image' | 'file' | 'system'
    deliveryStatus?: 'sent' | 'delivered' | 'read' | 'failed'
    attachments?: MessageAttachment[]
    replyTo?: number
    replies?: EnhancedMessage[]
    replyCount?: number
    metadata?: {
      clientId?: string
      mentions?: Array<string | number>
      isForwarded?: boolean
      originalMessageId?: number | string
      customData?: Record<string, any>
    }
  }

  // Props
  export let messages: EnhancedMessage[] = []
  export let newMessageText = ''
  export let loading = false
  export let error = ''
  export let placeholderText = 'Type a message...'
  export let emptyStateText = 'No messages to display'

  // Track which message has an open dropdown menu
  let activeDropdownId: string | number | null = null

  // Toggle dropdown menu visibility
  function toggleDropdown(messageId: string | number) {
    activeDropdownId = activeDropdownId === messageId ? null : messageId
  }

  // Close dropdown when clicking outside
  function handleClickOutside(event: MouseEvent) {
    if (activeDropdownId !== null) {
      activeDropdownId = null
    }
  }

  // Function to format file size for display
  export function formatFileSize(bytes?: number): string {
    if (bytes === undefined) return 'Unknown size'
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
  }

  // Handle message thread opening
  function handleThreadAction(message: EnhancedMessage) {
    dispatch('threadAction', message)
  }

  // Handle sending a message
  function handleSendMessage() {
    dispatch('sendMessage', newMessageText)
    // Always scroll to bottom after sending a message
    shouldAutoScroll = true
  }

  // Handle keydown event
  function handleKeydown(event: KeyboardEvent) {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault()
      handleSendMessage()
    }
  }

  // Auto-scroll to bottom when messages change, but only if we're at the bottom already
  afterUpdate(() => {
    if (messageContainer && shouldAutoScroll) {
      scrollToBottom()
    }
  })

  // Scroll to bottom function
  function scrollToBottom() {
    messageContainer.scrollTop = messageContainer.scrollHeight
  }

  // Initial setup when component mounts
  onMount(() => {
    scrollToBottom()

    // Add scroll event listener to detect when user scrolls up
    if (messageContainer) {
      messageContainer.addEventListener('scroll', handleScroll)
    }

    // Add click outside listener for dropdown menus
    document.addEventListener('click', handleClickOutside)

    return () => {
      if (messageContainer) {
        messageContainer.removeEventListener('scroll', handleScroll)
      }
      document.removeEventListener('click', handleClickOutside)
    }
  })

  // Handle scroll events to show/hide scroll button and determine auto-scroll behavior
  function handleScroll() {
    if (!messageContainer) return

    const { scrollTop, scrollHeight, clientHeight } = messageContainer
    const atBottom = scrollTop >= scrollHeight - clientHeight - 20

    // Only auto-scroll if we're near the bottom
    shouldAutoScroll = atBottom

    // Show scroll button if we're not at the bottom
    showScrollButton = !atBottom
  }
</script>

<div class="flex flex-col h-full w-full relative">
  <!-- Header slot for custom header content -->
  <slot name="header">
    <!-- Default header if no custom one is provided -->
    <div class="p-4 border-b border-border bg-background">
      <slot name="header-content">
        <h2 class="text-base font-semibold text-foreground">Chat</h2>
      </slot>
    </div>
  </slot>

  <!-- Message list -->
  <div
    bind:this={messageContainer}
    class="flex-1 p-4 overflow-y-auto flex flex-col gap-5 custom-scrollbar"
  >
    {#if loading}
      <div class="flex justify-center items-center h-full">
        <div class="text-muted-foreground">Loading messages...</div>
      </div>
    {:else if error}
      <div class="flex justify-center items-center h-full">
        <div class="text-destructive">{error}</div>
      </div>
    {:else if messages.length === 0}
      <div class="flex justify-center items-center h-full">
        <div class="text-muted-foreground">{emptyStateText}</div>
      </div>
    {:else}
      {#each messages as message (message.id)}
        <div class="flex flex-col gap-1">
          <!-- Main message -->
          <div
            class="flex items-start gap-3 p-2 rounded-lg transition-colors hover:bg-muted/40 group relative"
          >
            <div
              class="flex-shrink-0 w-8 h-8 bg-primary/10 rounded-full flex items-center justify-center text-sm font-medium text-primary"
            >
              {message.sender.avatar}
            </div>
            <div class="flex-1 min-w-0">
              <div class="flex items-baseline gap-2">
                <span class="font-medium text-sm">
                  <slot name="sender-name" {message}>
                    {message.sender.name}
                  </slot>
                </span>
                <span class="text-xs text-muted-foreground">
                  <slot name="timestamp" {message}>
                    {typeof message.timestamp === 'string'
                      ? formatMessageTimestamp(message.timestamp)
                      : formatMessageTimestamp(message.timestamp)}
                  </slot>
                </span>

                <!-- Three dots menu (appears on hover) -->
                <div class="ml-auto opacity-0 group-hover:opacity-100 transition-opacity">
                  <div class="relative">
                    <button
                      class="p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted/80 transition-colors"
                      aria-label="Message options"
                      on:click|stopPropagation={() => toggleDropdown(message.id)}
                    >
                      <MoreVertical size={16} />
                    </button>

                    <!-- Dropdown menu -->
                    {#if activeDropdownId === message.id}
                      <div
                        class="absolute right-0 top-full mt-1 w-48 rounded-md shadow-lg bg-popover border border-border z-10"
                        on:click|stopPropagation
                      >
                        <div class="py-1">
                          <button
                            class="flex items-center gap-2 w-full px-4 py-2 text-sm text-foreground hover:bg-muted/80 transition-colors"
                            on:click={() => {
                              navigator.clipboard.writeText(message.text)
                              activeDropdownId = null
                            }}
                          >
                            <Copy size={16} />
                            Copy message
                          </button>
                          <button
                            class="flex items-center gap-2 w-full px-4 py-2 text-sm text-foreground hover:bg-muted/80 transition-colors"
                            on:click={() => {
                              activeDropdownId = null
                            }}
                          >
                            <Bot size={16} />
                            AI assistant
                          </button>
                        </div>
                      </div>
                    {/if}
                  </div>
                </div>
              </div>
              <div class="mt-1 text-sm text-foreground">
                {message.text}
              </div>

              <!-- Render attachments if present -->
              {#if message.attachments && message.attachments.length > 0}
                <div class="mt-2 flex flex-col gap-2">
                  {#each message.attachments as attachment}
                    {#if attachment.type === 'image'}
                      <div class="border border-border rounded-md overflow-hidden max-w-xs">
                        <img
                          src={attachment.url}
                          alt={attachment.filename}
                          class="max-w-full h-auto"
                        />
                      </div>
                    {:else}
                      <div
                        class="flex items-center gap-2 border border-border rounded-md p-2 max-w-xs"
                      >
                        <PaperclipIcon size={16} class="text-muted-foreground" />
                        <div class="flex-1 overflow-hidden">
                          <div class="text-sm truncate">{attachment.filename}</div>
                          <div class="text-xs text-muted-foreground">
                            {formatFileSize(attachment.size)}
                          </div>
                        </div>
                      </div>
                    {/if}
                  {/each}
                </div>
              {/if}

              <!-- Message actions -->
              <div class="flex items-center gap-3 mt-1.5">
                <slot name="message-actions" {message} {handleThreadAction}>
                  <button
                    class="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
                    on:click={() => handleThreadAction(message)}
                    aria-label="Reply"
                  >
                    <Reply size={14} />
                    <span>
                      {#if message.replyCount && message.replyCount > 0}
                        {message.replyCount} {message.replyCount === 1 ? 'reply' : 'replies'}
                      {:else}
                        Reply
                      {/if}
                    </span>
                  </button>
                </slot>
              </div>
            </div>
          </div>
        </div>
      {/each}
    {/if}
  </div>

  <!-- Scroll to latest messages badge -->
  {#if showScrollButton}
    <div class="absolute bottom-24 left-1/2 transform -translate-x-1/2 flex justify-center">
      <button
        class="py-1.5 px-3 rounded-full bg-primary/90 text-primary-foreground shadow-md hover:bg-primary transition-colors flex items-center gap-1.5 text-xs font-medium"
        on:click={() => {
          scrollToBottom()
          shouldAutoScroll = true
        }}
        aria-label="Scroll to latest messages"
      >
        <ArrowDown size={14} />
        <span>Latest messages</span>
      </button>
    </div>
  {/if}

  <!-- Message input -->
  <div class="p-4 bg-background">
    <slot name="input">
      <div class="flex flex-col gap-2 rounded-lg bg-muted/60 shadow-sm">
        <div class="flex items-center justify-between px-3 pt-2">
          <div class="flex items-center space-x-1">
            <button
              class="p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted/80 transition-colors flex items-center justify-center"
              aria-label="Attach file"
            >
              <PaperclipIcon size={16} />
            </button>
            <button
              class="p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted/80 transition-colors flex items-center justify-center"
              aria-label="AI assistant"
            >
              <Bot size={16} />
            </button>
          </div>
        </div>
        <div class="flex gap-2 px-2 pb-2">
          <input
            type="text"
            class="flex-1 px-3.5 py-2.5 rounded-md border-none bg-transparent text-foreground text-sm focus:outline-none"
            placeholder={placeholderText}
            bind:value={newMessageText}
            on:keydown={handleKeydown}
          />
          <button
            class="p-2.5 rounded-md border-none bg-primary text-primary-foreground cursor-pointer hover:bg-primary/90 transition-colors flex items-center justify-center"
            on:click={handleSendMessage}
            disabled={!newMessageText.trim() || loading}
            aria-label="Send message"
          >
            <Send size={18} />
          </button>
        </div>
      </div>
    </slot>
  </div>
</div>
