import { ipcMain, IpcMainInvokeEvent } from 'electron'
import { TrackerListResponse, TrackerListItem, TrackerInstallResponse } from '../../shared/types'
import logger from '../../shared/logging'
import { Channels } from '../../shared/constants'
import { trackerService } from '../services/trackerService'
import https from 'https'
import http from 'http'
import fs from 'fs'
import path from 'path'
import url from 'url'
import { appConfig } from '../services/config'

/**
 * Fetch trackers from the server
 */
async function fetchTrackersFromServer(): Promise<Record<string, any>> {
  return new Promise((resolve, reject) => {
    // Parse the server URL from config
    const serverUrl = new url.URL(appConfig.serverURL)
    const isHttps = serverUrl.protocol === 'https:'

    // Extract hostname and port from server URL
    const hostname = serverUrl.hostname
    const port = serverUrl.port ? parseInt(serverUrl.port) : isHttps ? 443 : 80

    // Build the tracker endpoint path
    const trackerEndpoint = '/tracker-apps'

    logger.info(
      `Connecting to tracker endpoint: ${isHttps ? 'https' : 'http'}://${hostname}:${port}${trackerEndpoint}`
    )

    const options = {
      hostname: hostname,
      port: port,
      path: trackerEndpoint,
      method: 'GET',
      // Only disable certificate verification for localhost
      rejectUnauthorized: hostname !== 'localhost'
    }

    // Choose http or https based on the protocol
    const requestModule = isHttps ? https : http
    const req = requestModule.request(options, (res) => {
      let data = ''

      res.on('data', (chunk) => {
        data += chunk
      })

      res.on('end', () => {
        try {
          // Check for successful status code
          if (res.statusCode && res.statusCode >= 200 && res.statusCode < 300) {
            const parsedData = JSON.parse(data)
            resolve(parsedData)
          } else {
            reject(new Error(`Server returned status ${res.statusCode}: ${data}`))
          }
        } catch (error) {
          const errorMessage = error instanceof Error ? error.message : String(error)
          reject(new Error(`Failed to parse JSON: ${errorMessage}`))
        }
      })
    })

    req.on('error', (error) => {
      logger.error(`Error fetching trackers from server ${hostname}:${port}:`, error)
      reject(error)
    })

    // Set a timeout to prevent hanging connections
    req.setTimeout(10000, () => {
      req.destroy()
      reject(new Error('Request timeout after 10 seconds'))
    })

    req.end()
  })
}

/**
 * Transform server data to match the TrackerListItem interface
 */
function transformTrackerData(serverData: Record<string, any>): TrackerListItem[] {
  const result: TrackerListItem[] = []

  // Save the SVG icons to a temp directory for caching
  const iconDir = path.join(process.cwd(), 'resources', 'tracker-icons')

  // Ensure directory exists
  if (!fs.existsSync(iconDir)) {
    try {
      fs.mkdirSync(iconDir, { recursive: true })
    } catch (error) {
      logger.error('Failed to create icon directory:', error)
    }
  }

  for (const [name, data] of Object.entries(serverData)) {
    try {
      // Use the ID from data if available, otherwise fall back to the name
      const id = data.id || name

      // Create a path for the icon
      const iconFileName = `${name.toLowerCase().replace(/\s+/g, '-')}.svg`
      const iconPath = path.join('tracker-icons', iconFileName)
      const fullIconPath = path.join(iconDir, iconFileName)

      // Save the SVG content to a file
      try {
        fs.writeFileSync(fullIconPath, data.icon)
      } catch (error) {
        logger.error(`Failed to save icon for ${name}:`, error)
      }

      // Create the TrackerListItem
      const trackerItem: TrackerListItem = {
        id,
        name,
        version: data.version,
        description: data.description,
        iconPath,
        developer: 'OpenMined', // Default as requested
        verified: true, // Default as requested
        featured: true // Default as requested
      }

      result.push(trackerItem)
    } catch (error) {
      logger.error(`Error transforming tracker data for ${name}:`, error)
    }
  }

  return result
}

/**
 * Get the list of available trackers from the marketplace
 */
export async function getTrackerList(_event: IpcMainInvokeEvent): Promise<TrackerListResponse> {
  try {
    logger.info('Fetching tracker list from local server')

    try {
      // Try to fetch from local server
      const serverData = await fetchTrackersFromServer()
      const trackers = transformTrackerData(serverData)

      logger.info(`Successfully fetched ${trackers.length} trackers from server`)

      return {
        success: true,
        trackers
      }
    } catch (serverError) {
      logger.warn('Failed to fetch trackers from server, using fallback data:', serverError)

      // Fallback to hardcoded data if server request fails
      const trackers = [
        {
          id: 'google-documents-tracker',
          name: 'Google Documents Tracker',
          version: '0.1.0',
          description:
            'A data pipeline for analyzing Google Documents and retrieving their content',
          iconPath: 'gdocuments_tracker/icon.svg',
          developer: 'OpenMined',
          verified: true,
          featured: true
        },
        {
          id: 'google-calendar-tracker',
          name: 'Google Calendar Tracker',
          version: '0.1.0',
          description: 'A data pipeline project for Google Calendar Tracking',
          iconPath: 'calendar_tracker/icon.svg',
          developer: 'OpenMined',
          verified: true,
          featured: true
        },
        {
          id: 'asana-tracker-app',
          name: 'Asana Tracker App',
          version: '1.0.0',
          description:
            'Fetches and processes data from Asana including workspaces, projects, and tasks',
          iconPath: 'asana_tracker/icon.svg',
          developer: 'OpenMined',
          verified: true,
          featured: true
        },
        {
          id: 'google-sheets-tracker',
          name: 'Google Sheets Tracker',
          version: '0.1.0',
          description: 'A data pipeline project for tracking Google Spreadsheets',
          iconPath: 'gsheet_tracker/icon.svg',
          developer: 'OpenMined',
          verified: true,
          featured: true
        },
        {
          id: 'repository-architecture-tracker',
          name: 'Repository Architecture Tracker',
          version: '0.1.0',
          description:
            'A data pipeline for analyzing repository structures and generating architectural insights',
          iconPath: 'repository_tracker/icon.svg',
          developer: 'OpenMined',
          verified: true,
          featured: true
        },
        {
          id: 'mail-tracker',
          name: 'Mail Tracker',
          version: '0.1.0',
          description: 'A data pipeline for analyzing Gmail messages and conversations',
          iconPath: 'gmail_tracker/icon.svg',
          developer: 'OpenMined',
          verified: true,
          featured: true
        }
      ]

      return {
        success: true,
        trackers
      }
    }
  } catch (error) {
    logger.error('Error returning tracker list:', error)
    const errorMessage = error instanceof Error ? error.message : String(error)
    return {
      success: false,
      error: `Failed to get tracker list: ${errorMessage}`
    }
  }
}

/**
 * Download and install a tracker from the marketplace
 *
 * @param event - The IPC event
 * @param trackerId - The ID of the tracker to install
 * @returns Success status, message, and the installed tracker ID
 */
export async function installTracker(
  _event: IpcMainInvokeEvent,
  trackerId: string
): Promise<TrackerInstallResponse> {
  try {
    logger.info(`Installing tracker with ID: ${trackerId}`)

    if (!trackerId) {
      return {
        success: false,
        error: 'No tracker ID provided'
      }
    }

    // Call the tracker service to download and extract the tracker
    const result = await trackerService.downloadAndExtractTracker(trackerId)

    if (result.success) {
      logger.info(`Successfully installed tracker ${trackerId}`)
      return {
        success: true,
        message: result.message,
        trackerId: result.trackerId
      }
    } else {
      logger.error(`Failed to install tracker ${trackerId}: ${result.error}`)
      return {
        success: false,
        error: result.error
      }
    }
  } catch (error) {
    logger.error(`Error installing tracker ${trackerId}:`, error)
    const errorMessage = error instanceof Error ? error.message : String(error)
    return {
      success: false,
      error: `Failed to install tracker: ${errorMessage}`
    }
  }
}

/**
 * Register all tracker marketplace related IPC handlers
 */
export function registerTrackerMarketplaceHandlers(): void {
  logger.info('Registering tracker marketplace IPC handlers')

  // Handle tracker list requests
  ipcMain.handle(Channels.TrackerMarketplaceGetTrackerList, (event) => {
    logger.info('GetTrackerList handler called')
    return getTrackerList(event)
  })

  // Handle tracker installation requests
  ipcMain.handle(Channels.TrackerMarketplaceInstallTracker, (event, trackerId: string) => {
    logger.info('InstallTracker handler called')
    return installTracker(event, trackerId)
  })

  logger.info('Tracker marketplace IPC handlers registered successfully')
}
