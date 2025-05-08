<script lang="ts">
  import { onMount } from 'svelte'
  import { createEventDispatcher } from 'svelte'
  import { cn } from '../lib/utils'
  import { toast } from '../lib/toast'
  import type { TrackerListItem } from '../../../shared/types'
  import {
    X,
    Search,
    MessageSquare,
    Github,
    Mail,
    FileText,
    Headphones,
    Download,
    Package,
    AppWindow,
    ShieldCheck,
    Star,
    StarHalf,
    Filter,
    Sparkles,
    ArrowDownCircle
  } from 'lucide-svelte'

  // Modal props
  export let showModal = false

  // Create event dispatcher for modal events
  const dispatch = createEventDispatcher()

  // State variables for filtering and sorting
  let searchQuery = ''
  let selectedCategory = 'All'
  let selectedFilter = 'Featured'
  let isInstalling = false
  let currentTrackerId = null

  // Tracker data state
  let trackers: (TrackerListItem & { downloads: number; rating: number })[] = []
  let isLoading = true
  let loadError = ''

  // Fetch trackers from backend
  async function fetchTrackers() {
    try {
      isLoading = true
      loadError = ''

      const response = await window.api.trackerMarketplace.getTrackerList()

      if (response.success && response.trackers) {
        // Add download counts and ratings for sorting (these would ideally come from backend)
        trackers = response.trackers.map((tracker) => ({
          ...tracker, // Make sure to use the id from the response
          downloads: Math.floor(Math.random() * 10000),
          rating: 3.5 + Math.random() * 1.5
        }))
      } else {
        loadError = response.error || 'Failed to load trackers'
        console.error('Error loading trackers:', loadError)
      }
    } catch (error) {
      loadError = error.message || 'An unexpected error occurred'
      console.error('Exception while loading trackers:', error)
    } finally {
      isLoading = false
    }
  }

  // Get the icon URL for a tracker
  function getTrackerIconUrl(tracker) {
    if (!tracker || !tracker.iconPath) {
      return null
    }

    // Check if it's a path to our locally saved icons
    if (tracker.iconPath.startsWith('tracker-icons/')) {
      const iconName = tracker.iconPath.split('/').pop()
      return `../../../resources/${tracker.iconPath}`
    }

    // Fall back to GitHub URL for backward compatibility
    const url = `https://raw.githubusercontent.com/IonesioJunior/trackers/refs/heads/main/${tracker.iconPath}`
    return url
  }

  // Get appropriate icon component based on tracker name
  function getTrackerIcon(tracker) {
    if (!tracker) return AppWindow

    const name = tracker.name.toLowerCase()

    if (name.includes('mail') || name.includes('gmail')) {
      return Mail
    } else if (name.includes('document')) {
      return FileText
    } else if (name.includes('calendar')) {
      return MessageSquare
    } else if (name.includes('sheet') || name.includes('spreadsheet')) {
      return FileText
    } else if (name.includes('repository') || name.includes('github')) {
      return Github
    } else if (name.includes('asana')) {
      return Package
    } else {
      return AppWindow
    }
  }

  // Extract categories from descriptions
  $: categories = extractCategories(trackers)

  function extractCategories(trackersList) {
    const categorySet = new Set()
    trackersList.forEach((tracker) => {
      const matches = tracker.description.match(/\((.*?)\)/g) || []
      if (matches.length > 0) {
        matches.forEach((m) => {
          const category = m.replace(/[()]/g, '').trim()
          if (category) categorySet.add(category)
        })
      }
    })
    return ['All', ...Array.from(categorySet).sort()]
  }

  // Filter options
  const filterOptions = ['Featured', 'Most Popular', 'Newest', 'Highest Rated']

  // Format download count
  function formatDownloads(count: number): string {
    if (count >= 1000) {
      return `${(count / 1000).toFixed(1)}k`
    }
    return count.toString()
  }

  // Helper function to extract tags from description
  function extractTags(description: string): string[] {
    const tags: string[] = []
    const words = description.toLowerCase().match(/\b\w+\b/g) || []
    const commonTags = [
      'google',
      'document',
      'calendar',
      'sheet',
      'email',
      'task',
      'repository',
      'code'
    ]

    for (const word of words) {
      if (commonTags.includes(word) && !tags.includes(word)) {
        tags.push(word)
      }
    }

    return tags
  }

  // Filter trackers based on search query and selected category
  $: filteredTrackers = trackers
    .filter((tracker) => {
      const tags = extractTags(tracker.description)

      const matchesSearch =
        searchQuery === '' ||
        tracker.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        tracker.description.toLowerCase().includes(searchQuery.toLowerCase()) ||
        tags.some((tag) => tag.includes(searchQuery.toLowerCase()))

      // Extract category from description or use 'Miscellaneous'
      const trackerCategory = tracker.description.match(/\((.*?)\)/)?.[1]?.trim() || 'Miscellaneous'

      const matchesCategory = selectedCategory === 'All' || trackerCategory === selectedCategory

      return matchesSearch && matchesCategory
    })
    .sort((a, b) => {
      // Sort based on selected filter
      if (selectedFilter === 'Featured') {
        // Featured first, then verified
        if (a.featured && !b.featured) return -1
        if (!a.featured && b.featured) return 1
        if (a.verified && !b.verified) return -1
        if (!a.verified && b.verified) return 1
        return a.name.localeCompare(b.name)
      } else if (selectedFilter === 'Most Popular') {
        // Sort by our simulated download counts
        return b.downloads - a.downloads
      } else if (selectedFilter === 'Highest Rated') {
        // Sort by our simulated ratings
        return b.rating - a.rating
      } else if (selectedFilter === 'Newest') {
        // Use version as a proxy
        const aVersion = a.version.split('.').map(Number)
        const bVersion = b.version.split('.').map(Number)

        for (let i = 0; i < Math.max(aVersion.length, bVersion.length); i++) {
          const aVal = i < aVersion.length ? aVersion[i] : 0
          const bVal = i < bVersion.length ? bVersion[i] : 0
          if (aVal !== bVal) {
            return bVal - aVal
          }
        }
        return 0
      }
      return 0
    })

  // Close modal and dispatch event
  function closeModal() {
    showModal = false
    dispatch('close')
  }

  // Track installed trackers
  let installedTrackerIds = []

  // Check which trackers are already installed
  async function checkInstalledTrackers() {
    try {
      const response = await window.api.apps.getAppTrackers()

      if (response.success && response.appTrackers) {
        // The id field is the folder name of the tracker
        installedTrackerIds = response.appTrackers.map((app) => {
          // Extract just the folder name from app.path if it exists
          let trackerId = app.id
          if (app.path) {
            const pathParts = app.path.split('/')
            // Use the last part of the path as the id for comparison
            const folderName = pathParts[pathParts.length - 1]
            trackerId = folderName
          }
          return trackerId
        })
      }
    } catch (error) {
      console.error('Failed to get installed trackers:', error)
    }
  }

  // Install tracker
  async function installTracker(tracker) {
    try {
      // Validation check
      if (!tracker || !tracker.id) {
        throw new Error('Invalid tracker data')
      }

      // Check if already installed
      const isInstalled = installedTrackerIds.includes(tracker.id)

      if (isInstalled) {
        return false
      }

      isInstalling = true
      currentTrackerId = tracker.id

      // Extra validation for the tracker ID
      if (!tracker.id || typeof tracker.id !== 'string') {
        throw new Error(
          `Invalid tracker ID: ${JSON.stringify(tracker.id)}. Expected a non-empty string.`
        )
      }

      // Call the backend API to install the tracker
      const result = await window.api.trackerMarketplace.installTracker(tracker.id)

      if (!result.success) {
        throw new Error(result.error || 'Failed to install tracker')
      }

      // Add to installed trackers list
      installedTrackerIds = [...installedTrackerIds, tracker.id]

      // Show success toast with detailed information
      toast.success(`${tracker.name} has been successfully installed and is ready to use.`, {
        title: 'Tracker Installed',
        duration: 4000
      })

      // Close modal and dispatch event with the installed tracker data
      closeModal()
      dispatch('installed', {
        tracker,
        installResult: result
      })

      return true
    } catch (error) {
      console.error('Failed to install tracker:', error)

      let errorMessage = 'An unexpected error occurred.'

      // Provide more specific error messages based on the error type
      if (
        error.message.includes('Connection refused') ||
        error.message.includes('Connection timed out')
      ) {
        errorMessage =
          'Could not connect to the tracker server. Please check your network connection and try again.'
      } else if (error.message.includes('not found')) {
        errorMessage = `Tracker "${tracker.name}" is not available on the server. It may have been removed.`
      } else if (error.message.includes('SyftBox configuration')) {
        errorMessage = 'System configuration issue. Please contact the administrator.'
      } else if (error.message) {
        // Use the actual error message if available
        errorMessage = error.message
      }

      // Show error toast with more specific information
      toast.error(errorMessage, {
        title: 'Installation Failed',
        duration: 5000
      })

      return false
    } finally {
      isInstalling = false
      currentTrackerId = null
    }
  }

  // Reset search and filters
  function resetFilters() {
    searchQuery = ''
    selectedCategory = 'All'
    selectedFilter = 'Featured'
  }

  // Handle click outside modal content to close
  function handleOutsideClick(event) {
    if (event.target === event.currentTarget) {
      closeModal()
    }
  }

  // Handle keydown for accessibility
  function handleKeydown(event) {
    if (event.key === 'Escape') {
      closeModal()
    }
  }

  // Fetch trackers when the component mounts
  onMount(() => {
    // First check which trackers are already installed
    checkInstalledTrackers().then(() => {
      // Then fetch trackers from backend
      fetchTrackers()
    })
  })
</script>

<!-- Svelte component markup with proper styling -->
{#if showModal}
  <!-- Modal backdrop -->
  <div
    class="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4"
    on:click={handleOutsideClick}
    on:keydown={handleKeydown}
    role="dialog"
    aria-modal="true"
    aria-labelledby="tracker-marketplace-title"
  >
    <!-- Modal content -->
    <div
      class="bg-card border border-border rounded-lg shadow-lg w-full max-w-5xl max-h-[90vh] flex flex-col overflow-hidden"
      role="document"
    >
      <!-- Modal header -->
      <div class="p-4 border-b border-border flex justify-between items-center">
        <h2
          id="tracker-marketplace-title"
          class="text-xl font-semibold text-foreground flex items-center gap-2"
        >
          <AppWindow size={20} />
          <span>Tracker Marketplace</span>
        </h2>
        <button
          class="h-8 w-8 rounded-md flex items-center justify-center text-muted-foreground hover:text-foreground hover:bg-muted/80 transition-colors"
          on:click={closeModal}
          aria-label="Close"
        >
          <X size={18} />
        </button>
      </div>

      <!-- Modal content - with custom scrollbar styling -->
      <div class="p-6 flex-1 overflow-auto custom-scrollbar">
        <!-- Search and filters -->
        <div class="flex flex-col md:flex-row gap-4 mb-6">
          <!-- Search input -->
          <div class="relative flex-1">
            <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
              <Search size={16} class="text-muted-foreground" />
            </div>
            <input
              type="text"
              placeholder="Search trackers..."
              bind:value={searchQuery}
              class="pl-10 pr-4 py-2 w-full rounded-md border border-input bg-background text-foreground text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary"
            />
          </div>

          <!-- Category filter -->
          <div class="relative w-full md:w-48">
            <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
              <Filter size={16} class="text-muted-foreground" />
            </div>
            <select
              bind:value={selectedCategory}
              class="pl-10 pr-4 py-2 w-full rounded-md border border-input bg-background text-foreground text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary appearance-none"
            >
              {#each categories as category}
                <option value={category}>{category}</option>
              {/each}
            </select>
            <div class="absolute inset-y-0 right-0 pr-3 flex items-center pointer-events-none">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                class="text-muted-foreground"><path d="m6 9 6 6 6-6" /></svg
              >
            </div>
          </div>

          <!-- Sort filter -->
          <div class="relative w-full md:w-48">
            <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                class="text-muted-foreground"
                ><path d="M11 5h10" /><path d="M11 9h7" /><path d="M11 13h4" /><path
                  d="m3 17 3 3 3-3"
                /><path d="M6 5v15" /></svg
              >
            </div>
            <select
              bind:value={selectedFilter}
              class="pl-10 pr-4 py-2 w-full rounded-md border border-input bg-background text-foreground text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary appearance-none"
            >
              {#each filterOptions as option}
                <option value={option}>{option}</option>
              {/each}
            </select>
            <div class="absolute inset-y-0 right-0 pr-3 flex items-center pointer-events-none">
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                class="text-muted-foreground"><path d="m6 9 6 6 6-6" /></svg
              >
            </div>
          </div>
        </div>

        <!-- Loading state -->
        {#if isLoading}
          <div class="flex flex-col items-center justify-center py-10">
            <div
              class="w-12 h-12 border-4 border-t-transparent border-primary rounded-full animate-spin mb-4"
            ></div>
            <p class="text-muted-foreground">Loading trackers...</p>
          </div>
          <!-- Error state (complete failure) -->
        {:else if loadError}
          <div class="flex flex-col items-center justify-center py-10 text-center">
            <div
              class="w-16 h-16 rounded-full bg-destructive/10 flex items-center justify-center mb-4"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="28"
                height="28"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
                class="text-destructive"
              >
                <circle cx="12" cy="12" r="10"></circle><line x1="12" y1="8" x2="12" y2="12"
                ></line><line x1="12" y1="16" x2="12.01" y2="16"></line>
              </svg>
            </div>
            <h3 class="text-lg font-medium text-foreground mb-2">Failed to load trackers</h3>
            <p class="text-muted-foreground max-w-md mb-6">{loadError}</p>
            <button
              class="flex items-center gap-2 px-4 py-2 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
              on:click={fetchTrackers}
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"></path>
                <path d="M3 3v5h5"></path>
              </svg>
              <span>Try Again</span>
            </button>
          </div>
          <!-- Content -->
        {:else}
          <!-- Featured trackers carousel (only shown if there are featured trackers) -->
          {#if selectedFilter === 'Featured' && filteredTrackers.some((t) => t.featured)}
            <div class="mb-8">
              <h3 class="text-lg font-medium text-foreground mb-4 flex items-center gap-2">
                <Sparkles size={18} class="text-primary" />
                <span>Developed by OpenMined</span>
              </h3>

              <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                {#each filteredTrackers.filter((t) => t.featured) as tracker (tracker.id)}
                  <div
                    class="bg-primary/5 border border-primary/20 rounded-lg p-4 flex gap-4 hover:shadow-md transition-shadow"
                  >
                    <!-- Tracker icon -->
                    <div
                      class="flex-shrink-0 w-12 h-12 bg-primary/10 rounded-full flex items-center justify-center text-primary"
                    >
                      <svelte:component this={getTrackerIcon(tracker)} size={24} />
                    </div>

                    <!-- Tracker details -->
                    <div class="flex-1">
                      <div class="flex items-center justify-between">
                        <div>
                          <h4 class="font-medium text-foreground">{tracker.name}</h4>
                          <p class="text-xs text-primary font-medium -mt-0.5">by OpenMined</p>
                        </div>
                        {#if tracker.verified}
                          <div
                            class="flex items-center text-xs text-emerald-600 dark:text-emerald-500 font-medium"
                          >
                            <ShieldCheck size={14} class="mr-1" />
                            <span>Verified</span>
                          </div>
                        {/if}
                      </div>

                      <p class="text-sm text-muted-foreground mt-1 line-clamp-2">
                        {tracker.description}
                      </p>

                      <div class="flex justify-between items-center mt-2">
                        <!-- Version information -->
                        <div class="flex items-center gap-1 text-xs text-muted-foreground">
                          <span>v{tracker.version}</span>
                        </div>

                        <button
                          class={cn(
                            'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium',
                            isInstalling && currentTrackerId === tracker.id
                              ? 'bg-primary/70 text-primary-foreground cursor-not-allowed'
                              : installedTrackerIds.includes(tracker.id)
                                ? 'bg-muted text-muted-foreground cursor-not-allowed'
                                : 'bg-primary text-primary-foreground hover:bg-primary/90',
                            'transition-colors'
                          )}
                          on:click={() => installTracker(tracker)}
                          disabled={isInstalling || installedTrackerIds.includes(tracker.id)}
                        >
                          {#if isInstalling && currentTrackerId === tracker.id}
                            <div
                              class="h-3 w-3 border-2 border-t-transparent border-white rounded-full animate-spin"
                            ></div>
                            <span>Installing...</span>
                          {:else if installedTrackerIds.includes(tracker.id)}
                            <span>Installed</span>
                          {:else}
                            <Download size={14} />
                            <span>Install</span>
                          {/if}
                        </button>
                      </div>
                    </div>
                  </div>
                {/each}
              </div>
            </div>
          {/if}

          <!-- All trackers grid -->
          <div>
            <div class="flex justify-between items-center mb-4">
              <h3 class="text-lg font-medium text-foreground">
                {#if searchQuery || selectedCategory !== 'All' || selectedFilter !== 'Featured'}
                  Search Results
                {:else}
                  All Trackers
                {/if}
              </h3>

              {#if searchQuery || selectedCategory !== 'All' || selectedFilter !== 'Featured'}
                <button
                  class="text-xs text-primary hover:text-primary/80 flex items-center gap-1"
                  on:click={resetFilters}
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="14"
                    height="14"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    ><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8" /><path
                      d="M3 3v5h5"
                    /></svg
                  >
                  Reset Filters
                </button>
              {/if}
            </div>

            {#if filteredTrackers.filter((t) => !t.featured || selectedFilter !== 'Featured').length === 0}
              <div
                class="flex flex-col items-center justify-center bg-card border border-border rounded-lg p-10 text-center min-h-[200px]"
              >
                <div
                  class="w-16 h-16 bg-muted rounded-full flex items-center justify-center text-muted-foreground mb-4"
                >
                  <Search size={24} />
                </div>
                <h3 class="text-xl font-medium text-foreground mb-2">No trackers found</h3>
                <p class="text-muted-foreground max-w-md">
                  We couldn't find any trackers matching your search criteria. Try adjusting your
                  filters or search query.
                </p>
                <button
                  class="mt-4 text-primary hover:text-primary/80 flex items-center gap-1 text-sm"
                  on:click={() => {
                    resetFilters()
                    fetchTrackers()
                  }}
                >
                  <svg
                    xmlns="http://www.w3.org/2000/svg"
                    width="14"
                    height="14"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="2"
                    stroke-linecap="round"
                    stroke-linejoin="round"
                  >
                    <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"></path>
                    <path d="M3 3v5h5"></path>
                  </svg>
                  Refresh results
                </button>
              </div>
            {:else}
              <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {#each filteredTrackers.filter((t) => !t.featured || selectedFilter !== 'Featured') as tracker (tracker.id)}
                  <div
                    class="bg-card border border-border rounded-lg p-4 flex flex-col h-[180px] hover:shadow-md transition-shadow"
                  >
                    <div class="flex gap-3 mb-3">
                      <!-- Tracker icon -->
                      <div
                        class="flex-shrink-0 w-10 h-10 bg-primary/10 rounded-full flex items-center justify-center text-primary"
                      >
                        <svelte:component this={getTrackerIcon(tracker)} size={20} />
                      </div>

                      <!-- Tracker name and developer -->
                      <div class="flex-1">
                        <div class="flex items-center">
                          <h4 class="font-medium text-foreground">{tracker.name}</h4>
                          {#if tracker.verified}
                            <div
                              class="ml-2 text-emerald-600 dark:text-emerald-500"
                              title="Verified"
                            >
                              <ShieldCheck size={14} />
                            </div>
                          {/if}
                        </div>
                        <p class="text-xs text-muted-foreground">
                          by {tracker.developer}
                        </p>
                      </div>
                    </div>

                    <!-- Tracker description -->
                    <p class="text-sm text-muted-foreground mb-3 line-clamp-2 flex-1">
                      {tracker.description}
                    </p>

                    <!-- Tracker stats and install button -->
                    <div class="flex items-center justify-between mt-auto">
                      <div class="flex items-center gap-4 text-xs text-muted-foreground">
                        <!-- Only show version -->
                        <div>
                          <span class="text-xs">v{tracker.version}</span>
                        </div>
                      </div>

                      <button
                        class={cn(
                          'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium',
                          isInstalling && currentTrackerId === tracker.id
                            ? 'bg-primary/70 text-primary-foreground cursor-not-allowed'
                            : installedTrackerIds.includes(tracker.id)
                              ? 'bg-muted text-muted-foreground cursor-not-allowed'
                              : 'bg-primary text-primary-foreground hover:bg-primary/90',
                          'transition-colors'
                        )}
                        on:click={() => installTracker(tracker)}
                        disabled={isInstalling || installedTrackerIds.includes(tracker.id)}
                      >
                        {#if isInstalling && currentTrackerId === tracker.id}
                          <div
                            class="h-3 w-3 border-2 border-t-transparent border-white rounded-full animate-spin"
                          ></div>
                          <span>Installing...</span>
                        {:else if installedTrackerIds.includes(tracker.id)}
                          <span>Installed</span>
                        {:else}
                          <Download size={14} />
                          <span>Install</span>
                        {/if}
                      </button>
                    </div>
                  </div>
                {/each}
              </div>
            {/if}
          </div>
        {/if}
      </div>

      <!-- Modal footer with optional information about how to create custom trackers -->
      <div class="p-4 border-t border-border bg-muted/40">
        <div class="flex items-center justify-between text-sm text-muted-foreground">
          <div class="flex items-center">
            <span>Showing {filteredTrackers.length} of {trackers.length} available trackers</span>
          </div>
          <button
            class="text-primary hover:text-primary/80 hover:underline"
            on:click={() => fetchTrackers()}
          >
            Refresh results
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}
