<script lang="ts">
  import {
    FileText,
    ChevronsLeft,
    ChevronLeft,
    ChevronRight,
    ChevronsRight,
    MoreVertical,
    Trash2
  } from 'lucide-svelte'
  import type { RAGDocument } from '../../../shared/types'
  import { toast } from '../lib/toast'

  // State for dropdown menus
  let openDropdownId: string | null = null

  // Toggle dropdown menu
  function toggleDropdown(documentId: string) {
    if (openDropdownId === documentId) {
      openDropdownId = null
    } else {
      openDropdownId = documentId
    }
  }

  // Close dropdown when clicking outside
  function handleClickOutside(event: MouseEvent) {
    if (openDropdownId && !(event.target as HTMLElement).closest('.document-dropdown')) {
      openDropdownId = null
    }
  }

  // Add document click handler when component is mounted
  import { onMount } from 'svelte'

  onMount(() => {
    document.addEventListener('click', handleClickOutside)
    return () => {
      document.removeEventListener('click', handleClickOutside)
    }
  })

  // Props - Documents to display
  export let documents: RAGDocument[] = []
  export let searchQuery: string = ''

  // Pagination state
  const ITEMS_PER_PAGE = 15
  let currentPage = 1
  let totalPages = Math.max(1, Math.ceil(documents.length / ITEMS_PER_PAGE))

  // Reactive statement to update totalPages when documents change
  $: totalPages = Math.max(1, Math.ceil(documents.length / ITEMS_PER_PAGE))

  // Reactive statement to limit currentPage to valid range
  $: if (currentPage > totalPages) currentPage = totalPages

  // Get current page of documents - using direct reactive statement to ensure reactivity
  $: paginatedDocuments = (() => {
    const startIndex = (currentPage - 1) * ITEMS_PER_PAGE
    const endIndex = Math.min(startIndex + ITEMS_PER_PAGE, documents.length)
    return documents.slice(startIndex, endIndex)
  })()

  // Pagination controls
  function goToFirstPage() {
    currentPage = 1
  }

  function goToPreviousPage() {
    if (currentPage > 1) {
      currentPage--
    }
  }

  function goToNextPage() {
    if (currentPage < totalPages) {
      currentPage++
    }
  }

  function goToLastPage() {
    currentPage = totalPages
  }

  // Format the score as a percentage (always expect value between 0-1)
  function formatScore(score: number): string {
    // Handle invalid values
    if (score === undefined || score === null || isNaN(score)) {
      return '0%'
    }

    // Convert from decimal (0-1) to percentage
    return `${Math.round(score * 100)}%`
  }

  // Format the date string (already formatted as a string from the server)
  function formatDate(dateString: string): string {
    return dateString
  }

  // Extract filename from file path
  function extractFilename(filePath: string): string {
    // Split by slashes and get the last part
    const parts = filePath.split(/[\/\\]/)
    return parts[parts.length - 1] || filePath
  }

  // Get source app from document metadata
  function getSourceApp(filePath: string, metadata?: Record<string, any>): string {
    // Check if metadata contains an app attribute
    if (metadata && metadata.app) {
      return metadata.app
    }

    // Fallback to extension-based logic if metadata.app is not available
    const ext = filePath.split('.').pop()?.toLowerCase() || ''

    const appMap: Record<string, string> = {
      md: 'Markdown',
      txt: 'Notepad',
      pdf: 'PDF Viewer',
      doc: 'Microsoft Word',
      docx: 'Microsoft Word',
      xls: 'Microsoft Excel',
      xlsx: 'Microsoft Excel',
      ppt: 'PowerPoint',
      pptx: 'PowerPoint',
      js: 'VS Code',
      ts: 'VS Code',
      html: 'Chrome',
      css: 'VS Code',
      json: 'VS Code'
    }

    return appMap[ext] || 'Other App'
  }

  // Helper to highlight search terms in content
  function highlightSearchTerms(content: string, query: string): string {
    if (!query.trim()) return content

    // Simple implementation - a production app would use a more robust approach
    const regex = new RegExp(`(${query.trim()})`, 'gi')
    return content.replace(
      regex,
      '<mark class="bg-yellow-200 dark:bg-yellow-800 px-1 rounded">$1</mark>'
    )
  }

  // Generate a unique ID for a document
  function getDocumentId(document: RAGDocument): string {
    // Create a simple hash from the file path and a snippet of content
    const fileHash = document.file.split('/').pop() || document.file
    const contentSnippet = document.content.substring(0, 20).replace(/\s+/g, '')
    return `doc-${fileHash}-${contentSnippet}`
  }
</script>

<div class="bg-card border border-border rounded-lg p-6">
  <!-- Results header -->
  <div class="flex justify-between items-center mb-4">
    <h3 class="text-base font-medium text-foreground">
      {documents.length
        ? `Found ${documents.length} document${documents.length === 1 ? '' : 's'}`
        : 'No documents found'}
      {#if documents.length > ITEMS_PER_PAGE}
        <span class="text-sm ml-1 text-muted-foreground">
          (showing {Math.min(ITEMS_PER_PAGE, paginatedDocuments.length)} per page)
        </span>
      {/if}
    </h3>

    {#if searchQuery}
      <div class="text-sm text-muted-foreground">
        Search: "{searchQuery}"
      </div>
    {/if}
  </div>

  <!-- Results list -->
  {#if documents.length > 0}
    <ul class="space-y-4">
      {#each paginatedDocuments as document}
        <li
          class="border border-border rounded-md p-4 hover:bg-muted/50 transition-colors relative"
        >
          <div class="flex items-start gap-3">
            <!-- Document icon -->
            <div class="mt-1 flex-shrink-0 p-1.5 bg-primary/10 rounded text-primary">
              <FileText size={16} />
            </div>

            <!-- Document content -->
            <div class="flex-grow min-w-0">
              <!-- Document header with filename, app, score and dropdown -->
              <div class="flex flex-nowrap items-center justify-between gap-2 mb-2">
                <div class="font-medium text-foreground truncate max-w-[60%]">
                  {extractFilename(document.file)}
                </div>
                <div class="flex flex-nowrap items-center gap-2 shrink-0">
                  <span
                    class="text-xs px-2 py-1 rounded bg-muted text-muted-foreground whitespace-nowrap"
                  >
                    {getSourceApp(document.file, document.metadata)}
                  </span>
                  <span
                    class="text-xs px-2 py-1 rounded bg-green-100/90 text-green-800 dark:bg-green-900/80 dark:text-green-100 whitespace-nowrap"
                  >
                    Score: {formatScore(document.score)}
                  </span>

                  <!-- Document dropdown menu -->
                  <div class="document-dropdown relative">
                    <button
                      class="p-1.5 rounded-full hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
                      on:click|stopPropagation={(e) => {
                        e.preventDefault()
                        toggleDropdown(getDocumentId(document))
                      }}
                      aria-label="Document options"
                    >
                      <MoreVertical size={16} />
                    </button>

                    {#if openDropdownId === getDocumentId(document)}
                      <div
                        class="absolute z-50 right-0 mt-1 w-48 rounded-md shadow-lg bg-popover border border-border py-1"
                        style="top: 100%;"
                      >
                        <button
                          class="w-full text-left flex items-center gap-2 px-4 py-2 text-sm text-destructive hover:bg-muted transition-colors"
                          on:click|stopPropagation={async () => {
                            const filename = extractFilename(document.file)

                            try {
                              const result = await window.api.apps.deleteDocument(filename)

                              if (result.success) {
                                // Remove the document from the documents array
                                documents = documents.filter(
                                  (doc) => extractFilename(doc.file) !== filename
                                )

                                // Show success notification
                                toast.success(`Document "${filename}" deleted successfully`, {
                                  title: 'Document Deleted',
                                  duration: 3000
                                })
                              } else {
                                console.error('Failed to delete document:', result.message)

                                // Show error notification
                                toast.error(result.message || 'Failed to delete document', {
                                  title: 'Delete Failed',
                                  duration: 4000
                                })
                              }
                            } catch (error) {
                              console.error('Error deleting document:', error)
                            }

                            // Close the dropdown
                            openDropdownId = null
                          }}
                        >
                          <Trash2 size={16} />
                          Delete Document
                        </button>
                      </div>
                    {/if}
                  </div>
                </div>
              </div>

              <!-- Document content with highlighted search terms -->
              <div class="text-sm text-muted-foreground mb-2">
                <p class="line-clamp-3">
                  {@html highlightSearchTerms(document.content, searchQuery)}
                </p>
              </div>

              <!-- Document timestamp -->
              <div class="text-xs text-muted-foreground mt-2">
                {formatDate(document.metadata?.date || '')}
              </div>
            </div>
          </div>
        </li>
      {/each}
    </ul>

    <!-- Pagination controls (only show if we have more than one page) -->
    {#if totalPages > 1}
      <div class="flex justify-center items-center mt-6 space-x-2">
        <!-- First page button -->
        <button
          on:click={goToFirstPage}
          disabled={currentPage === 1}
          class="p-1.5 rounded-md border border-input hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed"
          aria-label="First page"
        >
          <ChevronsLeft size={16} />
        </button>

        <!-- Previous page button -->
        <button
          on:click={goToPreviousPage}
          disabled={currentPage === 1}
          class="p-1.5 rounded-md border border-input hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed"
          aria-label="Previous page"
        >
          <ChevronLeft size={16} />
        </button>

        <!-- Page indicator -->
        <span class="text-sm text-muted-foreground">
          Page {currentPage} of {totalPages}
        </span>

        <!-- Next page button -->
        <button
          on:click={goToNextPage}
          disabled={currentPage === totalPages}
          class="p-1.5 rounded-md border border-input hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed"
          aria-label="Next page"
        >
          <ChevronRight size={16} />
        </button>

        <!-- Last page button -->
        <button
          on:click={goToLastPage}
          disabled={currentPage === totalPages}
          class="p-1.5 rounded-md border border-input hover:bg-muted disabled:opacity-50 disabled:cursor-not-allowed"
          aria-label="Last page"
        >
          <ChevronsRight size={16} />
        </button>
      </div>
    {/if}
  {:else if searchQuery}
    <!-- No results message -->
    <div class="flex flex-col items-center justify-center py-8 text-center">
      <div class="text-muted-foreground mb-2">
        <FileText size={32} />
      </div>
      <p class="text-muted-foreground">No documents found matching your search.</p>
      <p class="text-sm text-muted-foreground mt-2">
        Try using different keywords or reduce the specificity of your search.
      </p>
    </div>
  {:else}
    <!-- Empty state -->
    <div class="flex flex-col items-center justify-center py-8 text-center">
      <div class="text-muted-foreground mb-2">
        <FileText size={32} />
      </div>
      <p class="text-muted-foreground">Enter a search query to find relevant documents.</p>
    </div>
  {/if}
</div>
