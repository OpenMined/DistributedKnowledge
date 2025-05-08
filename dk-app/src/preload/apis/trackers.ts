import { ipcRenderer } from 'electron'
import { TrackersAPI } from '../../shared/ipc'
import { TrackerChannels } from '../../shared/channels'

export const trackersAPI: TrackersAPI = {
  getTemplates: (trackerId: string) => ipcRenderer.invoke(TrackerChannels.GetTemplates, trackerId),
  getDatasets: (trackerId: string) => ipcRenderer.invoke(TrackerChannels.GetDatasets, trackerId),
  getAppFolders: () => ipcRenderer.invoke(TrackerChannels.GetAppFolders),
  getTrackerForm: (trackerId: string) =>
    ipcRenderer.invoke(TrackerChannels.GetTrackerForm, trackerId),
  getTrackerConfig: (trackerId: string) =>
    ipcRenderer.invoke(TrackerChannels.GetTrackerConfig, trackerId),
  saveTrackerConfig: (trackerId: string, configData: Record<string, unknown>) =>
    ipcRenderer.invoke(TrackerChannels.SaveTrackerConfig, trackerId, configData),
  getAppSourceFiles: (trackerId: string) =>
    ipcRenderer.invoke(TrackerChannels.GetAppSourceFiles, trackerId),
  getAppFileContent: (trackerId: string, filePath: string) =>
    ipcRenderer.invoke(TrackerChannels.GetAppFileContent, trackerId, filePath),
  getAppConfig: () => ipcRenderer.invoke(TrackerChannels.GetAppConfig),
  updateAppConfig: (formValues: Record<string, unknown>) =>
    ipcRenderer.invoke(TrackerChannels.UpdateAppConfig, formValues),
  uploadTrackerFile: (trackerId: string, filePath: string, variableId: string) =>
    ipcRenderer.invoke(TrackerChannels.UploadTrackerFile, trackerId, filePath, variableId),
  showFileDialog: (trackerId: string, variableId: string, options?: { extensions?: string[] }) =>
    ipcRenderer.invoke(TrackerChannels.ShowFileDialog, trackerId, variableId, options)
}
