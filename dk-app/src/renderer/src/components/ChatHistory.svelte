<script lang="ts">
  import { onMount } from 'svelte'
  import { cn } from '../lib/utils'
  import BaseChatHistory, { type EnhancedMessage } from './BaseChatHistory.svelte'

  export let userId: number
  export let userName: string
  export let isOnline: boolean = false

  // Message store
  let messages: EnhancedMessage[] = []
  let loading = true
  let error = ''
  let newMessageText = ''

  // Fetch messages from Electron backend
  async function fetchMessages(id: number): Promise<void> {
    try {
      loading = true
      error = ''
      messages = await window.api.chat.getMessages(id)
    } catch (err) {
      console.error('Failed to fetch messages:', err)
      error = 'Failed to load messages. Please try again.'
      messages = []
    } finally {
      loading = false
    }
  }

  // Watch for userId changes to fetch messages
  $: {
    if (userId) {
      fetchMessages(userId)
    } else {
      messages = []
    }
  }

  // Set up real-time message listener
  onMount(() => {
    // Listen for new messages
    const newMessageListener = (event, data) => {
      const { cacheKey, message } = data

      // Only update messages if they match our current view
      if (cacheKey === userId.toString()) {
        // Check if message already exists to prevent duplicates
        const messageExists = messages.some((m) => m.id === message.id)
        if (!messageExists) {
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

  // Function to send a new message
  async function handleSendMessage(event): Promise<void> {
    const text = event.detail
    if (!text.trim() || !userId) return

    try {
      // Send the message to the main process
      const result = await window.api.chat.sendMessage(userId, text.trim())

      if (result.success) {
        // Message sent successfully, clear the input
        newMessageText = ''
        // Refresh the messages
        await fetchMessages(userId)
      } else {
        console.error('Failed to send message:', result.error)
      }
    } catch (err) {
      console.error('Error sending message:', err)
    }
  }

  // Function to reply to a message
  function handleThreadAction(event): void {
    const message = event.detail
    console.log(`Replying to message ${message.id}`)
  }

  // No custom header component needed, using slots instead
</script>

<BaseChatHistory
  {messages}
  {loading}
  {error}
  bind:newMessageText
  placeholderText="Type a message to {userName}..."
  emptyStateText="No messages with {userName} yet. Start the conversation!"
  on:sendMessage={handleSendMessage}
  on:threadAction={handleThreadAction}
>
  <svelte:fragment slot="header">
    <div class="p-4 border-b border-border bg-background flex items-center gap-3">
      <div
        class="flex-shrink-0 w-8 h-8 bg-primary/10 rounded-full flex items-center justify-center text-sm font-medium text-primary"
      >
        {userId === 0 ? 'You' : userName.charAt(0).toUpperCase()}
      </div>
      <div>
        <h2 class="text-base font-semibold text-foreground">{userName}</h2>
        <div class="flex items-center gap-1.5">
          <div
            class={cn(
              'w-2 h-2 rounded-full flex-shrink-0',
              userId === 0 || isOnline ? 'bg-success' : 'bg-muted-foreground'
            )}
          ></div>
          <p class="text-xs text-muted-foreground">
            {#if userId === 0}
              Active now
            {:else if isOnline}
              Online
            {:else}
              Offline
            {/if}
          </p>
        </div>
      </div>
    </div>
  </svelte:fragment>

  <svelte:fragment slot="sender-name" let:message>
    {message.sender.id === 0 ? 'You' : message.sender.name}
  </svelte:fragment>
</BaseChatHistory>
