<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte'
  import { X } from 'lucide-svelte'
  import * as LLMTypes from '@shared/llmTypes'
  import { LLMProvider } from '@shared/llmTypes'

  export let showModal = false

  let appConfig: {
    serverURL: string
    userID: string
    isConnected: boolean
  } = {
    serverURL: '',
    userID: '',
    isConnected: false
  }

  let llmConfig: {
    activeProvider: LLMProvider
    providers: {
      [key in LLMProvider]?: {
        apiKey: string
        baseUrl?: string
        defaultModel: string
        models: string[]
      }
    }
  } = {
    activeProvider: LLMProvider.OLLAMA,
    providers: {}
  }

  let isLoading = true
  let isSaving = false
  let errorMessage = ''
  let successMessage = ''
  let activeTab = 'general'

  const dispatch = createEventDispatcher<{
    close: void
  }>()

  onMount(async () => {
    try {
      // Load configurations
      const [basicConfig, llmData] = await Promise.all([
        window.api.config.get(),
        window.api.llm.getConfig()
      ])

      appConfig = basicConfig
      llmConfig = llmData

      isLoading = false
    } catch (error) {
      console.error('Failed to load configuration:', error)
      errorMessage = 'Failed to load configuration'
      isLoading = false
    }
  })

  async function saveProviderConfig(provider: LLMProvider) {
    try {
      isSaving = true
      errorMessage = ''
      successMessage = ''

      const config = { ...llmConfig.providers[provider] }

      // Sanitize empty API keys (keep them unchanged on the server)
      if (config.apiKey === '') {
        delete config.apiKey
      }

      const success = await window.api.llm.updateProviderConfig(provider, config)

      if (success) {
        successMessage = `${provider} configuration updated successfully`
        // Refresh config
        llmConfig = await window.api.llm.getConfig()
      } else {
        errorMessage = `Failed to update ${provider} configuration`
      }
    } catch (error) {
      console.error(`Failed to save ${provider} configuration:`, error)
      errorMessage = `Failed to save ${provider} configuration`
    } finally {
      isSaving = false
    }
  }

  async function setActiveProvider(provider: LLMProvider) {
    try {
      isSaving = true
      errorMessage = ''
      successMessage = ''

      const success = await window.api.llm.setActiveProvider(provider)

      if (success) {
        llmConfig.activeProvider = provider
        successMessage = `Active provider changed to ${provider}`
      } else {
        errorMessage = 'Failed to change active provider'
      }
    } catch (error) {
      console.error('Failed to set active provider:', error)
      errorMessage = 'Failed to set active provider'
    } finally {
      isSaving = false
    }
  }

  function closeModal() {
    dispatch('close')
  }

  function changeTab(tab: string) {
    activeTab = tab
  }
</script>

{#if showModal}
  <div class="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
    <div
      class="bg-background rounded-lg max-w-4xl w-full max-h-[90vh] overflow-hidden flex flex-col"
    >
      <div class="p-4 border-b border-border flex justify-between items-center">
        <h2 class="text-xl font-semibold">Settings</h2>
        <button
          class="hover:bg-accent rounded-md p-1"
          on:click={closeModal}
          aria-label="Close settings"
        >
          <X size={24} />
        </button>
      </div>

      {#if isLoading}
        <div class="flex-1 flex justify-center items-center py-12">
          <div
            class="animate-spin h-8 w-8 border-4 border-primary border-t-transparent rounded-full"
          ></div>
        </div>
      {:else}
        <div class="flex flex-1 overflow-hidden">
          <!-- Sidebar -->
          <div class="w-52 border-r border-border p-4 space-y-2">
            <button
              class={`w-full text-left px-3 py-2 rounded-md ${activeTab === 'general' ? 'bg-primary text-primary-foreground' : 'hover:bg-accent'}`}
              on:click={() => changeTab('general')}
            >
              General
            </button>
            <button
              class={`w-full text-left px-3 py-2 rounded-md ${activeTab === 'ai' ? 'bg-primary text-primary-foreground' : 'hover:bg-accent'}`}
              on:click={() => changeTab('ai')}
            >
              AI Configuration
            </button>
          </div>

          <!-- Content -->
          <div class="flex-1 p-6 overflow-auto custom-scrollbar">
            {#if errorMessage}
              <div class="bg-destructive/20 text-destructive p-3 rounded-md mb-4">
                {errorMessage}
              </div>
            {/if}

            {#if successMessage}
              <div class="bg-success/20 text-success p-3 rounded-md mb-4">
                {successMessage}
              </div>
            {/if}

            <!-- General Settings Tab -->
            {#if activeTab === 'general'}
              <div class="space-y-6">
                <h3 class="text-lg font-medium">App Configuration</h3>

                <div class="space-y-2">
                  <label for="serverURL" class="block text-sm font-medium text-foreground">
                    Server URL
                  </label>
                  <input
                    id="serverURL"
                    type="text"
                    class="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground"
                    disabled
                    value={appConfig.serverURL}
                  />
                  <p class="text-xs text-muted-foreground">
                    Server URL can only be changed in the config.json file
                  </p>
                </div>

                <div class="space-y-2">
                  <label for="userID" class="block text-sm font-medium text-foreground">
                    User ID
                  </label>
                  <input
                    id="userID"
                    type="text"
                    class="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground"
                    disabled
                    value={appConfig.userID}
                  />
                  <p class="text-xs text-muted-foreground">
                    User ID can only be changed in the config.json file
                  </p>
                </div>

                <div class="space-y-2">
                  <span class="block text-sm font-medium text-foreground"> Connection Status </span>
                  <div class="flex items-center gap-2">
                    <div
                      class={`w-3 h-3 rounded-full ${appConfig.isConnected ? 'bg-success' : 'bg-destructive'}`}
                    ></div>
                    <span>{appConfig.isConnected ? 'Connected' : 'Disconnected'}</span>
                  </div>
                </div>
              </div>
            {/if}

            <!-- AI Configuration Tab -->
            {#if activeTab === 'ai'}
              <div class="space-y-6">
                <h3 class="text-lg font-medium">LLM Configuration</h3>

                <div class="space-y-2">
                  <label class="block text-sm font-medium text-foreground"> Active Provider </label>
                  <div class="grid grid-cols-4 gap-2">
                    {#each Object.values(LLMProvider) as provider}
                      <button
                        class={`px-3 py-2 rounded-md text-sm ${
                          llmConfig.activeProvider === provider
                            ? 'bg-primary text-primary-foreground'
                            : 'bg-accent hover:bg-accent/80'
                        }`}
                        on:click={() => setActiveProvider(provider)}
                        disabled={isSaving}
                      >
                        {provider}
                      </button>
                    {/each}
                  </div>
                </div>

                <!-- Only show configuration for active provider -->
                <div class="space-y-6 pt-4">
                  {#if llmConfig.providers[llmConfig.activeProvider]}
                    {@const providerName = llmConfig.activeProvider}
                    {@const providerConfig = llmConfig.providers[providerName]}
                    <div class="border border-border rounded-md p-4">
                      <h4 class="text-md font-medium mb-3 capitalize">{providerName} Settings</h4>

                      <div class="space-y-4">
                        <!-- API Key -->
                        <div class="space-y-2">
                          <label
                            for="{providerName}-api-key"
                            class="block text-sm font-medium text-foreground"
                          >
                            API Key
                          </label>
                          <input
                            id="{providerName}-api-key"
                            type="password"
                            class="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground"
                            placeholder={providerName === 'ollama'
                              ? 'Not required for Ollama'
                              : '••••••••••••••••••••••••••••••'}
                            disabled={providerName === 'ollama'}
                            bind:value={providerConfig.apiKey}
                          />
                        </div>

                        <!-- Base URL (only for Ollama) -->
                        {#if providerName === 'ollama' && providerConfig.baseUrl !== undefined}
                          <div class="space-y-2">
                            <label
                              for="{providerName}-base-url"
                              class="block text-sm font-medium text-foreground"
                            >
                              Base URL
                            </label>
                            <input
                              id="{providerName}-base-url"
                              type="text"
                              class="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground"
                              bind:value={providerConfig.baseUrl}
                            />
                          </div>
                        {/if}

                        <!-- Default Model -->
                        <div class="space-y-2">
                          <label
                            for="{providerName}-default-model"
                            class="block text-sm font-medium text-foreground"
                          >
                            Default Model
                          </label>
                          <select
                            id="{providerName}-default-model"
                            class="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground"
                            bind:value={providerConfig.defaultModel}
                          >
                            {#each providerConfig.models as model}
                              <option value={model}>{model}</option>
                            {/each}
                          </select>
                        </div>

                        <button
                          class="w-full px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
                          on:click={() => saveProviderConfig(providerName as LLMProvider)}
                          disabled={isSaving}
                        >
                          {isSaving ? 'Saving...' : 'Save'}
                        </button>
                      </div>
                    </div>
                  {/if}
                </div>
              </div>
            {/if}
          </div>
        </div>
      {/if}
    </div>
  </div>
{/if}

<style>
  /* Custom scrollbar for the modal */
  .scrollbar-hide::-webkit-scrollbar {
    display: none;
  }
  .scrollbar-hide {
    -ms-overflow-style: none;
    scrollbar-width: none;
  }
</style>
