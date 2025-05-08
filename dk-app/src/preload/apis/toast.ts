import { ipcRenderer } from 'electron'
import { ToastAPI } from '../../shared/ipc'
import { ToastChannels } from '../../shared/channels'

export const toastAPI: ToastAPI = {
  show: (message: string, options?: { type?: string; duration?: number }) =>
    ipcRenderer.send(ToastChannels.Show, message, options)
}
