<script lang="ts">
  import { cn } from '../lib/utils'
  import { onMount } from 'svelte'
  import { toasts } from '../lib/stores/toast'
  import { toast } from '../lib/toast'

  // Import icons
  import {
    Globe,
    Users,
    Clock,
    XCircle,
    CheckCircle,
    ChevronDown,
    Settings,
    Power,
    FileText,
    PlusCircle,
    Shield,
    AlertTriangle,
    Search,
    MoreVertical,
    AppWindow,
    User,
    Calendar,
    ExternalLink
  } from 'lucide-svelte'

  // Tab state
  let activeTab = 'active' // 'active', 'pending', 'denied'

  // Mock data for API requests
  $: activeApis = [
    {
      id: 'api-1',
      name: 'Weather API',
      description: 'Real-time weather data for global locations',
      users: [
        { id: 'user-1', name: 'Alex Smith', avatar: 'AS' },
        { id: 'user-2', name: 'Jamie Lee', avatar: 'JL' },
        { id: 'user-3', name: 'Morgan Chen', avatar: 'MC' }
      ],
      documents: [
        { id: 'doc-1', name: 'Weather_schema.json', type: 'JSON' },
        { id: 'doc-2', name: 'Location_format.md', type: 'MD' }
      ],
      policy: {
        rateLimit: '100 calls/min',
        dailyQuota: '10,000 calls'
      },
      active: true
    },
    {
      id: 'api-2',
      name: 'News API',
      description: 'Latest news articles from worldwide sources',
      users: [
        { id: 'user-1', name: 'Alex Smith', avatar: 'AS' },
        { id: 'user-4', name: 'Taylor Wong', avatar: 'TW' }
      ],
      documents: [
        { id: 'doc-3', name: 'NEWS_catalog.json', type: 'JSON' },
        { id: 'doc-4', name: 'Source_list.csv', type: 'CSV' }
      ],
      policy: {
        rateLimit: '50 calls/min',
        dailyQuota: '5,000 calls'
      },
      active: true
    }
  ]

  $: pendingRequests = [
    {
      id: 'req-1',
      apiName: 'Financial Data API',
      description: 'Stock market and financial indicators data',
      user: { id: 'user-3', name: 'Morgan Chen', avatar: 'MC' },
      submittedDate: '2025-04-28',
      documents: [{ id: 'doc-5', name: 'Financial_indicators.json', type: 'JSON' }],
      requiredTrackers: [{ id: 'tracker-1', name: 'Market Analytics' }]
    },
    {
      id: 'req-2',
      apiName: 'Translation API',
      description: 'Real-time text translation for 100+ languages',
      user: { id: 'user-4', name: 'Taylor Wong', avatar: 'TW' },
      submittedDate: '2025-05-01',
      documents: [],
      requiredTrackers: []
    }
  ]

  $: deniedRequests = [
    {
      id: 'req-3',
      apiName: 'Social Media API',
      description: 'Social media integration and posting',
      user: { id: 'user-2', name: 'Jamie Lee', avatar: 'JL' },
      submittedDate: '2025-04-15',
      deniedDate: '2025-04-17',
      denialReason: 'Security policy violation: excessive permissions requested',
      documents: [{ id: 'doc-6', name: 'Social_permissions.json', type: 'JSON' }],
      requiredTrackers: [{ id: 'tracker-2', name: 'Social Connector' }]
    }
  ]

  // Loading state
  $: loading = false

  // Track which API has an open dropdown menu
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

  // Handle API deactivation
  function handleDeactivateApi(id: string) {
    // In a real app, this would call the backend
    activeApis = activeApis.map((api) => {
      if (api.id === id) {
        return { ...api, active: false }
      }
      return api
    })

    // Show toast notification
    toast.success(`API deactivated successfully`, {
      title: 'API Status Updated',
      duration: 3000
    })

    // Close dropdown
    activeDropdownId = null
  }

  // Handle API configuration
  function handleConfigureApi(id: string) {
    const api = activeApis.find((a) => a.id === id)
    if (!api) return

    toast.info(`Configuration opened for ${api.name}`, {
      title: 'Configure API',
      duration: 3000
    })

    // Close dropdown
    activeDropdownId = null
  }

  // Handle approving a pending request
  function handleApproveRequest(id: string) {
    // Find the request to approve
    const request = pendingRequests.find((req) => req.id === id)
    if (!request) return

    // Remove from pending
    pendingRequests = pendingRequests.filter((req) => req.id !== id)

    // Add to active APIs in a real implementation
    // Here we just show a toast
    toast.success(`Request for ${request.apiName} approved`, {
      title: 'Request Approved',
      duration: 3000
    })
  }

  // Handle denying a pending request
  function handleDenyRequest(id: string) {
    // Find the request to deny
    const request = pendingRequests.find((req) => req.id === id)
    if (!request) return

    // Remove from pending
    pendingRequests = pendingRequests.filter((req) => req.id !== id)

    // Add to denied requests with current date and a mock reason
    deniedRequests = [
      ...deniedRequests,
      {
        ...request,
        deniedDate: new Date().toISOString().split('T')[0],
        denialReason: 'Request denied by administrator'
      }
    ]

    // Show toast
    toast.info(`Request for ${request.apiName} denied`, {
      title: 'Request Denied',
      duration: 3000
    })
  }

  // Format date for display
  function formatDate(dateString: string): string {
    const date = new Date(dateString)
    return date.toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric'
    })
  }

  // Request a new API
  function handleRequestNewApi() {
    toast.info('This would open a form to request a new API', {
      title: 'New API',
      duration: 3000
    })
  }

  onMount(() => {
    // Add click outside listener for dropdown menus
    document.addEventListener('click', handleClickOutside)

    // Clean up event listener on component unmount
    return () => {
      document.removeEventListener('click', handleClickOutside)
    }
  })
</script>

<div class="flex flex-col h-full w-full bg-background">
  <!-- Header -->
  <div class="p-4 border-b border-border bg-background">
    <h2 class="text-base font-semibold text-foreground">APIs</h2>
  </div>

  <!-- Main content area -->
  <div class="flex-1 overflow-y-auto custom-scrollbar">
    {#if loading}
      <div class="flex justify-center items-center h-48">
        <div class="text-muted-foreground">Loading...</div>
      </div>
    {:else}
      <!-- Section title and action button -->
      <div class="p-6 pb-0 flex justify-between items-center">
        <h3 class="text-lg font-medium text-foreground">API Management</h3>
        <button
          class={cn(
            'flex items-center gap-2 px-4 py-2 rounded-md bg-primary text-primary-foreground',
            'hover:bg-primary/90 transition-colors text-sm font-medium'
          )}
          on:click={handleRequestNewApi}
        >
          <PlusCircle size={16} />
          <span>New API</span>
        </button>
      </div>

      <!-- Tabs -->
      <div class="px-6 pt-6">
        <div class="flex border-b border-border">
          <button
            class={cn(
              'px-4 py-2 text-sm font-medium',
              activeTab === 'active'
                ? 'text-primary border-b-2 border-primary'
                : 'text-muted-foreground hover:text-foreground'
            )}
            on:click={() => (activeTab = 'active')}
          >
            <div class="flex items-center gap-2">
              <CheckCircle size={16} />
              <span>Active APIs</span>
              <span
                class="bg-primary/10 text-primary text-xs font-medium px-2 py-0.5 rounded-full ml-1"
              >
                {activeApis.length}
              </span>
            </div>
          </button>

          <button
            class={cn(
              'px-4 py-2 text-sm font-medium',
              activeTab === 'pending'
                ? 'text-primary border-b-2 border-primary'
                : 'text-muted-foreground hover:text-foreground'
            )}
            on:click={() => (activeTab = 'pending')}
          >
            <div class="flex items-center gap-2">
              <Clock size={16} />
              <span>Pending Requests</span>
              <span
                class="bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200 text-xs font-medium px-2 py-0.5 rounded-full ml-1"
              >
                {pendingRequests.length}
              </span>
            </div>
          </button>

          <button
            class={cn(
              'px-4 py-2 text-sm font-medium',
              activeTab === 'denied'
                ? 'text-primary border-b-2 border-primary'
                : 'text-muted-foreground hover:text-foreground'
            )}
            on:click={() => (activeTab = 'denied')}
          >
            <div class="flex items-center gap-2">
              <XCircle size={16} />
              <span>Denied Requests</span>
              <span
                class="bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200 text-xs font-medium px-2 py-0.5 rounded-full ml-1"
              >
                {deniedRequests.length}
              </span>
            </div>
          </button>
        </div>
      </div>

      <!-- Tab content -->
      <div class="p-6">
        <!-- Active APIs Tab -->
        {#if activeTab === 'active'}
          {#if activeApis.length === 0}
            <!-- Empty state -->
            <div
              class="flex flex-col items-center justify-center bg-card border border-border rounded-lg p-10 text-center mt-6 min-h-[250px]"
            >
              <div
                class="w-16 h-16 bg-primary/10 rounded-full flex items-center justify-center text-primary mb-4"
              >
                <Globe size={32} />
              </div>
              <h3 class="text-xl font-medium text-foreground mb-2">No active APIs</h3>
              <p class="text-muted-foreground mb-6 max-w-md">
                You don't have any active APIs. Request new API access to get started.
              </p>
              <button
                class={cn(
                  'flex items-center gap-2 px-5 py-2.5 rounded-md bg-primary text-primary-foreground',
                  'hover:bg-primary/90 transition-colors font-medium'
                )}
                on:click={handleRequestNewApi}
              >
                <PlusCircle size={18} />
                <span>Request API Access</span>
              </button>
            </div>
          {:else}
            <!-- Active APIs Grid -->
            <div class="grid grid-cols-1 gap-6">
              {#each activeApis as api (api.id)}
                <div
                  class="bg-card border border-border rounded-lg shadow-sm hover:shadow-md transition-shadow"
                >
                  <!-- API Header -->
                  <div class="p-4 border-b border-border flex justify-between items-start">
                    <div class="flex items-start gap-3">
                      <div
                        class="mt-0.5 flex-shrink-0 w-10 h-10 bg-primary/10 rounded-full flex items-center justify-center text-primary"
                      >
                        <Globe size={20} />
                      </div>
                      <div>
                        <h4 class="font-medium text-foreground">{api.name}</h4>
                        <p class="text-sm text-muted-foreground mt-1">{api.description}</p>
                      </div>
                    </div>

                    <!-- Dropdown Menu -->
                    <div class="relative">
                      <button
                        class="p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted/80 transition-colors"
                        aria-label="API options"
                        on:click={(e) => toggleDropdown(api.id, e)}
                      >
                        <MoreVertical size={16} />
                      </button>

                      {#if activeDropdownId === api.id}
                        <div
                          class="absolute right-0 z-10 w-44 rounded-md shadow-lg bg-popover border border-border"
                          style="top: 2rem; right: 0;"
                          on:click|stopPropagation
                        >
                          <div class="py-1">
                            <button
                              class="flex items-center gap-2 w-full px-4 py-2 text-sm text-foreground hover:bg-muted/80 transition-colors"
                              on:click={() => handleConfigureApi(api.id)}
                            >
                              <Settings size={16} />
                              Configure
                            </button>
                            <button
                              class="flex items-center gap-2 w-full px-4 py-2 text-sm text-destructive hover:bg-muted/80 transition-colors"
                              on:click={() => handleDeactivateApi(api.id)}
                            >
                              <Power size={16} />
                              Deactivate
                            </button>
                          </div>
                        </div>
                      {/if}
                    </div>
                  </div>

                  <!-- API Body - Three columns layout -->
                  <div class="p-4 grid grid-cols-1 md:grid-cols-3 gap-6">
                    <!-- Users Column -->
                    <div>
                      <h5
                        class="font-medium text-sm flex items-center gap-2 mb-3 text-muted-foreground"
                      >
                        <Users size={16} />
                        <span>Users ({api.users.length})</span>
                      </h5>

                      <div class="bg-background rounded-md border border-border p-3">
                        <ul class="space-y-3">
                          {#each api.users as user (user.id)}
                            <li class="flex items-center gap-2">
                              <div
                                class="w-6 h-6 rounded-full bg-primary/10 flex items-center justify-center text-xs font-medium text-primary"
                              >
                                {user.avatar}
                              </div>
                              <span class="text-sm">{user.name}</span>
                            </li>
                          {/each}
                        </ul>
                      </div>
                    </div>

                    <!-- Documents Column -->
                    <div>
                      <h5
                        class="font-medium text-sm flex items-center gap-2 mb-3 text-muted-foreground"
                      >
                        <FileText size={16} />
                        <span>Documents ({api.documents.length})</span>
                      </h5>

                      <div class="bg-background rounded-md border border-border p-3">
                        {#if api.documents.length === 0}
                          <p class="text-sm text-muted-foreground text-center py-2">
                            No documents associated
                          </p>
                        {:else}
                          <ul class="space-y-3">
                            {#each api.documents as doc (doc.id)}
                              <li class="flex items-center justify-between">
                                <div class="flex items-center gap-2">
                                  <div
                                    class="w-6 h-6 rounded bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200 flex items-center justify-center text-xs font-medium"
                                  >
                                    {doc.type}
                                  </div>
                                  <span class="text-sm font-mono text-xs">{doc.name}</span>
                                </div>
                                <button class="text-muted-foreground hover:text-foreground">
                                  <ExternalLink size={14} />
                                </button>
                              </li>
                            {/each}
                          </ul>
                        {/if}
                      </div>
                    </div>

                    <!-- Policy Column -->
                    <div>
                      <h5
                        class="font-medium text-sm flex items-center gap-2 mb-3 text-muted-foreground"
                      >
                        <Shield size={16} />
                        <span>API Policy</span>
                      </h5>

                      <div class="bg-background rounded-md border border-border p-3">
                        <div class="space-y-3">
                          <div>
                            <p class="text-xs text-muted-foreground">Rate Limit</p>
                            <p class="text-sm font-mono">{api.policy.rateLimit}</p>
                          </div>
                          <div>
                            <p class="text-xs text-muted-foreground">Daily Quota</p>
                            <p class="text-sm font-mono">{api.policy.dailyQuota}</p>
                          </div>

                          <!-- API Status -->
                          <div class="pt-2 border-t border-border mt-2">
                            <p class="text-xs text-muted-foreground">Status</p>
                            <div class="flex items-center gap-2 mt-1">
                              <div class="w-2 h-2 rounded-full bg-success"></div>
                              <p class="text-sm">Active</p>
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        {/if}

        <!-- Pending Requests Tab -->
        {#if activeTab === 'pending'}
          {#if pendingRequests.length === 0}
            <!-- Empty state -->
            <div
              class="flex flex-col items-center justify-center bg-card border border-border rounded-lg p-10 text-center mt-6 min-h-[250px]"
            >
              <div
                class="w-16 h-16 bg-yellow-100 dark:bg-yellow-900 rounded-full flex items-center justify-center text-yellow-800 dark:text-yellow-200 mb-4"
              >
                <Clock size={32} />
              </div>
              <h3 class="text-xl font-medium text-foreground mb-2">No pending requests</h3>
              <p class="text-muted-foreground mb-6 max-w-md">
                There are no pending API requests at this time.
              </p>
              <button
                class={cn(
                  'flex items-center gap-2 px-5 py-2.5 rounded-md bg-primary text-primary-foreground',
                  'hover:bg-primary/90 transition-colors font-medium'
                )}
                on:click={handleRequestNewApi}
              >
                <PlusCircle size={18} />
                <span>New API</span>
              </button>
            </div>
          {:else}
            <!-- Pending Requests List -->
            <div class="space-y-4">
              {#each pendingRequests as request (request.id)}
                <div class="bg-card border border-border rounded-lg shadow-sm">
                  <!-- Request Header -->
                  <div class="p-4 border-b border-border flex justify-between items-start">
                    <div>
                      <h4 class="font-medium text-foreground flex items-center gap-2">
                        {request.apiName}
                        <span
                          class="bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200 text-xs px-2 py-0.5 rounded-full"
                        >
                          Pending
                        </span>
                      </h4>
                      <p class="text-sm text-muted-foreground mt-1">{request.description}</p>
                    </div>

                    <!-- Action Buttons -->
                    <div class="flex gap-2">
                      <button
                        class="flex items-center gap-1 px-3 py-1 text-xs font-medium rounded-md bg-destructive/10 text-destructive hover:bg-destructive/20 transition-colors"
                        on:click={() => handleDenyRequest(request.id)}
                      >
                        <XCircle size={14} />
                        <span>Deny</span>
                      </button>
                      <button
                        class="flex items-center gap-1 px-3 py-1 text-xs font-medium rounded-md bg-success/10 text-success hover:bg-success/20 transition-colors"
                        on:click={() => handleApproveRequest(request.id)}
                      >
                        <CheckCircle size={14} />
                        <span>Approve</span>
                      </button>
                    </div>
                  </div>

                  <!-- Request Details -->
                  <div class="p-4 grid grid-cols-1 md:grid-cols-3 gap-6">
                    <!-- Requestor Column -->
                    <div>
                      <h5
                        class="font-medium text-sm flex items-center gap-2 mb-3 text-muted-foreground"
                      >
                        <User size={16} />
                        <span>Requestor</span>
                      </h5>

                      <div class="bg-background rounded-md border border-border p-3">
                        <div class="flex items-center gap-2">
                          <div
                            class="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center text-sm font-medium text-primary"
                          >
                            {request.user.avatar}
                          </div>
                          <div>
                            <p class="text-sm">{request.user.name}</p>
                            <p class="text-xs text-muted-foreground">
                              <Calendar size={12} class="inline mr-1" />
                              Submitted: {formatDate(request.submittedDate)}
                            </p>
                          </div>
                        </div>
                      </div>
                    </div>

                    <!-- Documents Column -->
                    <div>
                      <h5
                        class="font-medium text-sm flex items-center gap-2 mb-3 text-muted-foreground"
                      >
                        <FileText size={16} />
                        <span
                          >RAG Documents {request.documents.length > 0
                            ? `(${request.documents.length})`
                            : ''}</span
                        >
                      </h5>

                      <div class="bg-background rounded-md border border-border p-3">
                        {#if request.documents.length === 0}
                          <div class="flex flex-col items-center justify-center py-3 text-center">
                            <FileText size={24} class="text-muted-foreground mb-2 opacity-50" />
                            <p class="text-xs text-muted-foreground">No documents required</p>
                          </div>
                        {:else}
                          <ul class="space-y-3">
                            {#each request.documents as doc (doc.id)}
                              <li class="flex items-center justify-between">
                                <div class="flex items-center gap-2">
                                  <div
                                    class="w-6 h-6 rounded bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200 flex items-center justify-center text-xs font-medium"
                                  >
                                    {doc.type}
                                  </div>
                                  <span class="text-sm font-mono text-xs">{doc.name}</span>
                                </div>
                                <button class="text-muted-foreground hover:text-foreground">
                                  <ExternalLink size={14} />
                                </button>
                              </li>
                            {/each}
                          </ul>
                        {/if}
                      </div>
                    </div>

                    <!-- Required Trackers Column -->
                    <div>
                      <h5
                        class="font-medium text-sm flex items-center gap-2 mb-3 text-muted-foreground"
                      >
                        <AppWindow size={16} />
                        <span
                          >Required Trackers {request.requiredTrackers.length > 0
                            ? `(${request.requiredTrackers.length})`
                            : ''}</span
                        >
                      </h5>

                      <div class="bg-background rounded-md border border-border p-3">
                        {#if request.requiredTrackers.length === 0}
                          <div class="flex flex-col items-center justify-center py-3 text-center">
                            <AppWindow size={24} class="text-muted-foreground mb-2 opacity-50" />
                            <p class="text-xs text-muted-foreground">No trackers required</p>
                          </div>
                        {:else}
                          <ul class="space-y-2">
                            {#each request.requiredTrackers as tracker (tracker.id)}
                              <li
                                class="flex items-center justify-between p-2 rounded-md bg-muted/50"
                              >
                                <span class="text-sm">{tracker.name}</span>
                                <div
                                  class="bg-success/10 text-success text-xs px-2 py-0.5 rounded-full"
                                >
                                  Installed
                                </div>
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
        {/if}

        <!-- Denied Requests Tab -->
        {#if activeTab === 'denied'}
          {#if deniedRequests.length === 0}
            <!-- Empty state -->
            <div
              class="flex flex-col items-center justify-center bg-card border border-border rounded-lg p-10 text-center mt-6 min-h-[250px]"
            >
              <div
                class="w-16 h-16 bg-red-100 dark:bg-red-900 rounded-full flex items-center justify-center text-red-800 dark:text-red-200 mb-4"
              >
                <XCircle size={32} />
              </div>
              <h3 class="text-xl font-medium text-foreground mb-2">No denied requests</h3>
              <p class="text-muted-foreground mb-6 max-w-md">
                There are no denied API requests in the history.
              </p>
            </div>
          {:else}
            <!-- Denied Requests List -->
            <div class="space-y-4">
              {#each deniedRequests as request (request.id)}
                <div class="bg-card border border-border rounded-lg shadow-sm">
                  <!-- Request Header -->
                  <div class="p-4 border-b border-border">
                    <div class="flex justify-between items-start">
                      <div>
                        <h4 class="font-medium text-foreground flex items-center gap-2">
                          {request.apiName}
                          <span
                            class="bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200 text-xs px-2 py-0.5 rounded-full"
                          >
                            Denied
                          </span>
                        </h4>
                        <p class="text-sm text-muted-foreground mt-1">{request.description}</p>
                      </div>

                      <!-- Request Again Button -->
                      <button
                        class="flex items-center gap-1 px-3 py-1 text-xs font-medium rounded-md bg-primary/10 text-primary hover:bg-primary/20 transition-colors"
                      >
                        <PlusCircle size={14} />
                        <span>Request Again</span>
                      </button>
                    </div>

                    <!-- Denial reason -->
                    {#if request.denialReason}
                      <div
                        class="mt-3 flex items-start gap-2 bg-red-50 dark:bg-red-950/50 text-red-800 dark:text-red-200 p-2 rounded-md text-sm"
                      >
                        <AlertTriangle size={16} class="flex-shrink-0 mt-0.5" />
                        <p>{request.denialReason}</p>
                      </div>
                    {/if}
                  </div>

                  <!-- Request Details -->
                  <div class="p-4 grid grid-cols-1 md:grid-cols-3 gap-6">
                    <!-- Requestor Column -->
                    <div>
                      <h5
                        class="font-medium text-sm flex items-center gap-2 mb-3 text-muted-foreground"
                      >
                        <User size={16} />
                        <span>Requestor</span>
                      </h5>

                      <div class="bg-background rounded-md border border-border p-3">
                        <div class="flex items-center gap-2">
                          <div
                            class="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center text-sm font-medium text-primary"
                          >
                            {request.user.avatar}
                          </div>
                          <div>
                            <p class="text-sm">{request.user.name}</p>
                            <div class="flex flex-col gap-1 mt-1">
                              <p class="text-xs text-muted-foreground">
                                <Calendar size={12} class="inline mr-1" />
                                Submitted: {formatDate(request.submittedDate)}
                              </p>
                              <p class="text-xs text-muted-foreground">
                                <XCircle size={12} class="inline mr-1" />
                                Denied: {formatDate(request.deniedDate)}
                              </p>
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>

                    <!-- Documents Column -->
                    <div>
                      <h5
                        class="font-medium text-sm flex items-center gap-2 mb-3 text-muted-foreground"
                      >
                        <FileText size={16} />
                        <span
                          >RAG Documents {request.documents.length > 0
                            ? `(${request.documents.length})`
                            : ''}</span
                        >
                      </h5>

                      <div class="bg-background rounded-md border border-border p-3">
                        {#if request.documents.length === 0}
                          <div class="flex flex-col items-center justify-center py-3 text-center">
                            <FileText size={24} class="text-muted-foreground mb-2 opacity-50" />
                            <p class="text-xs text-muted-foreground">No documents required</p>
                          </div>
                        {:else}
                          <ul class="space-y-3">
                            {#each request.documents as doc (doc.id)}
                              <li class="flex items-center justify-between">
                                <div class="flex items-center gap-2">
                                  <div
                                    class="w-6 h-6 rounded bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200 flex items-center justify-center text-xs font-medium"
                                  >
                                    {doc.type}
                                  </div>
                                  <span class="text-sm font-mono text-xs">{doc.name}</span>
                                </div>
                                <button class="text-muted-foreground hover:text-foreground">
                                  <ExternalLink size={14} />
                                </button>
                              </li>
                            {/each}
                          </ul>
                        {/if}
                      </div>
                    </div>

                    <!-- Required Trackers Column -->
                    <div>
                      <h5
                        class="font-medium text-sm flex items-center gap-2 mb-3 text-muted-foreground"
                      >
                        <AppWindow size={16} />
                        <span
                          >Required Trackers {request.requiredTrackers.length > 0
                            ? `(${request.requiredTrackers.length})`
                            : ''}</span
                        >
                      </h5>

                      <div class="bg-background rounded-md border border-border p-3">
                        {#if request.requiredTrackers.length === 0}
                          <div class="flex flex-col items-center justify-center py-3 text-center">
                            <AppWindow size={24} class="text-muted-foreground mb-2 opacity-50" />
                            <p class="text-xs text-muted-foreground">No trackers required</p>
                          </div>
                        {:else}
                          <ul class="space-y-2">
                            {#each request.requiredTrackers as tracker (tracker.id)}
                              <li
                                class="flex items-center justify-between p-2 rounded-md bg-muted/50"
                              >
                                <span class="text-sm">{tracker.name}</span>
                                <div
                                  class="bg-success/10 text-success text-xs px-2 py-0.5 rounded-full"
                                >
                                  Installed
                                </div>
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
        {/if}
      </div>
    {/if}
  </div>
</div>
