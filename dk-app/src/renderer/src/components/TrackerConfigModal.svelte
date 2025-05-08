<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte'
  import { X, Save, Check, AlertCircle, Info } from 'lucide-svelte'
  import { cn } from '../lib/utils'
  import { toast } from '../lib/toast'

  export let showModal = false
  export let trackerId: string = ''

  /**
   * Represents a form field option (for select, radio, checkbox)
   */
  interface FormFieldOption {
    value: string
    label: string
  }

  /**
   * Type definition for all supported input types in the form
   */
  type FormFieldType = 'text' | 'password' | 'number' | 'radio' | 'select' | 'checkbox' | 'file'

  /**
   * Represents a field in the form.json configuration
   */
  interface FormField {
    id: string // Unique identifier for the field
    variable_id: string // Variable name used in config.json
    type: FormFieldType // Type of the input
    title?: string // Section title (optional)
    label?: string // Field label
    description?: string // Help text
    required?: boolean // If field is required
    placeholder?: string // Placeholder text
    min?: number // Min value for number input
    max?: number // Max value for number input
    accept?: string // File extensions for file input (e.g., ".jpg,.png")
    options?: FormFieldOption[] // Options for select, radio, checkbox
  }

  /**
   * Represents the full form definition from form.json
   */
  interface FormDefinition {
    formTitle: string // Form title displayed at the top
    fields: FormField[] // Array of form fields
  }

  /**
   * Represents basic information about a tracker app
   */
  interface TrackerInfo {
    name: string // Tracker name
    description: string // Tracker description
    version: string // Tracker version
    enabled: boolean // Whether tracker is active
    icon: string // Icon identifier
    customIconPath: string // Path to custom icon
    dataPath: string // Path to tracker data folder
  }

  let trackerInfo: TrackerInfo = {
    name: '',
    description: '',
    version: '',
    enabled: false,
    icon: '',
    customIconPath: '',
    dataPath: ''
  }

  // This will hold the form definition from form.json
  let formDefinition: FormDefinition | null = null

  // This will hold the actual config values
  let configValues: Record<string, any> = {}

  let isLoading = true
  let isSaving = false
  let errorMessage = ''
  let successMessage = ''
  let activeTab = 'general'

  // Track if data has been loaded successfully
  let dataLoaded = false

  // Track last trackerId that was loaded to prevent redundant loading
  let lastLoadedTrackerId = ''

  const dispatch = createEventDispatcher<{
    close: void
    configUpdated: void
  }>()

  // Use onMount to start initial loading
  onMount(() => {
    console.log('TrackerConfigModal component mounted')
  })

  // Watch showModal and trackerId changes
  $: if (showModal && trackerId && trackerId !== lastLoadedTrackerId) {
    console.log(
      `Modal state or trackerId changed. trackerId: ${trackerId}, Type: ${typeof trackerId}, Loading fresh data...`
    )
    lastLoadedTrackerId = trackerId
    loadTrackerData()
  }

  async function loadTrackerData() {
    try {
      console.log(`Loading tracker data for ID: ${trackerId}`)
      isLoading = true
      errorMessage = ''
      dataLoaded = false
      formDefinition = null // Reset form definition
      configValues = {} // Reset config values

      // Make sure window.api exists
      if (!window.api) {
        console.error('window.api is undefined')
        errorMessage = 'API not available'
        return
      }

      if (!window.api.apps) {
        console.error('window.api.apps is undefined')
        errorMessage = 'Apps API not available'
        return
      }

      console.log('Step 1: Fetching app trackers...')
      // Get basic tracker info from the tracker list
      const appTrackersResponse = await window.api.apps.getAppTrackers()

      if (!appTrackersResponse.success || !appTrackersResponse.appTrackers) {
        console.error('Failed to fetch app trackers:', appTrackersResponse.error)
        errorMessage = 'Failed to fetch tracker information'
        return
      }

      // Debug: Print all app trackers with their IDs and paths to better understand the structure
      console.log('All app trackers:')
      appTrackersResponse.appTrackers.forEach((app, index) => {
        console.log(`[${index}] ID: ${app.id}, Name: ${app.name}, Path: ${app.path || 'none'}`)
      })

      // Find the tracker with matching ID
      const matchingTracker = appTrackersResponse.appTrackers.find((app) => app.id === trackerId)

      if (!matchingTracker) {
        console.error(`No tracker found with ID: ${trackerId}`)
        errorMessage = `No tracker found with ID: ${trackerId}`
        return
      }

      console.log('Step 2: Found matching tracker:', matchingTracker.name)
      console.log('Matching tracker full details:', JSON.stringify(matchingTracker, null, 2))

      // Try to get custom icon for the tracker if available, passing the path to avoid another getAppTrackers call
      let customIconPath = ''
      try {
        customIconPath =
          (await window.api.apps.getAppIconPath(trackerId, matchingTracker.path)) || ''
      } catch (error) {
        console.warn('Failed to get custom icon path:', error)
        // Non-critical error, continue without icon
      }

      // Store basic tracker info
      trackerInfo = {
        name: matchingTracker.name,
        description: matchingTracker.description,
        version: matchingTracker.version,
        enabled: matchingTracker.enabled,
        icon: matchingTracker.icon,
        customIconPath: customIconPath,
        dataPath: ''
      }

      // We have the path in the matchingTracker object - let's extract the folder name
      console.log('Step 3: Extracting folder name from path')

      if (!matchingTracker.path) {
        console.error(
          `No path information available for tracker "${matchingTracker.name}" (ID: ${trackerId})`
        )
        errorMessage = `No configuration path found for tracker: ${matchingTracker.name}`
        return
      }

      // Extract the folder name from the full path
      // The path looks like: /path/to/apps/folder-name
      const pathParts = matchingTracker.path.split('/')
      const folderName = pathParts[pathParts.length - 1]

      console.log(`Extracted folder name: "${folderName}" from path: ${matchingTracker.path}`)

      // Double-check if the folder exists
      const appFoldersResponse = await window.api.trackers.getAppFolders()

      if (
        !appFoldersResponse.success ||
        !appFoldersResponse.folders ||
        appFoldersResponse.folders.length === 0
      ) {
        console.error('Failed to fetch app folders:', appFoldersResponse.error)
        errorMessage = 'Failed to fetch app folders'
        return
      }

      console.log('Available folders:', appFoldersResponse.folders)

      // Verify the extracted folder exists in the available folders
      if (!appFoldersResponse.folders.includes(folderName)) {
        console.error(
          `Extracted folder "${folderName}" not found in available folders. This should not happen.`
        )
        console.log('Will continue with the extracted folder name anyway.')
      }

      // Use the folder name directly from the path
      const targetFolder = folderName
      console.log(`Using folder name "${targetFolder}" extracted from tracker path`)

      // Store data path
      trackerInfo.dataPath = targetFolder

      // Load form definition first, as it might be needed to initialize config values
      console.log(`Step 4: Loading form definition using folder: ${targetFolder}`)
      await loadFormDefinition(targetFolder)

      // Then load config values
      console.log(`Step 5: Loading configuration values using folder: ${targetFolder}`)
      await loadConfigValues(targetFolder)

      // Mark data as successfully loaded
      dataLoaded = true
      console.log('Data loading complete')
    } catch (error) {
      console.error('Failed to load tracker information:', error)
      errorMessage = 'Failed to load tracker information'
    } finally {
      isLoading = false
    }
  }

  async function loadFormDefinition(folderName: string) {
    try {
      console.log(`Loading form definition from folder: ${folderName}`)
      const formResponse = await window.api.trackers.getTrackerForm(folderName)

      if (formResponse.success && formResponse.form) {
        formDefinition = formResponse.form
        console.log('Form definition loaded successfully:', formDefinition.formTitle)

        // Validate form definition has required properties
        if (!formDefinition.fields || !Array.isArray(formDefinition.fields)) {
          console.error(
            `Invalid form definition for ${folderName}: Missing or invalid 'fields' array`
          )
          errorMessage = 'Invalid form definition format'
          formDefinition = null
          return
        }

        // Log field count for debugging
        console.log(`Form has ${formDefinition.fields.length} fields defined`)
      } else {
        console.warn(`No form definition found for folder ${folderName}:`, formResponse.error)
        errorMessage = `No form definition found for this tracker: ${formResponse.error || ''}`
        formDefinition = null
      }
    } catch (error) {
      console.error(`Failed to load form definition from folder ${folderName}:`, error)
      errorMessage = `Error loading form definition: ${error.message || 'Unknown error'}`
      formDefinition = null
    }
  }

  /**
   * Loads configuration values for the given tracker folder
   * @param folderName The folder name of the tracker
   */
  async function loadConfigValues(folderName: string) {
    try {
      console.log(`Loading config values from folder: ${folderName}`)
      const configResponse = await window.api.trackers.getTrackerConfig(folderName)

      // Also load the app-level config.json to check for variable_id values
      let appConfig = {}
      try {
        const appConfigResponse = await window.api.trackers.getAppConfig()
        if (appConfigResponse.success) {
          appConfig = appConfigResponse.config || {}
          console.log('App config loaded successfully:', Object.keys(appConfig).length, 'entries')
        } else {
          console.warn('App config response not successful:', appConfigResponse.error)
        }
      } catch (error) {
        console.warn('Failed to load app-level config.json:', error)
        // Non-critical error, continue without app config
      }

      if (configResponse.success) {
        const loadedConfig = configResponse.config || {}
        console.log(
          'Config values loaded successfully:',
          Object.keys(loadedConfig).length,
          'entries'
        )

        // Create a new object for proper reactivity
        const newConfigValues: Record<string, any> = {}

        // Copy existing values from tracker config
        Object.keys(loadedConfig).forEach((key) => {
          newConfigValues[key] = loadedConfig[key]
          console.log(`Loaded config value for ${key} from tracker config`)
        })

        // Initialize values if form definition is available
        if (formDefinition && formDefinition.fields) {
          console.log('Processing', formDefinition.fields.length, 'form fields')

          formDefinition.fields.forEach((field) => {
            if (!field.variable_id) {
              console.warn(`Field is missing required variable_id:`, field)
              return
            }

            const variableId = field.variable_id

            // First check if the value exists in app-level config.json
            if (variableId in appConfig) {
              console.log(
                `Found value for ${variableId} in app-level config.json:`,
                appConfig[variableId]
              )
              newConfigValues[variableId] = appConfig[variableId]
            }
            // If not in app config and not already in tracker config, initialize with default
            else if (!(variableId in newConfigValues)) {
              console.log(
                `Initializing field ${variableId} with default value based on type: ${field.type}`
              )

              // Set default values based on field type
              if (field.type === 'checkbox') {
                // For checkboxes, initialize with an empty array
                newConfigValues[variableId] = []
              } else if (field.type === 'radio' || field.type === 'select') {
                // For radio buttons and selects, use the first option value or empty string
                newConfigValues[variableId] =
                  field.options && field.options.length ? field.options[0].value : ''
              } else if (field.type === 'number') {
                // For number inputs, initialize with the placeholder value if available, or 0
                newConfigValues[variableId] = field.placeholder ? parseInt(field.placeholder) : 0
              } else if (field.type === 'file') {
                // For file inputs, initialize with empty string
                newConfigValues[variableId] = ''
              } else {
                // For text inputs, initialize with empty string
                newConfigValues[variableId] = ''
              }
            } else {
              console.log(
                `Value for ${variableId} already exists in tracker config:`,
                newConfigValues[variableId]
              )
            }
          })
        } else {
          console.warn('No form definition available, skipping field initialization')
        }

        // Update the configValues with the new object
        configValues = newConfigValues
        console.log('Final config values:', Object.keys(configValues).length, 'entries')
      } else {
        console.warn(`Failed to load config from folder ${folderName}:`, configResponse.error)
        errorMessage = `Could not load configuration: ${configResponse.error || 'Unknown error'}`
        configValues = {}
      }
    } catch (error) {
      console.error(`Failed to load config values from folder ${folderName}:`, error)
      errorMessage = `Error loading configuration: ${error.message || 'Unknown error'}`
      configValues = {}
    }
  }

  async function saveTrackerConfig() {
    try {
      isSaving = true
      errorMessage = ''
      successMessage = ''

      // Check if we have a valid folder path
      if (!trackerInfo.dataPath) {
        errorMessage = 'No valid folder path for saving configuration'
        isSaving = false
        return
      }

      // Basic validation for required fields
      let isValid = true
      let missingFields: string[] = []

      if (formDefinition) {
        formDefinition.fields.forEach((field) => {
          if (field.required) {
            const value = configValues[field.variable_id]
            if (
              value === undefined ||
              value === null ||
              value === '' ||
              (Array.isArray(value) && value.length === 0)
            ) {
              isValid = false
              missingFields.push(field.label || field.variable_id)
            }
          }
        })
      }

      if (!isValid) {
        errorMessage = `Please fill in all required fields: ${missingFields.join(', ')}`
        isSaving = false
        return
      }

      // Use the folder name from trackerInfo, not the numeric ID
      console.log(`Saving configuration to folder: ${trackerInfo.dataPath}`)
      const saveResult = await window.api.trackers.saveTrackerConfig(
        trackerInfo.dataPath,
        configValues
      )

      if (saveResult.success) {
        // Also update the app-level config.json with form values
        if (formDefinition && formDefinition.fields) {
          try {
            console.log('Updating app-level config.json with form values')

            // Extract the variable_ids from form.json and create a subset of configValues to update
            const formValuesToSync = {}
            formDefinition.fields.forEach((field) => {
              // Only include values that are defined in the form
              if (field.variable_id && field.variable_id in configValues) {
                formValuesToSync[field.variable_id] = configValues[field.variable_id]
              }
            })

            if (Object.keys(formValuesToSync).length > 0) {
              console.log('Form values to sync with app config:', formValuesToSync)
              const updateResult = await window.api.trackers.updateAppConfig(formValuesToSync)

              if (updateResult.success) {
                console.log('Successfully updated app-level config.json')
              } else {
                console.warn('Failed to update app-level config.json:', updateResult.error)
              }
            } else {
              console.log('No form values to sync with app config')
            }
          } catch (error) {
            console.error('Error updating app-level config.json:', error)
            // This is a non-critical error, so we'll continue
          }
        }

        successMessage = 'Tracker configuration saved successfully'
        dataLoaded = true

        // Notify parent component that config was updated
        dispatch('configUpdated')

        toast.success('Tracker configuration saved', {
          title: 'Success',
          duration: 3000
        })

        // Close modal after saving
        setTimeout(() => {
          closeModal()
        }, 1500)
      } else {
        throw new Error(saveResult.error || 'Failed to save configuration')
      }
    } catch (error) {
      console.error('Failed to save tracker configuration:', error)
      errorMessage = `Failed to save tracker configuration: ${error.message || 'Unknown error'}`
    } finally {
      isSaving = false
    }
  }

  // Helper for handling checkbox changes
  function handleCheckboxChange(variableId: string, value: string, event: Event) {
    const checked = (event.target as HTMLInputElement).checked
    let values = [...(configValues[variableId] || [])]

    if (checked) {
      if (!values.includes(value)) {
        values.push(value)
      }
    } else {
      values = values.filter((v) => v !== value)
    }

    configValues[variableId] = values
    configValues = { ...configValues } // Trigger reactivity
  }

  // Check if a checkbox is checked
  function isChecked(variableId: string, value: string): boolean {
    const values = configValues[variableId] || []
    return values.includes(value)
  }

  // Handle file selection using native dialog
  async function handleFileUpload(_event: Event, field: FormField) {
    try {
      errorMessage = ''

      // Get file extensions from accept attribute if available
      let extensions: string[] = []
      if (field.accept) {
        extensions = field.accept.split(',').map((ext) => ext.trim())
      }

      console.log(
        `Opening file dialog for field: ${field.variable_id}, accepting: ${extensions.join(', ')}`
      )

      // Show native file dialog via main process
      const dialogResult = await window.api.trackers.showFileDialog(
        trackerInfo.dataPath, // Use the folder name directly instead of trackerId
        field.variable_id,
        { extensions }
      )

      // Handle dialog result
      if (!dialogResult.success) {
        if (dialogResult.canceled) {
          console.log('File selection was canceled by user')
          return
        }

        console.error('Failed to select file:', dialogResult.error)
        errorMessage = `Failed to select file: ${dialogResult.error}`
        return
      }

      // Update the config value with the file path
      configValues[field.variable_id] = dialogResult.filePath
      // Force reactivity update
      configValues = { ...configValues }

      console.log(`File uploaded successfully: ${dialogResult.filePath}`)
      toast.success('File uploaded successfully', {
        title: 'Success',
        duration: 3000
      })
    } catch (error) {
      console.error('Error handling file selection:', error)
      errorMessage = `Failed to select file: ${error.message || 'Unknown error'}`
    }
  }

  function getFileNameFromPath(path: string) {
    if (!path) return 'No file selected'

    // Extract the file name from the path
    // Handle both relative paths and absolute paths from syftboxConfig.data_dir
    const parts = path.split('/')

    // Check if this looks like a full path (starting with / or containing syftbox)
    if (path.startsWith('/') || path.includes('syftbox')) {
      // Show the basename with a tooltip/title containing the full path
      return parts[parts.length - 1]
    } else {
      // For relative paths, just return the filename
      return parts[parts.length - 1]
    }
  }

  function closeModal() {
    console.log('Closing modal and resetting state')

    // Reset all state
    isLoading = true
    errorMessage = ''
    successMessage = ''
    formDefinition = null
    configValues = {}
    dataLoaded = false

    // Reset the last loaded tracker ID to ensure fresh loading next time
    lastLoadedTrackerId = ''

    // Reset tracker info
    trackerInfo = {
      name: '',
      description: '',
      version: '',
      enabled: false,
      icon: '',
      customIconPath: '',
      dataPath: ''
    }

    // Notify parent component
    dispatch('close')
  }

  function changeTab(tab: string) {
    activeTab = tab
  }
</script>

{#if showModal}
  <div class="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
    <div
      class="bg-background rounded-lg max-w-4xl w-full max-h-[90vh] overflow-hidden flex flex-col shadow-xl"
    >
      <!-- Debug info -->
      {#if isLoading || (import.meta.env.DEV && !formDefinition)}
        <div class="absolute top-0 right-0 bg-black/80 text-white text-xs p-2 m-1 rounded">
          <div>Loading: {isLoading}</div>
          <div>TrackerId: {trackerId || 'none'}</div>
          {#if trackerInfo.dataPath}
            <div>Folder: {trackerInfo.dataPath}</div>
          {/if}
        </div>
      {/if}
      <div class="p-4 border-b border-border flex justify-between items-center">
        <h2 class="text-xl font-semibold">Tracker Configuration</h2>
        <button
          class="p-2 hover:bg-accent rounded-md text-muted-foreground hover:text-foreground transition-colors"
          on:click={closeModal}
          aria-label="Close configuration"
        >
          <X size={20} />
        </button>
      </div>

      {#if isLoading}
        <div class="flex-1 flex flex-col justify-center items-center py-12">
          <div
            class="animate-spin h-8 w-8 border-4 border-primary border-t-transparent rounded-full mb-4"
          ></div>
          <div class="text-sm text-muted-foreground">
            Loading configuration for {trackerInfo.name || trackerId}...
          </div>
          <!-- Additional debug info in development environment -->
          <div class="mt-4 text-xs text-muted-foreground opacity-70">
            Tracker ID: {trackerId}
          </div>
        </div>
      {:else}
        <div class="flex flex-1 overflow-hidden">
          <!-- Sidebar -->
          <div class="w-52 border-r border-border p-4 space-y-2">
            <button
              class={cn(
                'w-full text-left px-3 py-2 rounded-md transition-colors',
                activeTab === 'general' ? 'bg-primary text-primary-foreground' : 'hover:bg-accent'
              )}
              on:click={() => changeTab('general')}
            >
              General
            </button>
            <button
              class={cn(
                'w-full text-left px-3 py-2 rounded-md transition-colors',
                activeTab === 'advanced' ? 'bg-primary text-primary-foreground' : 'hover:bg-accent'
              )}
              on:click={() => changeTab('advanced')}
            >
              Advanced
            </button>
          </div>

          <!-- Content -->
          <div class="flex-1 p-6 overflow-auto custom-scrollbar">
            {#if errorMessage}
              <div
                class="bg-destructive/20 text-destructive p-3 rounded-md mb-4 flex items-center gap-2"
              >
                <AlertCircle size={18} />
                <span>{errorMessage}</span>
              </div>
            {/if}

            {#if successMessage}
              <div class="bg-success/20 text-success p-3 rounded-md mb-4 flex items-center gap-2">
                <Check size={18} />
                <span>{successMessage}</span>
              </div>
            {/if}

            <!-- General Settings Tab -->
            {#if activeTab === 'general'}
              <div class="space-y-6">
                {#if formDefinition}
                  <!-- Dynamic Form Rendering -->
                  <div class="space-y-6">
                    <h3 class="text-lg font-medium">
                      {formDefinition.formTitle || 'Tracker Configuration'}
                    </h3>

                    {#each formDefinition.fields as field}
                      <div class="space-y-3 border-b border-border/50 pb-5 mb-2 last:border-b-0">
                        {#if field.title}
                          <h4 class="text-md font-medium">{field.title}</h4>
                        {/if}

                        <!-- Text Input -->
                        {#if field.type === 'text'}
                          <div class="space-y-2">
                            <label for={field.id} class="block text-sm font-medium text-foreground">
                              {field.label || field.variable_id}
                              {#if field.required}<span class="text-destructive ml-1">*</span>{/if}
                            </label>
                            {#if field.description}
                              <p class="text-xs text-muted-foreground">{field.description}</p>
                            {/if}
                            <input
                              id={field.id}
                              type="text"
                              class="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground"
                              placeholder={field.placeholder || ''}
                              bind:value={configValues[field.variable_id]}
                              required={field.required}
                            />
                          </div>
                        {/if}

                        <!-- Password Input -->
                        {#if field.type === 'password'}
                          <div class="space-y-2">
                            <label for={field.id} class="block text-sm font-medium text-foreground">
                              {field.label || field.variable_id}
                              {#if field.required}<span class="text-destructive ml-1">*</span>{/if}
                            </label>
                            {#if field.description}
                              <p class="text-xs text-muted-foreground">{field.description}</p>
                            {/if}
                            <input
                              id={field.id}
                              type="password"
                              class="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground"
                              placeholder={field.placeholder || ''}
                              bind:value={configValues[field.variable_id]}
                              required={field.required}
                            />
                          </div>
                        {/if}

                        <!-- Number Input -->
                        {#if field.type === 'number'}
                          <div class="space-y-2">
                            <label for={field.id} class="block text-sm font-medium text-foreground">
                              {field.label || field.variable_id}
                              {#if field.required}<span class="text-destructive ml-1">*</span>{/if}
                            </label>
                            {#if field.description}
                              <p class="text-xs text-muted-foreground">{field.description}</p>
                            {/if}
                            <input
                              id={field.id}
                              type="number"
                              class="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground"
                              placeholder={field.placeholder || ''}
                              bind:value={configValues[field.variable_id]}
                              required={field.required}
                              min={field.min !== undefined ? field.min : null}
                              max={field.max !== undefined ? field.max : null}
                            />
                          </div>
                        {/if}

                        <!-- Radio Buttons -->
                        {#if field.type === 'radio' && field.options}
                          <div class="space-y-2">
                            <div class="flex items-center justify-between">
                              <label class="block text-sm font-medium text-foreground">
                                {field.label || field.variable_id}
                                {#if field.required}<span class="text-destructive ml-1">*</span
                                  >{/if}
                              </label>
                            </div>
                            {#if field.description}
                              <p class="text-xs text-muted-foreground">{field.description}</p>
                            {/if}
                            <div class="space-y-2 mt-1">
                              {#each field.options as option}
                                <div class="flex items-center gap-2">
                                  <input
                                    type="radio"
                                    id="{field.id}-{option.value}"
                                    name={field.variable_id}
                                    value={option.value}
                                    bind:group={configValues[field.variable_id]}
                                    required={field.required}
                                    class="h-4 w-4 text-primary border-border"
                                  />
                                  <label for="{field.id}-{option.value}" class="text-sm">
                                    {option.label}
                                  </label>
                                </div>
                              {/each}
                            </div>
                          </div>
                        {/if}

                        <!-- Select Dropdown -->
                        {#if field.type === 'select' && field.options}
                          <div class="space-y-2">
                            <label for={field.id} class="block text-sm font-medium text-foreground">
                              {field.label || field.variable_id}
                              {#if field.required}<span class="text-destructive ml-1">*</span>{/if}
                            </label>
                            {#if field.description}
                              <p class="text-xs text-muted-foreground">{field.description}</p>
                            {/if}
                            <select
                              id={field.id}
                              class="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground"
                              bind:value={configValues[field.variable_id]}
                              required={field.required}
                            >
                              {#each field.options as option}
                                <option value={option.value}>{option.label}</option>
                              {/each}
                            </select>
                          </div>
                        {/if}

                        <!-- Checkbox Group -->
                        {#if field.type === 'checkbox' && field.options}
                          <div class="space-y-2">
                            <label class="block text-sm font-medium text-foreground">
                              {field.label || field.variable_id}
                              {#if field.required}<span class="text-destructive ml-1">*</span>{/if}
                            </label>
                            {#if field.description}
                              <p class="text-xs text-muted-foreground">{field.description}</p>
                            {/if}
                            <div class="space-y-2 mt-1">
                              {#each field.options as option}
                                <div class="flex items-center gap-2">
                                  <input
                                    type="checkbox"
                                    id="{field.id}-{option.value}"
                                    checked={isChecked(field.variable_id, option.value)}
                                    on:change={(e) =>
                                      handleCheckboxChange(field.variable_id, option.value, e)}
                                    class="h-4 w-4 text-primary border-border rounded"
                                  />
                                  <label for="{field.id}-{option.value}" class="text-sm">
                                    {option.label}
                                  </label>
                                </div>
                              {/each}
                            </div>
                          </div>
                        {/if}

                        <!-- File Input -->
                        {#if field.type === 'file'}
                          <div class="space-y-2">
                            <label for={field.id} class="block text-sm font-medium text-foreground">
                              {field.label || field.variable_id}
                              {#if field.required}<span class="text-destructive ml-1">*</span>{/if}
                            </label>
                            {#if field.description}
                              <p class="text-xs text-muted-foreground">{field.description}</p>
                            {/if}

                            <div class="flex flex-col space-y-2">
                              <!-- Custom file input UI -->
                              <div class="flex items-center">
                                <button
                                  type="button"
                                  class="px-4 py-2 bg-accent hover:bg-accent/80 text-accent-foreground rounded-md transition-colors"
                                  on:click={(e) => handleFileUpload(e, field)}
                                >
                                  Browse...
                                </button>
                                <div class="ml-3 text-sm text-muted-foreground truncate max-w-xs">
                                  {configValues[field.variable_id]
                                    ? getFileNameFromPath(configValues[field.variable_id])
                                    : 'No file selected'}
                                </div>
                              </div>

                              {#if configValues[field.variable_id]}
                                <div class="text-xs text-muted-foreground">
                                  <span title={configValues[field.variable_id]}>
                                    Path: {configValues[field.variable_id].length > 50
                                      ? configValues[field.variable_id].substring(0, 20) +
                                        '...' +
                                        configValues[field.variable_id].substring(
                                          configValues[field.variable_id].length - 25
                                        )
                                      : configValues[field.variable_id]}
                                  </span>
                                </div>
                              {/if}
                            </div>
                          </div>
                        {/if}
                      </div>
                    {/each}
                  </div>
                {:else}
                  <!-- No Form Definition Found -->
                  <div class="bg-muted/30 rounded-lg p-6 text-center">
                    <div class="flex justify-center mb-4">
                      <Info size={48} class="text-muted-foreground" />
                    </div>
                    <h4 class="text-lg font-medium mb-2">No Configuration Form Found</h4>
                    <p class="text-sm text-muted-foreground max-w-md mx-auto">
                      This tracker doesn't have a form.json file defining its configuration options.
                      Please check the tracker's documentation for more information.
                    </p>
                  </div>
                {/if}

                <!-- Add action buttons at the end of general tab -->
                <div class="border-t border-border mt-6 pt-4 flex justify-end">
                  <div class="flex gap-3">
                    <button
                      class="px-4 py-2 border border-border rounded-md hover:bg-accent transition-colors min-w-[100px] focus:outline-none focus:ring-2 focus:ring-primary/30"
                      on:click={closeModal}
                      type="button"
                    >
                      Cancel
                    </button>
                    <button
                      class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed min-w-[140px] focus:outline-none focus:ring-2 focus:ring-primary flex items-center justify-center gap-2"
                      on:click={saveTrackerConfig}
                      disabled={isSaving}
                      type="button"
                    >
                      {#if isSaving}
                        <div
                          class="animate-spin h-4 w-4 border-2 border-primary-foreground border-t-transparent rounded-full"
                        ></div>
                        <span>Saving...</span>
                      {:else}
                        <Save size={16} />
                        <span>Save Changes</span>
                      {/if}
                    </button>
                  </div>
                </div>
              </div>
            {/if}

            <!-- Advanced Settings Tab -->
            {#if activeTab === 'advanced'}
              <div class="space-y-6">
                <h3 class="text-lg font-medium">Advanced Settings</h3>

                <div class="space-y-3">
                  <label for="dataPath" class="block text-sm font-medium text-foreground">
                    Data Directory
                  </label>
                  <div class="flex gap-2">
                    <input
                      id="dataPath"
                      type="text"
                      class="flex-1 px-3 py-2 border border-border rounded-md bg-muted text-muted-foreground"
                      value={trackerInfo.dataPath}
                      readonly
                    />
                  </div>
                  <p class="text-xs text-muted-foreground">
                    Data directory location cannot be changed through this interface.
                  </p>
                </div>

                <div class="space-y-3 mt-6">
                  <label class="block text-sm font-medium text-foreground">
                    Tracker Information
                  </label>
                  <div class="bg-muted p-3 rounded-md text-sm">
                    <div><span class="font-medium">ID:</span> {trackerId}</div>
                    <div><span class="font-medium">Name:</span> {trackerInfo.name}</div>
                    <div><span class="font-medium">Version:</span> {trackerInfo.version}</div>
                    <div>
                      <span class="font-medium">Status:</span>
                      {trackerInfo.enabled ? 'Enabled' : 'Disabled'}
                    </div>
                  </div>
                </div>

                <div class="border-t border-border pt-6 mt-8">
                  <h4 class="text-md font-medium text-destructive mb-3">Danger Zone</h4>
                  <div class="space-y-4">
                    <div
                      class="flex items-center justify-between p-4 border border-destructive/30 rounded-md"
                    >
                      <div>
                        <p class="font-medium">Reset Tracker Settings</p>
                        <p class="text-sm text-muted-foreground">
                          Reset all settings to default values. This will not delete any data.
                        </p>
                      </div>
                      <button
                        class="px-3 py-1 bg-destructive/10 hover:bg-destructive/20 text-destructive rounded-md transition-colors"
                      >
                        Reset
                      </button>
                    </div>

                    <div
                      class="flex items-center justify-between p-4 border border-destructive/30 rounded-md"
                    >
                      <div>
                        <p class="font-medium">Uninstall Tracker</p>
                        <p class="text-sm text-muted-foreground">
                          Remove this tracker from your system. This action cannot be undone.
                        </p>
                      </div>
                      <button
                        class="px-3 py-1 bg-destructive hover:bg-destructive/90 text-destructive-foreground rounded-md transition-colors"
                      >
                        Uninstall
                      </button>
                    </div>
                  </div>
                </div>

                <!-- Add action buttons at the end of advanced tab -->
                <div class="border-t border-border mt-6 pt-4 flex justify-end">
                  <div class="flex gap-3">
                    <button
                      class="px-4 py-2 border border-border rounded-md hover:bg-accent transition-colors min-w-[100px] focus:outline-none focus:ring-2 focus:ring-primary/30"
                      on:click={closeModal}
                      type="button"
                    >
                      Cancel
                    </button>
                    <button
                      class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed min-w-[140px] focus:outline-none focus:ring-2 focus:ring-primary flex items-center justify-center gap-2"
                      on:click={saveTrackerConfig}
                      disabled={isSaving}
                      type="button"
                    >
                      {#if isSaving}
                        <div
                          class="animate-spin h-4 w-4 border-2 border-primary-foreground border-t-transparent rounded-full"
                        ></div>
                        <span>Saving...</span>
                      {:else}
                        <Save size={16} />
                        <span>Save Changes</span>
                      {/if}
                    </button>
                  </div>
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
  :global(.scrollbar-hide::-webkit-scrollbar) {
    display: none;
  }
  :global(.scrollbar-hide) {
    -ms-overflow-style: none;
    scrollbar-width: none;
  }

  /* Disable text highlighting on input focus */
  input,
  select,
  textarea {
    user-select: none;
  }

  input:focus,
  select:focus,
  textarea:focus {
    outline: none;
    box-shadow: 0 0 0 2px rgba(var(--primary-rgb), 0.3);
  }
</style>
