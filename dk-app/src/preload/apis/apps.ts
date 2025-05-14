import { ipcRenderer } from 'electron'
import { AppsAPI } from '../../shared/ipc'
import { AppsChannels } from '../../shared/channels'

export const appsAPI: AppsAPI = {
  getAppTrackers: () => ipcRenderer.invoke(AppsChannels.GetAppTrackers),
  toggleAppTracker: (id: string) => ipcRenderer.invoke(AppsChannels.ToggleAppTracker, id),
  getDocumentCount: () => ipcRenderer.invoke(AppsChannels.GetDocumentCount),
  getDocuments: () => ipcRenderer.invoke(AppsChannels.GetDocuments),
  cleanupDocuments: () => ipcRenderer.invoke(AppsChannels.CleanupDocuments),
  installAppTracker: (metadata: Record<string, unknown>) =>
    ipcRenderer.invoke(AppsChannels.InstallAppTracker, metadata),
  updateAppTracker: (id: string) => ipcRenderer.invoke(AppsChannels.UpdateAppTracker, id),
  uninstallAppTracker: (id: string) => ipcRenderer.invoke(AppsChannels.UninstallAppTracker, id),
  getAppIconPath: (appId: string, appPath?: string) =>
    ipcRenderer.invoke(AppsChannels.GetAppIconPath, appId, appPath),
  searchRAGDocuments: (params: { query: string; numResults: number }) =>
    ipcRenderer.invoke(AppsChannels.SearchRAGDocuments, params),
  deleteDocument: (filename: string) => ipcRenderer.invoke(AppsChannels.DeleteDocument, filename),
  getApiManagement: () => ipcRenderer.invoke(AppsChannels.GetApiManagement),
  updateApiStatus: (params: { id: string; active: boolean }) =>
    ipcRenderer.invoke(AppsChannels.UpdateApiStatus, params),
  approveApiRequest: (requestId: string) =>
    ipcRenderer.invoke(AppsChannels.ApproveApiRequest, requestId),
  denyApiRequest: (params: { requestId: string; reason?: string }) =>
    ipcRenderer.invoke(AppsChannels.DenyApiRequest, params),
  getPolicies: (params) => ipcRenderer.invoke(AppsChannels.GetPolicies, params),
  getPolicy: (id) => ipcRenderer.invoke(AppsChannels.GetPolicy, id),
  getAPIsByPolicy: (policyId, params) =>
    ipcRenderer.invoke(AppsChannels.GetAPIsByPolicy, policyId, params),
  createPolicy: (policy) => ipcRenderer.invoke(AppsChannels.CreatePolicy, policy),
  updatePolicy: (id, updates) => ipcRenderer.invoke(AppsChannels.UpdatePolicy, id, updates),
  deletePolicy: (id) => ipcRenderer.invoke(AppsChannels.DeletePolicy, id),
  changeAPIPolicy: (apiId, params) =>
    ipcRenderer.invoke(AppsChannels.ChangeAPIPolicy, apiId, params),
  createApi: (apiData) => ipcRenderer.invoke(AppsChannels.CreateApi, apiData),
  deleteApi: (id: string) => ipcRenderer.invoke(AppsChannels.DeleteApi, id)
}
