<script lang="ts">
  import { onDestroy } from 'svelte'
  import { toasts } from '../../lib/stores/toast'
  import Toast from './Toast.svelte'

  let toastsList = []

  const unsubscribe = toasts.subscribe((value) => {
    toastsList = value
  })

  onDestroy(() => {
    unsubscribe()
  })

  function handleClose(id: string) {
    toasts.remove(id)
  }
</script>

<div
  class="fixed bottom-4 right-4 z-[9999] flex max-h-screen w-full flex-col-reverse gap-2 sm:max-w-[420px]"
  aria-live="assertive"
>
  {#each toastsList as toast (toast.id)}
    <Toast {toast} onClose={() => handleClose(toast.id)} />
  {/each}
</div>
