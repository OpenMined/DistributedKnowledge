<script lang="ts">
  import { cn } from '../lib/utils'
  import { onMount } from 'svelte'
  import { toasts } from '../lib/stores/toast'
  import { toast } from '../lib/toast'
  // Import icons for documents and more
  import { MoreVertical, FileText, Trash2, Search } from 'lucide-svelte'

  // Import components
  import DocumentSearchForm from './DocumentSearchForm.svelte'
  import DocumentResults from './DocumentResults.svelte'
  import type { RAGDocument } from '../../../shared/types'

  // Data from electron backend
  $: documentCount = 0
  $: loading = true
  $: errorMessage = ''

  // Search state
  let searchQuery = ''
  let resultLimit = 5
  let searchResults: RAGDocument[] = []
  let hasSearched = false
  let isSearching = false

  // Track which element has an open dropdown menu
  let activeDropdownId: string | number | null = null

  // Function to refresh document data
  async function refreshDocuments() {
    try {
      loading = true
      errorMessage = ''

      // Get document count from backend
      console.log('Calling getDocumentCount() to check RAG server status...')
      const docCountResponse = await window.api.apps.getDocumentCount()
      console.log('DEBUG - Document count response:', JSON.stringify(docCountResponse, null, 2))

      if (docCountResponse.success && docCountResponse.stats) {
        documentCount = docCountResponse.stats.count
        console.log(`Document count: ${documentCount}`)

        // Store error message if it exists
        if (docCountResponse.stats.error) {
          console.log(`ERROR MESSAGE FROM SERVER: "${docCountResponse.stats.error}"`)
          console.log('Full error context:', docCountResponse)
          errorMessage = docCountResponse.stats.error
        }
      } else {
        console.log('Document count response unsuccessful or missing stats:', docCountResponse)
      }
    } catch (error) {
      console.error('Failed to load document data:', error)
      errorMessage = 'Failed to load document data'
    } finally {
      loading = false
    }
  }

  // Function to handle search
  async function handleSearch(event: CustomEvent<{ query: string; limit: number }>) {
    const { query, limit } = event.detail
    searchQuery = query
    resultLimit = limit
    isSearching = true

    // Debug logging
    console.log(`Search requested for "${query}" with limit ${limit}`)

    try {
      // Call the backend to search documents using the RAG server
      const response = await window.api.apps.searchRAGDocuments({
        query,
        numResults: limit
      })

      console.log(`Search results received:`, response)

      if (response.success && response.results) {
        // Get the documents array from the results
        const documents = response.results.documents || []

        // Update searchResults array
        searchResults = documents

        // Set hasSearched flag to true to show results component
        hasSearched = true

        // If no results found for a search with a query, show a toast notification
        if (query.trim() && documents.length === 0) {
          toast.info(`No documents found matching "${query}"`, {
            title: 'No Results',
            duration: 3000
          })
        }
      } else {
        // Handle API error
        console.error('Error from search API:', response.error)
        toast.error('Failed to search documents. Please try again.', {
          title: 'Search Error',
          duration: 4000
        })
        searchResults = []
      }
    } catch (error) {
      console.error('Error performing search:', error)

      // Show error toast
      toast.error('Failed to search documents. Please try again.', {
        title: 'Search Error',
        duration: 4000
      })

      // Return empty results if there was an error
      searchResults = []
    } finally {
      isSearching = false
    }
  }

  // Load data on mount
  onMount(async () => {
    // Initial data fetch
    await refreshDocuments()

    // Add click outside listener for dropdown menus
    document.addEventListener('click', handleClickOutside)

    // Set up a simpler refresh interval that just updates the UI from backend cache
    // This is much lighter than the previous approach that made direct API calls
    const refreshInterval = setInterval(async () => {
      try {
        await refreshDocumentCount() // Only get the count from backend cache
      } catch (error) {
        console.error('Failed to refresh document count:', error)
      }
    }, 5000) // Every 5 seconds

    // Clean up interval and event listener on component unmount
    return () => {
      clearInterval(refreshInterval)
      document.removeEventListener('click', handleClickOutside)
    }
  })

  // Close dropdown when clicking outside
  function handleClickOutside(event: MouseEvent) {
    if (activeDropdownId !== null) {
      activeDropdownId = null
    }
  }

  // Update document count directly
  async function refreshDocumentCount() {
    try {
      console.log('Refreshing document count - calling getDocumentCount()...')
      const docCountResponse = await window.api.apps.getDocumentCount()
      console.log(
        'DEBUG - RefreshDocumentCount response:',
        JSON.stringify(docCountResponse, null, 2)
      )

      if (docCountResponse.success && docCountResponse.stats) {
        documentCount = docCountResponse.stats.count
        console.log(`Updated document count: ${documentCount}`)

        if (docCountResponse.stats.error) {
          console.log(`REFRESH ERROR FROM SERVER: "${docCountResponse.stats.error}"`)
          errorMessage = docCountResponse.stats.error

          // Look for specific RAG server not available message
          if (docCountResponse.stats.error.includes('RAG server is not available')) {
            console.log('RAG SERVER NOT AVAILABLE ERROR DETECTED!')
            console.log('Full response context:', docCountResponse)
          }
        } else {
          // Clear error when server responds with no error
          console.log('No error in response, clearing error message')
          errorMessage = ''
        }
      } else {
        console.error('Invalid document count response:', docCountResponse)
      }
    } catch (error) {
      console.error('Failed to update document count:', error)
      console.error('Error details:', error)
      errorMessage = 'Failed to update document count'
    }
  }

  // Cleanup all documents
  async function cleanupDocuments() {
    try {
      console.log('Making API call to cleanupDocuments()...')
      const result = await window.api.apps.cleanupDocuments()
      console.log('Received API response:', result)

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

        // Clear search results if there are any
        if (hasSearched) {
          searchResults = []
        }

        return true
      } else {
        console.error('Failed to cleanup documents:', result.error || result.message)
        errorMessage = result.error || result.message || 'Failed to cleanup documents'

        // Show error toast notification using helper
        toast.error(result.error || result.message || 'Failed to cleanup documents', {
          title: 'Cleanup Failed',
          duration: 4000
        })

        return false
      }
    } catch (error) {
      console.error('Failed to cleanup documents:', error)
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
    <h2 class="text-base font-semibold text-foreground">Documents</h2>
  </div>

  <!-- Main content area -->
  <div class="flex-1 p-6 overflow-y-auto custom-scrollbar">
    {#if loading}
      <div class="flex justify-center items-center h-48">
        <div class="text-muted-foreground">Loading...</div>
      </div>
    {:else}
      <!-- Two-column layout for desktop, stacked for mobile -->
      <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <!-- Left column: Document count and search form -->
        <div class="lg:col-span-1 space-y-6">
          <!-- Documents count card -->
          <div
            class="bg-card border border-border rounded-lg shadow-sm p-6 w-full hover:shadow-md transition-shadow"
          >
            <div class="flex flex-col items-center text-center">
              <div class="flex justify-end w-full">
                <div class="relative">
                  <button
                    class="p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted/80 transition-colors"
                    aria-label="Document options"
                    on:click={(e) => {
                      e.stopPropagation()
                      activeDropdownId = activeDropdownId === 'documents' ? null : 'documents'
                    }}
                  >
                    <MoreVertical size={16} />
                  </button>

                  <!-- Dropdown menu for documents -->
                  {#if activeDropdownId === 'documents'}
                    <div
                      class="absolute right-0 z-10 w-48 rounded-md shadow-lg bg-popover border border-border"
                      style="top: 2rem; right: 0;"
                      on:click|stopPropagation
                    >
                      <div class="py-1">
                        <button
                          class="flex items-center gap-2 w-full px-4 py-2 text-sm text-destructive hover:bg-muted/80 transition-colors"
                          on:click={() => {
                            // Show confirmation toast with action button
                            const toastId = toast.action(
                              'Are you sure you want to delete all documents? This action cannot be undone.',
                              {
                                title: 'Confirm Document Cleanup',
                                type: 'warning',
                                duration: 0, // No auto-dismiss
                                action: {
                                  label: 'Yes, Delete All',
                                  onClick: async () => {
                                    // Close the confirmation toast
                                    toast.dismiss(toastId)

                                    // Show in-progress toast
                                    const processingToastId = toast.info(
                                      'Removing all documents...',
                                      {
                                        title: 'Processing',
                                        duration: 10000 // Longer duration in case the API call takes time
                                      }
                                    )

                                    // Perform the cleanup (ensure we wait for it to complete)
                                    await cleanupDocuments()

                                    // Dismiss the processing toast if it's still showing
                                    toast.dismiss(processingToastId)
                                  }
                                },
                                onDismiss: () => {
                                  console.log('Cleanup confirmation dismissed')
                                }
                              }
                            )

                            // Close dropdown menu
                            activeDropdownId = null
                          }}
                        >
                          <Trash2 size={16} />
                          Clean up
                        </button>
                      </div>
                    </div>
                  {/if}
                </div>
              </div>
              <span class="text-6xl font-bold text-primary">{documentCount}</span>
              <span class="text-sm text-muted-foreground mt-3">Currently loaded documents</span>
              {#if errorMessage}
                <div class="mt-3 p-2 rounded-md bg-yellow-100 text-yellow-800 text-xs max-w-xs">
                  {errorMessage}
                </div>
              {/if}
            </div>
          </div>

          <!-- Document search form -->
          <DocumentSearchForm bind:searchQuery bind:resultLimit on:search={handleSearch} />

          <!-- About RAG document tracking -->
          <div class="bg-card border border-border rounded-lg p-6">
            <h4 class="text-md font-medium text-foreground mb-3">About Document Tracking</h4>
            <p class="text-sm text-muted-foreground mb-4">
              The system automatically tracks documents from all enabled trackers. These documents
              are used to enhance the response quality of the RAG (Retrieval Augmented Generation)
              system by providing relevant context from your files and applications.
            </p>
            <ul class="text-sm text-muted-foreground list-disc pl-5 space-y-2">
              <li>Documents are processed and stored securely on your device</li>
              <li>No document data is sent to external servers without your permission</li>
              <li>You can manage document sources through the Trackers section</li>
              <li>Clean up all documents at any time to start fresh</li>
            </ul>
          </div>
        </div>

        <!-- Right column: Search results -->
        <div class="lg:col-span-2">
          {#if isSearching}
            <!-- Searching state -->
            <div class="bg-card border border-border rounded-lg p-6">
              <div class="flex flex-col items-center justify-center py-12">
                <div class="mb-4">
                  <div class="animate-spin rounded-full h-10 w-10 border-b-2 border-primary"></div>
                </div>
                <p class="text-muted-foreground">Searching documents...</p>
              </div>
            </div>
          {:else if hasSearched}
            <!-- Document results -->
            <DocumentResults documents={searchResults} {searchQuery} />
          {:else}
            <!-- No search yet state -->
            <div class="bg-card border border-border rounded-lg p-6 text-center">
              <div class="flex flex-col items-center justify-center py-8">
                <div class="bg-primary/10 rounded-full p-4 mb-4 text-primary">
                  <Search size={24} />
                </div>
                <h3 class="text-lg font-medium text-foreground mb-2">Search Your Documents</h3>
                <p class="text-muted-foreground max-w-md mx-auto mb-6">
                  Use the search form to find relevant documents and content from your tracked
                  applications.
                </p>
                <button
                  type="button"
                  on:click={() => {
                    // Empty query to show all results, with a higher limit since we'll paginate
                    // We request 100 documents which will be paginated by the DocumentResults component
                    handleSearch(
                      new CustomEvent('search', {
                        detail: { query: '', limit: 100 }
                      })
                    )
                  }}
                  class="inline-flex items-center justify-center rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 focus:outline-none focus:ring-2 focus:ring-ring"
                >
                  View All Documents
                </button>
              </div>
            </div>
          {/if}
        </div>
      </div>
    {/if}
  </div>
</div>
