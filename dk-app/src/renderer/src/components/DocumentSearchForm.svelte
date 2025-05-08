<script lang="ts">
  import { createEventDispatcher } from 'svelte'
  import { Search, ChevronUp, ChevronDown } from 'lucide-svelte'

  // Search form properties
  export let searchQuery: string = ''
  export let resultLimit: number = 5

  // Min and max values for the number input
  const minResults = 1
  const maxResults = 20

  // Create event dispatcher for search events
  const dispatch = createEventDispatcher<{
    search: { query: string; limit: number }
  }>()

  // Handle search submission
  function handleSearch() {
    console.log(
      `DocumentSearchForm: Submitting search with query "${searchQuery}" and limit ${resultLimit}`
    )

    dispatch('search', {
      query: searchQuery.trim(),
      limit: resultLimit
    })
  }

  // Handle Enter key press in the search input
  function handleKeyDown(event: KeyboardEvent) {
    if (event.key === 'Enter') {
      event.preventDefault()
      handleSearch()
    }
  }

  // Increment/decrement result limit
  function incrementLimit() {
    if (resultLimit < maxResults) {
      resultLimit++
    }
  }

  function decrementLimit() {
    if (resultLimit > minResults) {
      resultLimit--
    }
  }
</script>

<form
  class="bg-card border border-border rounded-lg p-6 mb-6"
  on:submit|preventDefault={handleSearch}
>
  <h3 class="text-base font-medium text-foreground mb-4">Search Documents</h3>

  <div class="space-y-4">
    <!-- Search query input -->
    <div class="space-y-2">
      <label for="search-query" class="text-sm font-medium text-foreground">Search Query</label>
      <div class="relative">
        <input
          id="search-query"
          type="text"
          bind:value={searchQuery}
          on:keydown={handleKeyDown}
          placeholder="Enter search terms..."
          class="w-full rounded-md border border-input bg-background px-3 py-2 pl-9 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none"
        />
        <div class="absolute left-3 top-2.5 text-muted-foreground">
          <Search size={16} />
        </div>
      </div>
      <p class="text-xs text-muted-foreground">Search across all your tracked documents</p>
    </div>

    <!-- Minimalistic number of results input -->
    <div>
      <div class="flex items-center justify-between">
        <label for="result-limit" class="text-sm font-medium text-foreground">Results to show</label
        >
        <div
          class="flex items-center gap-1 bg-background border border-input rounded-md overflow-hidden"
        >
          <button
            type="button"
            aria-label="Decrease limit"
            disabled={resultLimit <= minResults}
            on:click={decrementLimit}
            class="p-1 text-muted-foreground hover:text-foreground hover:bg-muted/80 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <ChevronDown size={16} />
          </button>

          <span class="w-8 text-center text-sm text-foreground">
            {resultLimit}
          </span>

          <button
            type="button"
            aria-label="Increase limit"
            disabled={resultLimit >= maxResults}
            on:click={incrementLimit}
            class="p-1 text-muted-foreground hover:text-foreground hover:bg-muted/80 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <ChevronUp size={16} />
          </button>
        </div>
      </div>
    </div>

    <!-- Search button -->
    <button
      type="submit"
      class="inline-flex items-center justify-center rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 focus:outline-none focus:ring-2 focus:ring-ring w-full"
    >
      <Search size={16} class="mr-2" />
      Search Documents
    </button>
  </div>
</form>
