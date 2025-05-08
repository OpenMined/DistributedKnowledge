import { ipcRenderer } from 'electron'
import { AppsAPI } from '../../shared/ipc'
import { AppsChannels } from '../../shared/channels'

export const appsAPI: AppsAPI = {
  getAppTrackers: () => ipcRenderer.invoke(AppsChannels.GetAppTrackers),
  toggleAppTracker: (id: number) => ipcRenderer.invoke(AppsChannels.ToggleAppTracker, id),
  getDocumentCount: () => ipcRenderer.invoke(AppsChannels.GetDocumentCount),
  cleanupDocuments: () => ipcRenderer.invoke(AppsChannels.CleanupDocuments),
  installAppTracker: (metadata: Record<string, unknown>) =>
    ipcRenderer.invoke(AppsChannels.InstallAppTracker, metadata),
  updateAppTracker: (id: number) => ipcRenderer.invoke(AppsChannels.UpdateAppTracker, id),
  uninstallAppTracker: (id: number) => ipcRenderer.invoke(AppsChannels.UninstallAppTracker, id),
  getAppIconPath: (appId: string, appPath?: string) =>
    ipcRenderer.invoke(AppsChannels.GetAppIconPath, appId, appPath),
  searchRAGDocuments: (params: { query: string; numResults: number }) =>
    ipcRenderer.invoke(AppsChannels.SearchRAGDocuments, params),
  deleteDocument: (filename: string) => ipcRenderer.invoke(AppsChannels.DeleteDocument, filename)
}
