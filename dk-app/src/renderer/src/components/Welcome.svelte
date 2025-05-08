<script lang="ts">
  import { onMount } from 'svelte'
  import { toast } from '../lib/toast'

  let userId = ''

  onMount(async () => {
    try {
      const config = await window.api.config.get()
      userId = config.userID || 'User'

      // Check if we're not coming from onboarding before showing welcome toast
      const { success, status, configExists } = await window.api.onboarding.getStatus()

      // If config doesn't exist, don't show toast (user should be seeing onboarding)
      if (!configExists) {
        console.log('Config does not exist, not showing welcome toast')
        return
      }

      const isShowingOnboarding = success && status && status.isFirstRun && !status.completed

      // Only show welcome toast if not in onboarding
      if (!isShowingOnboarding) {
        setTimeout(() => {
          toast.info(`Welcome back, ${userId}!`, {
            title: 'Hello',
            duration: 5000
          })
        }, 1000)
      }
    } catch (error) {
      console.error('Failed to load user ID:', error)
      userId = 'User'
    }
  })

  function navigateToTrackers() {
    // Dispatch a custom event to navigate to the TrackerSection
    const event = new CustomEvent('navigate', {
      detail: { section: 'apps' }
    })
    window.dispatchEvent(event)
  }
</script>

<div class="flex items-center justify-center h-full w-full">
  <div class="text-center animate-[fadeIn_0.5s_ease-in-out]">
    <img
      src="./dk_logo.png"
      alt="Logo"
      width="150"
      height="150"
      class="mx-auto mb-6 animate-float"
    />
    <h1 class="text-3xl font-bold text-foreground mb-4">Welcome {userId}</h1>
    <p class="text-base font-normal text-muted-foreground">
      Discover AI tools in Trackers to supercharge your workflow!
    </p>
    <button
      class="mt-6 px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors"
      on:click={navigateToTrackers}
    >
      Get Started
    </button>
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

  :global(.animate-float) {
    animation: float 3s ease-in-out infinite;
  }
</style>
