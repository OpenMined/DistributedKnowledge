<script lang="ts">
  import { createEventDispatcher } from 'svelte'
  import { cn } from '../lib/utils'
  import { createLogger } from '../lib/utils/logger'
  import { toast } from '../lib/toast'
  import { X, Plus, Trash2, Check, Info, Shield } from 'lucide-svelte'

  // Create a logger specifically for policy creation
  const logger = createLogger('PolicyCreateModal')

  // Event dispatcher for component events
  const dispatch = createEventDispatcher<{
    close: void
    created: { id: string; name: string }
  }>()

  // Props
  export let showModal: boolean = false

  // State
  let isLoading = false
  let isSaving = false
  let errorMessage = ''
  let successMessage = ''
  let activeTab = 'basic' // 'basic', 'rules'

  // Form data
  let formData = {
    name: '',
    description: '',
    type: 'rate', // Default policy type
    rules: [
      {
        type: 'rate',
        limit: 100,
        period: 'minute',
        action: 'throttle'
      }
    ]
  }

  // Policy type options
  const policyTypes = [
    { value: 'free', label: 'Free (Unlimited)' },
    { value: 'rate', label: 'Rate Limited' },
    { value: 'token', label: 'Token Limited' },
    { value: 'time', label: 'Time Limited' },
    { value: 'credit', label: 'Credit Based' },
    { value: 'composite', label: 'Composite (Multiple Rules)' }
  ]

  // Period options for rules
  const periodOptions = [
    { value: 'minute', label: 'Per Minute' },
    { value: 'hour', label: 'Per Hour' },
    { value: 'day', label: 'Per Day' },
    { value: 'week', label: 'Per Week' },
    { value: 'month', label: 'Per Month' },
    { value: 'year', label: 'Per Year' }
  ]

  // Action options for rules
  const actionOptions = [
    { value: 'block', label: 'Block (Reject Request)' },
    { value: 'throttle', label: 'Throttle (Slow Down)' },
    { value: 'notify', label: 'Notify Only' },
    { value: 'log', label: 'Log Only' }
  ]

  // Ensure rule types always match policy type for non-composite policies
  $: if (formData.type !== 'composite' && formData.type !== 'free' && formData.rules.length > 0) {
    formData.rules = formData.rules.map((rule) => ({
      ...rule,
      type: formData.type
    }))
  }

  // Form validation
  $: isValid =
    formData.name.trim() !== '' &&
    (formData.type === 'free' ||
      (formData.rules.length > 0 &&
        formData.rules.every(
          (rule) => rule.limit > 0 && (formData.type === 'composite' || rule.type === formData.type)
        )))

  // Handle closing the modal
  function closeModal() {
    resetForm()
    dispatch('close')
  }

  // Handle policy type change
  function handlePolicyTypeChange() {
    // If switching to free policy, clear rules
    if (formData.type === 'free') {
      formData.rules = []
    }
    // If switching from free policy, add default rule
    else if (formData.rules.length === 0) {
      formData.rules = [
        {
          type: formData.type === 'composite' ? 'rate' : formData.type,
          limit: 100,
          period: 'minute',
          action: 'throttle'
        }
      ]
    }
    // Update rule types if not composite
    else if (formData.type !== 'composite') {
      formData.rules = formData.rules.map((rule) => ({
        ...rule,
        type: formData.type
      }))
    }
  }

  // Add a new rule
  function addRule() {
    // Ensure the rule type is explicitly set
    const newRuleType = formData.type === 'composite' ? 'rate' : formData.type

    // Log rule creation for debugging
    logger.debug('Adding new rule with type:', {
      policyType: formData.type,
      newRuleType: newRuleType
    })

    formData.rules = [
      ...formData.rules,
      {
        type: newRuleType,
        limit: 100,
        period: 'minute',
        action: 'throttle'
      }
    ]
  }

  // Remove a rule
  function removeRule(index: number) {
    formData.rules = formData.rules.filter((_, i) => i !== index)
  }

  function changeTab(tab: string) {
    activeTab = tab
  }

  function resetForm() {
    formData = {
      name: '',
      description: '',
      type: 'rate',
      rules: [
        {
          type: 'rate',
          limit: 100,
          period: 'minute',
          action: 'throttle'
        }
      ]
    }
    errorMessage = ''
    successMessage = ''
    activeTab = 'basic'
  }

  function validateForm() {
    // Reset error message
    errorMessage = ''

    // Validate required fields
    if (!formData.name.trim()) {
      errorMessage = 'Policy name is required'
      activeTab = 'basic'
      return false
    }

    if (
      formData.type !== 'free' &&
      (formData.rules.length === 0 || !formData.rules.every((r) => r.limit > 0))
    ) {
      errorMessage = 'Rules are required and must have valid limits'
      activeTab = 'rules'
      return false
    }

    // Validate rule types match policy type
    if (formData.type !== 'free' && formData.type !== 'composite') {
      const invalidRules = formData.rules.filter((r) => r.type !== formData.type)
      if (invalidRules.length > 0) {
        // Force update rule types to match policy type
        formData.rules = formData.rules.map((rule) => ({
          ...rule,
          type: formData.type
        }))

        logger.warn('Fixed rule types to match policy type before submission', {
          policyType: formData.type,
          ruleCount: formData.rules.length
        })
      }
    }

    return true
  }

  // Handle form submission
  async function createPolicy() {
    if (!validateForm()) {
      return
    }

    isSaving = true
    errorMessage = ''
    successMessage = ''

    try {
      logger.debug('Creating policy', {
        name: formData.name,
        type: formData.type,
        rulesCount: formData.rules.length
      })

      // Pre-process rules to ensure type is correct
      let processedRules = []
      if (formData.type !== 'free') {
        processedRules = formData.rules.map((rule) => ({
          ...rule,
          type: formData.type === 'composite' ? rule.type || 'rate' : formData.type
        }))

        // Log the rules for debugging
        logger.debug('Processed rules before submission:', processedRules)
      }

      // Call the API to create the policy
      const response = await window.api.apps.createPolicy({
        name: formData.name,
        description: formData.description,
        type: formData.type,
        rules: formData.type === 'free' ? [] : processedRules
      })

      logger.debug('Policy creation response:', response)

      if (response.success) {
        // Policy creation was successful
        const policyData = response.data || { id: `policy-${Date.now()}`, name: formData.name }
        logger.debug('Policy created successfully', policyData)

        // Show success toast and message
        successMessage = 'Policy created successfully'
        toast.success('Policy created successfully', {
          title: 'Success',
          duration: 3000
        })

        // Emit created event and close modal after a short delay
        setTimeout(() => {
          resetForm()
          dispatch('created', {
            id: policyData.id,
            name: policyData.name
          })
          closeModal()
        }, 1000)
      } else {
        // Policy creation failed
        logger.error('Failed to create policy:', response)

        // Show error toast and message
        errorMessage = response.message || response.error || 'Failed to create policy'
        toast.error(errorMessage, {
          title: 'Error',
          duration: 5000
        })
      }
    } catch (error) {
      logger.error('Failed to create policy', error)

      // Show error toast and message
      errorMessage = `Error: ${error.message}`
      toast.error(errorMessage, {
        title: 'Error',
        duration: 5000
      })
    } finally {
      isSaving = false
    }
  }
</script>

{#if showModal}
  <div class="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
    <div
      class="bg-background rounded-lg max-w-4xl w-full max-h-[90vh] overflow-hidden flex flex-col"
    >
      <div class="p-4 border-b border-border flex justify-between items-center">
        <h2 class="text-xl font-semibold">Create New Policy</h2>
        <button
          class="hover:bg-accent rounded-md p-1"
          on:click={closeModal}
          aria-label="Close modal"
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
          <!-- Tabs -->
          <div class="w-52 border-r border-border p-4 space-y-2">
            <button
              class={cn(
                'w-full text-left px-3 py-2 rounded-md',
                activeTab === 'basic' ? 'bg-primary text-primary-foreground' : 'hover:bg-accent'
              )}
              on:click={() => changeTab('basic')}
            >
              Basic Information
            </button>
            <button
              class={cn(
                'w-full text-left px-3 py-2 rounded-md',
                activeTab === 'rules' ? 'bg-primary text-primary-foreground' : 'hover:bg-accent',
                formData.type === 'free' ? 'opacity-50 cursor-not-allowed' : ''
              )}
              on:click={() => formData.type !== 'free' && changeTab('rules')}
              disabled={formData.type === 'free'}
            >
              Policy Rules {formData.type === 'free' ? '(N/A)' : ''}
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

            <!-- Basic Tab -->
            {#if activeTab === 'basic'}
              <div class="space-y-6">
                <h3 class="text-lg font-medium">Basic Information</h3>
                <div class="border border-border rounded-md p-4">
                  <div class="space-y-4">
                    <!-- Policy Name -->
                    <div class="space-y-2">
                      <label for="policy-name" class="block text-sm font-medium text-foreground">
                        Policy Name <span class="text-destructive">*</span>
                      </label>
                      <input
                        id="policy-name"
                        type="text"
                        class="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground"
                        placeholder="Enter policy name"
                        bind:value={formData.name}
                      />
                    </div>

                    <!-- Description -->
                    <div class="space-y-2">
                      <label
                        for="policy-description"
                        class="block text-sm font-medium text-foreground"
                      >
                        Description
                      </label>
                      <textarea
                        id="policy-description"
                        class="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground resize-none h-32"
                        placeholder="Enter policy description"
                        bind:value={formData.description}
                      ></textarea>
                    </div>

                    <!-- Policy Type -->
                    <div class="space-y-2">
                      <label for="policy-type" class="block text-sm font-medium text-foreground">
                        Policy Type <span class="text-destructive">*</span>
                      </label>
                      <select
                        id="policy-type"
                        class="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground"
                        bind:value={formData.type}
                        on:change={handlePolicyTypeChange}
                      >
                        {#each policyTypes as type}
                          <option value={type.value}>{type.label}</option>
                        {/each}
                      </select>

                      <div class="flex items-start mt-2">
                        <Info size={16} class="mr-2 text-blue-500 mt-0.5 flex-shrink-0" />
                        <p class="text-xs text-muted-foreground">
                          {#if formData.type === 'free'}
                            Free policies have no limits and allow unlimited requests.
                          {:else if formData.type === 'rate'}
                            Rate limiting restricts the number of requests in a time period.
                          {:else if formData.type === 'token'}
                            Token limiting restricts the total number of tokens processed.
                          {:else if formData.type === 'time'}
                            Time limiting restricts the total execution time.
                          {:else if formData.type === 'credit'}
                            Credit-based policies allocate a certain amount of credits for usage.
                          {:else if formData.type === 'composite'}
                            Composite policies combine multiple rule types for complex access
                            patterns.
                          {/if}
                        </p>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            {/if}

            <!-- Rules Tab -->
            {#if activeTab === 'rules' && formData.type !== 'free'}
              <div class="space-y-6">
                <h3 class="text-lg font-medium">Policy Rules</h3>
                <div class="border border-border rounded-md p-4">
                  <div class="flex items-center justify-between mb-4">
                    <div class="flex items-start">
                      <Info size={18} class="mr-2 text-blue-500 mt-0.5 flex-shrink-0" />
                      <p class="text-sm text-muted-foreground">
                        Define rules to control how this policy limits API usage.
                        {#if formData.type === 'composite'}
                          You can combine different rule types for more complex policies.
                        {/if}
                      </p>
                    </div>

                    <button
                      class="flex items-center gap-1 bg-primary/10 hover:bg-primary/20 text-primary px-3 py-1.5 rounded-md text-sm"
                      on:click={addRule}
                      disabled={isSaving}
                    >
                      <Plus size={16} />
                      <span>Add Rule</span>
                    </button>
                  </div>

                  <div class="space-y-4 max-h-[400px] overflow-y-auto pr-2">
                    {#each formData.rules as rule, i}
                      <div class="border border-border rounded-md p-4 bg-accent/50 relative">
                        <!-- Remove rule button - only show if more than one rule -->
                        {#if formData.rules.length > 1}
                          <button
                            class="absolute right-3 top-3 text-destructive hover:bg-destructive/10 rounded-full p-1"
                            on:click={() => removeRule(i)}
                            title="Remove rule"
                          >
                            <Trash2 size={16} />
                          </button>
                        {/if}

                        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                          <!-- Rule type - only editable in composite policy -->
                          {#if formData.type === 'composite'}
                            <div class="space-y-1">
                              <label for={`rule-type-${i}`} class="block text-sm font-medium"
                                >Rule Type</label
                              >
                              <select
                                id={`rule-type-${i}`}
                                bind:value={rule.type}
                                class="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground text-sm"
                              >
                                <option value="rate">Rate Limit</option>
                                <option value="token">Token Limit</option>
                                <option value="time">Time Limit</option>
                                <option value="credit">Credit Limit</option>
                              </select>
                            </div>
                          {:else}
                            <!-- For non-composite policies, display rule type without binding -->
                            <div class="text-xs text-muted-foreground">
                              Rule type: <span class="font-medium">{formData.type}</span>
                            </div>
                          {/if}

                          <!-- Limit value -->
                          <div class="space-y-1">
                            <label for={`rule-limit-${i}`} class="block text-sm font-medium">
                              {rule.type === 'rate'
                                ? 'Request Limit'
                                : rule.type === 'token'
                                  ? 'Token Limit'
                                  : rule.type === 'time'
                                    ? 'Time Limit (ms)'
                                    : rule.type === 'credit'
                                      ? 'Credit Limit'
                                      : 'Limit'}
                            </label>
                            <input
                              id={`rule-limit-${i}`}
                              type="number"
                              bind:value={rule.limit}
                              min="1"
                              class="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground text-sm"
                            />
                          </div>

                          <!-- Time period -->
                          <div class="space-y-1">
                            <label for={`rule-period-${i}`} class="block text-sm font-medium"
                              >Time Period</label
                            >
                            <select
                              id={`rule-period-${i}`}
                              bind:value={rule.period}
                              class="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground text-sm"
                            >
                              {#each periodOptions as period}
                                <option value={period.value}>{period.label}</option>
                              {/each}
                            </select>
                          </div>

                          <!-- Action -->
                          <div class="space-y-1">
                            <label for={`rule-action-${i}`} class="block text-sm font-medium"
                              >Action</label
                            >
                            <select
                              id={`rule-action-${i}`}
                              bind:value={rule.action}
                              class="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground text-sm"
                            >
                              {#each actionOptions as action}
                                <option value={action.value}>{action.label}</option>
                              {/each}
                            </select>
                          </div>
                        </div>
                      </div>
                    {/each}

                    {#if formData.rules.length === 0}
                      <div class="text-center py-10 bg-muted/20 rounded-md">
                        <p class="text-muted-foreground">
                          No rules defined. Click "Add Rule" to create a rule for this policy.
                        </p>
                      </div>
                    {/if}
                  </div>
                </div>
              </div>
            {/if}
          </div>
        </div>

        <!-- Footer with actions -->
        <div class="p-4 border-t border-border flex justify-between">
          <button
            class="px-4 py-2 bg-muted hover:bg-muted/80 text-muted-foreground rounded-md"
            on:click={closeModal}
            disabled={isSaving}
          >
            Cancel
          </button>
          <button
            class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
            on:click={createPolicy}
            disabled={isSaving}
          >
            {#if isSaving}
              <div
                class="animate-spin h-4 w-4 border-2 border-primary-foreground border-t-transparent rounded-full"
              ></div>
              Creating...
            {:else}
              <Shield size={16} />
              Create Policy
            {/if}
          </button>
        </div>
      {/if}
    </div>
  </div>
{/if}

<style>
  /* Custom scrollbar styling */
  .custom-scrollbar::-webkit-scrollbar {
    width: 10px;
    height: 10px;
  }

  .custom-scrollbar::-webkit-scrollbar-track {
    background: hsl(var(--background));
    border-radius: 5px;
  }

  .custom-scrollbar::-webkit-scrollbar-thumb {
    background: hsl(var(--muted));
    border-radius: 5px;
  }

  .custom-scrollbar::-webkit-scrollbar-thumb:hover {
    background: hsl(var(--muted-foreground));
  }
</style>
