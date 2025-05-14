<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte'
  import { X, Plus, Trash2, Check, Info, Search, Shield } from 'lucide-svelte'
  import { toast } from '../lib/toast'
  import { cn } from '../lib/utils'

  export let showModal = false

  // API Creation form data
  let formData = {
    name: '',
    description: '',
    policyId: '',
    documentIds: [] as string[],
    externalUsers: [] as { userId: string; accessLevel: string }[],
    isActive: true
  }

  // Form state
  let isLoading = false // Start with false to show the form immediately
  let isSaving = false
  let errorMessage = ''
  let successMessage = ''
  let policies: { id: string; name: string; type: string }[] = []
  let availableDocuments: { id: string; name: string; type: string }[] = []
  let availableUsers: {
    id: string
    name: string
    avatar: string
    online?: boolean
    status?: string
  }[] = []
  let activeTab = 'basic' // 'basic', 'documents', 'users', 'policy'
  let isDataLoading = true // Track backend data loading separately

  // For document/user selection
  let selectedDocumentId = ''
  let documentSearchQuery = ''
  let documentFilteredResults: typeof availableDocuments = []

  // For user selection
  let selectedUserId = ''
  let userSearchQuery = ''
  let userFilteredResults: typeof availableUsers = []

  // Access level hardcoded to 'read'

  const dispatch = createEventDispatcher<{
    close: void
    created: { apiId: string }
  }>()

  onMount(async () => {
    // Set loading to false immediately to make the form interactive
    // We'll load data in the background
    isLoading = false

    try {
      // Request data in parallel but don't block UI on it
      Promise.all([
        window.api.apps.getPolicies?.() || { success: false, data: [] },
        window.api.apps.getDocuments() || { success: false, data: [] },
        window.api.apps.getUsers?.() || { success: false, data: [] }
      ])
        .then(([policiesData, documentsData, usersData]) => {
          if (policiesData.success) {
            // Process the policies data - it comes in data.policies rather than directly in data
            policies = policiesData.data?.policies || []
            console.log('Loaded policies:', policies) // Debug log
            if (policies.length > 0) {
              formData.policyId = policies[0].id
            }
          }

          if (documentsData.success) {
            availableDocuments = documentsData.data || []
            // Update filtered results for documents
            documentFilteredResults = [...availableDocuments]
          }

          if (usersData.success) {
            availableUsers = usersData.data || []
            // Update filtered results for users
            userFilteredResults = [...availableUsers]
          } else {
            // Fetch users from sidebar API if they're not available from apps API
            window.api.sidebar
              .getUsers()
              .then((sidebarUsers) => {
                if (sidebarUsers && sidebarUsers.length > 0) {
                  availableUsers = sidebarUsers.map((user) => ({
                    id: String(user.id),
                    name: user.name,
                    avatar: user.name.substring(0, 2).toUpperCase(),
                    online: user.online,
                    status: user.status
                  }))
                  // Update filtered results for users
                  userFilteredResults = [...availableUsers]
                }
              })
              .catch((sidebarError) => {
                console.error('Failed to load users from sidebar API:', sidebarError)
              })
          }

          // Mark data loading as complete
          isDataLoading = false
        })
        .catch((error) => {
          console.error('Failed to load required data:', error)
          errorMessage = 'Failed to load required data'
          isDataLoading = false
        })
    } catch (error) {
      console.error('Error setting up data loading:', error)
      errorMessage = 'Error setting up data loading'
    }
  })

  async function createAPI() {
    if (!validateForm()) {
      return
    }

    try {
      isSaving = true
      errorMessage = ''
      successMessage = ''

      console.log('Creating API with form data:', formData)

      // Make API request to create the API
      const response = await window.api.apps.createApi(formData)
      console.log('API creation response:', response)

      if (response && response.success) {
        successMessage = response.message || 'API created successfully'
        toast.success(successMessage, {
          title: 'Success',
          duration: 3000
        })

        // If the API was created successfully and there's a policy ID selected, apply it
        if (response.data?.id && formData.policyId) {
          try {
            // Update the API with the selected policy
            console.log(`Applying policy ${formData.policyId} to API ${response.data.id}`)
            const policyUpdateData = {
              policy_id: formData.policyId,
              effective_immediately: true,
              change_reason: 'Initial policy assignment during API creation'
            }

            // Call the policy update endpoint
            const policyResponse = await window.api.apps.changeAPIPolicy?.(
              response.data.id,
              policyUpdateData
            )

            if (policyResponse && policyResponse.success) {
              console.log('Successfully applied policy to API:', policyResponse)
            } else {
              console.error(
                'Failed to apply policy to API:',
                policyResponse?.error || 'Unknown error'
              )
            }
          } catch (policyError) {
            console.error('Error applying policy to API:', policyError)
          }
        }

        // Close the modal and notify parent component
        setTimeout(() => {
          resetForm()
          if (response.data) {
            // Include all API data in the created event
            console.log('API created with data:', response.data)
            dispatch('created', {
              apiId: response.data.id,
              name: response.data.name,
              users: response.data.users || [],
              documents: response.data.documents || []
            })
          }
          closeModal()
        }, 1500)
      } else {
        const errorMsg = response?.error || 'Failed to create API. Please try again.'
        console.error('API creation failed:', errorMsg)
        errorMessage = errorMsg
        toast.error(errorMsg, {
          title: 'Error',
          duration: 5000
        })
      }
    } catch (error) {
      console.error('Exception in API creation:', error)
      const errorMsg = error?.message || 'An unexpected error occurred'
      errorMessage = errorMsg
      toast.error(errorMsg, {
        title: 'Error',
        duration: 5000
      })
    } finally {
      isSaving = false
    }
  }

  function validateForm() {
    // Reset error message
    errorMessage = ''

    // Validate required fields
    if (!formData.name.trim()) {
      errorMessage = 'API name is required'
      activeTab = 'basic'
      return false
    }

    if (!formData.policyId) {
      errorMessage = 'Please select a policy'
      activeTab = 'policy'
      return false
    }

    return true
  }

  function resetForm() {
    formData = {
      name: '',
      description: '',
      policyId: policies.length > 0 ? policies[0].id : '',
      documentIds: [],
      externalUsers: [],
      isActive: true
    }
    activeTab = 'basic'
    errorMessage = ''
    successMessage = ''
  }

  function closeModal() {
    resetForm()
    dispatch('close')
  }

  function addDocument() {
    if (!selectedDocumentId) return

    // Don't add if already in the list
    if (formData.documentIds.includes(selectedDocumentId)) {
      return
    }

    formData.documentIds = [...formData.documentIds, selectedDocumentId]
    selectedDocumentId = ''
  }

  function removeDocument(documentId: string) {
    formData.documentIds = formData.documentIds.filter((id) => id !== documentId)
  }

  function addUser() {
    if (!selectedUserId) return

    // Don't add if already in the list
    if (formData.externalUsers.some((u) => u.userId === selectedUserId)) {
      return
    }

    // Ensure access_level is one of the valid values: 'read', 'write', 'admin'
    // According to CLAUDE.md, these are the only valid values
    formData.externalUsers = [
      ...formData.externalUsers,
      {
        userId: selectedUserId,
        accessLevel: 'read' // Default to read permission - safer option
      }
    ]

    selectedUserId = ''
  }

  function removeUser(userId: string) {
    formData.externalUsers = formData.externalUsers.filter((u) => u.userId !== userId)
  }

  function changeTab(tab: string) {
    activeTab = tab
  }

  // Filter documents based on search query
  function filterDocuments() {
    if (!documentSearchQuery.trim()) {
      documentFilteredResults = availableDocuments
      return
    }

    const query = documentSearchQuery.toLowerCase().trim()
    documentFilteredResults = availableDocuments.filter(
      (doc) => doc.name.toLowerCase().includes(query) || doc.type.toLowerCase().includes(query)
    )
  }

  // Filter users based on search query
  function filterUsers() {
    if (!userSearchQuery.trim()) {
      userFilteredResults = availableUsers
      return
    }

    const query = userSearchQuery.toLowerCase().trim()
    userFilteredResults = availableUsers.filter(
      (user) =>
        user.name.toLowerCase().includes(query) ||
        (user.online && query === 'online') ||
        (user.status === 'offline' && query === 'offline')
    )
  }
</script>

{#if showModal}
  <div class="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
    <div
      class="bg-background rounded-lg max-w-4xl w-full max-h-[90vh] overflow-hidden flex flex-col"
    >
      <div class="p-4 border-b border-border flex justify-between items-center">
        <h2 class="text-xl font-semibold">Create New API</h2>
        <button
          class="hover:bg-accent rounded-md p-1"
          on:click={closeModal}
          aria-label="Close modal"
        >
          <X size={24} />
        </button>
      </div>

      <!-- Content is always shown now, but we add an overlay for loading data -->
      <div class="flex flex-1 overflow-hidden relative">
        <!-- Loading overlay that only covers secondary data, not basic form inputs -->
        {#if isDataLoading}
          <div
            class="absolute inset-0 bg-background/50 flex items-center justify-center z-10 pointer-events-none"
          >
            <div
              class="flex flex-col items-center gap-2 bg-background/80 p-4 rounded-md border border-border"
            >
              <div
                class="animate-spin h-6 w-6 border-4 border-primary border-t-transparent rounded-full"
              ></div>
              <p class="text-sm text-muted-foreground">Loading data...</p>
            </div>
          </div>
        {/if}
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
                activeTab === 'documents' ? 'bg-primary text-primary-foreground' : 'hover:bg-accent'
              )}
              on:click={() => changeTab('documents')}
            >
              Documents
            </button>
            <button
              class={cn(
                'w-full text-left px-3 py-2 rounded-md',
                activeTab === 'users' ? 'bg-primary text-primary-foreground' : 'hover:bg-accent'
              )}
              on:click={() => changeTab('users')}
            >
              User Access
            </button>
            <button
              class={cn(
                'w-full text-left px-3 py-2 rounded-md',
                activeTab === 'policy' ? 'bg-primary text-primary-foreground' : 'hover:bg-accent'
              )}
              on:click={() => changeTab('policy')}
            >
              Policy
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
                    <!-- API Name -->
                    <div class="space-y-2">
                      <label for="api-name" class="block text-sm font-medium text-foreground">
                        API Name <span class="text-destructive">*</span>
                      </label>
                      <input
                        id="api-name"
                        type="text"
                        class="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground"
                        placeholder="Enter API name"
                        bind:value={formData.name}
                      />
                    </div>

                    <!-- Description -->
                    <div class="space-y-2">
                      <label for="description" class="block text-sm font-medium text-foreground">
                        Description
                      </label>
                      <textarea
                        id="description"
                        class="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground resize-none h-32"
                        placeholder="Enter API description"
                        bind:value={formData.description}
                      ></textarea>
                    </div>

                    <!-- Active Status -->
                    <div class="flex items-center space-x-2">
                      <input
                        id="is-active"
                        type="checkbox"
                        class="h-4 w-4 rounded border-border"
                        bind:checked={formData.isActive}
                      />
                      <label for="is-active" class="text-sm font-medium text-foreground">
                        Make API active immediately
                      </label>
                    </div>
                  </div>
                </div>
              </div>
            {/if}

            <!-- Documents Tab -->
            {#if activeTab === 'documents'}
              <div class="space-y-6">
                <h3 class="text-lg font-medium">Associated Documents</h3>
                <div class="border border-border rounded-md p-4">
                  <!-- Add Document -->
                  <div class="space-y-2 mb-6">
                    <label for="document-select" class="block text-sm font-medium text-foreground">
                      Add Document
                    </label>

                    {#if availableDocuments.length === 0}
                      <div class="p-3 mb-3 bg-yellow-100 text-yellow-800 rounded-md text-sm">
                        No documents available. Documents are automatically tracked from enabled
                        trackers.
                      </div>
                    {/if}

                    <div class="space-y-3">
                      <div class="relative w-full">
                        <input
                          type="text"
                          class="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground pr-10"
                          placeholder="Search documents..."
                          bind:value={documentSearchQuery}
                          on:input={filterDocuments}
                        />
                        <div
                          class="absolute right-3 top-1/2 transform -translate-y-1/2 text-muted-foreground"
                        >
                          <Search size={16} />
                        </div>
                      </div>

                      {#if documentFilteredResults.length > 0 && documentSearchQuery}
                        <div
                          class="max-h-60 overflow-y-auto border border-border rounded-md divide-y divide-border"
                        >
                          {#each documentFilteredResults as document}
                            <button
                              class="w-full text-left px-3 py-2 hover:bg-accent/50"
                              on:click={() => {
                                selectedDocumentId = document.id
                                addDocument()
                                documentSearchQuery = ''
                                filterDocuments()
                              }}
                            >
                              <div class="flex items-center justify-between">
                                <div class="flex-1 truncate">{document.name}</div>
                                <span
                                  class="text-xs font-medium px-2 py-0.5 bg-primary/10 text-primary rounded-full ml-2"
                                >
                                  {document.type}
                                </span>
                              </div>
                            </button>
                          {/each}
                        </div>
                      {:else if documentSearchQuery && documentFilteredResults.length === 0}
                        <div
                          class="p-3 bg-muted/50 rounded-md text-sm text-muted-foreground text-center"
                        >
                          No documents match your search
                        </div>
                      {/if}
                    </div>
                  </div>

                  <!-- Document List -->
                  <div>
                    <h4 class="text-sm font-medium mb-2">Selected Documents</h4>
                    {#if formData.documentIds.length === 0}
                      <p class="text-sm text-muted-foreground italic">No documents selected</p>
                    {:else}
                      <div class="space-y-2">
                        {#each formData.documentIds as documentId}
                          {@const document = availableDocuments.find((d) => d.id === documentId)}
                          {#if document}
                            <div
                              class="flex justify-between items-center p-2 bg-accent/50 rounded-md"
                            >
                              <div class="flex items-center gap-2">
                                <span
                                  class="text-xs font-medium px-2 py-0.5 bg-primary/10 text-primary rounded-full"
                                >
                                  {document.type}
                                </span>
                                <span>{document.name}</span>
                              </div>
                              <button
                                class="text-destructive hover:bg-destructive/10 rounded-full p-1"
                                on:click={() => removeDocument(documentId)}
                                title="Remove document"
                              >
                                <Trash2 size={16} />
                              </button>
                            </div>
                          {/if}
                        {/each}
                      </div>
                    {/if}
                  </div>
                </div>
              </div>
            {/if}

            <!-- Users Tab -->
            {#if activeTab === 'users'}
              <div class="space-y-6">
                <h3 class="text-lg font-medium">
                  External User Access <span class="text-xs text-muted-foreground ml-2"
                    >(Read-only)</span
                  >
                </h3>
                <div class="border border-border rounded-md p-4">
                  <div
                    class="p-3 mb-3 bg-blue-900/20 text-blue-300 dark:bg-blue-950/30 dark:text-blue-200 rounded-md text-sm"
                  >
                    All users are granted read-only access to the API by default. The backend
                    enforces that access levels must be one of: 'read', 'write', or 'admin'.
                  </div>

                  <!-- Add User -->
                  <div class="space-y-2 mb-6">
                    <label for="user-select" class="block text-sm font-medium text-foreground">
                      Add User (Read Access)
                    </label>
                    <div class="space-y-3">
                      <div class="relative w-full">
                        <input
                          type="text"
                          class="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground pr-10"
                          placeholder="Search users..."
                          bind:value={userSearchQuery}
                          on:input={filterUsers}
                        />
                        <div
                          class="absolute right-3 top-1/2 transform -translate-y-1/2 text-muted-foreground"
                        >
                          <Search size={16} />
                        </div>
                      </div>

                      {#if userFilteredResults.length > 0 && userSearchQuery}
                        <div
                          class="max-h-60 overflow-y-auto border border-border rounded-md divide-y divide-border"
                        >
                          {#each userFilteredResults as user}
                            <button
                              class="w-full text-left px-3 py-2 hover:bg-accent/50"
                              on:click={() => {
                                selectedUserId = user.id
                                addUser()
                                userSearchQuery = ''
                                filterUsers()
                              }}
                            >
                              <div class="flex items-center gap-2">
                                <div class="relative flex-shrink-0">
                                  <div
                                    class="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center text-primary"
                                  >
                                    {user.avatar || user.name.charAt(0)}
                                  </div>
                                  {#if user.online}
                                    <div
                                      class="absolute -bottom-1 -right-1 w-3 h-3 bg-green-500 rounded-full border border-background"
                                    ></div>
                                  {:else if user.status === 'offline'}
                                    <div
                                      class="absolute -bottom-1 -right-1 w-3 h-3 bg-gray-400 rounded-full border border-background"
                                    ></div>
                                  {/if}
                                </div>
                                <div class="flex flex-col">
                                  <span class="font-medium">{user.name}</span>
                                  <span class="text-xs text-muted-foreground">
                                    {#if user.online}<span class="text-green-500">• Online</span
                                      >{:else if user.status === 'offline'}<span
                                        class="text-gray-400">• Offline</span
                                      >{/if}
                                  </span>
                                </div>
                              </div>
                            </button>
                          {/each}
                        </div>
                      {:else if userSearchQuery && userFilteredResults.length === 0}
                        <div
                          class="p-3 bg-muted/50 rounded-md text-sm text-muted-foreground text-center"
                        >
                          No users match your search
                        </div>
                      {/if}
                    </div>
                  </div>

                  <!-- User List -->
                  <div>
                    <h4 class="text-sm font-medium mb-2">Users with Access</h4>
                    {#if formData.externalUsers.length === 0}
                      <p class="text-sm text-muted-foreground italic">No users added</p>
                    {:else}
                      <div class="space-y-2">
                        {#each formData.externalUsers as userAccess}
                          {@const user = availableUsers.find((u) => u.id === userAccess.userId)}
                          {#if user}
                            <div
                              class="flex justify-between items-center p-2 bg-accent/50 rounded-md"
                            >
                              <div class="flex items-center gap-2">
                                <div class="relative">
                                  <div
                                    class="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center text-primary"
                                  >
                                    {user.avatar || user.name.charAt(0)}
                                  </div>
                                  {#if user.online}
                                    <div
                                      class="absolute -bottom-1 -right-1 w-3 h-3 bg-green-500 rounded-full border border-background"
                                    ></div>
                                  {:else if user.status === 'offline'}
                                    <div
                                      class="absolute -bottom-1 -right-1 w-3 h-3 bg-gray-400 rounded-full border border-background"
                                    ></div>
                                  {/if}
                                </div>
                                <div>
                                  <div>{user.name}</div>
                                  <div class="text-xs text-muted-foreground">
                                    Read Access {#if user.online}<span
                                        class="text-green-500 font-medium">• Online</span
                                      >{:else if user.status === 'offline'}<span
                                        class="text-gray-400">• Offline</span
                                      >{/if}
                                  </div>
                                </div>
                              </div>
                              <button
                                class="text-destructive hover:bg-destructive/10 rounded-full p-1"
                                on:click={() => removeUser(userAccess.userId)}
                                title="Remove user access"
                              >
                                <Trash2 size={16} />
                              </button>
                            </div>
                          {/if}
                        {/each}
                      </div>
                    {/if}
                  </div>
                </div>
              </div>
            {/if}

            <!-- Policy Tab -->
            {#if activeTab === 'policy'}
              <div class="space-y-6">
                <h3 class="text-lg font-medium">API Policy</h3>
                <div class="border border-border rounded-md p-4">
                  <div class="space-y-4">
                    <div class="flex items-start mb-4">
                      <Info size={18} class="mr-2 text-blue-500 mt-0.5 flex-shrink-0" />
                      <p class="text-sm text-muted-foreground">
                        Policies define usage limits and quotas for this API. Select an appropriate
                        policy based on your expected usage patterns and resource constraints.
                      </p>
                    </div>

                    <!-- Policy selection -->
                    <div class="space-y-2">
                      <label for="policy-select" class="block text-sm font-medium text-foreground">
                        Select Policy <span class="text-destructive">*</span>
                      </label>

                      {#if policies.length === 0}
                        <div class="bg-background border border-border rounded-md p-4 text-center">
                          <p class="text-muted-foreground text-sm">
                            No policies available. Create policies in the Policies section first.
                          </p>
                        </div>
                      {:else}
                        <!-- Custom policy selector -->
                        <div class="w-full">
                          <div class="w-full border border-border rounded-lg overflow-hidden">
                            <div class="divide-y divide-border">
                              {#each policies as policy}
                                <button
                                  type="button"
                                  class={cn(
                                    'w-full px-4 py-3 text-left flex items-center gap-3 hover:bg-accent/40 transition-colors',
                                    formData.policyId === policy.id
                                      ? 'bg-primary/10 border-l-4 border-l-primary'
                                      : 'border-l-4 border-l-transparent'
                                  )}
                                  on:click={() => (formData.policyId = policy.id)}
                                >
                                  <div
                                    class={cn(
                                      'flex-shrink-0 w-10 h-10 rounded-full flex items-center justify-center',
                                      formData.policyId === policy.id
                                        ? 'bg-primary text-primary-foreground'
                                        : 'bg-muted text-muted-foreground'
                                    )}
                                  >
                                    <Shield size={18} />
                                  </div>
                                  <div class="flex-1 min-w-0">
                                    <div class="flex items-center justify-between">
                                      <p
                                        class={cn(
                                          'text-sm font-medium truncate',
                                          formData.policyId === policy.id
                                            ? 'text-primary'
                                            : 'text-foreground'
                                        )}
                                      >
                                        {policy.name}
                                      </p>
                                      {#if policy.type}
                                        <span
                                          class="text-xs bg-primary/10 text-primary px-2 py-0.5 rounded-full capitalize ml-2"
                                        >
                                          {policy.type}
                                        </span>
                                      {/if}
                                    </div>
                                    {#if policy.description}
                                      <p class="text-xs text-muted-foreground truncate mt-0.5">
                                        {policy.description}
                                      </p>
                                    {/if}
                                    {#if Array.isArray(policy.rules) && policy.rules.length > 0}
                                      <p class="text-xs text-muted-foreground mt-1">
                                        {policy.rules.length} rule{policy.rules.length !== 1
                                          ? 's'
                                          : ''}
                                      </p>
                                    {/if}
                                  </div>
                                  {#if formData.policyId === policy.id}
                                    <div class="flex-shrink-0 text-primary">
                                      <Check size={18} />
                                    </div>
                                  {/if}
                                </button>
                              {/each}
                            </div>
                          </div>
                          {#if policies.length === 0}
                            <div class="text-sm text-center text-muted-foreground py-4">
                              No policies available
                            </div>
                          {/if}
                        </div>
                      {/if}
                    </div>

                    <!-- Selected policy details -->
                    {#if formData.policyId}
                      {@const selectedPolicy = policies.find((p) => p.id === formData.policyId)}
                      {#if selectedPolicy}
                        <div class="mt-4">
                          <h4 class="text-sm font-medium mb-2 flex items-center gap-1.5">
                            <Info size={14} />
                            <span>Selected Policy Details</span>
                          </h4>
                          <div class="rounded-lg border border-border overflow-hidden">
                            <!-- Policy header -->
                            <div
                              class="p-4 bg-accent/20 border-b border-border flex justify-between items-center"
                            >
                              <div class="flex items-center gap-3">
                                <div
                                  class="w-10 h-10 rounded-full bg-primary text-primary-foreground flex items-center justify-center"
                                >
                                  <Shield size={18} />
                                </div>
                                <div>
                                  <div class="flex items-center gap-2">
                                    <h4 class="font-medium">{selectedPolicy.name}</h4>
                                    {#if selectedPolicy.type}
                                      <span
                                        class="text-xs bg-primary/20 text-primary px-2 py-0.5 rounded-full capitalize"
                                      >
                                        {selectedPolicy.type}
                                      </span>
                                    {/if}
                                  </div>
                                  {#if selectedPolicy.description}
                                    <p class="text-xs text-muted-foreground">
                                      {selectedPolicy.description}
                                    </p>
                                  {/if}
                                </div>
                              </div>
                            </div>

                            <!-- Policy rules -->
                            <div class="p-4">
                              {#if selectedPolicy.rules && Array.isArray(selectedPolicy.rules) && selectedPolicy.rules.length > 0}
                                <h5 class="text-sm font-medium mb-3 flex items-center">
                                  <span>Policy Rules</span>
                                  <span class="ml-2 text-xs bg-accent px-1.5 py-0.5 rounded-full">
                                    {selectedPolicy.rules.length}
                                  </span>
                                </h5>
                                <div class="space-y-3">
                                  {#each selectedPolicy.rules as rule}
                                    {#if rule}
                                      <div
                                        class="bg-background border border-border rounded-md p-3"
                                      >
                                        <div class="flex items-center gap-2 mb-1">
                                          <div class="w-1 h-5 bg-primary rounded-full"></div>
                                          <span class="font-medium text-sm capitalize"
                                            >{(rule.type || '').toString().replace('_', ' ')}</span
                                          >
                                        </div>
                                        <div class="ml-3">
                                          <div class="text-sm font-mono">
                                            {#if typeof rule.limit === 'number'}
                                              <span class="font-bold">{rule.limit}</span>
                                              {rule.period ? `per ${rule.period}` : ''}
                                            {:else if rule.value}
                                              {rule.value}
                                            {:else}
                                              {JSON.stringify(rule)}
                                            {/if}
                                          </div>
                                          {#if rule.action}
                                            <div
                                              class="text-xs text-muted-foreground mt-1 flex items-center gap-1"
                                            >
                                              <span>Action:</span>
                                              <span
                                                class="capitalize bg-accent/50 px-1.5 py-0.5 rounded-sm"
                                                >{rule.action}</span
                                              >
                                            </div>
                                          {/if}
                                        </div>
                                      </div>
                                    {/if}
                                  {/each}
                                </div>
                              {:else}
                                <div
                                  class="py-8 text-center border border-dashed border-border rounded-md bg-muted/10"
                                >
                                  <p class="text-sm text-muted-foreground">
                                    No rules defined for this policy
                                  </p>
                                </div>
                              {/if}
                            </div>
                          </div>
                        </div>
                      {/if}
                    {/if}
                  </div>
                </div>
              </div>
            {/if}
          </div>
        </div>
      </div>

      <!-- Footer with actions -->
      <div class="p-4 border-t border-border flex justify-between">
        <button
          class="px-4 py-2 bg-muted hover:bg-muted/80 text-muted-foreground rounded-md"
          on:click={closeModal}
        >
          Cancel
        </button>
        <button
          class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
          on:click={createAPI}
          disabled={isSaving}
        >
          {#if isSaving}
            <div
              class="animate-spin h-4 w-4 border-2 border-primary-foreground border-t-transparent rounded-full"
            ></div>
            Creating...
          {:else}
            <Check size={18} />
            Create API
          {/if}
        </button>
      </div>
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
