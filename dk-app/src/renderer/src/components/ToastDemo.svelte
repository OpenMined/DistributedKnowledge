<script lang="ts">
  import { toast } from '../lib/toast'

  function showDefaultToast() {
    toast.show('This is a default toast message')
  }

  function showSuccessToast() {
    toast.success('Your message was sent successfully!', {
      title: 'Success'
    })
  }

  function showErrorToast() {
    toast.error('Failed to connect to the server.', {
      title: 'Connection Error',
      duration: 8000
    })
  }

  function showWarningToast() {
    toast.warning('Your session will expire in 5 minutes.', {
      title: 'Session Expiring'
    })
  }

  function showInfoToast() {
    toast.info('A new version is available for download.', {
      title: 'Update Available'
    })
  }

  function showActionToast() {
    toast.action('Do you want to save your changes?', {
      title: 'Unsaved Changes',
      duration: 10000,
      action: {
        label: 'Save Now',
        onClick: () => {
          console.log('Saving changes...')
          toast.success('Changes saved successfully!')
        }
      }
    })
  }
</script>

<div class="p-6 space-y-6">
  <h1 class="text-2xl font-bold">Toast Component Demo</h1>

  <div class="grid grid-cols-2 md:grid-cols-3 gap-4">
    <button
      class="px-4 py-2 bg-background border border-border rounded-md hover:bg-accent/50"
      on:click={showDefaultToast}
    >
      Default Toast
    </button>

    <button
      class="px-4 py-2 bg-success text-success-foreground border border-success rounded-md hover:opacity-90"
      on:click={showSuccessToast}
    >
      Success Toast
    </button>

    <button
      class="px-4 py-2 bg-destructive text-destructive-foreground border border-destructive rounded-md hover:opacity-90"
      on:click={showErrorToast}
    >
      Error Toast
    </button>

    <button
      class="px-4 py-2 bg-warning text-warning-foreground border border-warning rounded-md hover:opacity-90"
      on:click={showWarningToast}
    >
      Warning Toast
    </button>

    <button
      class="px-4 py-2 bg-primary text-primary-foreground border border-primary rounded-md hover:opacity-90"
      on:click={showInfoToast}
    >
      Info Toast
    </button>

    <button
      class="px-4 py-2 bg-background border border-border rounded-md hover:bg-accent/50"
      on:click={showActionToast}
    >
      Action Toast
    </button>
  </div>
</div>
