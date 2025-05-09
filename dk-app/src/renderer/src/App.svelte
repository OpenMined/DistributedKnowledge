<script lang="ts">
  import { onMount } from 'svelte'
  import Titlebar from './components/Titlebar.svelte'
  import Sidebar from './components/Sidebar.svelte'
  import Welcome from './components/Welcome.svelte'
  import ChatHistory from './components/ChatHistory.svelte'
  import ChannelChatHistory from './components/ChannelChatHistory.svelte'
  import AIChatHistory from './components/AIChatHistory.svelte'
  import AppsSection from './components/AppsSection.svelte'
  import DocumentsSection from './components/DocumentsSection.svelte'
  import APIsSection from './components/APIsSection.svelte'
  import OnboardingWizard from './components/OnboardingWizard.svelte'
  import Toaster from './components/ui/Toaster.svelte'
  import { Search, FileText, AppWindow, Globe } from 'lucide-svelte'
  import { cn } from './lib/utils.ts'
  let currentView = 'welcome'
  let selectedUser = { id: 0, name: '', online: false }
  let selectedChannel = ''
  let darkMode = false
  let showThreadSidebar = false
  let activeThread = null
  let threadReplyText = ''

  // Onboarding state
  let showOnboarding = false // Default to NOT showing onboarding
  let onboardingCompleted = false
  let configCheckComplete = false // Flag to track if we've checked config status

  // Search functionality
  let searchQuery = ''
  let searchResults = []
  let isSearching = false
  let searchCategory = 'all' // 'all', 'documents', 'trackers', 'apis'

  // Function to handle theme changes
  function handleThemeChange(event: CustomEvent<boolean>): void {
    darkMode = event.detail
  }

  // Function to handle user selection
  function handleUserSelect(
    event: CustomEvent<{ id: number; name: string; online: boolean }>
  ): void {
    selectedUser = event.detail
    currentView = 'user-chat'
    // Reset thread view when changing chats
    showThreadSidebar = false
    activeThread = null
    // Store last selected view in localStorage
    localStorage.setItem('current-view', currentView)
    localStorage.setItem('selected-user', JSON.stringify(selectedUser))
  }

  // Function to handle channel selection
  function handleChannelSelect(event: CustomEvent<string>): void {
    selectedChannel = event.detail
    currentView = 'channel-chat'
    // Reset thread view when changing channels
    showThreadSidebar = false
    activeThread = null
    // Store last selected view in localStorage
    localStorage.setItem('current-view', currentView)
    localStorage.setItem('selected-channel', selectedChannel)
  }

  // Function to handle AI Assistant selection
  function handleAIAssistantSelect(): void {
    currentView = 'ai-chat'
    // Reset thread view when changing to AI chat
    showThreadSidebar = false
    activeThread = null
    // Store last selected view in localStorage
    localStorage.setItem('current-view', currentView)
  }

  // Function to handle Apps selection
  function handleAppsSelect(): void {
    currentView = 'apps'
    // Reset thread view when changing to Apps section
    showThreadSidebar = false
    activeThread = null
    // Store last selected view in localStorage
    localStorage.setItem('current-view', currentView)
  }

  // Function to handle Documents selection
  function handleDocumentsSelect(): void {
    currentView = 'documents'
    // Reset thread view when changing to Documents section
    showThreadSidebar = false
    activeThread = null
    // Store last selected view in localStorage
    localStorage.setItem('current-view', currentView)
  }

  // Function to handle APIs selection
  function handleAPIsSelect(): void {
    currentView = 'apis'
    // Reset thread view when changing to APIs section
    showThreadSidebar = false
    activeThread = null
    // Store last selected view in localStorage
    localStorage.setItem('current-view', currentView)
  }

  // Function to handle Search Data selection
  function handleSearchDataSelect(): void {
    currentView = 'search-data'
    // Reset thread view when changing to Search Data section
    showThreadSidebar = false
    activeThread = null
    // Clear previous search
    searchQuery = ''
    searchResults = []
    isSearching = false
    // Store last selected view in localStorage
    localStorage.setItem('current-view', currentView)
  }

  // Function to handle search submission
  async function handleSearch(category = searchCategory) {
    if (!searchQuery.trim()) return

    isSearching = true
    searchCategory = category

    try {
      // In a real implementation, this would call an API to search through data
      // For now, we'll simulate a search with a timeout
      // await new Promise(resolve => setTimeout(resolve, 1000));

      // Placeholder for search results - in a real implementation this would be replaced with actual API calls
      console.log(`Searching for "${searchQuery}" in category: ${category}`)

      // Clear previous results
      searchResults = []

      // Here, you would implement the actual search logic based on the category
      // For now, we'll just simulate some placeholder results

      // This is just a placeholder - replace with actual implementation
      // searchResults = await window.api.search.searchData(searchQuery, category);
    } catch (error) {
      console.error('Error performing search:', error)
    } finally {
      isSearching = false
    }
  }

  // Function to set search category and perform search
  function setSearchCategory(category: string) {
    searchCategory = category
    if (searchQuery.trim()) {
      handleSearch(category)
    }
  }

  // Function to handle thread viewing
  function handleOpenThread(event: CustomEvent<any>): void {
    activeThread = event.detail
    showThreadSidebar = true
    threadReplyText = ''
  }

  // Function to close thread sidebar
  function closeThreadSidebar(): void {
    showThreadSidebar = false
    activeThread = null
    threadReplyText = ''
  }

  // Function to send a reply in a thread
  async function sendThreadReply(): Promise<void> {
    if (!threadReplyText.trim() || !activeThread || !selectedChannel) return

    try {
      const result = await window.api.channel.sendReply(
        selectedChannel,
        activeThread.id,
        threadReplyText
      )

      if (result.success && result.reply) {
        // Update the active thread with the new reply
        if (!activeThread.replies) {
          activeThread.replies = []
        }

        activeThread.replies = [...activeThread.replies, result.reply]
        activeThread.replyCount = activeThread.replies.length

        // Clear the input
        threadReplyText = ''
      }
    } catch (error) {
      console.error('Failed to send reply:', error)
    }
  }

  // Handle enter key in thread reply
  function handleThreadReplyKeydown(event: KeyboardEvent): void {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault()
      sendThreadReply()
    }
  }

  // Handle onboarding completion
  function handleOnboardingComplete() {
    onboardingCompleted = true
    showOnboarding = false
    currentView = 'welcome'
  }

  // Initialize state
  onMount(async () => {
    // Initialize theme state
    const savedTheme = localStorage.getItem('dark-mode')
    if (savedTheme !== null) {
      darkMode = savedTheme === 'true'
    } else {
      // Check if user prefers dark mode
      darkMode = window.matchMedia('(prefers-color-scheme: dark)').matches
    }

    // Check if onboarding is needed by seeing if config.json exists
    try {
      console.log('Checking onboarding status and config existence...')

      // First directly check if config file exists
      const configPath = await window.api.onboarding.getStatus()

      console.log('Raw response from onboarding status check:', configPath)

      // Check if the response has the expected format
      if (configPath && typeof configPath === 'object') {
        const { success, status, configExists } = configPath
        console.log('Parsed response:', { success, status, configExists })

        if (success) {
          // If configExists is explicitly false, we MUST show onboarding
          if (configExists === false) {
            console.log('CONFIG FILE DOES NOT EXIST - MUST show onboarding')
            showOnboarding = true
            onboardingCompleted = false
          } else if (status) {
            // Config file exists, use status from backend
            showOnboarding = status.isFirstRun && !status.completed
            onboardingCompleted = status.completed
            console.log('Using status from backend:', {
              isFirstRun: status.isFirstRun,
              completed: status.completed,
              showOnboarding,
              onboardingCompleted
            })
          }
        } else {
          // If the backend call failed, show onboarding as a fallback
          console.error('Backend call failed - showing onboarding as fallback')
          showOnboarding = true
        }
      } else {
        // If we got an unexpected response format, show onboarding as a fallback
        console.error('Unexpected response format - showing onboarding as fallback')
        showOnboarding = true
      }

      configCheckComplete = true
      console.log('Final onboarding decision:', {
        showOnboarding,
        onboardingCompleted,
        configCheckComplete
      })
    } catch (error) {
      console.error('Failed to check onboarding status:', error)

      // For any error, show onboarding as a safety measure
      console.log('Error occurred - showing onboarding as fallback')
      showOnboarding = true
      configCheckComplete = true
    }

    // Always start with welcome screen (unless onboarding is showing)
    if (!showOnboarding) {
      currentView = 'welcome'
    }

    // Listen for navigation events from Welcome component
    window.addEventListener('navigate', (event) => {
      if (event.detail && event.detail.section) {
        if (event.detail.section === 'apps') {
          currentView = 'apps'
        }
      }
    })
  })

  const ipcHandle = (): void => window.electron.ipcRenderer.send('ping')
</script>

{#if configCheckComplete}
  {#if showOnboarding}
    <!-- Onboarding wizard component (for first-time setup) -->
    <OnboardingWizard on:complete={handleOnboardingComplete} />
  {:else}
    <!-- Normal app UI (when config exists) -->
    <Titlebar />
    <Sidebar
      on:userSelect={handleUserSelect}
      on:channelSelect={handleChannelSelect}
      on:aiAssistantSelect={handleAIAssistantSelect}
      on:appsSelect={handleAppsSelect}
      on:documentsSelect={handleDocumentsSelect}
      on:apisSelect={handleAPIsSelect}
      on:searchDataSelect={handleSearchDataSelect}
      on:themeChange={handleThemeChange}
    />
  {/if}
{:else}
  <!-- Simple floating logo while checking config -->
  <div class="flex items-center justify-center h-screen bg-background">
    <img src="./dk_logo.png" alt="Loading" width="150" height="150" class="animate-float" />
  </div>
{/if}

{#if configCheckComplete && !showOnboarding}
  <main
    class={cn(
      'pt-8 h-screen box-border transition-all duration-200 ease-in-out',
      'pl-[50px]',
      showThreadSidebar ? 'pr-[300px]' : ''
    )}
  >
    {#if currentView === 'welcome'}
      <Welcome />
    {:else if currentView === 'user-chat'}
      <ChatHistory
        userId={selectedUser.id}
        userName={selectedUser.name}
        isOnline={selectedUser.online}
      />
    {:else if currentView === 'channel-chat'}
      <ChannelChatHistory channelName={selectedChannel} on:openThread={handleOpenThread} />
    {:else if currentView === 'ai-chat'}
      <AIChatHistory />
    {:else if currentView === 'apps'}
      <AppsSection />
    {:else if currentView === 'documents'}
      <DocumentsSection />
    {:else if currentView === 'apis'}
      <APIsSection />
    {:else if currentView === 'search-data'}
      <div class="flex flex-col items-center justify-center h-full">
        <!-- Logo or app branding could go here -->
        <div class="w-full max-w-2xl px-6 -mt-24">
          <h1 class="text-3xl font-bold mb-2 text-center text-foreground">Search</h1>
          <p class="text-sm text-muted-foreground text-center mb-8">
            Find information across documents, trackers, APIs and more
          </p>

          <!-- Google-inspired search bar -->
          <form on:submit|preventDefault={() => handleSearch()} class="w-full">
            <div class="relative">
              <div
                class="flex items-center w-full bg-background border border-border hover:border-input focus-within:border-primary focus-within:ring-2 focus-within:ring-primary/20 rounded-full shadow-sm transition-all duration-200"
              >
                <Search class="absolute left-4 text-muted-foreground" size={20} />
                <input
                  type="text"
                  bind:value={searchQuery}
                  placeholder="Type to search..."
                  class="w-full h-14 pl-12 pr-14 py-4 bg-transparent border-none rounded-full focus:outline-none text-base"
                  on:keydown={(e) => e.key === 'Enter' && handleSearch()}
                  autofocus
                />
                {#if searchQuery}
                  <button
                    type="button"
                    class="absolute right-14 p-1 rounded-full hover:bg-accent/50"
                    aria-label="Clear search"
                    on:click={() => {
                      searchQuery = ''
                      searchResults = []
                    }}
                  >
                    <svg
                      xmlns="http://www.w3.org/2000/svg"
                      width="18"
                      height="18"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="2"
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      class="text-muted-foreground"
                    >
                      <line x1="18" y1="6" x2="6" y2="18"></line>
                      <line x1="6" y1="6" x2="18" y2="18"></line>
                    </svg>
                  </button>
                {/if}
                <button
                  type="button"
                  class="absolute right-4 p-1 rounded-full hover:bg-accent/50"
                  aria-label="Search options"
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="20"
                    height="20"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    class="text-muted-foreground"
                  >
                    <circle cx="12" cy="12" r="1"></circle>
                    <circle cx="19" cy="12" r="1"></circle>
                    <circle cx="5" cy="12" r="1"></circle>
                  </svg>
                </button>
              </div>
            </div>
          </form>

          <!-- Search buttons -->
          <div class="flex justify-center flex-wrap gap-3 mt-8">
            <button
              class={cn(
                'px-4 py-2 text-sm font-medium rounded-md transition-colors',
                searchCategory === 'all'
                  ? 'bg-primary text-primary-foreground hover:bg-primary/90'
                  : 'bg-accent hover:bg-accent/80'
              )}
              on:click={() => setSearchCategory('all')}
            >
              Search Everything
            </button>
            <button
              class={cn(
                'px-4 py-2 text-sm font-medium rounded-md transition-colors',
                searchCategory === 'documents'
                  ? 'bg-primary text-primary-foreground hover:bg-primary/90'
                  : 'bg-accent hover:bg-accent/80'
              )}
              on:click={() => setSearchCategory('documents')}
            >
              Search Documents
            </button>
            <button
              class={cn(
                'px-4 py-2 text-sm font-medium rounded-md transition-colors',
                searchCategory === 'trackers'
                  ? 'bg-primary text-primary-foreground hover:bg-primary/90'
                  : 'bg-accent hover:bg-accent/80'
              )}
              on:click={() => setSearchCategory('trackers')}
            >
              Search Trackers
            </button>
            <button
              class={cn(
                'px-4 py-2 text-sm font-medium rounded-md transition-colors',
                searchCategory === 'apis'
                  ? 'bg-primary text-primary-foreground hover:bg-primary/90'
                  : 'bg-accent hover:bg-accent/80'
              )}
              on:click={() => setSearchCategory('apis')}
            >
              Search APIs
            </button>
          </div>

          {#if isSearching}
            <!-- Loading spinner -->
            <div class="flex justify-center mt-12">
              <div
                class="w-6 h-6 border-2 border-t-primary border-r-transparent border-b-transparent border-l-transparent rounded-full animate-spin"
              ></div>
              <span class="ml-3 text-muted-foreground">Searching...</span>
            </div>
          {:else if searchQuery && searchResults.length === 0}
            <!-- No results view -->
            <div class="text-center mt-12">
              <p class="text-muted-foreground">No results found for "{searchQuery}"</p>
              <p class="text-sm text-muted-foreground mt-2">
                Try different keywords or search in another category
              </p>
            </div>
          {:else if !searchQuery}
            <!-- Category cards when no search is active -->
            <div class="grid grid-cols-2 md:grid-cols-3 gap-4 mt-12">
              <button
                on:click={() => setSearchCategory('documents')}
                class="flex flex-col items-center justify-center p-6 bg-card border border-border rounded-lg hover:border-primary/50 hover:shadow-sm transition-all"
              >
                <FileText size={24} class="text-primary mb-2" />
                <span class="text-sm font-medium">Documents</span>
              </button>
              <button
                on:click={() => setSearchCategory('trackers')}
                class="flex flex-col items-center justify-center p-6 bg-card border border-border rounded-lg hover:border-primary/50 hover:shadow-sm transition-all"
              >
                <AppWindow size={24} class="text-primary mb-2" />
                <span class="text-sm font-medium">Trackers</span>
              </button>
              <button
                on:click={() => setSearchCategory('apis')}
                class="flex flex-col items-center justify-center p-6 bg-card border border-border rounded-lg hover:border-primary/50 hover:shadow-sm transition-all"
              >
                <Globe size={24} class="text-primary mb-2" />
                <span class="text-sm font-medium">APIs</span>
              </button>
            </div>
          {/if}
        </div>
      </div>
    {/if}
  </main>

  {#if showThreadSidebar}
    <aside
      class="fixed right-0 top-8 bottom-0 w-[300px] bg-background border-l border-border overflow-y-auto flex flex-col"
      aria-label="Thread sidebar"
    >
      <header class="p-3 border-b border-border flex justify-between items-center">
        <h2 class="text-sm font-medium">Thread</h2>
        <button
          class="p-1 rounded-md hover:bg-accent/50"
          on:click={closeThreadSidebar}
          aria-label="Close thread sidebar"
        >
          &times
        </button>
      </header>

      {#if activeThread}
        <section class="p-3 border-b border-border" aria-label="Original message">
          <article class="flex items-start gap-2 mb-2">
            <div
              class="flex-shrink-0 w-8 h-8 bg-primary/10 rounded-full flex items-center justify-center text-sm font-medium text-primary"
              aria-hidden="true"
            >
              {activeThread.sender.avatar}
            </div>
            <div>
              <div class="flex items-baseline gap-2">
                <span class="font-medium text-sm">{activeThread.sender.name}</span>
                <time class="text-xs text-muted-foreground">{activeThread.timestamp}</time>
              </div>
              <p class="mt-1 text-sm text-foreground">
                {activeThread.text}
              </p>
            </div>
          </article>
        </section>

        <section class="flex-1 p-3 overflow-y-auto" aria-label="Thread replies">
          {#if activeThread.replies && activeThread.replies.length > 0}
            <ul class="list-none p-0 m-0">
              {#each activeThread.replies as reply (reply.id)}
                <li class="py-2 flex items-start gap-2">
                  <div
                    class="flex-shrink-0 w-6 h-6 bg-secondary/80 rounded-full flex items-center justify-center text-xs font-medium text-secondary-foreground"
                    aria-hidden="true"
                  >
                    {reply.sender.avatar}
                  </div>
                  <div class="flex-1 min-w-0">
                    <div class="flex items-baseline gap-2">
                      <span class="font-medium text-xs">{reply.sender.name}</span>
                      <time class="text-[10px] text-muted-foreground">{reply.timestamp}</time>
                    </div>
                    <p class="mt-0.5 text-xs text-foreground">
                      {reply.text}
                    </p>
                  </div>
                </li>
              {/each}
            </ul>
          {:else}
            <p class="text-center text-sm text-muted-foreground py-4">No replies yet</p>
          {/if}
        </section>

        <footer class="p-3 border-t border-border mt-auto">
          <form class="flex gap-2" on:submit|preventDefault={sendThreadReply}>
            <label for="thread-reply" class="sr-only">Reply in thread</label>
            <input
              id="thread-reply"
              type="text"
              class="flex-1 px-3 py-2 rounded-md border border-border bg-background text-foreground text-sm focus:outline-none focus:border-primary"
              placeholder="Reply in thread..."
              bind:value={threadReplyText}
              on:keydown={handleThreadReplyKeydown}
              aria-label="Reply in thread"
            />
            <button
              type="submit"
              class="px-3 py-2 rounded-md border-none bg-primary text-primary-foreground text-sm font-medium cursor-pointer hover:bg-primary/90"
              disabled={!threadReplyText.trim()}
              aria-label="Send reply"
            >
              Send
            </button>
          </form>
        </footer>
      {/if}
    </aside>
  {/if}
{/if}

<Toaster />

<style>
  /* Simple float animation for logo */
  @keyframes float {
    0% {
      transform: translateY(0px);
    }
    50% {
      transform: translateY(-10px);
    }
    100% {
      transform: translateY(0px);
    }
  }

  :global(.animate-float) {
    animation: float 3s ease-in-out infinite;
  }
</style>
