<script lang="ts">
  import { onMount } from 'svelte'
  import SearchBar from './SearchBar.svelte'

  let isMaximized = false

  function minimize(): void {
    window.api.window.minimize()
  }

  function maximize(): void {
    window.api.window.maximize()
  }

  function close(): void {
    window.api.window.close()
  }

  async function updateMaximizedState(): Promise<void> {
    isMaximized = await window.api.window.isMaximized()
  }

  onMount(() => {
    updateMaximizedState()

    // Update maximized state when window changes
    window.addEventListener('resize', updateMaximizedState)

    return () => {
      window.removeEventListener('resize', updateMaximizedState)
    }
  })
</script>

<header
  class="fixed top-0 left-0 right-0 h-8 bg-background border-b border-border flex items-center p-0 z-[100] select-none"
  style="-webkit-app-region: drag;"
  role="banner"
>
  <div class="w-[138px] flex items-center h-full">
    <h1 class="ml-3 text-xs font-medium text-foreground opacity-70"></h1>
  </div>
  <nav class="flex-1 flex items-center justify-center h-full" aria-label="Search">
    <div style="-webkit-app-region: no-drag;">
      <SearchBar placeholder="Search messages..." />
    </div>
  </nav>
  <section class="flex h-full" style="-webkit-app-region: no-drag;" aria-label="Window controls">
    <button
      class="h-full w-[46px] border-none bg-transparent flex justify-center items-center text-foreground opacity-70 cursor-pointer hover:bg-muted/50 hover:opacity-100"
      on:click={minimize}
      aria-label="Minimize"
      tabindex="0"
    >
      <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true">
        <rect x="2" y="5.5" width="8" height="1" fill="currentColor" />
      </svg>
    </button>
    <button
      class="h-full w-[46px] border-none bg-transparent flex justify-center items-center text-foreground opacity-70 cursor-pointer hover:bg-muted/50 hover:opacity-100"
      on:click={maximize}
      aria-label={isMaximized ? 'Restore' : 'Maximize'}
      tabindex="0"
    >
      {#if isMaximized}
        <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true">
          <path fill="currentColor" d="M3.5,4.5v3h3v-3H3.5z M2,3h6v6H2V3z M4.5,2v1h3v3h1V2H4.5z" />
        </svg>
      {:else}
        <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true">
          <rect x="2.5" y="2.5" width="7" height="7" stroke="currentColor" fill="none" />
        </svg>
      {/if}
    </button>
    <button
      class="h-full w-[46px] border-none bg-transparent flex justify-center items-center text-foreground opacity-70 cursor-pointer hover:bg-destructive hover:text-destructive-foreground hover:opacity-100"
      on:click={close}
      aria-label="Close"
      tabindex="0"
    >
      <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true">
        <path
          fill="currentColor"
          d="M6,5.3l2.6-2.6c0.2-0.2,0.5-0.2,0.7,0c0.2,0.2,0.2,0.5,0,0.7L6.7,6l2.6,2.6c0.2,0.2,0.2,0.5,0,0.7c-0.2,0.2-0.5,0.2-0.7,0L6,6.7L3.4,9.3c-0.2,0.2-0.5,0.2-0.7,0c-0.2-0.2-0.2-0.5,0-0.7L5.3,6L2.7,3.4c-0.2-0.2-0.2-0.5,0-0.7c0.2-0.2,0.5-0.2,0.7,0L6,5.3z"
        />
      </svg>
    </button>
  </section>
</header>
