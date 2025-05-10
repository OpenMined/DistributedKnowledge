import { ipcRenderer } from 'electron'
import { ConfigAPI } from '../../shared/ipc'
import { ConfigChannels, MCPChannels } from '../../shared/channels'

export const configAPI: ConfigAPI = {
  get: () => ipcRenderer.invoke(ConfigChannels.Get),
  save: (config: Record<string, unknown>) => ipcRenderer.invoke(ConfigChannels.Save, config),
  getMCPConfig: () => ipcRenderer.invoke(MCPChannels.GetConfig),
  saveMCPConfig: (config: Record<string, unknown>) =>
    ipcRenderer.invoke(MCPChannels.SaveConfig, config)
}
