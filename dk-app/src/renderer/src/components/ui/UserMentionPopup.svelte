<script lang="ts">
  import { cn } from '@lib/utils'
  import { fly } from 'svelte/transition'
  import { onMount, createEventDispatcher } from 'svelte'
  import logger from '@lib/utils/logger'

  const dispatch = createEventDispatcher()

  // Props
  export let inputText = ''
  export let visible = false
  export let users = []
  export let selectedUserIndex = 0
  export let currentUserId = '' // New prop to identify current user

  // Computed properties
  $: mentionText = inputText.includes('@')
    ? inputText.substring(inputText.lastIndexOf('@') + 1).split(' ')[0]
    : ''

  // Filter users to exclude the current user and apply text filter
  $: filteredUsers = mentionText
    ? users.filter(
        (user) =>
          user.id !== currentUserId && // Exclude current user
          user.name.toLowerCase().includes(mentionText.toLowerCase())
      )
    : users.filter((user) => user.id !== currentUserId) // Exclude current user

  // Hide popup if mention text contains a space (meaning it's complete)
  $: if (inputText.includes('@')) {
    const textAfterAt = inputText.substring(inputText.lastIndexOf('@'))
    if (textAfterAt.includes(' ') && textAfterAt.indexOf(' ') > 1) {
      visible = false
    }
  }

  $: {
    logger.debug('UserMention state:', {
      visible,
      inputText,
      mentionText,
      filteredCount: filteredUsers?.length || 0,
      usersCount: users?.length || 0
    })
  }

  // Methods
  function handleClickUser(user) {
    logger.debug('User selected:', user)
    dispatch('selectUser', user)
  }

  onMount(() => {
    logger.debug('UserMentionPopup mounted')
    logger.debug('Initial users:', users)
  })
</script>

{#if visible}
  <!-- Add debug info at the top -->
  <div
    class="absolute left-0 bottom-full w-full max-h-[300px] overflow-y-auto bg-background border border-border rounded-md shadow-md z-10 mb-1"
    transition:fly={{ y: 10, duration: 150 }}
  >
    <div class="py-1">
      {#if filteredUsers && filteredUsers.length > 0}
        {#each filteredUsers as user, i}
          <button
            class={cn(
              'flex justify-between items-center w-full px-4 py-2 text-left text-sm cursor-pointer',
              i === selectedUserIndex
                ? 'bg-muted text-foreground'
                : 'text-foreground hover:bg-muted/50'
            )}
            on:click={() => handleClickUser(user)}
          >
            <div class="flex items-center">
              <div
                class={cn(
                  'w-2 h-2 rounded-full mr-2',
                  user.online ? 'bg-green-500' : 'bg-gray-400'
                )}
              ></div>
              <span class="font-medium">{user.name}</span>
            </div>
            <span class="text-muted-foreground text-xs ml-2">
              {user.online ? 'online' : 'offline'}
            </span>
          </button>
        {/each}
      {:else}
        <div class="px-4 py-2 text-sm text-muted-foreground">No users found.</div>
      {/if}
    </div>
  </div>
{/if}
