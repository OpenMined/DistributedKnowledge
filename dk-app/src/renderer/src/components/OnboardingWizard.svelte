<script lang="ts">
  import { onMount, createEventDispatcher } from 'svelte'
  import * as LLMTypes from '@shared/llmTypes'
  import { LLMProvider } from '@shared/llmTypes'
  import { toast } from '../lib/toast'
  import { Sun, Moon } from 'lucide-svelte'

  // Window control functions
  let isMaximized = false
  let darkMode = false

  function minimize(): void {
    window.api.window.minimize()
  }

  function maximize(): void {
    window.api.window.maximize()
  }

  function close(): void {
    window.api.window.close()
  }

  async function updateMaximizedState(): Promise<void> {
    isMaximized = await window.api.window.isMaximized()
  }

  function toggleTheme(): void {
    darkMode = !darkMode
    localStorage.setItem('dark-mode', darkMode.toString())
    document.documentElement.classList.toggle('dark', darkMode)
  }

  // Steps state tracking
  const totalSteps = 5
  let currentStep = 1

  // Configuration object to collect throughout the wizard
  let config = {
    serverURL: 'https://distributedknowledge.org', // Default server URL
    userID: '',
    private_key: '',
    public_key: '',
    llm: {
      activeProvider: LLMProvider.OLLAMA,
      providers: {
        [LLMProvider.OLLAMA]: {
          baseUrl: 'http://localhost:11434',
          defaultModel: 'gemma3:4b',
          models: ['gemma3:4b', 'gemma:2b', 'qwen2.5:latest']
        },
        [LLMProvider.ANTHROPIC]: {
          apiKey: '',
          defaultModel: 'claude-3-opus-20240229',
          models: ['claude-3-opus-20240229', 'claude-3-sonnet-20240229', 'claude-3-haiku-20240307']
        },
        [LLMProvider.OPENAI]: {
          apiKey: '',
          defaultModel: 'gpt-4-turbo',
          models: ['gpt-4-turbo', 'gpt-4', 'gpt-3.5-turbo']
        },
        [LLMProvider.GEMINI]: {
          apiKey: '',
          defaultModel: 'gemini-pro',
          models: ['gemini-pro', 'gemini-pro-vision']
        }
      }
    }
  }

  // Keys generation state
  let isGeneratingKeys = false
  let keysGenerated = false

  // Service installation status
  let ollamaInstalled = false
  let syftboxInstalled = false
  let nomicEmbedModelInstalled = false
  let checkingServiceStatus = false
  let pullingNomicModel = false

  // LLM config state
  let selectedLLMProvider = LLMProvider.OLLAMA

  // Loading state
  let isLoading = false
  let saveErrorMessage = ''

  // Tour state
  let showTour = true

  // Step validation
  let serverError = ''
  let userIdError = ''
  let authKeysError = ''
  let llmError = ''

  // Step validation state
  let serverStepValid = false
  let servicesStepValid = false
  let llmStepValid = false

  // Track whether user has attempted to proceed on each step
  let serverStepAttempted = false
  let servicesStepAttempted = false
  let llmStepAttempted = false

  const dispatch = createEventDispatcher<{
    complete: void
  }>()

  // Check if fields are valid
  function validateServerStep(): boolean {
    serverError = ''
    userIdError = ''

    if (!config.serverURL) {
      serverError = 'Server URL is required'
      return false
    }

    // Simple URL validation
    try {
      new URL(config.serverURL)
    } catch (e) {
      serverError = 'Server URL must be a valid URL'
      return false
    }

    // Simple validation for userId - just check if it exists
    const userId = config.userID.trim()
    if (!userId) {
      userIdError = 'User ID is required'
      return false
    }

    return true
  }

  function updateServerStepValidation(): void {
    try {
      serverStepValid = validateServerStep()
    } catch (error) {
      console.error('Error in server step validation:', error)
      serverStepValid = false
    }
  }

  function validateServicesStep(): boolean {
    authKeysError = ''

    // First check authentication keys
    if (!config.private_key || !config.public_key) {
      authKeysError = 'Authentication keys generation failed. Please try again.'
      return false
    }

    // Then check if required services are installed and running
    if (!syftboxInstalled) {
      authKeysError = 'Syftbox must be installed to continue.'
      return false
    }

    if (!ollamaInstalled) {
      authKeysError = 'Ollama must be installed and running to continue.'
      return false
    }

    if (!nomicEmbedModelInstalled) {
      authKeysError = 'The nomic-embed-text model must be installed to continue.'
      return false
    }

    return true
  }

  function updateServicesStepValidation(): void {
    servicesStepValid = validateServicesStep()
  }

  function validateLLMStep(): boolean {
    llmError = ''

    // If using Ollama, ensure it's installed or show warning
    if (selectedLLMProvider === LLMProvider.OLLAMA && !ollamaInstalled) {
      llmError =
        'Warning: Ollama is not installed. You can continue but the application may not function correctly.'
      // We don't block progress, just warn the user
    }
    // If not using Ollama, API key is required
    else if (selectedLLMProvider !== LLMProvider.OLLAMA) {
      const apiKey = config.llm.providers[selectedLLMProvider]?.apiKey
      if (!apiKey) {
        llmError = 'API key is required for this provider'
        return false
      }
    }

    return true
  }

  function updateLLMStepValidation(): void {
    llmStepValid = validateLLMStep()
  }

  // Step navigation
  function nextStep() {
    try {
      // Mark the current step as attempted
      if (currentStep === 2) {
        serverStepAttempted = true
        // Prevent UI freezing by validating the server step without complex animations
        // Skip updateServerStepValidation() and directly validate
        const isValid = validateServerStep()
        serverStepValid = isValid
        if (!isValid) return
      } else if (currentStep === 3) {
        servicesStepAttempted = true
        updateServicesStepValidation()
        if (!servicesStepValid) return
      } else if (currentStep === 4) {
        llmStepAttempted = true
        updateLLMStepValidation()
        if (!llmStepValid) return
      }

      // Update step in local state and backend
      if (currentStep < totalSteps) {
        currentStep++
        window.api.onboarding.setStep(currentStep)
      }
    } catch (error) {
      console.error('Error in nextStep:', error)
    }
  }

  function prevStep() {
    if (currentStep > 1) {
      currentStep--
      window.api.onboarding.setStep(currentStep)
    }
  }

  // Auth keys generation
  async function generateAuthKeys() {
    isGeneratingKeys = true

    try {
      const result = await window.api.onboarding.generateAuthKeys()

      if (result.success && result.keys) {
        // Store the file paths in config
        config.private_key = result.keys.private_key
        config.public_key = result.keys.public_key

        keysGenerated = true
        updateServicesStepValidation()
      } else {
        authKeysError = result.error || 'Failed to generate keys'
        servicesStepValid = false
      }
    } catch (error) {
      authKeysError = 'Error generating authentication keys'
      servicesStepValid = false
      console.error(error)
    } finally {
      isGeneratingKeys = false
    }
  }

  // LLM provider selection
  function selectLLMProvider(provider: LLMProvider) {
    selectedLLMProvider = provider
    config.llm.activeProvider = provider
    updateLLMStepValidation()
  }

  // Complete onboarding and save config
  async function completeOnboarding() {
    isLoading = true
    saveErrorMessage = ''

    try {
      // First save the config
      const saveResult = await window.api.onboarding.saveConfig(config)

      if (!saveResult.success) {
        saveErrorMessage = saveResult.error || 'Failed to save configuration'
        isLoading = false
        return
      }

      // Mark onboarding as complete
      const completeResult = await window.api.onboarding.complete()

      if (!completeResult.success) {
        saveErrorMessage = completeResult.error || 'Failed to complete onboarding'
        isLoading = false
        return
      }

      // Show success toast
      toast.success('Setup completed! The application is ready to use.', {
        title: 'Setup Successful',
        duration: 5000
      })

      // Dispatch complete event to parent
      dispatch('complete')
    } catch (error) {
      saveErrorMessage = 'Error completing onboarding'
      console.error(error)
    } finally {
      isLoading = false
    }
  }

  /**
   * Checks if Ollama and Syftbox services are installed
   */
  async function checkServiceStatus() {
    checkingServiceStatus = true

    try {
      // Call the IPC handler to check external services
      const result = await window.api.onboarding.checkExternalServices()

      if (result.success && result.status) {
        ollamaInstalled = result.status.ollama
        syftboxInstalled = result.status.syftbox
        nomicEmbedModelInstalled = result.status.nomicEmbedModel
      } else {
        console.error('Failed to check services:', result.error)
        // Keep previous values or set to false if undefined
        ollamaInstalled = ollamaInstalled || false
        syftboxInstalled = syftboxInstalled || false
        nomicEmbedModelInstalled = nomicEmbedModelInstalled || false
      }

      // Update validation status after checking services
      updateServicesStepValidation()

      return { ollamaInstalled, syftboxInstalled, nomicEmbedModelInstalled }
    } catch (error) {
      console.error('Failed to check service status:', error)
      return { ollamaInstalled: false, syftboxInstalled: false, nomicEmbedModelInstalled: false }
    } finally {
      checkingServiceStatus = false
    }
  }

  /**
   * Pull the nomic-embed-text model for Ollama with continuous status checking
   */
  async function pullNomicEmbedModel() {
    pullingNomicModel = true
    let downloadCheckInterval = null

    try {
      // Call the IPC handler to pull the model
      const result = await window.api.onboarding.pullNomicEmbedModel()

      if (result.success) {
        // Show success message
        toast.success(result.message || 'Started pulling nomic-embed-text model', {
          title: 'Download Started',
          duration: 5000
        })

        // Start a status check interval that runs every 3 seconds
        downloadCheckInterval = setInterval(async () => {
          // Check if the model is now downloaded
          const status = await checkServiceStatus()

          // If model is now installed, clear the interval and show success message
          if (status.nomicEmbedModelInstalled) {
            clearInterval(downloadCheckInterval)
            downloadCheckInterval = null
            pullingNomicModel = false

            // Show success toast
            toast.success(
              'nomic-embed-text model has been successfully downloaded and installed!',
              {
                title: 'Download Complete',
                duration: 5000
              }
            )

            // Update validation state since the model is now installed
            updateServicesStepValidation()
          }
        }, 3000) // Check every 3 seconds

        // Safety timeout - stop checking after 5 minutes (300000ms) if not completed
        setTimeout(() => {
          if (downloadCheckInterval) {
            clearInterval(downloadCheckInterval)
            downloadCheckInterval = null

            // Only show timeout message if we're still pulling (prevents message if completed normally)
            if (pullingNomicModel) {
              pullingNomicModel = false

              // Check one final time to see if it's actually installed
              checkServiceStatus().then((status) => {
                if (!status.nomicEmbedModelInstalled) {
                  // Show timeout toast
                  toast.error(
                    'Download is taking longer than expected. The model may still be downloading in the background.',
                    {
                      title: 'Download Status Unknown',
                      duration: 8000
                    }
                  )
                }
                // Update validation state regardless of result
                updateServicesStepValidation()
              })
            }
          }
        }, 300000)

        return true
      } else {
        console.error('Failed to pull model:', result.error)

        // Show error toast
        toast.error(result.error || 'Failed to pull nomic-embed-text model', {
          title: 'Download Failed',
          duration: 5000
        })

        pullingNomicModel = false
        return false
      }
    } catch (error) {
      console.error('Error pulling nomic-embed-text model:', error)

      // Show error toast
      toast.error('Error pulling nomic-embed-text model', {
        title: 'Download Failed',
        duration: 5000
      })

      pullingNomicModel = false
      return false
    }
  }

  /**
   * Auto-generates authentication keys on component mount
   */
  async function autoGenerateKeys() {
    isGeneratingKeys = true

    try {
      const result = await window.api.onboarding.generateAuthKeys()

      if (result.success && result.keys) {
        // Store the file paths in config
        config.private_key = result.keys.private_key
        config.public_key = result.keys.public_key

        keysGenerated = true
        updateServicesStepValidation()
      } else {
        authKeysError = result.error || 'Failed to generate keys'
        servicesStepValid = false
      }
    } catch (error) {
      authKeysError = 'Error generating authentication keys'
      servicesStepValid = false
      console.error(error)
    } finally {
      isGeneratingKeys = false
    }
  }

  // Update validation state when relevant data changes
  $: {
    config.serverURL
    updateServerStepValidation()
  }

  $: {
    config.private_key
    config.public_key
    ollamaInstalled
    syftboxInstalled
    nomicEmbedModelInstalled
    updateServicesStepValidation()
  }

  $: {
    selectedLLMProvider
    if (selectedLLMProvider !== LLMProvider.OLLAMA) {
      config.llm.providers[selectedLLMProvider]?.apiKey
    }
    updateLLMStepValidation()
  }

  onMount(async () => {
    try {
      console.log('OnboardingWizard component mounted - getting onboarding status')
      // Get onboarding status
      const { success, status, configExists } = await window.api.onboarding.getStatus()

      console.log('Received onboarding status:', { success, status, configExists })

      if (success && status) {
        console.log('Setting currentStep to:', status.currentStep || 1)
        currentStep = status.currentStep || 1
      }

      // Setup window maximized state tracking
      updateMaximizedState()

      // Update maximized state when window changes
      window.addEventListener('resize', updateMaximizedState)

      // Initialize theme from localStorage or system preference
      const savedTheme = localStorage.getItem('dark-mode')
      if (savedTheme !== null) {
        darkMode = savedTheme === 'true'
      } else {
        // Check if user prefers dark mode
        darkMode = window.matchMedia('(prefers-color-scheme: dark)').matches
      }
      document.documentElement.classList.toggle('dark', darkMode)

      // Auto-generate authentication keys
      await autoGenerateKeys()

      // Check if Ollama and Syftbox are installed
      await checkServiceStatus()

      // Initialize validation states
      updateServerStepValidation()
      updateServicesStepValidation()
      updateLLMStepValidation()

      return () => {
        window.removeEventListener('resize', updateMaximizedState)
      }
    } catch (error) {
      console.error('Failed to get onboarding status:', error)
    }
  })
</script>

<!-- Title bar -->
<header
  class="fixed top-0 left-0 right-0 h-8 bg-background border-b border-border flex items-center p-0 z-[100] select-none"
  style="-webkit-app-region: drag;"
  role="banner"
>
  <div class="w-[138px] flex items-center h-full">
    <h1 class="ml-3 text-xs font-medium text-foreground opacity-70">Setup</h1>
  </div>
  <nav class="flex-1 flex items-center justify-center h-full" aria-label="Search">
    <div style="-webkit-app-region: no-drag;">
      <!-- Showing the current step info in the title bar -->
      <span class="text-xs font-medium text-foreground opacity-70">
        Step {currentStep} of {totalSteps}
      </span>
    </div>
  </nav>
  <section class="flex h-full" style="-webkit-app-region: no-drag;" aria-label="Window controls">
    <button
      class="h-full w-[46px] border-none bg-transparent flex justify-center items-center text-foreground opacity-70 cursor-pointer hover:bg-muted/50 hover:opacity-100"
      on:click={toggleTheme}
      aria-label={darkMode ? 'Switch to light mode' : 'Switch to dark mode'}
      tabindex="0"
    >
      {#if darkMode}
        <Sun size={16} aria-hidden="true" />
      {:else}
        <Moon size={16} aria-hidden="true" />
      {/if}
    </button>
    <button
      class="h-full w-[46px] border-none bg-transparent flex justify-center items-center text-foreground opacity-70 cursor-pointer hover:bg-muted/50 hover:opacity-100"
      on:click={minimize}
      aria-label="Minimize"
      tabindex="0"
    >
      <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true">
        <rect x="2" y="5.5" width="8" height="1" fill="currentColor" />
      </svg>
    </button>
    <button
      class="h-full w-[46px] border-none bg-transparent flex justify-center items-center text-foreground opacity-70 cursor-pointer hover:bg-muted/50 hover:opacity-100"
      on:click={maximize}
      aria-label={isMaximized ? 'Restore' : 'Maximize'}
      tabindex="0"
    >
      {#if isMaximized}
        <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true">
          <path fill="currentColor" d="M3.5,4.5v3h3v-3H3.5z M2,3h6v6H2V3z M4.5,2v1h3v3h1V2H4.5z" />
        </svg>
      {:else}
        <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true">
          <rect x="2.5" y="2.5" width="7" height="7" stroke="currentColor" fill="none" />
        </svg>
      {/if}
    </button>
    <button
      class="h-full w-[46px] border-none bg-transparent flex justify-center items-center text-foreground opacity-70 cursor-pointer hover:bg-destructive hover:text-destructive-foreground hover:opacity-100"
      on:click={close}
      aria-label="Close"
      tabindex="0"
    >
      <svg width="12" height="12" viewBox="0 0 12 12" aria-hidden="true">
        <path
          fill="currentColor"
          d="M6,5.3l2.6-2.6c0.2-0.2,0.5-0.2,0.7,0c0.2,0.2,0.2,0.5,0,0.7L6.7,6l2.6,2.6c0.2,0.2,0.2,0.5,0,0.7c-0.2,0.2-0.5,0.2-0.7,0L6,6.7L3.4,9.3c-0.2,0.2-0.5,0.2-0.7,0c-0.2-0.2-0.2-0.5,0-0.7L5.3,6L2.7,3.4c-0.2-0.2-0.2-0.5,0-0.7c0.2-0.2,0.5-0.2,0.7,0L6,5.3z"
        />
      </svg>
    </button>
  </section>
</header>

<!-- Main onboarding container -->
<div
  class="fixed inset-0 bg-background flex flex-col items-center justify-center p-4 z-50 pt-10 overflow-auto custom-scrollbar"
>
  <!-- Progress indicator -->
  <div class="w-full max-w-2xl mb-8">
    <div class="flex justify-between items-center px-4">
      {#each Array(totalSteps) as _, idx}
        <div class="relative flex flex-col items-center">
          <div
            class={`w-8 h-8 rounded-full flex items-center justify-center transition-colors ${
              idx + 1 <= currentStep
                ? 'bg-primary text-primary-foreground'
                : 'bg-accent text-muted-foreground'
            }`}
          >
            {idx + 1}
          </div>
          {#if idx < totalSteps - 1}
            <div
              class={`absolute top-4 w-[calc(100%-32px)] h-0.5 transition-colors -right-1/2 z-[-1] ${
                idx + 2 <= currentStep ? 'bg-primary' : 'bg-accent'
              }`}
            ></div>
          {/if}
        </div>
      {/each}
    </div>
  </div>

  <!-- Step content -->
  <div class="bg-card rounded-lg shadow-xl border border-border w-full max-w-2xl flex flex-col">
    <!-- Step 1: Welcome -->
    {#if currentStep === 1}
      <div class="p-6 animate-[fadeIn_0.5s_ease-in-out]">
        <img
          src="./dk_logo.png"
          alt="Logo"
          width="100"
          height="100"
          class="mx-auto mb-4 animate-float"
        />
        <h2 class="text-2xl font-bold text-center mb-2">Welcome</h2>
        <p class="text-center text-muted-foreground mb-6">
          Let's set up your DK application in a few simple steps
        </p>
      </div>
    {/if}

    <!-- Step 2: Server Connection -->
    {#if currentStep === 2}
      <div class="p-6 animate-[fadeIn_0.5s_ease-in-out]">
        <h2 class="text-2xl font-bold mb-4">Server Connection</h2>
        <p class="text-muted-foreground mb-6">Configure how your app connects to the server</p>

        <div class="space-y-4">
          <div class="space-y-2">
            <label for="serverURL" class="block text-sm font-medium"> Server URL </label>
            <input
              id="serverURL"
              type="text"
              bind:value={config.serverURL}
              class="w-full px-3 py-2 rounded-md border border-border bg-background transition-colors hover:border-border focus:border-border"
              placeholder="https://example.com"
            />
            {#if serverStepAttempted && serverError}
              <p class="text-destructive text-sm">{serverError}</p>
            {/if}
            <p class="text-xs text-muted-foreground">
              The URL of the server your application will connect to
            </p>
          </div>

          <div class="space-y-2">
            <label for="userID" class="block text-sm font-medium"> User ID </label>
            <input
              id="userID"
              type="text"
              value={config.userID}
              on:input={(e) => (config.userID = e.target.value)}
              class="w-full px-3 py-2 rounded-md border border-border bg-background hover:border-border focus:border-border"
              placeholder="your-unique-username"
            />
            {#if serverStepAttempted && userIdError}
              <p class="text-destructive text-sm">{userIdError}</p>
            {/if}
            <p class="text-xs text-muted-foreground">Your unique identifier on the server</p>
          </div>

          <div class="mt-4 bg-accent/30 p-4 rounded-md">
            <h3 class="text-sm font-medium mb-2">Server Information</h3>
            <p class="text-xs text-muted-foreground">
              The default server URL is set to <code>https://distributedknowledge.org</code>. For
              local development, you can use <code>http://localhost:3000</code> instead.
            </p>
          </div>
        </div>
      </div>
    {/if}

    <!-- Step 3: Services Check -->
    {#if currentStep === 3}
      <div class="p-6 animate-[fadeIn_0.5s_ease-in-out]">
        <h2 class="text-2xl font-bold mb-4">Service Verification</h2>
        <p class="text-muted-foreground mb-6">
          All services below must be installed and running to continue
        </p>

        <div class="space-y-6">
          <!-- Authentication keys info panel (now automatic) -->
          <div class="p-4 bg-card border border-border rounded-md">
            <div class="flex items-center justify-between">
              <div>
                <h3 class="text-base font-semibold">Authentication Keys</h3>
                <p class="text-sm text-muted-foreground">Secure communication</p>
              </div>
              <div class="flex items-center">
                {#if isGeneratingKeys}
                  <span class="inline-block animate-spin mr-2 text-amber-500">⏳</span>
                  <span class="text-amber-500 font-medium">Generating...</span>
                {:else if keysGenerated}
                  <span class="text-success font-medium flex items-center">
                    <svg class="w-5 h-5 mr-1" viewBox="0 0 20 20" fill="currentColor">
                      <path
                        fill-rule="evenodd"
                        d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                        clip-rule="evenodd"
                      />
                    </svg>
                    Generated
                  </span>
                {:else}
                  <span class="text-destructive font-medium flex items-center">
                    <svg class="w-5 h-5 mr-1" viewBox="0 0 20 20" fill="currentColor">
                      <path
                        fill-rule="evenodd"
                        d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                        clip-rule="evenodd"
                      />
                    </svg>
                    Missing
                  </span>
                {/if}
              </div>
            </div>
          </div>

          <!-- Ollama status panel -->
          <div class="p-4 bg-card border border-border rounded-md">
            <div class="flex items-center justify-between">
              <div>
                <h3 class="text-base font-semibold">Ollama</h3>
                <p class="text-sm text-muted-foreground">Local LLM server</p>
              </div>
              <div class="flex items-center">
                {#if checkingServiceStatus}
                  <span class="inline-block animate-spin mr-2 text-amber-500">⏳</span>
                  <span class="text-amber-500 font-medium">Checking...</span>
                {:else if ollamaInstalled}
                  <span class="text-success font-medium flex items-center">
                    <svg class="w-5 h-5 mr-1" viewBox="0 0 20 20" fill="currentColor">
                      <path
                        fill-rule="evenodd"
                        d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                        clip-rule="evenodd"
                      />
                    </svg>
                    Running
                  </span>
                {:else}
                  <span class="text-amber-500 font-medium flex items-center">
                    <svg class="w-5 h-5 mr-1" viewBox="0 0 20 20" fill="currentColor">
                      <path
                        fill-rule="evenodd"
                        d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                        clip-rule="evenodd"
                      />
                    </svg>
                    Not Detected
                  </span>
                {/if}
              </div>
            </div>

            {#if ollamaInstalled}
              <div class="mt-3 border-t border-border pt-3">
                <div class="flex items-center justify-between text-sm">
                  <span class="text-muted-foreground">nomic-embed-text model:</span>
                  {#if checkingServiceStatus}
                    <span class="inline-block animate-spin mr-2 text-amber-500">⏳</span>
                  {:else if nomicEmbedModelInstalled}
                    <span class="text-success flex items-center">
                      <svg class="w-4 h-4 mr-1" viewBox="0 0 20 20" fill="currentColor">
                        <path
                          fill-rule="evenodd"
                          d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                          clip-rule="evenodd"
                        />
                      </svg>
                      Installed
                    </span>
                  {:else}
                    <span class="text-amber-500 flex items-center">
                      <svg class="w-4 h-4 mr-1" viewBox="0 0 20 20" fill="currentColor">
                        <path
                          fill-rule="evenodd"
                          d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                          clip-rule="evenodd"
                        />
                      </svg>
                      Not Installed
                    </span>
                  {/if}
                </div>
              </div>
            {/if}

            {#if !ollamaInstalled}
              <div class="mt-3 border-t border-border pt-3">
                <div class="flex items-start gap-2">
                  <div class="min-w-0 flex-1">
                    <p class="text-xs text-muted-foreground">
                      Ollama not found or not running! Make sure Ollama is installed and running to
                      use local models. You can continue with a cloud provider in the next step if
                      preferred.
                    </p>
                  </div>
                  <a
                    href="https://ollama.com/download"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="inline-flex items-center px-2.5 py-1.5 border border-transparent text-xs font-medium rounded bg-primary text-primary-foreground hover:bg-primary/90 focus:outline-none"
                  >
                    Install
                  </a>
                </div>
              </div>
            {:else if !nomicEmbedModelInstalled}
              <div class="mt-3 border-t border-border pt-3">
                <div class="flex items-start gap-2">
                  <div class="min-w-0 flex-1">
                    <p class="text-xs">
                      {#if pullingNomicModel}
                        <span class="text-amber-600 flex items-center">
                          <svg
                            class="animate-pulse w-3 h-3 mr-1"
                            viewBox="0 0 24 24"
                            fill="currentColor"
                          >
                            <path d="M13 10V3L4 14h7v7l9-11h-7z" />
                          </svg>
                          <span>Downloading model (~250MB)...</span>
                        </span>
                      {:else}
                        Required model for document searching is not installed.
                      {/if}
                    </p>
                  </div>
                  <button
                    class="inline-flex items-center px-2.5 py-1.5 border border-transparent text-xs font-medium rounded bg-primary text-primary-foreground hover:bg-primary/90 focus:outline-none disabled:opacity-75 disabled:cursor-not-allowed"
                    on:click={pullNomicEmbedModel}
                    disabled={pullingNomicModel}
                  >
                    {#if pullingNomicModel}
                      <div class="flex items-center">
                        <svg
                          class="animate-spin -ml-1 mr-2 h-4 w-4 text-white"
                          xmlns="http://www.w3.org/2000/svg"
                          fill="none"
                          viewBox="0 0 24 24"
                        >
                          <circle
                            class="opacity-25"
                            cx="12"
                            cy="12"
                            r="10"
                            stroke="currentColor"
                            stroke-width="4"
                          ></circle>
                          <path
                            class="opacity-75"
                            fill="currentColor"
                            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                          ></path>
                        </svg>
                        <span>Downloading...</span>
                      </div>
                    {:else}
                      Download Model
                    {/if}
                  </button>
                </div>
              </div>
            {/if}
          </div>

          <!-- Syftbox status panel -->
          <div class="p-4 bg-card border border-border rounded-md">
            <div class="flex items-center justify-between">
              <div>
                <h3 class="text-base font-semibold">Syftbox</h3>
                <p class="text-sm text-muted-foreground">Data processing service</p>
              </div>
              <div class="flex items-center">
                {#if checkingServiceStatus}
                  <span class="inline-block animate-spin mr-2 text-amber-500">⏳</span>
                  <span class="text-amber-500 font-medium">Checking...</span>
                {:else if syftboxInstalled}
                  <span class="text-success font-medium flex items-center">
                    <svg class="w-5 h-5 mr-1" viewBox="0 0 20 20" fill="currentColor">
                      <path
                        fill-rule="evenodd"
                        d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                        clip-rule="evenodd"
                      />
                    </svg>
                    Installed
                  </span>
                {:else}
                  <span class="text-amber-500 font-medium flex items-center">
                    <svg class="w-5 h-5 mr-1" viewBox="0 0 20 20" fill="currentColor">
                      <path
                        fill-rule="evenodd"
                        d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                        clip-rule="evenodd"
                      />
                    </svg>
                    Not Detected
                  </span>
                {/if}
              </div>
            </div>

            {#if !syftboxInstalled}
              <div class="mt-3 border-t border-border pt-3">
                <div class="flex items-start gap-2">
                  <div class="min-w-0 flex-1">
                    <p class="text-xs text-muted-foreground">
                      Syftbox configuration not found! You need to install Syftbox and ensure its
                      config file exists at ~/.syftbox/config.json. Some features will be limited
                      without it.
                    </p>
                  </div>
                  <a
                    href="https://syftbox.openmined.org/"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="inline-flex items-center px-2.5 py-1.5 border border-transparent text-xs font-medium rounded bg-primary text-primary-foreground hover:bg-primary/90 focus:outline-none"
                  >
                    Install
                  </a>
                </div>
              </div>
            {/if}
          </div>

          {#if servicesStepAttempted && authKeysError}
            <div class="mt-2 p-3 bg-destructive/20 text-destructive rounded-md text-sm">
              {authKeysError}
            </div>
          {/if}

          <div class="mt-4">
            <button
              class="w-full py-2 bg-accent hover:bg-accent/80 text-accent-foreground font-medium rounded-md transition-colors"
              on:click={checkServiceStatus}
              disabled={checkingServiceStatus}
            >
              {#if checkingServiceStatus}
                <span class="inline-block animate-spin mr-2">⏳</span>
                Checking Services...
              {:else}
                Recheck Services
              {/if}
            </button>
          </div>

          <div class="mt-2 bg-accent/30 p-4 rounded-md">
            <h3 class="text-sm font-medium mb-2">About Required Services</h3>
            <p class="text-xs text-muted-foreground">
              <strong>Ollama</strong> provides local AI models that run on your machine. Recommended
              for privacy.
              <br />
              <strong>Syftbox</strong> is required for tracker applications and document processing.
              <br /><br />
              You can continue without these services, but some features will be limited.
            </p>
          </div>
        </div>
      </div>
    {/if}

    <!-- Step 4: LLM Selection -->
    {#if currentStep === 4}
      <div class="p-6 animate-[fadeIn_0.5s_ease-in-out]">
        <h2 class="text-2xl font-bold mb-4">AI Model Configuration</h2>
        <p class="text-muted-foreground mb-6">
          Select and configure your preferred AI language model provider
        </p>

        <div class="space-y-4">
          <h3 class="text-lg font-medium">Choose Provider</h3>

          <div class="grid grid-cols-2 gap-2">
            {#each Object.values(LLMProvider) as provider}
              <button
                class={`px-4 py-3 rounded-md text-sm font-medium transition-colors ${
                  selectedLLMProvider === provider
                    ? 'bg-primary text-primary-foreground'
                    : 'bg-accent hover:bg-accent/80'
                }`}
                on:click={() => selectLLMProvider(provider)}
              >
                {provider}
              </button>
            {/each}
          </div>

          <!-- Provider-specific settings -->
          <div class="mt-6 border border-border rounded-md p-4">
            <h3 class="text-lg font-medium mb-4 capitalize">{selectedLLMProvider} Settings</h3>

            {#if selectedLLMProvider !== LLMProvider.OLLAMA}
              <div class="space-y-2">
                <label for="apiKey" class="block text-sm font-medium"> API Key </label>
                <input
                  id="apiKey"
                  type="password"
                  bind:value={config.llm.providers[selectedLLMProvider].apiKey}
                  class="w-full px-3 py-2 rounded-md border border-border bg-background transition-colors hover:border-border focus:border-border"
                  placeholder="Enter your API key"
                />
                <p class="text-xs text-muted-foreground">
                  Your API key for authenticating with the {selectedLLMProvider} service
                </p>
              </div>
            {:else}
              <div class="space-y-2">
                <label for="baseUrl" class="block text-sm font-medium"> Base URL </label>
                <input
                  id="baseUrl"
                  type="text"
                  bind:value={config.llm.providers[selectedLLMProvider].baseUrl}
                  class="w-full px-3 py-2 rounded-md border border-border bg-background transition-colors hover:border-border focus:border-border"
                  placeholder="http://localhost:11434"
                />
                <p class="text-xs text-muted-foreground">
                  The URL where your Ollama instance is running
                </p>
              </div>
            {/if}

            <div class="space-y-2 mt-4">
              <label for="defaultModel" class="block text-sm font-medium"> Default Model </label>
              <select
                id="defaultModel"
                bind:value={config.llm.providers[selectedLLMProvider].defaultModel}
                class="w-full px-3 py-2 rounded-md border border-border bg-background transition-colors hover:border-border focus:border-border"
              >
                {#each config.llm.providers[selectedLLMProvider].models as model}
                  <option value={model}>{model}</option>
                {/each}
              </select>
              <p class="text-xs text-muted-foreground">The AI model that will be used by default</p>
            </div>
          </div>

          {#if llmStepAttempted && llmError}
            <p class="text-destructive text-sm">{llmError}</p>
          {/if}

          <div class="mt-4 bg-accent/30 p-4 rounded-md">
            <h3 class="text-sm font-medium mb-2">About AI Providers</h3>
            <p class="text-xs text-muted-foreground">
              <strong>Ollama:</strong> Free, local AI models that run on your machine. Great for
              development and privacy.<br />
              <strong>OpenAI:</strong> Powerful cloud-based models like GPT-4. Requires an API key.<br
              />
              <strong>Anthropic:</strong> Claude models with strong reasoning capabilities. Requires
              an API key.<br />
              <strong>Gemini:</strong> Google's AI models. Requires an API key.
            </p>
          </div>
        </div>
      </div>
    {/if}

    <!-- Step 5: Completion & Tour Offer -->
    {#if currentStep === 5}
      <div class="p-6 animate-[fadeIn_0.5s_ease-in-out]">
        <div class="text-center mb-6">
          <div
            class="w-16 h-16 bg-success/20 text-success rounded-full flex items-center justify-center mx-auto mb-4"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="3"
              stroke-linecap="round"
              stroke-linejoin="round"
              class="w-8 h-8"
            >
              <polyline points="20 6 9 17 4 12"></polyline>
            </svg>
          </div>
          <h2 class="text-2xl font-bold mb-2">Setup Complete!</h2>
          <p class="text-muted-foreground">
            Your Distributed Knowledge application is now configured and ready to use
          </p>
        </div>

        <div class="space-y-4">
          <div class="border border-border rounded-md p-4">
            <h3 class="text-lg font-medium mb-2">Configuration Summary</h3>
            <ul class="space-y-2 text-sm">
              <li><strong>Server:</strong> {config.serverURL}</li>
              <li><strong>User ID:</strong> {config.userID}</li>
              <li>
                <strong>Authentication:</strong>
                {config.private_key ? 'Keys generated successfully' : 'Failed to generate keys'}
              </li>
              <li>
                <strong>Required Services:</strong>
                <span class={ollamaInstalled ? 'text-success' : 'text-amber-500'}>
                  Ollama {ollamaInstalled ? '✓' : '⚠️'}
                </span>,
                <span class={syftboxInstalled ? 'text-success' : 'text-amber-500'}>
                  Syftbox {syftboxInstalled ? '✓' : '⚠️'}
                </span>,
                {#if ollamaInstalled}
                  <span class={nomicEmbedModelInstalled ? 'text-success' : 'text-amber-500'}>
                    nomic-embed-text {nomicEmbedModelInstalled ? '✓' : '⚠️'}
                  </span>
                {/if}
              </li>
              <li><strong>AI Provider:</strong> {config.llm.activeProvider}</li>
            </ul>
          </div>

          <div class="flex items-center">
            <input
              id="showTour"
              type="checkbox"
              bind:checked={showTour}
              class="h-4 w-4 rounded border-border text-primary focus:ring-primary"
            />
            <label for="showTour" class="ml-2 text-sm font-medium"> Start guided tour </label>
          </div>

          {#if saveErrorMessage}
            <div class="p-3 bg-destructive/20 text-destructive rounded-md text-sm">
              {saveErrorMessage}
            </div>
          {/if}
        </div>
      </div>
    {/if}

    <!-- Navigation buttons -->
    <div class="px-6 py-4 bg-card border-t border-border flex justify-between">
      {#if currentStep > 1}
        <button
          class="px-4 py-2 bg-accent text-accent-foreground font-medium rounded-md hover:bg-accent/80 transition-colors"
          on:click={prevStep}
          disabled={isLoading}
        >
          Back
        </button>
      {:else}
        <div></div>
      {/if}

      {#if currentStep < totalSteps}
        <button
          class="px-4 py-2 bg-primary text-primary-foreground font-medium rounded-md hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          on:click={() => {
            // This will trigger the validation checks and error messages
            nextStep()
            // Removed animation for server step to avoid UI freezing
            if ((currentStep === 3 && !servicesStepValid) || (currentStep === 4 && !llmStepValid)) {
              const button = document.activeElement
              if (button) {
                button.classList.add('animate-shake')
                setTimeout(() => {
                  button.classList.remove('animate-shake')
                }, 500)
              }
            }
          }}
          disabled={(currentStep === 2 && !serverStepValid) ||
            (currentStep === 3 && !servicesStepValid) ||
            (currentStep === 4 && !llmStepValid)}
        >
          {currentStep === 1 ? 'Get Started' : 'Continue'}
        </button>
      {:else}
        <button
          class="px-4 py-2 bg-primary text-primary-foreground font-medium rounded-md hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          on:click={completeOnboarding}
          disabled={isLoading}
        >
          {isLoading ? 'Saving...' : 'Get Started'}
        </button>
      {/if}
    </div>
  </div>
</div>

<style>
  @keyframes fadeIn {
    from {
      opacity: 0;
      transform: translateY(10px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  @keyframes float {
    0% {
      transform: translateY(0px);
    }
    50% {
      transform: translateY(-10px);
    }
    100% {
      transform: translateY(0px);
    }
  }

  @keyframes shake {
    0%,
    100% {
      transform: translateX(0);
    }
    20%,
    60% {
      transform: translateX(-5px);
    }
    40%,
    80% {
      transform: translateX(5px);
    }
  }

  :global(.animate-float) {
    animation: float 3s ease-in-out infinite;
  }

  :global(.animate-shake) {
    animation: shake 0.5s ease-in-out;
  }

  /* Custom scrollbar styling */
  :global(.custom-scrollbar::-webkit-scrollbar) {
    width: 8px;
    height: 8px;
  }

  :global(.custom-scrollbar::-webkit-scrollbar-track) {
    background-color: transparent;
  }

  :global(.custom-scrollbar::-webkit-scrollbar-thumb) {
    background-color: var(--border);
    border-radius: 4px;
  }

  :global(.custom-scrollbar::-webkit-scrollbar-thumb:hover) {
    background-color: var(--muted-foreground);
  }

  /* Remove highlight/focus borders for inputs */
  :global(input[type='text']),
  :global(input[type='password']),
  :global(select) {
    outline: none !important;
    box-shadow: none !important;
    -webkit-appearance: none !important;
    appearance: none !important;
  }

  :global(input[type='text']:focus),
  :global(input[type='password']:focus),
  :global(select:focus) {
    outline: none !important;
    box-shadow: none !important;
    border-color: var(--border) !important;
    ring-width: 0 !important;
  }
</style>
