import { toasts, type Toast } from './stores/toast'

type ToastOptions = Omit<Toast, 'id' | 'type' | 'message'>

export const toast = {
  /**
   * Show a default toast
   */
  show(message: string, options?: ToastOptions) {
    return toasts.add({ message, type: 'default', ...options })
  },

  /**
   * Show a success toast
   */
  success(message: string, options?: ToastOptions) {
    return toasts.add({ message, type: 'success', ...options })
  },

  /**
   * Show an error toast
   */
  error(message: string, options?: ToastOptions) {
    return toasts.add({ message, type: 'error', ...options })
  },

  /**
   * Show a warning toast
   */
  warning(message: string, options?: ToastOptions) {
    return toasts.add({ message, type: 'warning', ...options })
  },

  /**
   * Show an info toast
   */
  info(message: string, options?: ToastOptions) {
    return toasts.add({ message, type: 'info', ...options })
  },

  /**
   * Create a toast with a call to action
   */
  action(
    message: string,
    {
      action,
      ...options
    }: ToastOptions & {
      action: {
        label: string
        onClick: () => void
      }
    }
  ) {
    return toasts.add({
      message,
      type: 'default',
      template: 'action',
      action,
      ...options
    })
  },

  /**
   * Dismiss a toast by ID
   */
  dismiss(id: string) {
    toasts.remove(id)
  },

  /**
   * Clear all toasts
   */
  clear() {
    toasts.clear()
  }
}
