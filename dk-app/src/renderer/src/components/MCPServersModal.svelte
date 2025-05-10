<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte'
  import { X } from 'lucide-svelte'
  import { toast } from '../lib/toast'

  export let showModal = false

  // MCP Server configuration
  let mcpConfig: {
    mcpServers: {
      [key: string]: {
        command: string
        args: string[]
      }
    }
  } = {
    mcpServers: {}
  }

  let isLoading = true
  let isSaving = false
  let errorMessage = ''
  let successMessage = ''
  let activeTab = '' // Will be set to the first server name when loaded
  let newServerName = ''
  let showAddServerForm = false

  const dispatch = createEventDispatcher<{
    close: void
  }>()

  onMount(async () => {
    try {
      // Load MCP configuration
      const mcpData = await window.api.config.getMCPConfig()

      mcpConfig = mcpData

      // Set the active tab to the first server if exists
      const serverNames = Object.keys(mcpConfig.mcpServers)
      if (serverNames.length > 0) {
        activeTab = serverNames[0]
      }

      isLoading = false
    } catch (error) {
      console.error('Failed to load MCP configuration:', error)
      errorMessage = 'Failed to load MCP configuration'
      isLoading = false
    }
  })

  async function saveServerConfig(serverName: string) {
    try {
      isSaving = true
      errorMessage = ''
      successMessage = ''

      // Get the configuration to save
      const serverConfig = mcpConfig.mcpServers[serverName]

      if (!serverConfig) {
        errorMessage = `Server configuration for ${serverName} not found`
        isSaving = false
        return
      }

      // Validate command and arguments
      if (!serverConfig.command) {
        errorMessage = 'Command is required'
        isSaving = false
        return
      }

      // Save the configuration
      const success = await window.api.config.saveMCPConfig(mcpConfig)

      if (success) {
        successMessage = `${serverName} configuration updated successfully`

        // Refresh config
        mcpConfig = await window.api.config.getMCPConfig()
      } else {
        errorMessage = `Failed to update ${serverName} configuration`
      }
    } catch (error) {
      console.error(`Failed to save ${serverName} configuration:`, error)
      errorMessage = `Failed to save ${serverName} configuration`
    } finally {
      isSaving = false
    }
  }

  async function addNewServer() {
    if (!newServerName || newServerName.trim() === '') {
      errorMessage = 'Server name is required'
      return
    }

    const normalizedName = newServerName.trim().toLowerCase()

    if (mcpConfig.mcpServers[normalizedName]) {
      errorMessage = `Server "${normalizedName}" already exists`
      return
    }

    // Add new server with default configuration
    mcpConfig.mcpServers[normalizedName] = {
      command: '',
      args: []
    }

    // Try to save the updated configuration
    try {
      const success = await window.api.config.saveMCPConfig(mcpConfig)

      if (success) {
        successMessage = `Server "${normalizedName}" added successfully`
        errorMessage = ''
        newServerName = ''
        showAddServerForm = false
        activeTab = normalizedName

        // Refresh config
        mcpConfig = await window.api.config.getMCPConfig()
      } else {
        errorMessage = `Failed to add server "${normalizedName}"`
      }
    } catch (error) {
      console.error('Failed to add new server:', error)
      errorMessage = 'Failed to add new server'
    }
  }

  async function deleteServer(serverName: string) {
    // Show confirmation toast with action button
    const toastId = toast.action(
      `Are you sure you want to delete the "${serverName}" server? This action cannot be undone.`,
      {
        title: 'Confirm Server Deletion',
        type: 'warning',
        duration: 0, // No auto-dismiss
        action: {
          label: 'Yes, Delete',
          onClick: async () => {
            // Close the confirmation toast
            toast.dismiss(toastId)

            // Show in-progress toast
            const processingToastId = toast.info(`Removing "${serverName}" server...`, {
              title: 'Processing',
              duration: 3000
            })

            try {
              // Delete the server from configuration
              delete mcpConfig.mcpServers[serverName]

              // Save the updated configuration
              const success = await window.api.config.saveMCPConfig(mcpConfig)

              // Dismiss the processing toast
              toast.dismiss(processingToastId)

              if (success) {
                // Show success toast
                toast.success(`Server "${serverName}" deleted successfully`, {
                  title: 'Server Deleted',
                  duration: 3000
                })

                // Clear messages in the UI
                successMessage = ''
                errorMessage = ''

                // Refresh config
                mcpConfig = await window.api.config.getMCPConfig()

                // Update active tab
                const serverNames = Object.keys(mcpConfig.mcpServers)
                if (serverNames.length > 0) {
                  activeTab = serverNames[0]
                } else {
                  activeTab = ''
                }
              } else {
                // Show error toast
                toast.error(`Failed to delete server "${serverName}"`, {
                  title: 'Delete Failed',
                  duration: 4000
                })
                errorMessage = `Failed to delete server "${serverName}"`
              }
            } catch (error) {
              // Dismiss the processing toast if it's still showing
              toast.dismiss(processingToastId)

              console.error('Failed to delete server:', error)

              // Show error toast
              toast.error('An error occurred while deleting the server', {
                title: 'Delete Failed',
                duration: 4000
              })

              errorMessage = 'Failed to delete server'
            }
          }
        },
        onDismiss: () => {
          console.log('Server deletion confirmation dismissed')
        }
      }
    )
  }

  function closeModal() {
    dispatch('close')
  }

  function changeTab(tab: string) {
    activeTab = tab
    errorMessage = ''
    successMessage = ''
  }

  function addArgument(serverName: string) {
    // Create a copy of the current args array
    const updatedArgs = [...mcpConfig.mcpServers[serverName].args]

    // Add the new empty argument
    updatedArgs.push('')

    // Update the args array with the new copy to trigger reactivity
    mcpConfig = {
      ...mcpConfig,
      mcpServers: {
        ...mcpConfig.mcpServers,
        [serverName]: {
          ...mcpConfig.mcpServers[serverName],
          args: updatedArgs
        }
      }
    }
  }

  function removeArgument(serverName: string, index: number) {
    // Create a copy of the current args array
    const updatedArgs = [...mcpConfig.mcpServers[serverName].args]

    // Remove the argument at the specified index
    updatedArgs.splice(index, 1)

    // Update the args array with the new copy to trigger reactivity
    mcpConfig = {
      ...mcpConfig,
      mcpServers: {
        ...mcpConfig.mcpServers,
        [serverName]: {
          ...mcpConfig.mcpServers[serverName],
          args: updatedArgs
        }
      }
    }
  }
</script>

{#if showModal}
  <div class="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
    <div
      class="bg-background rounded-lg max-w-4xl w-full max-h-[90vh] overflow-hidden flex flex-col"
    >
      <div class="p-4 border-b border-border flex justify-between items-center">
        <h2 class="text-xl font-semibold">MCP Servers</h2>
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
            {#each Object.keys(mcpConfig.mcpServers) as serverName}
              <button
                class={`w-full text-left px-3 py-2 rounded-md ${activeTab === serverName ? 'bg-primary text-primary-foreground' : 'hover:bg-accent'}`}
                on:click={() => changeTab(serverName)}
              >
                {serverName}
              </button>
            {/each}

            <!-- Add server button -->
            <div class="pt-2">
              {#if showAddServerForm}
                <div class="space-y-2">
                  <input
                    type="text"
                    placeholder="Server name"
                    class="w-full px-3 py-2 text-sm border border-border rounded-md bg-background text-foreground"
                    bind:value={newServerName}
                    on:keydown={(e) => {
                      if (e.key === 'Enter') {
                        e.preventDefault()
                        addNewServer()
                      }
                    }}
                  />
                  <div class="flex gap-2">
                    <button
                      class="bg-primary text-primary-foreground px-3 py-1 text-sm rounded-md"
                      on:click={addNewServer}
                    >
                      Add
                    </button>
                    <button
                      class="bg-accent text-accent-foreground px-3 py-1 text-sm rounded-md"
                      on:click={() => {
                        showAddServerForm = false
                        newServerName = ''
                        errorMessage = ''
                      }}
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              {:else}
                <button
                  class="w-full px-3 py-2 text-sm rounded-md border border-dashed border-border hover:bg-accent flex items-center justify-center"
                  on:click={() => (showAddServerForm = true)}
                >
                  + Add Server
                </button>
              {/if}
            </div>
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

            {#if activeTab && mcpConfig.mcpServers[activeTab]}
              <div class="space-y-6">
                <h3 class="text-lg font-medium">{activeTab} Server Configuration</h3>

                <div class="border border-border rounded-md p-4">
                  <div class="space-y-4">
                    <!-- Command -->
                    <div class="space-y-2">
                      <label for="command" class="block text-sm font-medium text-foreground">
                        Command
                      </label>
                      <input
                        id="command"
                        type="text"
                        class="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground"
                        placeholder="Command to run (e.g., npx, node, python)"
                        value={mcpConfig.mcpServers[activeTab].command}
                        on:input={(e) => {
                          // Update command with proper reactivity
                          mcpConfig = {
                            ...mcpConfig,
                            mcpServers: {
                              ...mcpConfig.mcpServers,
                              [activeTab]: {
                                ...mcpConfig.mcpServers[activeTab],
                                command: e.currentTarget.value
                              }
                            }
                          }
                        }}
                      />
                    </div>

                    <!-- Arguments -->
                    <div class="space-y-2">
                      <label class="block text-sm font-medium text-foreground"> Arguments </label>

                      {#if mcpConfig.mcpServers[activeTab].args.length === 0}
                        <p class="text-sm text-muted-foreground italic">No arguments defined</p>
                      {:else}
                        {#each mcpConfig.mcpServers[activeTab].args as arg, index}
                          <div class="flex gap-2 mb-2">
                            <input
                              type="text"
                              class="flex-1 px-3 py-2 border border-border rounded-md bg-background text-foreground"
                              placeholder="Argument value"
                              value={mcpConfig.mcpServers[activeTab].args[index]}
                              on:input={(e) => {
                                // Create a copy of the current args array
                                const updatedArgs = [...mcpConfig.mcpServers[activeTab].args]
                                // Update the value at the current index
                                updatedArgs[index] = e.currentTarget.value
                                // Update the args array with the new copy to trigger reactivity
                                mcpConfig = {
                                  ...mcpConfig,
                                  mcpServers: {
                                    ...mcpConfig.mcpServers,
                                    [activeTab]: {
                                      ...mcpConfig.mcpServers[activeTab],
                                      args: updatedArgs
                                    }
                                  }
                                }
                              }}
                            />
                            <button
                              class="px-3 py-2 bg-destructive/10 hover:bg-destructive/20 text-destructive rounded-md"
                              on:click={() => removeArgument(activeTab, index)}
                              title="Remove argument"
                            >
                              ✕
                            </button>
                          </div>
                        {/each}
                      {/if}

                      <button
                        class="w-full mt-2 px-3 py-2 bg-accent hover:bg-accent/80 text-accent-foreground rounded-md text-sm"
                        on:click={() => addArgument(activeTab)}
                      >
                        + Add Argument
                      </button>
                    </div>

                    <!-- Save button -->
                    <div class="flex justify-between pt-4">
                      <button
                        class="px-4 py-2 bg-destructive text-destructive-foreground rounded-md hover:bg-destructive/90"
                        on:click={() => deleteServer(activeTab)}
                      >
                        Delete Server
                      </button>

                      <button
                        class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
                        on:click={() => saveServerConfig(activeTab)}
                        disabled={isSaving}
                      >
                        {isSaving ? 'Saving...' : 'Save'}
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            {:else if Object.keys(mcpConfig.mcpServers).length === 0}
              <div class="flex flex-col items-center justify-center h-64 text-center">
                <p class="text-lg text-muted-foreground mb-4">No MCP servers configured</p>
                <p class="text-sm text-muted-foreground mb-4">
                  Add a new server using the button in the sidebar
                </p>
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
