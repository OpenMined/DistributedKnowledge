<script lang="ts">
  import { onMount, createEventDispatcher } from 'svelte'
  import {
    Settings as SettingsIcon,
    ChevronRight,
    ChevronLeft,
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

  let expanded = false
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
    sidebarChange: boolean
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

  function toggleSidebar(): void {
    expanded = !expanded
    localStorage.setItem('sidebar-expanded', expanded.toString())
    dispatch('sidebarChange', expanded)
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

  // Use localStorage to persist the sidebar and theme state across sessions
  onMount(async () => {
    const savedState = localStorage.getItem('sidebar-expanded')
    if (savedState !== null) {
      expanded = savedState === 'true'
    } else {
      expanded = false // Default to collapsed
    }
    dispatch('sidebarChange', expanded)

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
  class={cn(
    'fixed left-0 top-8 bottom-0 bg-secondary border-r border-border flex flex-col transition-all duration-200 ease-in-out z-50 overflow-hidden',
    expanded ? 'w-[220px]' : 'w-[50px]'
  )}
  aria-label="Main navigation"
>
  <div class="flex flex-col justify-between h-full py-2">
    <div class="flex flex-col gap-2 overflow-y-auto custom-scrollbar">
      <button
        class="flex items-center justify-between px-3 py-2.5 w-full border-none bg-transparent cursor-pointer text-foreground rounded-md hover:bg-accent"
        on:click={toggleSidebar}
        aria-label={expanded ? 'Collapse sidebar' : 'Expand sidebar'}
        tabindex="0"
      >
        <div class="flex items-center justify-center">
          <img src="./dk_logo.png" alt="Logo" width="28" height="28" />
        </div>
        {#if expanded}
          <ChevronLeft size={18} class="opacity-70" aria-hidden="true" />
        {/if}
      </button>

      {#if !expanded}
        <!-- AI Assistant icon in collapsed sidebar -->
        <button
          class="flex items-center justify-center px-3 py-2.5 w-full border-none bg-transparent cursor-pointer text-foreground rounded-md hover:bg-accent"
          on:click={handleAIAssistantClick}
          aria-label="AI Assistant"
          tabindex="0"
          style="display: none;"
        >
          <Bot size={20} aria-hidden="true" />
        </button>

        <!-- Broadcast icon in collapsed sidebar -->
        <button
          class="flex items-center justify-center px-3 py-2.5 w-full border-none bg-transparent cursor-pointer text-foreground rounded-md hover:bg-accent"
          on:click={() => toggleSidebar()}
          aria-label="Broadcast"
          tabindex="0"
          style="display: none;"
        >
          <Radio size={20} aria-hidden="true" />
        </button>

        <!-- Documents icon in collapsed sidebar -->
        <button
          class="flex items-center justify-center px-3 py-2.5 w-full border-none bg-transparent cursor-pointer text-foreground rounded-md hover:bg-accent"
          on:click={handleDocumentsClick}
          aria-label="Documents"
          tabindex="0"
        >
          <FileText size={20} aria-hidden="true" />
        </button>

        <!-- Trackers icon in collapsed sidebar -->
        <button
          class="flex items-center justify-center px-3 py-2.5 w-full border-none bg-transparent cursor-pointer text-foreground rounded-md hover:bg-accent"
          on:click={handleAppsClick}
          aria-label="Trackers"
          tabindex="0"
        >
          <AppWindow size={20} aria-hidden="true" />
        </button>

        <!-- APIs icon in collapsed sidebar -->
        <button
          class="flex items-center justify-center px-3 py-2.5 w-full border-none bg-transparent cursor-pointer text-foreground rounded-md hover:bg-accent"
          on:click={handleAPIsClick}
          aria-label="APIs"
          tabindex="0"
        >
          <Globe size={20} aria-hidden="true" />
        </button>

        <!-- Search Data icon in collapsed sidebar -->
        <button
          class="flex items-center justify-center px-3 py-2.5 w-full border-none bg-transparent cursor-pointer text-foreground rounded-md hover:bg-accent"
          on:click={handleSearchDataClick}
          aria-label="Search Data"
          tabindex="0"
        >
          <Search size={20} aria-hidden="true" />
        </button>

        <!-- Direct Messages icon in collapsed sidebar (hidden) -->
        <button
          class="flex items-center justify-center px-3 py-2.5 w-full border-none bg-transparent cursor-pointer text-foreground rounded-md hover:bg-accent"
          on:click={() => toggleSidebar()}
          aria-label="Direct Messages"
          tabindex="0"
          style="display: none;"
        >
          <MessageSquare size={20} aria-hidden="true" />
        </button>
      {/if}

      {#if expanded}
        <section class="px-3 py-2 overflow-y-auto custom-scrollbar flex flex-col gap-4">
          {#if loading}
            <p class="text-xs text-muted-foreground italic px-1">Loading...</p>
          {:else}
            <!-- AI Assistant section -->
            <section
              class="flex flex-col gap-0.5"
              aria-labelledby="ai-assistant-heading"
              style="display: none;"
            >
              <h2 id="ai-assistant-heading" class="sr-only">AI Assistant</h2>
              <button
                class="flex items-center text-xs font-semibold text-muted-foreground px-1 py-1 cursor-pointer hover:text-foreground hover:bg-accent rounded-md"
                on:click={handleAIAssistantClick}
                aria-label="AI Assistant"
                tabindex="0"
              >
                <div class="flex items-center gap-2">
                  <Bot size={14} class="opacity-80" aria-hidden="true" />
                  <span>AI Assistant</span>
                </div>
              </button>
            </section>

            <!-- Broadcast section -->
            <section
              class="flex flex-col gap-0.5"
              aria-labelledby="broadcast-heading"
              style="display: none;"
            >
              <h2 id="broadcast-heading" class="sr-only">Broadcast Channels</h2>
              <button
                class="flex items-center justify-between text-xs font-semibold text-muted-foreground px-1 py-1 cursor-pointer hover:text-foreground hover:bg-accent rounded-md"
                on:click={toggleBroadcast}
                aria-expanded={broadcastExpanded}
                aria-controls="broadcast-channels"
                tabindex="0"
              >
                <div class="flex items-center gap-2">
                  <Radio size={14} class="opacity-80" aria-hidden="true" />
                  <span>Broadcast</span>
                </div>
                <div>
                  {#if broadcastExpanded}
                    <ChevronDown size={14} aria-hidden="true" />
                  {:else}
                    <ChevronRight size={14} aria-hidden="true" />
                  {/if}
                </div>
              </button>
              {#if broadcastExpanded}
                <ul id="broadcast-channels" class="flex flex-col gap-0.5 pl-5 mt-0.5 list-none">
                  {#each channels as channel (channel.id)}
                    <li>
                      <button
                        class="w-full text-left px-1.5 py-1 text-xs text-foreground opacity-80 cursor-pointer rounded-md hover:bg-accent hover:opacity-100"
                        on:click={() => handleChannelClick(channel.id)}
                        aria-label={`Channel ${channel.name}`}
                        tabindex="0"
                      >
                        <span># {channel.name}</span>
                      </button>
                    </li>
                  {/each}
                </ul>
              {/if}
            </section>

            <!-- Documents section -->
            <section class="flex flex-col gap-0.5" aria-labelledby="documents-heading">
              <h2 id="documents-heading" class="sr-only">Documents</h2>
              <button
                class="flex items-center text-xs font-semibold text-muted-foreground px-1 py-1 cursor-pointer hover:text-foreground hover:bg-accent rounded-md"
                on:click={handleDocumentsClick}
                aria-label="Documents"
                tabindex="0"
              >
                <div class="flex items-center gap-2">
                  <FileText size={14} class="opacity-80" aria-hidden="true" />
                  <span>Documents</span>
                </div>
              </button>
            </section>

            <!-- Trackers section -->
            <section class="flex flex-col gap-0.5" aria-labelledby="trackers-heading">
              <h2 id="trackers-heading" class="sr-only">Trackers</h2>
              <button
                class="flex items-center text-xs font-semibold text-muted-foreground px-1 py-1 cursor-pointer hover:text-foreground hover:bg-accent rounded-md"
                on:click={handleAppsClick}
                aria-label="Trackers"
                tabindex="0"
              >
                <div class="flex items-center gap-2">
                  <AppWindow size={14} class="opacity-80" aria-hidden="true" />
                  <span>Trackers</span>
                </div>
              </button>
            </section>

            <!-- APIs section -->
            <section class="flex flex-col gap-0.5" aria-labelledby="apis-heading">
              <h2 id="apis-heading" class="sr-only">APIs</h2>
              <button
                class="flex items-center text-xs font-semibold text-muted-foreground px-1 py-1 cursor-pointer hover:text-foreground hover:bg-accent rounded-md"
                on:click={handleAPIsClick}
                aria-label="APIs"
                tabindex="0"
              >
                <div class="flex items-center gap-2">
                  <Globe size={14} class="opacity-80" aria-hidden="true" />
                  <span>APIs</span>
                </div>
              </button>
            </section>

            <!-- Search Data section -->
            <section class="flex flex-col gap-0.5" aria-labelledby="search-data-heading">
              <h2 id="search-data-heading" class="sr-only">Search Data</h2>
              <button
                class="flex items-center text-xs font-semibold text-muted-foreground px-1 py-1 cursor-pointer hover:text-foreground hover:bg-accent rounded-md"
                on:click={handleSearchDataClick}
                aria-label="Search Data"
                tabindex="0"
              >
                <div class="flex items-center gap-2">
                  <Search size={14} class="opacity-80" aria-hidden="true" />
                  <span>Search Data</span>
                </div>
              </button>
            </section>

            <!-- Direct Messages section (hidden) -->
            <section
              class="flex flex-col gap-0.5"
              aria-labelledby="direct-messages-heading"
              style="display: none;"
            >
              <h2 id="direct-messages-heading" class="sr-only">Direct Messages</h2>
              <button
                class="flex items-center justify-between text-xs font-semibold text-muted-foreground px-1 py-1 cursor-pointer hover:text-foreground hover:bg-accent rounded-md"
                on:click={toggleDirectMessages}
                aria-expanded={directMessagesExpanded}
                aria-controls="direct-message-users"
                tabindex="0"
              >
                <div class="flex items-center gap-2">
                  <MessageSquare size={14} class="opacity-80" aria-hidden="true" />
                  <span>Direct Messages</span>
                </div>
                <div>
                  {#if directMessagesExpanded}
                    <ChevronDown size={14} aria-hidden="true" />
                  {:else}
                    <ChevronRight size={14} aria-hidden="true" />
                  {/if}
                </div>
              </button>
              {#if directMessagesExpanded}
                <ul id="direct-message-users" class="flex flex-col gap-0.5 pl-5 mt-0.5 list-none">
                  {#each users as user (user.id)}
                    <li>
                      <button
                        class="w-full text-left flex items-center gap-2 px-1.5 py-1 text-xs text-foreground cursor-pointer rounded-md hover:bg-accent"
                        on:click={() => handleUserClick(user)}
                        aria-label={`${user.name} ${user.online ? 'online' : 'offline'}`}
                        tabindex="0"
                      >
                        <span
                          class={cn(
                            'w-2 h-2 rounded-full flex-shrink-0',
                            user.online ? 'bg-success' : 'bg-muted-foreground'
                          )}
                          aria-hidden="true"
                        ></span>
                        <span class="whitespace-nowrap overflow-hidden text-ellipsis">
                          {user.name}
                        </span>
                      </button>
                    </li>
                  {/each}
                </ul>
              {/if}
            </section>
          {/if}
        </section>
      {/if}
    </div>

    <footer class="flex flex-col gap-1 mt-auto">
      <button
        class="flex items-center gap-2.5 px-3 py-2.5 border-none bg-transparent cursor-pointer text-foreground rounded-md opacity-80 hover:bg-accent hover:opacity-100"
        aria-label={darkMode ? 'Switch to light mode' : 'Switch to dark mode'}
        on:click={toggleTheme}
        tabindex="0"
      >
        {#if darkMode}
          <Sun size={20} aria-hidden="true" />
        {:else}
          <Moon size={20} aria-hidden="true" />
        {/if}
        {#if expanded}
          <span class="text-sm font-medium">{darkMode ? 'Light Mode' : 'Dark Mode'}</span>
        {/if}
      </button>
      <button
        class="flex items-center gap-2.5 px-3 py-2.5 border-none bg-transparent cursor-pointer text-foreground rounded-md opacity-80 hover:bg-accent hover:opacity-100"
        aria-label="Settings"
        on:click={toggleSettings}
        tabindex="0"
      >
        <SettingsIcon size={20} aria-hidden="true" />
        {#if expanded}
          <span class="text-sm font-medium">Settings</span>
        {/if}
      </button>
    </footer>
  </div>
</nav>

<!-- Settings Modal -->
<Settings showModal={showSettings} on:close={() => (showSettings = false)} />
