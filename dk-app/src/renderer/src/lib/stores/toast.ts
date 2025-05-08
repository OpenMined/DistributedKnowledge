import { writable } from 'svelte/store'
import { v4 as uuidv4 } from 'uuid'
import { onMount, onDestroy } from 'svelte'

export type ToastType = 'default' | 'success' | 'error' | 'warning' | 'info'
export type ToastTemplate = 'simple' | 'action'

export interface Toast {
  id: string
  message: string
  type: ToastType
  template: ToastTemplate
  title?: string
  duration?: number
  action?: {
    label: string
    onClick: () => void
  }
  onDismiss?: () => void
}

export interface ToastStore {
  toasts: Toast[]
  add: (toast: Omit<Toast, 'id'>) => string
  remove: (id: string) => void
  update: (id: string, toast: Partial<Toast>) => void
  clear: () => void
}

const createToastStore = () => {
  const { subscribe, update } = writable<Toast[]>([])

  const store = {
    subscribe,
    add: (toast: Omit<Toast, 'id'>) => {
      const id = uuidv4()
      const fullToast: Toast = {
        id,
        duration: 5000,
        template: 'simple',
        type: 'default',
        ...toast
      }

      update((toasts) => [...toasts, fullToast])

      if (fullToast.duration && fullToast.duration > 0) {
        setTimeout(() => {
          store.remove(id)
          if (fullToast.onDismiss) {
            fullToast.onDismiss()
          }
        }, fullToast.duration)
      }

      return id
    },
    remove: (id: string) => {
      update((toasts) => toasts.filter((t) => t.id !== id))
    },
    update: (id: string, toast: Partial<Toast>) => {
      update((toasts) => toasts.map((t) => (t.id === id ? { ...t, ...toast } : t)))
    },
    clear: () => {
      update(() => [])
    }
  }

  // Set up listener for toasts from the main process
  if (window.electron) {
    window.electron.ipcRenderer.on('toast', (_event, message, options) => {
      store.add({
        message,
        ...options
      })
    })
  }

  return store
}

export const toasts = createToastStore()
