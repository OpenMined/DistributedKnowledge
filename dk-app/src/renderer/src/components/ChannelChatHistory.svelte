<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte'
  import { MessageSquare } from 'lucide-svelte'
  import BaseChatHistory, { type EnhancedMessage } from './BaseChatHistory.svelte'

  export let channelName: string

  // Event dispatcher
  const dispatch = createEventDispatcher()

  // Channel messages from backend
  let messages: EnhancedMessage[] = []
  let newMessageText = ''
  let loading = true
  let error = ''

  // Fetch channel messages when component mounts or channelName changes
  async function fetchChannelMessages() {
    loading = true
    error = ''
    try {
      messages = await window.api.channel.getMessages(channelName)
    } catch (err) {
      console.error('Failed to fetch channel messages:', err)
      error = 'Failed to load channel messages. Please try again.'
      messages = []
    } finally {
      loading = false
    }
  }

  onMount(() => {
    fetchChannelMessages()

    // Listen for new messages
    const newMessageListener = (event, data) => {
      const { cacheKey, message } = data

      // Only update messages if they match our current channel
      if (cacheKey === channelName) {
        // Check if message already exists to prevent duplicates
        const messageExists = messages.some((m) => m.id === message.id)
        if (!messageExists) {
          // Add replies field if not present (for channel messages)
          if (!message.replies) {
            message.replies = []
            message.replyCount = 0
          }

          messages = [...messages, message]

          // Sort messages by timestamp
          messages.sort((a, b) => {
            const timestampA =
              typeof a.timestamp === 'string'
                ? new Date(a.timestamp).getTime()
                : a.timestamp.getTime()
            const timestampB =
              typeof b.timestamp === 'string'
                ? new Date(b.timestamp).getTime()
                : b.timestamp.getTime()
            return timestampA - timestampB
          })
        }
      }
    }

    // Register the event listener
    window.api.chat.onNewMessage(newMessageListener)

    // Clean up the event listener on component unmount
    return () => {
      window.api.chat.removeNewMessageListener(newMessageListener) // Remove listener
    }
  })

  // Watch for changes to channelName
  $: if (channelName) {
    fetchChannelMessages()
  }

  // Function to open thread in sidebar
  function handleThreadAction(event) {
    const message = event.detail
    dispatch('openThread', message)
  }

  // Function to send a new message
  async function handleSendMessage(event) {
    const text = event.detail
    if (!text.trim()) return

    try {
      const result = await window.api.channel.sendMessage(channelName, text)
      if (result.success && result.message) {
        // Optionally add the message to the UI immediately for a more responsive feel
        messages = [...messages, result.message]
        newMessageText = ''
      }
    } catch (error) {
      console.error('Failed to send message:', error)
    }
  }

  // No custom header component needed, using slots instead
</script>

<BaseChatHistory
  {messages}
  {loading}
  {error}
  bind:newMessageText
  placeholderText="Type a message in #{channelName}..."
  emptyStateText="No messages in this channel yet. Be the first to say something!"
  on:sendMessage={handleSendMessage}
  on:threadAction={handleThreadAction}
>
  <svelte:fragment slot="header">
    <div class="p-4 border-b border-border bg-background">
      <h2 class="text-base font-semibold text-foreground">#{channelName}</h2>
    </div>
  </svelte:fragment>

  <svelte:fragment slot="message-actions" let:message let:handleThreadAction>
    <button
      class="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
      on:click={() => handleThreadAction(message)}
      aria-label={message.replyCount && message.replyCount > 0 ? 'View thread' : 'Reply in thread'}
    >
      <MessageSquare size={14} />
      <span>
        {#if message.replyCount && message.replyCount > 0}
          {message.replyCount} {message.replyCount === 1 ? 'reply' : 'replies'}
        {:else}
          Reply in thread
        {/if}
      </span>
    </button>
  </svelte:fragment>
</BaseChatHistory>
