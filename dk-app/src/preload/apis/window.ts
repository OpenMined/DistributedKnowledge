import { ipcRenderer } from 'electron'
import { WindowAPI } from '../../shared/ipc'
import { WindowChannels } from '../../shared/channels'

export const windowAPI: WindowAPI = {
  minimize: () => ipcRenderer.send(WindowChannels.Minimize),
  maximize: () => ipcRenderer.send(WindowChannels.Maximize),
  close: () => ipcRenderer.send(WindowChannels.Close),
  isMaximized: () => ipcRenderer.invoke(WindowChannels.IsMaximized)
}
