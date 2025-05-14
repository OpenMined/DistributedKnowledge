<script lang="ts">
  import { cn } from '../lib/utils'
  import { onMount } from 'svelte'
  import { toasts } from '../lib/stores/toast'
  import { toast } from '../lib/toast'
  import { createLogger } from '../lib/utils/logger'

  // Create a logger for the Policies section
  const policyLogger = createLogger('PoliciesSection')

  import {
    Shield,
    AlertTriangle,
    PlusCircle,
    Settings,
    MoreVertical,
    Users,
    Globe,
    Trash2
  } from 'lucide-svelte'

  // Import the policy creation modal
  import PolicyCreateModal from './PolicyCreateModal.svelte'

  // Loading state
  $: loading = false

  // Mock policy data
  let policies = []

  // Load policies data
  async function loadPolicies() {
    loading = true

    try {
      // Try to get API base URL from config
      let apiBaseUrl = ''
      try {
        if (window.api?.config) {
          const config = await window.api.config.get()
          if (config && config.dk_api) {
            apiBaseUrl = config.dk_api.endsWith('/') ? config.dk_api.slice(0, -1) : config.dk_api
            policyLogger.info('Using dk_api from config', { apiBaseUrl })
          }
        }
      } catch (configError) {
        policyLogger.warn('Failed to get config', configError)
      }

      // Use IPC API to get policies data
      policyLogger.info('Fetching policies data via IPC')
      const response = await window.api.apps.getPolicies({
        active: true // Default to active-only policies
      })

      policyLogger.debug('Policies response:', response)

      if (response.success && response.data?.policies) {
        // Process real policies data
        policies = response.data.policies.map((policy) => {
          return {
            id: policy.id,
            name: policy.name,
            description: policy.description || `${policy.type} policy`,
            rules: policy.rules.map((rule) => ({
              type: rule.type,
              value:
                rule.limit +
                (rule.period
                  ? ` ${
                      rule.period === 'minute'
                        ? 'requests/minute'
                        : rule.period === 'day'
                          ? 'requests/day'
                          : rule.period === 'hour'
                            ? 'requests/hour'
                            : rule.period === 'month'
                              ? 'requests/month'
                              : rule.period === 'year'
                                ? 'requests/year'
                                : 'requests/' + rule.period
                    }`
                  : '')
            })),
            // In a real app, you would fetch this from the API
            appliedTo: []
          }
        })

        // Always try to fetch applied APIs for each policy
        try {
          policyLogger.info('Fetching APIs for each policy')

          // Fetch a list of all APIs first to improve performance
          let allApis = []
          try {
            const apiManagementResponse = await window.api.apps.getApiManagement()
            if (
              apiManagementResponse.success &&
              apiManagementResponse.data &&
              apiManagementResponse.data.activeApis
            ) {
              allApis = apiManagementResponse.data.activeApis
              policyLogger.info(`Fetched ${allApis.length} total APIs for policy matching`)
            }
          } catch (allApisError) {
            policyLogger.warn('Error fetching all APIs:', allApisError)
          }

          // Process each policy to find its associated APIs
          for (const policy of policies) {
            try {
              // First try the direct API call to get APIs by policy
              policyLogger.debug(`Fetching APIs for policy ${policy.id}`)
              const apisResponse = await window.api.apps.getAPIsByPolicy(policy.id)

              if (
                apisResponse.success &&
                apisResponse.data &&
                Array.isArray(apisResponse.data.apis) &&
                apisResponse.data.apis.length > 0
              ) {
                // Use the API response if it's successful and contains data
                policy.appliedTo = apisResponse.data.apis.map((api) => ({
                  id: api.id,
                  name: api.name
                }))
                policyLogger.debug(
                  `Found ${policy.appliedTo.length} APIs using policy ${policy.name} via API call`
                )
              } else {
                // Fall back to filtering the APIs we already have
                if (allApis.length > 0) {
                  const matchingApis = allApis.filter(
                    (api) =>
                      api.policy && (api.policy.id === policy.id || api.policy_id === policy.id)
                  )

                  if (matchingApis.length > 0) {
                    policy.appliedTo = matchingApis.map((api) => ({
                      id: api.id,
                      name: api.name
                    }))
                    policyLogger.debug(
                      `Found ${policy.appliedTo.length} APIs using policy ${policy.name} via filtering`
                    )
                  } else {
                    policy.appliedTo = []
                    policyLogger.debug(`No APIs found using policy ${policy.name}`)
                  }
                } else {
                  policy.appliedTo = []
                }
              }
            } catch (policyError) {
              policyLogger.warn(`Error fetching APIs for policy ${policy.id}:`, policyError)
              policy.appliedTo = []
            }
          }

          policyLogger.info('Finished processing API assignments for all policies')
        } catch (apiError) {
          policyLogger.warn('Failed to get applied APIs data', apiError)
        }

        policyLogger.info('Loaded policies from API', { count: policies.length })
      } else {
        // API returned an error or empty data
        policyLogger.warn('Failed to load policies data or no policies found', {
          success: response.success,
          error: response.error,
          dataExists: !!response.data
        })

        // Fall back to mock data
        useMockData()
      }
    } catch (error) {
      policyLogger.error('Exception in loadPolicies', error)

      toast.error(`Error: ${error.message}`, {
        title: 'Policies Error',
        duration: 5000
      })

      // Fall back to mock data
      useMockData()
    } finally {
      loading = false
    }
  }

  // Helper function to use mock data
  function useMockData() {
    policies = [
      {
        id: 'policy-1',
        name: 'Standard Rate Limiting',
        description: 'Default rate limiting policy for most APIs',
        rules: [
          { type: 'rate_limit', value: '100 requests/minute' },
          { type: 'daily_quota', value: '10,000 requests/day' }
        ],
        appliedTo: [
          { id: 'api-1', name: 'Weather API' },
          { id: 'api-2', name: 'News API' }
        ]
      },
      {
        id: 'policy-2',
        name: 'Premium API Access',
        description: 'Higher limits for premium APIs and users',
        rules: [
          { type: 'rate_limit', value: '500 requests/minute' },
          { type: 'daily_quota', value: '50,000 requests/day' },
          { type: 'burst_limit', value: '100 requests/second' }
        ],
        appliedTo: [{ id: 'api-3', name: 'Financial Data API' }]
      }
    ]

    // Show information toast
    toast.info('Using mock policy data for display purposes', {
      title: 'Development Mode',
      duration: 5000
    })
  }

  // Track which policy has an open dropdown menu
  let activeDropdownId: string | null = null

  // Toggle dropdown menu visibility
  function toggleDropdown(id: string, event: MouseEvent) {
    event.stopPropagation() // Prevent event from bubbling up
    activeDropdownId = activeDropdownId === id ? null : id
  }

  // Close dropdown when clicking outside
  function handleClickOutside(event: MouseEvent) {
    if (activeDropdownId !== null) {
      activeDropdownId = null
    }
  }

  // Handle policy modification
  async function handleEditPolicy(id: string) {
    const policy = policies.find((p) => p.id === id)
    if (!policy) return

    try {
      // Fetch the detailed policy info from the API
      const response = await window.api.apps.getPolicy(id)

      if (response.success && response.data) {
        // In a real app, you would open an edit dialog with the policy details
        // For now, just log the details and show a toast
        policyLogger.debug('Policy details:', response.data)

        toast.info(`Edit policy: ${policy.name}`, {
          title: 'Edit Policy',
          duration: 3000
        })
      } else {
        toast.error('Failed to retrieve policy details', {
          title: 'Error',
          duration: 3000
        })
      }
    } catch (error) {
      policyLogger.error('Failed to fetch policy details:', error)
      toast.error(`Error: ${error.message}`, {
        title: 'Policy Error',
        duration: 5000
      })
    }

    // Close dropdown
    activeDropdownId = null
  }

  // Handle policy deletion with toast-based confirmation
  async function handleDeletePolicy(id: string) {
    const policy = policies.find((p) => p.id === id)
    if (!policy) return

    // Close dropdown
    activeDropdownId = null

    // Show confirmation toast with action button
    const toastId = toast.action(
      `Are you sure you want to delete policy "${policy.name}"? This action cannot be undone.`,
      {
        title: 'Confirm Policy Deletion',
        type: 'warning',
        duration: 0, // No auto-dismiss
        action: {
          label: 'Yes, Delete',
          onClick: async () => {
            // Close the confirmation toast
            toast.dismiss(toastId)

            // Show in-progress toast
            const processingToastId = toast.info(`Deleting policy "${policy.name}"...`, {
              title: 'Processing',
              duration: 10000 // Longer duration in case the API call takes time
            })

            try {
              policyLogger.info(`Attempting to delete policy: ${id}`)
              const response = await window.api.apps.deletePolicy(id)

              // Dismiss the processing toast
              toast.dismiss(processingToastId)

              if (response.success) {
                // Remove policy from the list
                policies = policies.filter((p) => p.id !== id)

                toast.success(`Successfully deleted policy: ${policy.name}`, {
                  title: 'Policy Deleted',
                  duration: 3000
                })

                // Reload policies to refresh the list
                await loadPolicies()
              } else {
                // Handle specific error cases
                if (response.error && response.error.includes('policy is in use')) {
                  toast.error(
                    `Cannot delete policy "${policy.name}" because it is currently being used by one or more APIs. Please remove the policy from all APIs first.`,
                    {
                      title: 'Policy In Use',
                      duration: 5000
                    }
                  )
                } else {
                  toast.error(`Failed to delete policy: ${response.error || 'Unknown error'}`, {
                    title: 'Delete Error',
                    duration: 5000
                  })
                }
              }
            } catch (error) {
              // Dismiss the processing toast if it's still showing
              toast.dismiss(processingToastId)

              policyLogger.error('Exception in handleDeletePolicy:', error)
              toast.error(`Error: ${error.message}`, {
                title: 'Policy Deletion Error',
                duration: 5000
              })
            }
          }
        },
        onDismiss: () => {
          policyLogger.debug('Policy deletion cancelled by user')
        }
      }
    )
  }

  // Policy creation modal state
  let showCreatePolicyModal = false

  // Create new policy function
  function handleCreatePolicy() {
    showCreatePolicyModal = true
  }

  // Handle policy creation from modal
  function handlePolicyCreated(event) {
    // Reload policies to include the new one
    loadPolicies()
  }

  onMount(() => {
    // Add click outside listener for dropdown menus
    document.addEventListener('click', handleClickOutside)

    // Load policies data when component mounts
    policyLogger.info('Component mounted, loading policies data')
    loadPolicies()

    // Clean up event listener on component unmount
    return () => {
      document.removeEventListener('click', handleClickOutside)
    }
  })
</script>

<div class="flex flex-col h-full w-full bg-background">
  <!-- Policy Creation Modal -->
  <PolicyCreateModal
    showModal={showCreatePolicyModal}
    on:close={() => (showCreatePolicyModal = false)}
    on:created={handlePolicyCreated}
  />

  <!-- Header -->
  <div class="p-4 border-b border-border bg-background">
    <h2 class="text-base font-semibold text-foreground">Policies</h2>
  </div>

  <!-- Main content area -->
  <div class="flex-1 overflow-y-auto custom-scrollbar">
    {#if loading}
      <div class="flex justify-center items-center h-48">
        <div class="text-muted-foreground">Loading...</div>
      </div>
    {:else}
      <!-- Section title and action button -->
      <div class="p-6 pb-4 flex justify-between items-center">
        <h3 class="text-lg font-medium text-foreground">API Policies</h3>
        <button
          class={cn(
            'flex items-center gap-2 px-4 py-2 rounded-md bg-primary text-primary-foreground',
            'hover:bg-primary/90 transition-colors text-sm font-medium'
          )}
          on:click={handleCreatePolicy}
        >
          <PlusCircle size={16} />
          <span>New Policy</span>
        </button>
      </div>

      <!-- Policies List -->
      <div class="px-6 pb-6">
        {#if policies.length === 0}
          <!-- Empty state -->
          <div
            class="flex flex-col items-center justify-center bg-card border border-border rounded-lg p-10 text-center min-h-[250px]"
          >
            <div
              class="w-16 h-16 bg-primary/10 rounded-full flex items-center justify-center text-primary mb-4"
            >
              <Shield size={32} />
            </div>
            <h3 class="text-xl font-medium text-foreground mb-2">No policies defined</h3>
            <p class="text-muted-foreground mb-6 max-w-md">
              Create your first API policy to manage access rates and quotas.
            </p>
            <button
              class={cn(
                'flex items-center gap-2 px-5 py-2.5 rounded-md bg-primary text-primary-foreground',
                'hover:bg-primary/90 transition-colors font-medium'
              )}
              on:click={handleCreatePolicy}
            >
              <PlusCircle size={18} />
              <span>Create Policy</span>
            </button>
          </div>
        {:else}
          <!-- Policies Grid -->
          <div class="grid grid-cols-1 gap-6">
            {#each policies as policy (policy.id)}
              <div
                class="bg-card border border-border rounded-lg shadow-sm hover:shadow-md transition-shadow"
              >
                <!-- Policy Header -->
                <div class="p-4 border-b border-border flex justify-between items-start">
                  <div class="flex items-start gap-3">
                    <div
                      class="mt-0.5 flex-shrink-0 w-10 h-10 bg-primary/10 rounded-full flex items-center justify-center text-primary"
                    >
                      <Shield size={20} />
                    </div>
                    <div>
                      <h4 class="font-medium text-foreground">{policy.name}</h4>
                      <p class="text-sm text-muted-foreground mt-1">{policy.description}</p>
                    </div>
                  </div>

                  <!-- Dropdown Menu -->
                  <div class="relative">
                    <button
                      class="p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted/80 transition-colors"
                      aria-label="Policy options"
                      on:click={(e) => toggleDropdown(policy.id, e)}
                    >
                      <MoreVertical size={16} />
                    </button>

                    {#if activeDropdownId === policy.id}
                      <div
                        class="absolute right-0 z-10 w-44 rounded-md shadow-lg bg-popover border border-border"
                        style="top: 2rem; right: 0;"
                        on:click|stopPropagation
                      >
                        <div class="py-1">
                          <button
                            class="flex items-center gap-2 w-full px-4 py-2 text-sm text-foreground hover:bg-muted/80 transition-colors"
                            on:click={() => handleEditPolicy(policy.id)}
                          >
                            <Settings size={16} />
                            Edit Policy
                          </button>
                          <button
                            class="flex items-center gap-2 w-full px-4 py-2 text-sm text-destructive hover:bg-muted/80 transition-colors"
                            on:click={() => handleDeletePolicy(policy.id)}
                          >
                            <Trash2 size={16} />
                            Delete Policy
                          </button>
                        </div>
                      </div>
                    {/if}
                  </div>
                </div>

                <!-- Policy Body - Two columns layout -->
                <div class="p-4 grid grid-cols-1 md:grid-cols-2 gap-6">
                  <!-- Rules Column -->
                  <div>
                    <h5
                      class="font-medium text-sm flex items-center gap-2 mb-3 text-muted-foreground"
                    >
                      <Shield size={16} />
                      <span>Policy Rules ({policy.rules.length})</span>
                    </h5>

                    <div class="bg-background rounded-md border border-border p-3">
                      {#if policy.rules.length === 0}
                        <div class="py-3 px-2 text-sm text-muted-foreground text-center">
                          No rules defined for this policy
                        </div>
                      {:else}
                        <ul class="space-y-3">
                          {#each policy.rules as rule, index}
                            <li class="flex items-center gap-2">
                              <div class="w-1.5 h-1.5 rounded-full bg-primary"></div>
                              <div>
                                <p class="text-xs text-muted-foreground capitalize">
                                  {rule.type.replace('_', ' ')}
                                </p>
                                <p class="text-sm font-mono">{rule.value}</p>
                              </div>
                            </li>
                            {#if index < policy.rules.length - 1}
                              <div class="border-t border-border my-1"></div>
                            {/if}
                          {/each}
                        </ul>
                      {/if}
                    </div>
                  </div>

                  <!-- Applied To Column -->
                  <div>
                    <h5
                      class="font-medium text-sm flex items-center gap-2 mb-3 text-muted-foreground"
                    >
                      <Globe size={16} />
                      <span>Applied to APIs ({policy.appliedTo.length})</span>
                    </h5>

                    <div class="bg-background rounded-md border border-border p-3">
                      {#if policy.appliedTo.length === 0}
                        <div class="py-3 px-2 text-sm text-muted-foreground text-center">
                          No APIs using this policy
                        </div>
                      {:else}
                        <ul class="space-y-2">
                          {#each policy.appliedTo as api (api.id)}
                            <li
                              class="flex items-center justify-between p-2 rounded-md bg-muted/50"
                            >
                              <span class="text-sm">{api.name}</span>
                              <button class="text-muted-foreground hover:text-foreground text-xs">
                                View
                              </button>
                            </li>
                          {/each}
                        </ul>
                      {/if}
                    </div>
                  </div>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  </div>
</div>
