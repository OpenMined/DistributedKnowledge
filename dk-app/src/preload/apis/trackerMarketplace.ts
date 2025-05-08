import { ipcRenderer } from 'electron'
import { TrackerMarketplaceAPI } from '../../shared/ipc'
import { TrackerMarketplaceChannels } from '../../shared/channels'
import logger from '../../shared/logging'

export const trackerMarketplaceAPI: TrackerMarketplaceAPI = {
  getTrackerList: async () => {
    logger.debug('Preload: Calling getTrackerList IPC...')
    try {
      const result = await ipcRenderer.invoke(TrackerMarketplaceChannels.GetTrackerList)
      logger.debug('Preload: getTrackerList result received')
      return result
    } catch (error) {
      logger.error('Preload: Error invoking getTrackerList:', error)
      // Return a standardized error response rather than throwing
      return {
        success: false,
        error: `Failed to get tracker list: ${error instanceof Error ? error.message : String(error)}`
      }
    }
  },
  installTracker: async (trackerId: string) => {
    logger.debug(`Preload: Calling installTracker IPC for tracker ${trackerId}...`)
    try {
      const result = await ipcRenderer.invoke(TrackerMarketplaceChannels.InstallTracker, trackerId)
      logger.debug('Preload: installTracker result received:', result)
      return result
    } catch (error) {
      logger.error('Preload: Error invoking installTracker:', error)
      // Return a standardized error response rather than throwing
      return {
        success: false,
        error: `Failed to install tracker: ${error instanceof Error ? error.message : String(error)}`
      }
    }
  }
}
