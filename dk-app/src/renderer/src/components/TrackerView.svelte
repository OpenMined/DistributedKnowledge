<script lang="ts">
  import { cn } from '../lib/utils'
  import { onMount } from 'svelte'
  import { toast } from '../lib/toast'
  import { marked } from 'marked'
  import DOMPurify from 'dompurify'
  import Prism from 'prismjs'
  import TrackerConfigModal from './TrackerConfigModal.svelte'

  // Import Prism languages
  import 'prismjs/components/prism-markdown'
  import 'prismjs/components/prism-python'
  import 'prismjs/components/prism-bash'
  import 'prismjs/themes/prism-okaidia.css' // Atom-like theme
  import 'prismjs/plugins/line-numbers/prism-line-numbers.css'
  import 'prismjs/plugins/line-numbers/prism-line-numbers.js'

  // Helper function to extract basename from a path (Node's path.basename equivalent for browser)
  function basename(path: string): string {
    if (!path) return ''
    return path.split('/').filter(Boolean).pop() || ''
  }

  // Import icons
  import {
    Activity,
    AlertCircle,
    Check,
    ChevronDown,
    ExternalLink,
    Settings,
    FileText,
    Code,
    LayoutList,
    FileCode,
    ChevronsLeft,
    Github,
    Mail,
    Headphones,
    MessageSquare,
    Folder,
    File,
    Maximize2,
    Minimize2
  } from 'lucide-svelte'

  // Props
  export let trackerId: string = ''
  export let onBackClick: () => void
  export let currentTracker = null // Pass tracker data from parent to avoid redundant fetch

  // State variables
  let loading = true
  let tracker: TrackerDetails | null = null
  let activeTab = 'principal'
  let activeAccordionItems: Set<string> = new Set()
  let datasets: Record<string, string> = {}
  let loadingDatasets = false

  // State for Code tab
  let appFiles: AppFileTree[] = []
  let selectedFile: string | null = null
  let fileContent: string = ''
  let loadingFiles = false
  let expandedDirs: Set<string> = new Set() // Track expanded directories
  let isCodeSectionExpanded = false // Track if code section is in fullscreen mode

  interface AppFileTree {
    name: string
    path: string
    type: 'file' | 'directory'
    children?: AppFileTree[]
  }

  // Function to toggle directory expansion
  function toggleDirectory(dirPath: string) {
    if (expandedDirs.has(dirPath)) {
      expandedDirs.delete(dirPath)
    } else {
      expandedDirs.add(dirPath)
    }
    expandedDirs = new Set(expandedDirs) // Trigger reactivity
  }

  // Function to toggle the code section fullscreen mode
  function toggleCodeSectionExpansion() {
    isCodeSectionExpanded = !isCodeSectionExpanded

    // When expanding, ensure Prism syntax highlighting is re-applied
    if (isCodeSectionExpanded && fileContent && selectedFile) {
      setTimeout(() => {
        Prism.highlightAll()
      }, 0)
    }
  }

  interface TrackerDetails {
    id: string
    name: string
    description: string
    version: string
    enabled: boolean
    icon: string
    customIconPath?: string
    templates: Record<string, Template>
  }

  interface Template {
    id: string
    content: string
    filename: string
  }

  // No mock data - we'll use empty objects instead when necessary

  // Function to fetch tracker data
  async function fetchTrackerData() {
    try {
      loading = true

      // Use the currentTracker prop instead of fetching trackers again
      let basicTrackerInfo = currentTracker

      // Safely extract the folder name from the path or fall back to trackerId
      let trackerFolderName = trackerId // Default fallback

      if (basicTrackerInfo && basicTrackerInfo.path) {
        // Try to extract the folder name from the path safely
        const folderName = basename(basicTrackerInfo.path)
        if (folderName) {
          console.log(`Extracted folder name "${folderName}" from path: ${basicTrackerInfo.path}`)
          trackerFolderName = folderName
        }
      }

      // If we have no basicTrackerInfo yet, we need to get it just this once
      if (!basicTrackerInfo) {
        console.log('No tracker data provided - fetching tracker info once')
        const appTrackersResponse = await window.api.apps.getAppTrackers()

        if (appTrackersResponse.success && appTrackersResponse.appTrackers) {
          // Find the tracker with matching ID
          const matchingTracker = appTrackersResponse.appTrackers.find(
            (app) => app.id === trackerId
          )

          if (matchingTracker) {
            // Try to get custom icon for the tracker if available, passing the path to avoid another getAppTrackers call
            const customIconPath = await window.api.apps.getAppIconPath(
              trackerId,
              matchingTracker.path
            )

            // Extract folder name from path for later use with templates and datasets
            if (matchingTracker.path) {
              const pathParts = matchingTracker.path.split('/')
              trackerFolderName = pathParts[pathParts.length - 1]
              console.log(
                `Extracted folder name "${trackerFolderName}" from path: ${matchingTracker.path}`
              )
            }

            // Store basic tracker info
            basicTrackerInfo = {
              id: matchingTracker.id,
              name: matchingTracker.name,
              description: matchingTracker.description,
              version: matchingTracker.version,
              enabled: matchingTracker.enabled,
              icon: matchingTracker.icon,
              path: matchingTracker.path,
              folderName: trackerFolderName,
              ...(customIconPath ? { customIconPath } : {})
            }

            console.log('Found matching tracker:', basicTrackerInfo.name)
            console.log('Matching tracker path:', basicTrackerInfo.path)
            console.log('Matching tracker folder name:', basicTrackerInfo.folderName)
          }
        }
      }

      // If we couldn't find basic info, create a default object
      if (!basicTrackerInfo) {
        console.warn(`No tracker found with ID: ${trackerId}, using default values`)
        basicTrackerInfo = {
          id: trackerId,
          name: 'Unknown Tracker',
          description: 'No description available for this tracker',
          version: '0.0.0',
          enabled: false,
          icon: 'FileCode',
          folderName: null
        }
      }

      // Get the available tracker folders to verify our folder exists
      const appFoldersResponse = await window.api.trackers.getAppFolders()

      // Check if we have any folders
      if (
        appFoldersResponse.success &&
        appFoldersResponse.folders &&
        appFoldersResponse.folders.length > 0
      ) {
        console.log('Available app folders:', appFoldersResponse.folders)

        // Determine which folder to use for this tracker
        let folderToUse = null

        // First option: Use the folder name we extracted from the tracker path
        if (
          basicTrackerInfo.folderName &&
          appFoldersResponse.folders.includes(basicTrackerInfo.folderName)
        ) {
          folderToUse = basicTrackerInfo.folderName
          console.log(`Using exact matching folder: ${folderToUse}`)
        }
        // Second option: Try to find a folder that contains the tracker name (converted to kebab-case)
        else if (basicTrackerInfo.name) {
          const expectedFolderName = basicTrackerInfo.name
            .toLowerCase()
            .replace(/\s+/g, '-')
            .replace(/[^a-z0-9-]/g, '')

          const matchingFolder = appFoldersResponse.folders.find(
            (folder) => folder.includes(expectedFolderName) || expectedFolderName.includes(folder)
          )

          if (matchingFolder) {
            folderToUse = matchingFolder
            console.log(`Using partial matching folder: ${folderToUse}`)
          }
        }

        // Use the trackerId directly as the folder name if we couldn't find a match
        // This is more precise than just using the first folder
        if (!folderToUse) {
          // If trackerId is in available folders, use it directly
          if (appFoldersResponse.folders.includes(trackerId)) {
            folderToUse = trackerId
            console.log(`Using trackerId directly as folder name: ${folderToUse}`)
          }
          // Otherwise check if our current trackerFolderName matches any available folder
          else if (trackerFolderName && appFoldersResponse.folders.includes(trackerFolderName)) {
            folderToUse = trackerFolderName
            console.log(`Using trackerFolderName as folder: ${folderToUse}`)
          }
          // As a last resort, don't use any folder - this will prevent showing data from the wrong folder
          else {
            console.warn(
              `No matching folder found for trackerId: ${trackerId} - NOT using fallback to first folder`
            )
          }
        }

        // If we found a folder to use, fetch templates and datasets
        if (folderToUse) {
          console.log(`Using folder ${folderToUse} to fetch templates and datasets`)

          // Fetch templates using the folder name
          const templatesResponse = await window.api.trackers.getTemplates(folderToUse)

          if (templatesResponse.success && templatesResponse.templates) {
            // Combine the basic info with the templates
            tracker = {
              ...basicTrackerInfo,
              templates: templatesResponse.templates,
              folderName: folderToUse // Store the folder name for later use
            }
            console.log('Successfully fetched templates for folder:', folderToUse)
          } else {
            console.warn('No templates found for tracker folder:', folderToUse)
            // Instead of using mock data, use an empty object to indicate no templates available
            tracker = {
              ...basicTrackerInfo,
              templates: {}, // Empty object instead of mock data
              folderName: folderToUse // Still store the folder name
            }
          }

          // Also fetch datasets for the Files tab
          await fetchDatasets(folderToUse)
        } else {
          console.warn('Could not determine folder for tracker, no templates will be shown')
          tracker = {
            ...basicTrackerInfo,
            templates: {} // Empty object instead of mock data
          }
        }
      } else {
        console.warn('No app folders found, no templates will be shown')
        tracker = {
          ...basicTrackerInfo,
          templates: {} // Empty object instead of mock data
        }
      }

      loading = false
    } catch (error) {
      console.error('Failed to fetch tracker details:', error)
      console.error('Error details:', {
        trackerId,
        currentTracker,
        trackerFolderName,
        error: error instanceof Error ? error.message : String(error)
      })

      toast.error('Failed to load tracker details. Check console for details.', {
        title: 'Error',
        duration: 3000
      })

      // Create a basic tracker object with empty templates in case of error
      tracker = {
        id: trackerId,
        name: 'Error Loading Tracker',
        description:
          'There was an error loading this tracker. Try refreshing the page or checking your connection.',
        version: '0.0.0',
        enabled: false,
        icon: 'AlertCircle', // Use alert icon to indicate error state
        templates: {} // Empty object will trigger our "no templates available" UI
      }
      loading = false
    }
  }

  // Function to fetch datasets
  async function fetchDatasets(folderId: string) {
    try {
      loadingDatasets = true

      // Call the API to get datasets
      const result = await window.api.trackers.getDatasets(folderId)

      if (result.success && result.datasets) {
        datasets = result.datasets
        console.log('Successfully fetched datasets:', datasets)
      } else {
        console.warn('No datasets found or error occurred:', result.error)
        // If no datasets found, use an empty object
        datasets = {}
      }

      loadingDatasets = false
    } catch (error) {
      console.error('Error fetching datasets:', error)
      datasets = {}
      loadingDatasets = false
    }
  }

  // Function to fetch app files
  async function fetchAppFiles(folderId: string) {
    try {
      loadingFiles = true
      console.log(`Fetching app files for folder/id: ${folderId}`)

      // Call the API to get app files
      const result = await window.api.trackers.getAppSourceFiles(folderId)

      if (result.success && result.files) {
        appFiles = result.files
        console.log(`Successfully fetched ${appFiles.length} app files for: ${folderId}`)

        // Keep all directories collapsed by default
        expandedDirs = new Set() // Start with all directories collapsed
      } else {
        console.warn(`No app files found for ${folderId} or error occurred:`, result.error)
        // If no files found, use an empty array
        appFiles = []
        toast.error('Failed to load app files', {
          title: 'Error',
          duration: 3000
        })
      }

      loadingFiles = false
    } catch (error) {
      console.error(`Error fetching app files for ${folderId}:`, error)
      appFiles = []
      loadingFiles = false
      toast.error('Error loading app files', {
        title: 'Error',
        duration: 3000
      })
    }
  }

  // Function to toggle accordion item
  function toggleAccordionItem(itemId: string) {
    if (activeAccordionItems.has(itemId)) {
      activeAccordionItems.delete(itemId)
    } else {
      activeAccordionItems.add(itemId)
    }
    activeAccordionItems = new Set(activeAccordionItems) // Trigger reactivity
  }

  // Function to handle template action
  function handleTemplateAction(templateId: string) {
    console.log(`Action triggered for template: ${templateId}`)
    toast.info(`Action triggered for template: ${templateId}`, {
      title: 'Template Action',
      duration: 3000
    })
  }

  // Track the config modal state - imported from parent or create new
  let showConfigModal = false

  // Function to configure tracker - match behavior of handleConfigureApp in AppsSection
  function handleConfigureTracker() {
    console.log(`Configuring tracker: ${trackerId}`)

    // Reset modal state first
    showConfigModal = false

    // Use setTimeout to ensure DOM updates before showing modal
    setTimeout(() => {
      // Show the modal
      showConfigModal = true
      console.log(`Modal opened for tracker: ${trackerId}`)
    }, 0)
  }

  // Handle config modal close
  function handleConfigModalClose() {
    console.log('Config modal closed')
    // Reset the visibility
    showConfigModal = false
  }

  // Handle config updated
  function handleConfigUpdated() {
    // Refresh data after config update
    fetchTrackerData()
  }

  // Function to toggle tracker enabled state
  async function toggleTrackerEnabled() {
    if (!tracker) return

    try {
      // Use the same API call as the AppsSection component
      const response = await window.api.apps.toggleAppTracker(trackerId)

      if (response.success && response.appTracker) {
        // Update tracker state with the response
        tracker.enabled = response.appTracker.enabled

        toast.success(`Tracker ${tracker.enabled ? 'enabled' : 'disabled'} successfully`, {
          title: 'Tracker Status Updated',
          duration: 3000
        })
      } else {
        throw new Error('Failed to toggle tracker status')
      }
    } catch (error) {
      console.error('Failed to toggle tracker:', error)
      toast.error('Failed to update tracker status', {
        title: 'Error',
        duration: 3000
      })
    }
  }

  // Function to sanitize markdown content and highlight template variables
  function sanitizeMarkdown(markdown: string): string {
    // First highlight template variables with {{ }}
    let highlightedMarkdown = markdown.replace(
      /\{\{([^}]*)\}\}/g,
      '<span class="text-primary font-medium">{{$1}}</span>'
    )

    // Then highlight template tags with {% %}
    highlightedMarkdown = highlightedMarkdown.replace(
      /\{%([^%]*?)%\}/g,
      '<span class="text-red-500 dark:text-red-400 font-medium">{%$1%}</span>'
    )

    // Convert markdown to HTML
    const html = marked(highlightedMarkdown)
    // Sanitize HTML to prevent XSS
    return DOMPurify.sanitize(html, { ADD_ATTR: ['class'] })
  }

  // Function to highlight code based on file extension
  function getHighlightedCode(content: string, filePath: string): string {
    if (!content) return ''

    // Get the file extension
    const extension = filePath.split('.').pop()?.toLowerCase()

    try {
      // Based on file extension, highlight the code with appropriate language
      if (extension === 'py') {
        return Prism.highlight(content, Prism.languages.python, 'python')
      } else if (extension === 'md') {
        return Prism.highlight(content, Prism.languages.markdown, 'markdown')
      } else if (extension === 'sh' || extension === 'bash') {
        return Prism.highlight(content, Prism.languages.bash, 'bash')
      } else {
        // For other file types, return content with HTML entities escaped
        return content
          .replace(/&/g, '&amp;')
          .replace(/</g, '&lt;')
          .replace(/>/g, '&gt;')
          .replace(/"/g, '&quot;')
          .replace(/'/g, '&#039;')
      }
    } catch (error) {
      console.error('Error highlighting code:', error)
      // In case of error, just escape HTML entities
      return content
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;')
    }
  }

  // Function to fetch file content
  async function fetchFileContent(filePath: string) {
    try {
      selectedFile = filePath
      fileContent = 'Loading...'

      // Determine the folder ID to use
      let folderToUse = tracker?.folderName || trackerId

      if (!folderToUse) {
        console.warn('No folder or trackerId provided for file content fetch')
        fileContent = `Error: No tracker folder information available to fetch file content`
        return
      }

      console.log(`Fetching content for file: ${filePath} using folder: ${folderToUse}`)

      // Call the API to get file content
      const result = await window.api.trackers.getAppFileContent(folderToUse, filePath)

      if (result.success && result.content !== undefined) {
        fileContent = result.content
        console.log(`Successfully fetched content (${result.content.length} bytes) for ${filePath}`)

        // Ensure syntax highlighting is applied after content is loaded
        setTimeout(() => {
          Prism.highlightAll()
        }, 0)
      } else {
        console.warn(`Failed to get file content: ${result.error}`)
        fileContent = `Error: Could not load content for ${filePath}. ${result.error || 'The file might be missing, empty, or you may not have permission to access it.'}`
        toast.error(`Failed to load file content: ${result.error || 'Access error'}`, {
          title: 'File Error',
          duration: 3000
        })
      }
    } catch (error) {
      console.error('Error fetching file content:', error)
      fileContent = `Error loading file content for ${filePath}. This could be due to a network issue, invalid path, or server error.`
      toast.error('Error loading file content. Please try again later.', {
        title: 'Error',
        duration: 3000
      })
    }
  }

  // Load data when component mounts or trackerId changes
  $: if (trackerId) {
    fetchTrackerData()
  }

  onMount(() => {
    fetchTrackerData()
  })

  // When activeTab changes to 'code', fetch app files
  $: if (activeTab === 'code' && tracker && appFiles.length === 0) {
    // Use the folder name we already determined in fetchTrackerData
    if (tracker.folderName) {
      console.log(`Using folder name from tracker: ${tracker.folderName} to fetch app files`)
      fetchAppFiles(tracker.folderName)
    } else if (trackerId) {
      // If we somehow don't have a folder name yet but have a tracker ID,
      // rely on the path resolution in fetchAppFiles but fetch available folders first
      console.log(
        `No folder name available, trying to find matching folder for tracker ID: ${trackerId}`
      )
      window.api.trackers
        .getAppFolders()
        .then((response) => {
          if (response.success && response.folders) {
            // If trackerId is in available folders, use it directly
            if (response.folders.includes(trackerId)) {
              console.log(`Found folder matching trackerId: ${trackerId}`)
              fetchAppFiles(trackerId)
            }
            // Otherwise don't fetch files - this prevents showing files from the wrong folder
            else {
              console.warn(`No matching folder found for trackerId: ${trackerId}`)
              appFiles = [] // Set empty array to avoid showing loading state indefinitely
              loadingFiles = false
            }
          } else {
            console.warn('No app folders found')
            appFiles = [] // Set empty array to avoid showing loading state indefinitely
            loadingFiles = false
          }
        })
        .catch((error) => {
          console.error('Error fetching app folders:', error)
          appFiles = [] // Set empty array to avoid showing loading state indefinitely
          loadingFiles = false
        })
    } else {
      console.warn('No tracker or trackerId available')
      appFiles = [] // Set empty array to avoid showing loading state indefinitely
      loadingFiles = false
    }
  }

  // Rehighlight code when file content changes
  $: if (fileContent && selectedFile) {
    // Use setTimeout to ensure DOM is updated before highlighting
    setTimeout(() => {
      Prism.highlightAll()
    }, 0)
  }

  // Initialize Prism.js after component mounts
  onMount(() => {
    Prism.highlightAll()
  })
</script>

<div class="flex flex-col h-full w-full bg-background">
  <!-- Header -->
  <header
    class="sticky top-0 z-10 p-4 border-b border-border bg-background flex items-center justify-between shadow-sm"
  >
    <div class="flex items-center gap-3">
      <button
        class="p-2 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted/80 transition-colors focus:outline-none focus:ring-2 focus:ring-primary/30"
        on:click={onBackClick}
        aria-label="Back to trackers list"
      >
        <ChevronsLeft size={20} />
      </button>
      <h2 class="text-lg font-semibold text-foreground">Tracker Details</h2>
    </div>
  </header>

  <!-- Main content area -->
  <div class="flex-1 overflow-y-auto custom-scrollbar">
    {#if loading}
      <div class="flex justify-center items-center h-48">
        <div class="text-muted-foreground">Loading tracker details...</div>
      </div>
    {:else if tracker}
      <!-- Tracker header section -->
      <section
        class="px-6 py-8 border-b border-border bg-gradient-to-r from-background to-muted/20"
      >
        <div class="max-w-7xl mx-auto">
          <div class="flex flex-col md:flex-row md:justify-between md:items-start gap-6">
            <div class="flex items-start gap-5">
              <!-- Tracker icon with toggle beneath it -->
              <div class="flex flex-col items-center gap-2">
                <div
                  class="flex-shrink-0 w-16 h-16 bg-primary/10 rounded-xl flex items-center justify-center text-primary shadow-sm"
                >
                  {#if tracker.customIconPath}
                    <img src={tracker.customIconPath} alt="{tracker.name} icon" class="w-10 h-10" />
                  {:else if tracker.icon === 'Github'}
                    <Github size={28} />
                  {:else if tracker.icon === 'Mail'}
                    <Mail size={28} />
                  {:else if tracker.icon === 'FileText'}
                    <FileText size={28} />
                  {:else if tracker.icon === 'Headphones'}
                    <Headphones size={28} />
                  {:else if tracker.icon === 'MessageSquare'}
                    <MessageSquare size={28} />
                  {:else}
                    <FileCode size={28} />
                  {/if}
                </div>

                <!-- Toggle switch directly under the icon -->
                <button
                  class="flex items-center transition-colors"
                  on:click={toggleTrackerEnabled}
                  aria-label={tracker.enabled ? 'Deactivate tracker' : 'Activate tracker'}
                >
                  <div
                    class={cn(
                      'w-10 h-5 rounded-full flex items-center transition-colors px-0.5',
                      tracker.enabled ? 'bg-success' : 'bg-muted'
                    )}
                  >
                    <div
                      class={cn(
                        'w-4 h-4 bg-white rounded-full shadow-sm transition-transform',
                        tracker.enabled ? 'translate-x-5' : 'translate-x-0'
                      )}
                    ></div>
                  </div>
                </button>
              </div>

              <!-- Tracker info -->
              <div class="flex-1">
                <div class="flex flex-wrap items-center gap-3">
                  <h3 class="text-2xl font-semibold text-foreground">{tracker.name}</h3>
                  <div
                    class={cn(
                      'px-2.5 py-0.5 text-xs font-medium rounded-full',
                      tracker.enabled
                        ? 'bg-success/15 text-success border border-success/20'
                        : 'bg-muted text-muted-foreground border border-muted-foreground/20'
                    )}
                  >
                    {tracker.enabled ? 'Active' : 'Deactivated'}
                  </div>
                </div>
                <div
                  class="flex items-center gap-3 mt-1.5 text-sm font-medium text-muted-foreground"
                >
                  <span class="bg-muted px-2 py-0.5 rounded">v{tracker.version}</span>
                </div>
                <p class="mt-3 text-sm text-muted-foreground max-w-2xl leading-relaxed">
                  {tracker.description}
                </p>
              </div>
            </div>

            <!-- Actions -->
            <div class="flex flex-col gap-3 md:items-end">
              <!-- Configure button -->
              <button
                class={cn(
                  'flex items-center gap-2 px-4 py-2 rounded-md',
                  'bg-primary text-primary-foreground hover:bg-primary/90',
                  'transition-colors text-sm font-medium shadow-sm'
                )}
                on:click={handleConfigureTracker}
              >
                <Settings size={18} />
                <span>Configure</span>
              </button>
            </div>
          </div>
        </div>
      </section>

      <!-- Tabs section -->
      <section class="container max-w-7xl mx-auto px-4 sm:px-6 py-6">
        <nav class="flex border-b border-border mb-6" aria-label="Tracker sections">
          <button
            class={cn(
              'px-5 py-3 text-sm font-medium transition-colors relative',
              activeTab === 'principal'
                ? 'text-primary'
                : 'text-muted-foreground hover:text-foreground'
            )}
            on:click={() => (activeTab = 'principal')}
            aria-selected={activeTab === 'principal'}
            aria-controls="panel-templates"
          >
            <div class="flex items-center gap-2.5">
              <FileText size={18} />
              <span>Templates</span>
            </div>
            {#if activeTab === 'principal'}
              <div class="absolute bottom-0 left-0 right-0 h-0.5 bg-primary" />
            {/if}
          </button>

          <button
            class={cn(
              'px-5 py-3 text-sm font-medium transition-colors relative',
              activeTab === 'secondary'
                ? 'text-primary'
                : 'text-muted-foreground hover:text-foreground'
            )}
            on:click={() => (activeTab = 'secondary')}
            aria-selected={activeTab === 'secondary'}
            aria-controls="panel-files"
          >
            <div class="flex items-center gap-2.5">
              <LayoutList size={18} />
              <span>Files</span>
            </div>
            {#if activeTab === 'secondary'}
              <div class="absolute bottom-0 left-0 right-0 h-0.5 bg-primary" />
            {/if}
          </button>

          <button
            class={cn(
              'px-5 py-3 text-sm font-medium transition-colors relative',
              activeTab === 'code' ? 'text-primary' : 'text-muted-foreground hover:text-foreground'
            )}
            on:click={() => (activeTab = 'code')}
            aria-selected={activeTab === 'code'}
            aria-controls="panel-code"
          >
            <div class="flex items-center gap-2.5">
              <Code size={18} />
              <span>Code</span>
            </div>
            {#if activeTab === 'code'}
              <div class="absolute bottom-0 left-0 right-0 h-0.5 bg-primary" />
            {/if}
          </button>
        </nav>

        <!-- Tab content -->
        <div class="w-full pb-12" role="tabpanel" aria-live="polite">
          {#if activeTab === 'principal'}
            <!-- Principal Tab: Accordion of templates -->
            <div id="panel-templates" class="space-y-5">
              {#if Object.keys(tracker.templates).length === 0}
                <div
                  class="bg-card border border-border rounded-lg p-8 text-center flex flex-col items-center justify-center"
                >
                  <AlertCircle size={40} class="text-amber-500/80 mb-4" />
                  <p class="text-foreground font-medium text-lg">
                    No templates available for this tracker
                  </p>
                  <p class="text-muted-foreground text-sm mt-3 max-w-md">
                    The tracker configuration might be missing template files or they haven't been
                    properly set up yet. Check the tracker's documentation for more information.
                  </p>
                  <div class="mt-5 flex flex-col gap-3 max-w-md text-sm text-left">
                    <p class="text-muted-foreground font-medium">Possible solutions:</p>
                    <ul class="list-disc pl-5 text-muted-foreground space-y-2">
                      <li>Ensure the tracker is properly installed and enabled</li>
                      <li>Check if the tracker requires additional configuration</li>
                      <li>Try reinstalling the tracker if issues persist</li>
                      <li>Contact the tracker developer for support</li>
                    </ul>
                  </div>
                  <button
                    class="mt-6 px-4 py-2 text-sm bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors"
                    on:click={handleConfigureTracker}
                  >
                    <div class="flex items-center gap-2">
                      <Settings size={16} />
                      <span>Configure Tracker</span>
                    </div>
                  </button>
                </div>
              {:else}
                {#each Object.entries(tracker.templates) as [templateId, template]}
                  <div class="bg-card border border-border rounded-lg overflow-hidden shadow-sm">
                    <!-- Accordion header -->
                    <div
                      class="w-full p-4 flex justify-between items-center text-left hover:bg-muted/30 transition-colors cursor-pointer"
                      on:click={() => toggleAccordionItem(templateId)}
                      role="button"
                      tabindex="0"
                      aria-expanded={activeAccordionItems.has(templateId)}
                      aria-controls={`content-${templateId}`}
                    >
                      <div class="flex items-center gap-3">
                        <div
                          class="bg-primary/10 w-8 h-8 rounded-md flex items-center justify-center text-primary"
                        >
                          <FileText size={18} />
                        </div>
                        <div>
                          <span class="font-medium text-foreground">{templateId}</span>
                          <div class="text-xs text-muted-foreground mt-0.5">
                            {template.filename}
                          </div>
                        </div>
                      </div>
                      <ChevronDown
                        size={18}
                        class={cn(
                          'text-muted-foreground transition-transform duration-200',
                          activeAccordionItems.has(templateId) && 'transform rotate-180'
                        )}
                      />
                    </div>

                    <!-- Accordion content -->
                    {#if activeAccordionItems.has(templateId)}
                      <div
                        id={`content-${templateId}`}
                        class="p-5 pt-0 pb-6 border-t border-border bg-card/30"
                      >
                        <div
                          class="prose prose-sm dark:prose-invert max-w-none mt-4 overflow-x-auto"
                        >
                          {@html sanitizeMarkdown(template.content)}
                        </div>
                      </div>
                    {/if}
                  </div>
                {/each}
              {/if}
            </div>
          {:else if activeTab === 'secondary'}
            <!-- Secondary Tab: List of files -->
            <div
              id="panel-files"
              class="bg-card border border-border rounded-lg overflow-hidden shadow-sm"
            >
              <div class="p-4 border-b border-border bg-muted/30">
                <div class="grid grid-cols-12 gap-4 text-sm font-medium text-muted-foreground">
                  <div class="col-span-5 sm:col-span-6">Filename</div>
                  <div class="col-span-5 sm:col-span-4">Template ID</div>
                  <div class="col-span-2 sm:col-span-2 text-right">Action</div>
                </div>
              </div>

              <div class="divide-y divide-border">
                {#if loadingDatasets}
                  <div class="p-8 text-center flex flex-col items-center justify-center">
                    <div class="animate-pulse mb-4 w-8 h-8 rounded-full bg-muted"></div>
                    <p class="text-muted-foreground">Loading files...</p>
                  </div>
                {:else if Object.keys(datasets).length === 0}
                  <div class="p-8 text-center flex flex-col items-center justify-center">
                    <LayoutList size={40} class="text-muted-foreground/50 mb-4" />
                    <p class="text-muted-foreground font-medium">
                      No files available for this tracker.
                    </p>
                    <p class="text-muted-foreground text-sm mt-2 max-w-md">
                      This tracker might not have created any output files yet or they might be
                      stored in a different location. Try using the tracker first or check its
                      configuration settings.
                    </p>
                  </div>
                {:else}
                  {#each Object.entries(datasets) as [filename, templateId]}
                    <div class="p-4 hover:bg-muted/20 transition-colors">
                      <div class="grid grid-cols-12 gap-4 text-sm items-center">
                        <div class="col-span-5 sm:col-span-6 flex items-center gap-3">
                          <div
                            class="bg-muted/50 w-8 h-8 rounded-md flex items-center justify-center text-muted-foreground"
                          >
                            <FileText size={16} />
                          </div>
                          <span class="truncate font-medium" title={filename}>
                            {filename}
                          </span>
                        </div>
                        <div class="col-span-5 sm:col-span-4 flex items-center">
                          <code class="px-2 py-1 bg-muted rounded-md text-xs font-mono">
                            {templateId}
                          </code>
                        </div>
                        <div class="col-span-2 sm:col-span-2 flex justify-end">
                          <button
                            class="p-2 rounded-md text-muted-foreground hover:text-primary hover:bg-muted/80 transition-colors"
                            on:click={() => handleTemplateAction(templateId)}
                            title="View template details"
                          >
                            <ExternalLink size={16} />
                          </button>
                        </div>
                      </div>
                    </div>
                  {/each}
                {/if}
              </div>
            </div>
          {:else if activeTab === 'code'}
            <!-- Code Tab: File Tree Explorer -->
            <div id="panel-code">
              {#if isCodeSectionExpanded}
                <!-- Solid background overlay for fullscreen mode -->
                <div
                  class="fixed top-0 right-0 bottom-0 left-0 bg-black z-40"
                  on:click={toggleCodeSectionExpansion}
                ></div>
              {/if}
              <div
                class={cn(
                  'bg-card border border-border overflow-hidden transition-all duration-300 shadow-sm',
                  isCodeSectionExpanded
                    ? 'fixed top-8 right-0 bottom-0 left-0 z-50 mx-0'
                    : 'relative rounded-lg'
                )}
              >
                <div
                  class={cn(
                    'flex flex-row',
                    isCodeSectionExpanded ? 'h-[calc(100vh-8px)] w-full' : 'h-[calc(100vh-380px)]'
                  )}
                >
                  <!-- Code header bar -->
                  <div
                    class="absolute top-0 left-0 right-0 flex justify-between items-center h-12 bg-muted border-b border-border px-4 z-10"
                  >
                    <div class="flex items-center gap-2 text-sm font-medium text-foreground">
                      <FileCode size={16} class="text-primary" />
                      <span>Source Code Explorer</span>
                    </div>

                    <!-- Controls for fullscreen/minimize -->
                    <div class="flex items-center">
                      <button
                        class="p-2 rounded-md text-muted-foreground hover:text-primary transition-colors flex items-center gap-1.5"
                        on:click={toggleCodeSectionExpansion}
                        aria-label={isCodeSectionExpanded ? 'Exit fullscreen' : 'Fullscreen mode'}
                      >
                        {#if isCodeSectionExpanded}
                          <Minimize2 size={16} />
                          <span class="text-xs font-medium">Exit Fullscreen</span>
                        {:else}
                          <Maximize2 size={16} />
                          <span class="text-xs font-medium">Fullscreen</span>
                        {/if}
                      </button>
                    </div>
                  </div>

                  <!-- File tree sidebar -->
                  <div
                    class={cn(
                      'bg-muted/30 border-r border-border overflow-y-auto custom-scrollbar pt-12' /* Adjusted padding-top for header */,
                      isCodeSectionExpanded ? 'w-1/5' : 'w-1/3'
                    )}
                  >
                    {#if loadingFiles}
                      <div class="p-8 text-center flex flex-col items-center justify-center">
                        <div class="animate-pulse mb-4 w-8 h-8 rounded-full bg-muted"></div>
                        <p class="text-muted-foreground">Loading source files...</p>
                      </div>
                    {:else if appFiles.length === 0}
                      <div class="p-8 text-center flex flex-col items-center justify-center">
                        <Folder size={40} class="text-muted-foreground/50 mb-4" />
                        <p class="text-muted-foreground font-medium">No source files available.</p>
                        <p class="text-muted-foreground text-sm mt-2 max-w-md">
                          This tracker doesn't provide access to its source code or there was an
                          error when trying to load the files. This could be due to permission
                          issues or the tracker's configuration.
                        </p>
                        <button
                          class="mt-4 px-3 py-1.5 text-xs bg-muted text-muted-foreground rounded-md hover:bg-muted/80 transition-colors"
                          on:click={() =>
                            toast.info('Attempting to reload source files...', {
                              title: 'Retry',
                              duration: 2000
                            }) && fetchAppFiles(tracker?.folderName || trackerId)}
                        >
                          Try again
                        </button>
                      </div>
                    {:else}
                      <div class="p-3">
                        <!-- Adjusted padding -->
                        <!-- Recursive directory tree component -->
                        {#each appFiles as item}
                          <div class="mb-1.5">
                            {#if item.type === 'directory'}
                              <div
                                class="px-3 py-2 text-sm font-medium text-foreground rounded-md hover:bg-muted transition-colors cursor-pointer"
                                on:click={() => toggleDirectory(item.path)}
                              >
                                <div class="flex items-center gap-2.5">
                                  <Folder
                                    size={18}
                                    class={expandedDirs.has(item.path)
                                      ? 'text-primary'
                                      : 'text-muted-foreground'}
                                  />
                                  <span>{item.name}</span>
                                  <ChevronDown
                                    size={16}
                                    class={cn(
                                      'ml-auto transition-transform duration-200',
                                      expandedDirs.has(item.path) ? 'transform rotate-180' : ''
                                    )}
                                  />
                                </div>
                              </div>
                              {#if item.children?.length && expandedDirs.has(item.path)}
                                <div class="ml-4 border-l border-border/70 pl-3 mt-1 mb-2">
                                  {#each item.children as child}
                                    {#if child.type === 'directory'}
                                      <div class="mb-1.5">
                                        <div
                                          class="px-3 py-1.5 text-sm font-medium text-foreground rounded-md hover:bg-muted transition-colors cursor-pointer"
                                          on:click|stopPropagation={() =>
                                            toggleDirectory(child.path)}
                                        >
                                          <div class="flex items-center gap-2.5">
                                            <Folder
                                              size={16}
                                              class={expandedDirs.has(child.path)
                                                ? 'text-primary'
                                                : 'text-muted-foreground'}
                                            />
                                            <span>{child.name}</span>
                                            <ChevronDown
                                              size={14}
                                              class={cn(
                                                'ml-auto transition-transform duration-200',
                                                expandedDirs.has(child.path)
                                                  ? 'transform rotate-180'
                                                  : ''
                                              )}
                                            />
                                          </div>
                                        </div>
                                        {#if child.children?.length && expandedDirs.has(child.path)}
                                          <div
                                            class="ml-3 border-l border-border/70 pl-3 mt-1 mb-1"
                                          >
                                            {#each child.children as subchild}
                                              {#if subchild.type === 'directory'}
                                                <div
                                                  class="px-3 py-1.5 text-sm font-medium text-foreground rounded-md hover:bg-muted transition-colors cursor-pointer mb-1"
                                                  on:click|stopPropagation={() =>
                                                    toggleDirectory(subchild.path)}
                                                >
                                                  <div class="flex items-center gap-2">
                                                    <Folder
                                                      size={16}
                                                      class={expandedDirs.has(subchild.path)
                                                        ? 'text-primary'
                                                        : 'text-muted-foreground'}
                                                    />
                                                    <span>{subchild.name}</span>
                                                    <ChevronDown
                                                      size={14}
                                                      class={cn(
                                                        'ml-auto transition-transform duration-200',
                                                        expandedDirs.has(subchild.path)
                                                          ? 'transform rotate-180'
                                                          : ''
                                                      )}
                                                    />
                                                  </div>
                                                </div>
                                                {#if subchild.children?.length && expandedDirs.has(subchild.path)}
                                                  <div
                                                    class="ml-3 border-l border-border/70 pl-3 mt-1 mb-1"
                                                  >
                                                    {#each subchild.children as grandchild}
                                                      <div
                                                        class={cn(
                                                          'px-3 py-1.5 text-sm rounded-md hover:bg-muted transition-colors cursor-pointer mb-1',
                                                          selectedFile === grandchild.path
                                                            ? 'bg-primary/10 text-primary font-medium'
                                                            : 'text-muted-foreground'
                                                        )}
                                                        on:click|stopPropagation={() =>
                                                          fetchFileContent(grandchild.path)}
                                                      >
                                                        <div class="flex items-center gap-2">
                                                          <File
                                                            size={16}
                                                            class={selectedFile === grandchild.path
                                                              ? 'text-primary'
                                                              : 'text-muted-foreground'}
                                                          />
                                                          <span>{grandchild.name}</span>
                                                        </div>
                                                      </div>
                                                    {/each}
                                                  </div>
                                                {/if}
                                              {:else}
                                                <div
                                                  class={cn(
                                                    'px-3 py-1.5 text-sm rounded-md hover:bg-muted transition-colors cursor-pointer mb-1',
                                                    selectedFile === subchild.path
                                                      ? 'bg-primary/10 text-primary font-medium'
                                                      : 'text-muted-foreground'
                                                  )}
                                                  on:click|stopPropagation={() =>
                                                    fetchFileContent(subchild.path)}
                                                >
                                                  <div class="flex items-center gap-2">
                                                    <File
                                                      size={16}
                                                      class={selectedFile === subchild.path
                                                        ? 'text-primary'
                                                        : 'text-muted-foreground'}
                                                    />
                                                    <span>{subchild.name}</span>
                                                  </div>
                                                </div>
                                              {/if}
                                            {/each}
                                          </div>
                                        {/if}
                                      </div>
                                    {:else}
                                      <div
                                        class={cn(
                                          'px-3 py-1.5 text-sm rounded-md hover:bg-muted transition-colors cursor-pointer mb-1',
                                          selectedFile === child.path
                                            ? 'bg-primary/10 text-primary font-medium'
                                            : 'text-muted-foreground'
                                        )}
                                        on:click|stopPropagation={() =>
                                          fetchFileContent(child.path)}
                                      >
                                        <div class="flex items-center gap-2">
                                          <File
                                            size={16}
                                            class={selectedFile === child.path
                                              ? 'text-primary'
                                              : 'text-muted-foreground'}
                                          />
                                          <span>{child.name}</span>
                                        </div>
                                      </div>
                                    {/if}
                                  {/each}
                                </div>
                              {/if}
                            {:else}
                              <div
                                class={cn(
                                  'px-3 py-2 text-sm rounded-md hover:bg-muted transition-colors cursor-pointer',
                                  selectedFile === item.path
                                    ? 'bg-primary/10 text-primary font-medium'
                                    : 'text-muted-foreground'
                                )}
                                on:click={() => fetchFileContent(item.path)}
                              >
                                <div class="flex items-center gap-2.5">
                                  <File
                                    size={18}
                                    class={selectedFile === item.path
                                      ? 'text-primary'
                                      : 'text-muted-foreground'}
                                  />
                                  <span>{item.name}</span>
                                </div>
                              </div>
                            {/if}
                          </div>
                        {/each}
                      </div>
                    {/if}
                  </div>

                  <!-- File content pane -->
                  <div
                    class={cn(
                      'overflow-y-auto custom-scrollbar pt-12' /* Adjusted padding-top for header */,
                      isCodeSectionExpanded ? 'w-4/5' : 'w-2/3'
                    )}
                  >
                    {#if !selectedFile}
                      <div
                        class="flex flex-col items-center justify-center h-full p-6 text-center text-muted-foreground"
                      >
                        <FileCode size={48} class="text-muted-foreground/50 mb-4" />
                        <p class="font-medium">
                          Select a file from the sidebar to view its content
                        </p>
                        <p class="text-sm mt-2 max-w-md">
                          Browse through the source code of this tracker to understand how it works
                        </p>
                      </div>
                    {:else}
                      <div class="px-4 py-3 border-b border-border bg-muted/20">
                        <!-- Adjusted padding -->
                        <div class="flex items-center gap-2.5 text-sm font-medium">
                          <FileCode size={16} class="text-primary" />
                          <span class="text-foreground">{selectedFile}</span>
                        </div>
                      </div>
                      {#if fileContent === 'Loading...'}
                        <div class="p-8 text-center flex flex-col items-center justify-center">
                          <div class="animate-pulse mb-4 w-8 h-8 rounded-full bg-muted"></div>
                          <p class="text-muted-foreground">Loading file content...</p>
                        </div>
                      {:else}
                        <div class="p-4">
                          <pre
                            class="text-sm font-mono p-4 rounded-md overflow-x-auto custom-scrollbar line-numbers shadow-sm"><code
                              class="language-{selectedFile?.split('.').pop()?.toLowerCase() ||
                                'plain'}"
                              >{@html getHighlightedCode(fileContent, selectedFile || '')}</code
                            ></pre>
                        </div>
                      {/if}
                    {/if}
                  </div>
                </div>
              </div>
            </div>
          {/if}
        </div>
      </section>
    {:else}
      <!-- Error state -->
      <div class="flex flex-col items-center justify-center min-h-[400px] p-8">
        <div class="bg-destructive/10 p-5 rounded-full mb-6">
          <AlertCircle size={40} class="text-destructive" />
        </div>
        <h3 class="text-xl font-semibold text-foreground mb-3">Tracker not found</h3>
        <p class="text-muted-foreground text-center max-w-md mb-2">
          The requested tracker could not be found or has been removed.
        </p>
        <p class="text-muted-foreground text-center text-sm max-w-md mb-8">
          Please check if the tracker is installed correctly or contact the administrator.
        </p>
        <button
          class="px-5 py-2.5 bg-primary text-primary-foreground rounded-md text-sm font-medium hover:bg-primary/90 transition-colors shadow-sm focus:outline-none focus:ring-2 focus:ring-primary/30"
          on:click={onBackClick}
        >
          Return to Trackers
        </button>
      </div>
    {/if}
  </div>
</div>

<!-- Tracker Config Modal -->
<TrackerConfigModal
  showModal={showConfigModal}
  {trackerId}
  on:close={handleConfigModalClose}
  on:configUpdated={handleConfigUpdated}
/>

<style>
  /* Remove default margin from pre elements */
  :global(pre[class*='language-']) {
    margin: 0;
  }

  /* Atom-inspired syntax highlighting styles */
  :global(.token.comment),
  :global(.token.prolog),
  :global(.token.doctype),
  :global(.token.cdata) {
    color: #7d8799;
  }

  :global(.token.punctuation) {
    color: #abb2bf;
  }

  :global(.token.namespace) {
    opacity: 0.7;
  }

  :global(.token.property),
  :global(.token.tag),
  :global(.token.constant),
  :global(.token.symbol),
  :global(.token.deleted) {
    color: #e06c75;
  }

  :global(.token.boolean),
  :global(.token.number) {
    color: #d19a66;
  }

  :global(.token.selector),
  :global(.token.attr-name),
  :global(.token.string),
  :global(.token.char),
  :global(.token.builtin),
  :global(.token.inserted) {
    color: #98c379;
  }

  :global(.token.operator),
  :global(.token.entity),
  :global(.token.url),
  :global(.language-css .token.string),
  :global(.style .token.string),
  :global(.token.variable) {
    color: #56b6c2;
  }

  :global(.token.atrule),
  :global(.token.attr-value),
  :global(.token.function),
  :global(.token.class-name) {
    color: #61afef;
  }

  :global(.token.keyword) {
    color: #c678dd;
  }

  :global(.token.regex),
  :global(.token.important) {
    color: #e5c07b;
  }

  :global(.line-numbers .line-numbers-rows) {
    border-right: 1px solid rgba(171, 178, 191, 0.2);
  }

  /* Enhance code blocks to more closely match Atom */
  :global(pre[class*='language-']) {
    background: #282c34;
    border-radius: 4px;
  }

  /* Hide scrollbars but keep functionality */
  .scrollbar-hide {
    -ms-overflow-style: none; /* IE and Edge */
    scrollbar-width: none; /* Firefox */
  }
  .scrollbar-hide::-webkit-scrollbar {
    display: none; /* Chrome, Safari and Opera */
  }

  /* Fullscreen transition effect */
  div.fixed {
    animation: expandAnimation 0.3s ease-in-out;
  }

  @keyframes expandAnimation {
    from {
      opacity: 0.8;
      transform: scale(0.98);
    }
    to {
      opacity: 1;
      transform: scale(1);
    }
  }
</style>
