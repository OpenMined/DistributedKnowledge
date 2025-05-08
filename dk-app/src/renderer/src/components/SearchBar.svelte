<script lang="ts">
  import { Search } from 'lucide-svelte'

  export let placeholder: string = 'Search...'
  let searchQuery = ''
  let isFocused = false

  function handleSearch(): void {
    if (searchQuery.trim()) {
      console.log('Searching for:', searchQuery)
      // In a real app, this would trigger a search request
    }
  }

  function handleKeyPress(event: KeyboardEvent): void {
    if (event.key === 'Enter') {
      event.preventDefault()
      handleSearch()
    }
  }
</script>

<form
  class="relative"
  role="search"
  aria-label="Search messages"
  on:submit|preventDefault={handleSearch}
>
  <div
    class="flex items-center h-6 w-[240px] px-2 rounded-md border transition-all duration-200 bg-background/80 backdrop-blur-sm {isFocused
      ? 'border-primary/70 shadow-sm'
      : 'border-border'}"
  >
    <label for="search-input" class="sr-only">Search</label>
    <Search size={16} class="text-muted-foreground" aria-hidden="true" />
    <input
      id="search-input"
      type="search"
      class="h-full w-full bg-transparent border-none outline-none px-2 text-sm text-foreground placeholder:text-muted-foreground"
      {placeholder}
      bind:value={searchQuery}
      on:keydown={handleKeyPress}
      on:focus={() => (isFocused = true)}
      on:blur={() => (isFocused = false)}
      aria-label={placeholder}
    />
  </div>
</form>
