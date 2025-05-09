<script lang="ts">
  import { onMount, createEventDispatcher } from 'svelte'
  import {
    Settings as SettingsIcon,
    ChevronDown,
    Radio,
    MessageSquare,
    Sun,
    Moon,
    Bot,
    AppWindow,
    FileText,
    Globe,
    Search
  } from 'lucide-svelte'
  import { cn } from '../lib/utils'
  import Settings from './Settings.svelte'

  let darkMode = false
  let users: Array<{ id: number; name: string; online: boolean }> = []
  let channels: Array<{ id: string; name: string }> = []
  let loading = true
  let refreshInterval: ReturnType<typeof setInterval> | null = null
  let isConnected = false
  let broadcastExpanded = true
  let directMessagesExpanded = true
  let showSettings = false

  // Event dispatcher to communicate with parent components
  const dispatch = createEventDispatcher<{
    userSelect: { id: number; name: string; online: boolean }
    channelSelect: string
    aiAssistantSelect: boolean
    appsSelect: boolean
    documentsSelect: boolean
    apisSelect: boolean
    searchDataSelect: boolean
    themeChange: boolean
  }>()

  async function fetchSidebarData(): Promise<void> {
    try {
      // Fetch users and channels using IPC
      const [usersData, channelsData, configData] = await Promise.all([
        window.api.sidebar.getUsers(),
        window.api.sidebar.getChannels(),
        window.api.config.get()
      ])

      // Filter out current user's own name from the users list
      users = usersData.filter((user) => user.name !== configData.userID)
      channels = channelsData
      isConnected = configData.isConnected
    } catch (error) {
      console.error('Failed to fetch sidebar data:', error)
    } finally {
      loading = false
    }
  }

  // Periodically check connection status
  async function checkConnectionStatus(): Promise<void> {
    try {
      const configData = await window.api.config.get()
      isConnected = configData.isConnected
    } catch (error) {
      console.error('Failed to check connection status:', error)
      isConnected = false
    }
  }

  // Function to refresh user status
  async function refreshUserStatus(): Promise<void> {
    if (isConnected) {
      try {
        const newUsers = await window.api.sidebar.getUsers()
        const configData = await window.api.config.get()

        // Always update the users array with whatever the server returns, even if empty
        // This ensures we correctly show when users go offline
        if (newUsers) {
          console.log(
            `Refreshed user status: ${newUsers.length} users, ${newUsers.filter((u) => u.online).length} online`
          )
          // Filter out current user's own name from the users list
          users = newUsers.filter((user) => user.name !== configData.userID)
        }
      } catch (error) {
        console.error('Failed to refresh user status:', error)
      }
    }
  }

  // Function to periodically refresh user status and connection state
  function startRefreshInterval(): void {
    // Clear any existing interval
    if (refreshInterval) {
      clearInterval(refreshInterval)
    }

    // Set up a new interval to refresh user status and connection state every 5 seconds
    refreshInterval = setInterval(async () => {
      // First check if we're connected
      await checkConnectionStatus()
      // Then refresh user status if we are
      await refreshUserStatus()
    }, 5000) // 5 seconds - refresh more frequently to see status changes
  }

  function toggleTheme(): void {
    darkMode = !darkMode
    localStorage.setItem('dark-mode', darkMode.toString())
    document.documentElement.classList.toggle('dark', darkMode)
    dispatch('themeChange', darkMode)
  }

  function toggleSettings(): void {
    showSettings = !showSettings
  }

  function handleUserClick(user: { id: number; name: string; online: boolean }): void {
    dispatch('userSelect', user)
  }

  function handleChannelClick(channelName: string): void {
    dispatch('channelSelect', channelName)
  }

  function handleAIAssistantClick(): void {
    dispatch('aiAssistantSelect', true)
  }

  function handleAppsClick(): void {
    dispatch('appsSelect', true)
  }

  function handleDocumentsClick(): void {
    dispatch('documentsSelect', true)
  }

  function handleAPIsClick(): void {
    dispatch('apisSelect', true)
  }

  function handleSearchDataClick(): void {
    dispatch('searchDataSelect', true)
  }

  function toggleBroadcast(): void {
    broadcastExpanded = !broadcastExpanded
    localStorage.setItem('broadcast-expanded', broadcastExpanded.toString())
  }

  function toggleDirectMessages(): void {
    directMessagesExpanded = !directMessagesExpanded
    localStorage.setItem('direct-messages-expanded', directMessagesExpanded.toString())
  }

  // Use localStorage to persist the theme state across sessions
  onMount(async () => {
    const savedTheme = localStorage.getItem('dark-mode')
    if (savedTheme !== null) {
      darkMode = savedTheme === 'true'
    } else {
      // Check if user prefers dark mode
      darkMode = window.matchMedia('(prefers-color-scheme: dark)').matches
    }
    document.documentElement.classList.toggle('dark', darkMode)

    // Load saved states for broadcast and direct messages sections
    const savedBroadcastState = localStorage.getItem('broadcast-expanded')
    if (savedBroadcastState !== null) {
      broadcastExpanded = savedBroadcastState === 'true'
    }

    const savedDirectMessagesState = localStorage.getItem('direct-messages-expanded')
    if (savedDirectMessagesState !== null) {
      directMessagesExpanded = savedDirectMessagesState === 'true'
    }

    // Fetch sidebar data from electron backend
    await fetchSidebarData()

    // Do an initial refresh of user status
    await refreshUserStatus()

    // Start refreshing user status periodically
    startRefreshInterval()

    // Clean up interval when component is destroyed
    return () => {
      if (refreshInterval) {
        clearInterval(refreshInterval)
      }
    }
  })
</script>

<nav
  class="fixed left-0 top-8 bottom-0 bg-secondary border-r border-border flex flex-col w-[50px] z-50 overflow-hidden"
  aria-label="Main navigation"
>
  <div class="flex flex-col justify-between h-full py-2">
    <div class="flex flex-col gap-2">
      <!-- Logo -->
      <div class="flex items-center justify-center px-3 py-2.5">
        <img src="./dk_logo.png" alt="Logo" width="28" height="28" />
      </div>

      <!-- Icon buttons -->
      <div class="flex flex-col gap-2">
        <!-- AI Assistant icon -->
        <button
          class="flex items-center justify-center p-2.5 w-full border-none bg-transparent cursor-pointer text-foreground rounded-md hover:bg-accent"
          on:click={handleAIAssistantClick}
          aria-label="AI Assistant"
          tabindex="0"
        >
          <Bot size={20} aria-hidden="true" />
        </button>

        <!-- Documents icon -->
        <button
          class="flex items-center justify-center p-2.5 w-full border-none bg-transparent cursor-pointer text-foreground rounded-md hover:bg-accent"
          on:click={handleDocumentsClick}
          aria-label="Documents"
          tabindex="0"
        >
          <FileText size={20} aria-hidden="true" />
        </button>

        <!-- Trackers icon -->
        <button
          class="flex items-center justify-center p-2.5 w-full border-none bg-transparent cursor-pointer text-foreground rounded-md hover:bg-accent"
          on:click={handleAppsClick}
          aria-label="Trackers"
          tabindex="0"
        >
          <AppWindow size={20} aria-hidden="true" />
        </button>
      </div>
    </div>

    <footer class="flex flex-col gap-1 mt-auto">
      <!-- Theme toggle -->
      <button
        class="flex items-center justify-center p-2.5 w-full border-none bg-transparent cursor-pointer text-foreground rounded-md opacity-80 hover:bg-accent hover:opacity-100"
        aria-label={darkMode ? 'Switch to light mode' : 'Switch to dark mode'}
        on:click={toggleTheme}
        tabindex="0"
      >
        {#if darkMode}
          <Sun size={20} aria-hidden="true" />
        {:else}
          <Moon size={20} aria-hidden="true" />
        {/if}
      </button>

      <!-- Settings -->
      <button
        class="flex items-center justify-center p-2.5 w-full border-none bg-transparent cursor-pointer text-foreground rounded-md opacity-80 hover:bg-accent hover:opacity-100"
        aria-label="Settings"
        on:click={toggleSettings}
        tabindex="0"
      >
        <SettingsIcon size={20} aria-hidden="true" />
      </button>
    </footer>
  </div>
</nav>

<!-- Settings Modal -->
<Settings showModal={showSettings} on:close={() => (showSettings = false)} />
