<script lang="ts">
  import { onMount } from 'svelte'
  import { createLogger } from '../lib/utils/logger'

  // Create a logger specifically for the API Management
  const logger = createLogger('APIManagementSection')

  // Import components
  import ApiManagementSidebar from './ApiManagementSidebar.svelte'
  import APIsSection from './APIsSection.svelte'
  import PoliciesSection from './PoliciesSection.svelte'

  // State
  let activeSection: 'apis' | 'policies' = 'apis'
</script>

<div class="flex h-full w-full bg-background">
  <!-- Sidebar -->
  <ApiManagementSidebar bind:activeSection />

  <!-- Content Area -->
  <div class="flex-1 overflow-hidden">
    {#if activeSection === 'apis'}
      <APIsSection />
    {:else if activeSection === 'policies'}
      <PoliciesSection />
    {/if}
  </div>
</div>
