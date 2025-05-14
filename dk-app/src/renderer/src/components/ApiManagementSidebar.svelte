<script lang="ts">
  import { cn } from '../lib/utils'
  import { Globe, Shield } from 'lucide-svelte'

  // Props
  export let activeSection: 'apis' | 'policies' = 'apis'
</script>

<div class="flex flex-col h-full border-r border-border w-52 shrink-0">
  <div class="px-3 py-4 border-b border-border">
    <h2 class="text-sm font-semibold text-foreground">API Management</h2>
  </div>

  <div class="flex-1 px-1 py-2">
    <nav class="space-y-1">
      <!-- APIs Section -->
      <button
        class={cn(
          'flex items-center w-full py-2 px-3 text-sm rounded-md',
          activeSection === 'apis'
            ? 'bg-accent text-accent-foreground font-medium'
            : 'text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-colors'
        )}
        on:click={() => (activeSection = 'apis')}
      >
        <Globe size={16} class="mr-2" />
        <span>APIs</span>
      </button>

      <!-- Policies Section -->
      <button
        class={cn(
          'flex items-center w-full py-2 px-3 text-sm rounded-md',
          activeSection === 'policies'
            ? 'bg-accent text-accent-foreground font-medium'
            : 'text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-colors'
        )}
        on:click={() => (activeSection = 'policies')}
      >
        <Shield size={16} class="mr-2" />
        <span>Policies</span>
      </button>
    </nav>
  </div>
</div>
