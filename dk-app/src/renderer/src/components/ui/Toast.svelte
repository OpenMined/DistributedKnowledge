<script lang="ts">
  import { onMount } from 'svelte'
  import { fade, fly } from 'svelte/transition'
  import { cn } from '../../lib/utils'
  import type { Toast as ToastType } from '../../lib/stores/toast'
  import { X, CheckCircle, AlertCircle, Info, AlertTriangle } from 'lucide-svelte'

  export let toast: ToastType
  export let onClose: () => void

  let progressWidth = 100
  let interval: ReturnType<typeof setInterval> | null = null

  const typeClasses = {
    default: 'bg-background border-border text-foreground',
    success: 'bg-success border-success text-success-foreground',
    error: 'bg-destructive border-destructive text-destructive-foreground',
    warning: 'bg-warning border-warning text-warning-foreground',
    info: 'bg-primary border-primary text-primary-foreground'
  }

  const typeIcons = {
    default: null,
    success: CheckCircle,
    error: AlertCircle,
    warning: AlertTriangle,
    info: Info
  }

  onMount(() => {
    if (toast.duration && toast.duration > 0) {
      const updateTime = 10 // ms
      const steps = toast.duration / updateTime
      const decrement = 100 / steps

      interval = setInterval(() => {
        progressWidth -= decrement
        if (progressWidth <= 0) {
          if (interval) clearInterval(interval)
          onClose()
        }
      }, updateTime)
    }

    return () => {
      if (interval) clearInterval(interval)
    }
  })

  function handleAction() {
    if (toast.action?.onClick) {
      toast.action.onClick()
    }
    onClose()
  }

  const Icon = typeIcons[toast.type]
</script>

<div
  class={cn(
    'relative w-full max-w-sm overflow-hidden rounded-lg border p-4 shadow-lg',
    'animate-in slide-in-from-bottom-full duration-300 ease-in-out',
    typeClasses[toast.type]
  )}
  in:fly={{ y: 20, duration: 300 }}
  out:fade={{ duration: 200 }}
>
  <div class="flex items-start gap-3">
    {#if Icon}
      <div class="shrink-0 mt-0.5">
        <svelte:component this={Icon} class="size-5" />
      </div>
    {/if}

    <div class="flex-1 space-y-1">
      {#if toast.title}
        <div class="font-semibold">{toast.title}</div>
      {/if}
      <div class="text-sm">{toast.message}</div>

      {#if toast.template === 'action' && toast.action}
        <button
          class={cn(
            'mt-2 text-sm font-medium underline underline-offset-4',
            'hover:opacity-80 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2'
          )}
          on:click={handleAction}
        >
          {toast.action.label}
        </button>
      {/if}
    </div>

    <button
      class="shrink-0 rounded-md p-1 opacity-70 transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
      on:click={onClose}
      aria-label="Close toast"
    >
      <X class="size-4" />
    </button>
  </div>

  {#if toast.duration && toast.duration > 0}
    <div class="absolute bottom-0 left-0 h-1 bg-foreground/20 w-full">
      <div class="h-full bg-foreground/40" style={`width: ${progressWidth}%`}></div>
    </div>
  {/if}
</div>
