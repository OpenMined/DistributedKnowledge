<script lang="ts">
  import { cn } from '../lib/utils'
  import { onMount } from 'svelte'
  import { toasts } from '../lib/stores/toast'
  import { toast } from '../lib/toast'
  // Import icons for apps and toggle
  import {
    ToggleLeft,
    ToggleRight,
    MessageSquare,
    Github,
    Mail,
    FileText,
    Headphones,
    PlusCircle,
    ArrowUpCircle,
    MoreVertical,
    Settings,
    Trash2,
    Eye
  } from 'lucide-svelte'

  // Import TrackerView component
  import TrackerView from './TrackerView.svelte'
  import TrackerConfigModal from './TrackerConfigModal.svelte'
  import TrackerMarketplaceModal from './TrackerMarketplaceModal.svelte'

  // Data from electron backend
  $: documentCount = 0
  $: apps = []
  $: loading = true
  $: errorMessage = ''

  // Track which app has an open dropdown menu
  let activeDropdownId: string | number | null = null

  // Track which tracker is currently being viewed
  let selectedTrackerId: string | null = null

  // Track the config modal state
  let showConfigModal = false
  let configTrackerId: string | null = null

  // Track the marketplace modal state
  let showMarketplaceModal = false

  // Map of icon names to components
  const iconMap = {
    MessageSquare: MessageSquare,
    Github: Github,
    Mail: Mail,
    FileText: FileText,
    Headphones: Headphones
  }

  // Function to load app-specific icon
  async function getAppIcon(appId: string, appPath?: string): Promise<string | null> {
    try {
      // Try to find icon.svg in the app folder
      const iconPath = await window.api.apps.getAppIconPath(appId, appPath)
      return iconPath
    } catch (error) {
      return null
    }
  }

  // Function to refresh app tracker data
  async function refreshAppTrackers() {
    try {
      loading = true
      errorMessage = ''

      // Get document count from backend
      const docCountResponse = await window.api.apps.getDocumentCount()
      if (docCountResponse.success && docCountResponse.stats) {
        documentCount = docCountResponse.stats.count
        // Store error message if it exists
        if (docCountResponse.stats.error) {
          errorMessage = docCountResponse.stats.error
        }
      }

      // Get app trackers from backend
      const appTrackersResponse = await window.api.apps.getAppTrackers()
      if (appTrackersResponse.success && appTrackersResponse.appTrackers) {
        // Get app trackers from backend and load custom icons where available
        const appTrackers = appTrackersResponse.appTrackers

        // Process each app to load custom icons or fallback to icon from metadata
        apps = await Promise.all(
          appTrackers.map(async (app) => {
            // Try to get custom icon.svg from the app's directory, passing app.path to avoid another call to getAppTrackers
            const customIconPath = await getAppIcon(app.id, app.path)

            if (customIconPath) {
              // If custom icon found, use it
              return {
                ...app,
                customIconPath,
                icon: iconMap[app.icon] || MessageSquare // Still keep the backup component icon
              }
            } else {
              // Otherwise fallback to the icon component from metadata
              return {
                ...app,
                icon: iconMap[app.icon] || MessageSquare // Default to MessageSquare if icon not found
              }
            }
          })
        )
      }
    } catch (error) {
      errorMessage = 'Failed to load application data'
    } finally {
      loading = false
    }
  }

  // Load data on mount
  onMount(async () => {
    await refreshAppTrackers()

    // Set up interval to fetch data periodically without forcing refresh
    const fetchInterval = setInterval(async () => {
      try {
        // Get document count
        const docCountResponse = await window.api.apps.getDocumentCount()
        if (docCountResponse.success && docCountResponse.stats) {
          documentCount = docCountResponse.stats.count
          errorMessage = docCountResponse.stats.error || ''
        }

        // Get app trackers
        const appTrackersResponse = await window.api.apps.getAppTrackers()
        if (appTrackersResponse.success && appTrackersResponse.appTrackers) {
          // Process each app to load custom icons or fallback to icon from metadata
          const updatedApps = await Promise.all(
            appTrackersResponse.appTrackers.map(async (app) => {
              // Try to get custom icon.svg from the app's directory, passing app.path to avoid another call to getAppTrackers
              const customIconPath = await getAppIcon(app.id, app.path)

              return {
                ...app,
                ...(customIconPath ? { customIconPath } : {}),
                icon: iconMap[app.icon] || MessageSquare
              }
            })
          )
          apps = updatedApps
        }
      } catch (error) {
        // Error handled in finally block
      }
    }, 3000)

    // Add click outside listener for dropdown menus
    document.addEventListener('click', handleClickOutside)

    // Clean up interval and event listener on component unmount
    return () => {
      clearInterval(fetchInterval)
      document.removeEventListener('click', handleClickOutside)
    }
  })

  // Toggle app enabled state
  async function toggleApp(id) {
    try {
      const response = await window.api.apps.toggleAppTracker(id)

      if (response.success && response.appTracker) {
        // Update local app data with the response
        const appName = apps.find((app) => app.id === id)?.name || 'App'
        const isEnabled = response.appTracker.enabled

        apps = apps.map((app) =>
          app.id === id
            ? {
                ...app,
                enabled: isEnabled
              }
            : app
        )

        // Show toast notification based on the toggle state
        toasts.add({
          type: 'success',
          title: isEnabled ? 'App Enabled' : 'App Disabled',
          message: isEnabled
            ? `RAG is now considering documents from ${appName}`
            : `RAG is now ignoring documents from ${appName}`,
          duration: 4000
        })
      } else if (!response.success) {
        // Show error toast with the specific error message
        toasts.add({
          type: 'error',
          title: 'Toggle Failed',
          message: response.error || 'Failed to toggle app status. Please try again.',
          duration: 4000
        })
      }
    } catch (error) {
      // Show error toast
      toasts.add({
        type: 'error',
        title: 'Toggle Failed',
        message: 'Failed to toggle app status. Please try again.',
        duration: 4000
      })
    }
  }

  // Toggle dropdown menu visibility
  function toggleDropdown(appId: string | number, event: MouseEvent) {
    event.stopPropagation() // Prevent event from bubbling up
    activeDropdownId = activeDropdownId === appId ? null : appId
  }

  // Close dropdown when clicking outside
  function handleClickOutside(event: MouseEvent) {
    if (activeDropdownId !== null) {
      activeDropdownId = null
    }
  }

  // Uninstall app
  async function handleUninstallApp(id) {
    try {
      // Only allow uninstallation for disabled apps
      const app = apps.find((a) => a.id === id)
      if (app && app.enabled) {
        toasts.add({
          type: 'warning',
          title: 'Action Required',
          message: 'Please disable the app before uninstalling.',
          duration: 4000
        })
        return
      }

      // Call the backend to uninstall the app
      const result = await window.api.apps.uninstallAppTracker(id)

      if (result.success) {
        // Close the dropdown and update local app state
        activeDropdownId = null

        // Get the app name before removing it
        const appName = apps.find((app) => app.id === id)?.name || 'App'

        // Remove the uninstalled app from our local state
        apps = apps.filter((app) => app.id !== id)

        // Show success toast notification
        toasts.add({
          type: 'success',
          title: 'App Uninstalled',
          message: `${appName} has been successfully uninstalled`,
          duration: 4000
        })
      } else {
        // Show error toast instead of alert
        toasts.add({
          type: 'error',
          title: 'Uninstall Failed',
          message: result.message || 'Failed to uninstall app',
          duration: 4000
        })
      }
    } catch (error) {
      // Show error toast for the error
      toasts.add({
        type: 'error',
        title: 'Uninstall Failed',
        message: 'An error occurred while trying to uninstall the app',
        duration: 4000
      })
    }
  }

  // Configure app
  function handleConfigureApp(id) {
    // Reset modal state first
    showConfigModal = false

    // Use setTimeout to ensure DOM updates before showing modal
    setTimeout(() => {
      // Then set the tracker ID and show the modal
      configTrackerId = id
      showConfigModal = true
      activeDropdownId = null
    }, 0)
  }

  // Handle config modal close
  function handleConfigModalClose() {
    // Reset both the visibility and the tracker ID
    showConfigModal = false
    configTrackerId = null
  }

  // Handle config updated
  function handleConfigUpdated() {
    // Refresh the app trackers to get updated data
    refreshAppTrackers()
  }

  // View tracker details
  function handleViewTracker(id) {
    // Make sure we have the latest data for this tracker
    const selectedTracker = apps.find((app) => app.id === id)
    selectedTrackerId = id
    activeDropdownId = null
  }

  // Return to trackers list from the tracker view
  function handleBackToTrackers() {
    selectedTrackerId = null
  }

  // Handle install app button click - now opens the marketplace modal
  function handleInstallApp() {
    showMarketplaceModal = true
  }

  // Handle tracker installation from marketplace
  async function handleTrackerInstalled(event) {
    try {
      const { tracker, installResult } = event.detail

      // Show a loading notification while we refresh the trackers list
      const loadingToast = toasts.add({
        type: 'default',
        title: 'Refreshing',
        message: 'Updating tracker list...',
        duration: 10000 // Long duration that we'll manually dismiss
      })

      // Refresh the app trackers to show the newly installed tracker
      await refreshAppTrackers()

      // Remove the loading notification
      toasts.remove(loadingToast)

      // Show additional information about what to do next
      setTimeout(() => {
        toasts.add({
          type: 'info',
          title: 'Next Steps',
          message: `Configure ${tracker.name} by clicking the three dots menu and selecting 'Configure'`,
          duration: 6000
        })
      }, 1000)

      return true
    } catch (error) {
      // Error completing tracker installation

      // Show error toast notification
      toasts.add({
        type: 'error',
        title: 'Refresh Failed',
        message:
          'The tracker was installed but there was an error refreshing the tracker list. Please restart the application.',
        duration: 6000
      })

      return false
    }
  }

  // Handle marketplace modal close
  function handleMarketplaceModalClose() {
    showMarketplaceModal = false
  }

  // Handle app update
  async function handleUpdateApp(id) {
    try {
      const result = await window.api.apps.updateAppTracker(id)
      if (result.success) {
        // Get app name for toast notification
        const app = apps.find((app) => app.id === id)
        const appName = app?.name || 'App'
        const oldVersion = app?.version || 'unknown'
        const newVersion = result.appTracker.version || 'latest version'

        // Update local app data to reflect the update
        apps = apps.map((app) =>
          app.id === id
            ? {
                ...app,
                version: result.appTracker.version,
                hasUpdate: false
              }
            : app
        )

        // Show success toast
        toasts.add({
          type: 'success',
          title: 'App Updated',
          message: `${appName} has been updated from v${oldVersion} to v${newVersion}`,
          duration: 4000
        })
      }
    } catch (error) {
      // Handle update failure
      toasts.add({
        type: 'error',
        title: 'Update Failed',
        message: 'Failed to update app. Please try again later.',
        duration: 4000
      })
    }
  }

  // Update document count directly
  async function refreshDocumentCount() {
    try {
      const docCountResponse = await window.api.apps.getDocumentCount()
      if (docCountResponse.success && docCountResponse.stats) {
        documentCount = docCountResponse.stats.count
        if (docCountResponse.stats.error) {
          errorMessage = docCountResponse.stats.error
        } else {
          errorMessage = ''
        }
      }
    } catch (error) {
      errorMessage = 'Failed to update document count'
    }
  }

  // Cleanup all documents
  async function cleanupDocuments() {
    try {
      const result = await window.api.apps.cleanupDocuments()

      if (result.success) {
        // Reset document count to 0
        documentCount = 0
        errorMessage = ''

        // Show success toast notification using helper
        toast.success(result.message || 'All documents have been successfully removed.', {
          title: 'Documents Removed',
          duration: 4000
        })

        // Show additional information toast about RAG impact
        setTimeout(() => {
          toast.info('RAG system will now start fresh with no document history.', {
            title: 'RAG Status Updated',
            duration: 5000
          })
        }, 1000)

        // Refresh document count from server to ensure UI is in sync
        await refreshDocumentCount()

        return true
      } else {
        errorMessage = result.error || result.message || 'Failed to cleanup documents'

        // Show error toast notification using helper
        toast.error(result.error || result.message || 'Failed to cleanup documents', {
          title: 'Cleanup Failed',
          duration: 4000
        })

        return false
      }
    } catch (error) {
      errorMessage = 'Failed to cleanup documents'

      // Show error toast notification using helper
      toast.error('An error occurred while trying to remove documents', {
        title: 'Cleanup Failed',
        duration: 4000
      })

      return false
    }
  }
</script>

<div class="flex flex-col h-full w-full bg-background">
  <!-- Header similar to other components -->
  <div class="p-4 border-b border-border bg-background">
    <h2 class="text-base font-semibold text-foreground">Trackers</h2>
  </div>

  {#if selectedTrackerId}
    <!-- TrackerView component when a tracker is selected -->
    {#key selectedTrackerId}
      <TrackerView
        trackerId={selectedTrackerId}
        currentTracker={apps.find((app) => app.id === selectedTrackerId)}
        onBackClick={handleBackToTrackers}
      />
    {/key}
  {:else}
    <!-- Main content area - trackers list -->
    <div class="flex-1 p-6 overflow-y-auto custom-scrollbar">
      {#if loading}
        <div class="flex justify-center items-center h-48">
          <div class="text-muted-foreground">Loading...</div>
        </div>
      {:else}
        <!-- Apps grid - 3 cards per row -->
        <div class="mt-8">
          <div class="flex justify-between items-center mb-4">
            <h3 class="text-lg font-medium text-foreground">Available Trackers</h3>

            <!-- Install Tracker button -->
            <button
              class={cn(
                'flex items-center gap-2 px-4 py-2 rounded-md bg-primary text-primary-foreground',
                'hover:bg-primary/90 transition-colors text-sm font-medium'
              )}
              on:click={handleInstallApp}
            >
              <PlusCircle size={16} />
              <span>Install Tracker</span>
            </button>
          </div>

          {#if apps.length === 0}
            <!-- Empty state message -->
            <div
              class="flex flex-col items-center justify-center bg-card border border-border rounded-lg p-10 text-center mt-6 min-h-[250px]"
            >
              <div
                class="w-16 h-16 bg-primary/10 rounded-full flex items-center justify-center text-primary mb-4"
              >
                <PlusCircle size={32} />
              </div>
              <h3 class="text-xl font-medium text-foreground mb-2">No trackers installed</h3>
              <p class="text-muted-foreground mb-6 max-w-md">
                Install your first tracker to enhance your RAG experience with external data
                sources.
              </p>
              <button
                class={cn(
                  'flex items-center gap-2 px-5 py-2.5 rounded-md bg-primary text-primary-foreground',
                  'hover:bg-primary/90 transition-colors font-medium'
                )}
                on:click={handleInstallApp}
              >
                <PlusCircle size={18} />
                <span>Install Your First Tracker</span>
              </button>
            </div>
          {:else}
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {#each apps as app (app.id)}
                <div
                  class="bg-card border border-border rounded-lg shadow-sm p-4 hover:shadow-md transition-shadow flex flex-col h-[200px]"
                >
                  <div class="flex justify-between mb-3 flex-shrink-0">
                    <!-- App icon and info -->
                    <div class="flex items-start gap-3">
                      <div
                        class="mt-0.5 flex-shrink-0 w-8 h-8 bg-primary/10 rounded-full flex items-center justify-center text-primary"
                      >
                        {#if app.customIconPath}
                          <img src={app.customIconPath} alt="{app.name} icon" class="w-6 h-6" />
                        {:else}
                          <svelte:component this={app.icon} size={18} />
                        {/if}
                      </div>
                      <div class="flex-1">
                        <h4 class="font-medium text-foreground">{app.name}</h4>
                        <p
                          class="text-sm text-muted-foreground mt-1 h-[60px] line-clamp-3 overflow-hidden"
                        >
                          {app.description}
                        </p>
                      </div>
                    </div>

                    <!-- Three dots menu in top-right corner -->
                    <div class="relative">
                      <button
                        class="p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted/80 transition-colors"
                        aria-label="App options"
                        on:click={(e) => toggleDropdown(app.id, e)}
                      >
                        <MoreVertical size={16} />
                      </button>

                      <!-- Dropdown menu -->
                      {#if activeDropdownId === app.id}
                        <div
                          class="absolute right-0 z-10 w-40 rounded-md shadow-lg bg-popover border border-border"
                          style="top: 2rem; right: 0;"
                          on:click|stopPropagation
                        >
                          <div class="py-1">
                            <button
                              class="flex items-center gap-2 w-full px-4 py-2 text-sm text-foreground hover:bg-muted/80 transition-colors"
                              on:click={() => handleViewTracker(app.id)}
                            >
                              <Eye size={16} />
                              View
                            </button>
                            <button
                              class="flex items-center gap-2 w-full px-4 py-2 text-sm text-foreground hover:bg-muted/80 transition-colors"
                              on:click={() => handleConfigureApp(app.id)}
                            >
                              <Settings size={16} />
                              Configure
                            </button>
                            <button
                              class="flex items-center gap-2 w-full px-4 py-2 text-sm text-destructive hover:bg-muted/80 transition-colors"
                              on:click={() => handleUninstallApp(app.id)}
                            >
                              <Trash2 size={16} />
                              Uninstall
                            </button>
                          </div>
                        </div>
                      {/if}
                    </div>
                  </div>

                  <!-- Version info and toggle in bottom row, separated -->
                  <div class="flex justify-between items-center mt-auto flex-shrink-0">
                    <div class="flex items-center gap-2">
                      <span class="text-xs text-muted-foreground">Version {app.version}</span>
                      {#if app.hasUpdate}
                        <button
                          class="inline-flex items-center justify-center px-2 py-0.5 text-xs font-medium rounded bg-primary/10 text-primary hover:bg-primary/20 transition-colors"
                          aria-label="Update available"
                          title="Update available"
                          on:click={() => handleUpdateApp(app.id)}
                        >
                          <ArrowUpCircle size={12} class="mr-1" />
                          Update
                        </button>
                      {/if}
                    </div>

                    <!-- Toggle button on the right -->
                    <button
                      class="text-lg focus:outline-none"
                      on:click={() => toggleApp(app.id)}
                      aria-label={app.enabled ? 'Disable app' : 'Enable app'}
                    >
                      {#if app.enabled}
                        <ToggleRight class="text-primary" size={28} />
                      {:else}
                        <ToggleLeft class="text-muted-foreground" size={28} />
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
  {/if}
</div>

<!-- Tracker Config Modal (placed at the top level of the document) -->
<TrackerConfigModal
  showModal={showConfigModal}
  trackerId={configTrackerId}
  on:close={handleConfigModalClose}
  on:configUpdated={handleConfigUpdated}
/>

<!-- Tracker Marketplace Modal -->
{#if showMarketplaceModal}
  <TrackerMarketplaceModal
    showModal={showMarketplaceModal}
    on:close={handleMarketplaceModalClose}
    on:installed={handleTrackerInstalled}
  />
{/if}
