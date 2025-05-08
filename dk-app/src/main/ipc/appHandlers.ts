import { ipcMain } from 'electron'
import { Channels } from '../../shared/constants'
import { AppsChannels } from '../../shared/channels'
import path from 'path'
import fs from 'fs'
import {
  getAppTrackers,
  toggleAppTracker,
  installAppTracker,
  updateAppTracker,
  uninstallAppTracker
} from '../services/appService'
import { documentService } from '../services/documentService'
import logger from '../../shared/logging'

/**
 * Register IPC handlers for app tracker related functionality
 */
export function registerAppHandlers(): void {
  // Log handler initialization
  logger.info('Initializing app tracker handlers')

  // Get all app trackers
  ipcMain.handle(Channels.GetAppTrackers, () => {
    try {
      return {
        success: true,
        appTrackers: getAppTrackers()
      }
    } catch (error) {
      logger.error('Failed to get app trackers:', error)
      return {
        success: false,
        error: 'Failed to get app trackers'
      }
    }
  })

  // Toggle app tracker enabled state
  ipcMain.handle(Channels.ToggleAppTracker, (_, id) => {
    if (!id) return { success: false, error: 'Invalid app tracker ID' }

    try {
      const updatedTracker = toggleAppTracker(id)
      if (!updatedTracker) {
        return { success: false, error: 'App tracker not found' }
      }

      return {
        success: true,
        appTracker: updatedTracker
      }
    } catch (error) {
      logger.error('Failed to toggle app tracker:', error)
      return {
        success: false,
        error: 'Failed to toggle app tracker'
      }
    }
  })

  // Get document count - now using documentService
  ipcMain.handle(Channels.GetDocumentCount, () => {
    try {
      const stats = documentService.getDocumentCount()
      return {
        success: true,
        stats: {
          count: stats.count,
          error: stats.error
        }
      }
    } catch (error) {
      logger.error('Failed to get document count:', error)
      return {
        success: false,
        error: 'Failed to get document count'
      }
    }
  })

  // Cleanup documents (delete all documents) - now using documentService
  ipcMain.handle(Channels.CleanupDocuments, async () => {
    try {
      const result = await documentService.cleanupDocuments()
      return result
    } catch (error) {
      logger.error('Failed to cleanup documents:', error)
      return {
        success: false,
        error: 'Failed to cleanup documents'
      }
    }
  })

  // Install app tracker
  ipcMain.handle(Channels.InstallAppTracker, (_, metadata) => {
    try {
      // Check if proper metadata is provided
      if (!metadata || !metadata.name) {
        return {
          success: false,
          error: 'Invalid app metadata. Name is required.'
        }
      }

      const result = installAppTracker(metadata)
      return {
        success: result.success,
        message: result.message,
        appTracker: result.appTracker
      }
    } catch (error) {
      logger.error('Failed to install app tracker:', error)
      return {
        success: false,
        error: 'Failed to install app tracker'
      }
    }
  })

  // Update app tracker
  ipcMain.handle(Channels.UpdateAppTracker, (_, id) => {
    if (!id) return { success: false, error: 'Invalid app tracker ID' }

    try {
      const result = updateAppTracker(id)
      return result
    } catch (error) {
      logger.error('Failed to update app tracker:', error)
      return {
        success: false,
        error: 'Failed to update app tracker'
      }
    }
  })

  // Uninstall app tracker
  ipcMain.handle(Channels.UninstallAppTracker, (_, id) => {
    if (!id) return { success: false, error: 'Invalid app tracker ID' }

    try {
      const result = uninstallAppTracker(id)
      return result
    } catch (error) {
      logger.error('Failed to uninstall app tracker:', error)
      return {
        success: false,
        error: 'Failed to uninstall app tracker'
      }
    }
  })

  // Get app icon path
  ipcMain.handle(Channels.GetAppIconPath, (_, appId, appPath) => {
    if (!appId) return null

    try {
      // Use the provided appPath if available, otherwise look up the app
      let appIconPath = appPath

      // Find the app by ID only if we don't have the path
      if (!appIconPath) {
        const apps = getAppTrackers()
        const app = apps.find((app) => app.id === appId)

        // If app not found or has no path, return null
        if (!app || !app.path) return null

        appIconPath = app.path
      }

      // Check if icon.svg exists in the app directory
      const iconPath = path.join(appIconPath, 'icon.svg')
      if (fs.existsSync(iconPath)) {
        logger.debug(`Found icon for app ${appId} at ${iconPath}`)

        try {
          // Read the SVG file content
          const svgContent = fs.readFileSync(iconPath, 'utf8')

          // Ensure the SVG has viewBox attribute if not present
          if (
            !svgContent.includes('viewBox') &&
            !svgContent.includes('width') &&
            !svgContent.includes('height')
          ) {
            // Add a default viewBox attribute to make it scale properly
            const modifiedSvg = svgContent.replace(/<svg/, '<svg viewBox="0 0 24 24"')
            // Return data URL for direct rendering
            return `data:image/svg+xml;charset=utf8,${encodeURIComponent(modifiedSvg)}`
          }

          // Return data URL for direct rendering
          return `data:image/svg+xml;charset=utf8,${encodeURIComponent(svgContent)}`
        } catch (readError) {
          logger.error(`Failed to read SVG content for app ${appId}:`, readError)
          // Fallback to file URL if reading fails
          return `file://${iconPath}`
        }
      }

      logger.debug(`No icon found for app ${appId} at ${iconPath}`)
      return null
    } catch (error) {
      logger.error(`Failed to get app icon path for app ${appId}:`, error)
      return null
    }
  })

  // Search RAG Documents - now using documentService
  ipcMain.handle(Channels.SearchRAGDocuments, async (_, { query, numResults }) => {
    try {
      logger.info(
        `Received request to search RAG documents with query "${query}" and limit ${numResults}`
      )

      const results = await documentService.searchDocuments(query, numResults)

      return {
        success: true,
        results: results
      }
    } catch (error) {
      logger.error('Failed to search RAG documents:', error)
      return {
        success: false,
        error: 'Failed to search RAG documents',
        results: { documents: [] }
      }
    }
  })

  // Delete Document - now using documentService
  ipcMain.handle(AppsChannels.DeleteDocument, async (_, filename) => {
    try {
      if (!filename) {
        return {
          success: false,
          message: 'Filename is required'
        }
      }

      logger.info(`Received request to delete document with filename "${filename}"`)

      const result = await documentService.deleteDocument(filename)
      return result
    } catch (error) {
      logger.error('Failed to delete document:', error)
      const errorMessage = error instanceof Error ? error.message : String(error)
      return {
        success: false,
        message: `Failed to delete document: ${errorMessage}`
      }
    }
  })
}
