import { ipcRenderer } from 'electron'
import { SidebarAPI } from '../../shared/ipc'
import { SidebarChannels } from '../../shared/channels'

export const sidebarAPI: SidebarAPI = {
  getUsers: () => ipcRenderer.invoke(SidebarChannels.GetUsers),
  getChannels: () => ipcRenderer.invoke(SidebarChannels.GetChannels)
}
