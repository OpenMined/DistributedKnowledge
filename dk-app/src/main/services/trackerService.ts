import { readFile, access, readdir, mkdir, writeFile, unlink } from 'fs/promises'
import { join, resolve, dirname } from 'path'
import { constants, createWriteStream, existsSync, createReadStream } from 'fs'
import { app } from 'electron'
import axios from 'axios'
import https from 'https'
import http from 'http'
import url from 'url'
import logger, { createServiceLogger } from '../../shared/logging'
import { syftboxConfig } from './appService'
import { pipeline } from 'stream/promises'
import * as unzipper from 'unzipper'
// Define a module declaration file in the project instead of directly in this file
// Create a custom type definition for unzipper
type UnzipperDirectory = {
  files: Array<{ path: string; type: string; buffer: () => Promise<Buffer> }>
  extract: (options: { path: string }) => Promise<void>
}

// Use the custom types directly instead of module augmentation
const Extract = unzipper.Extract
const Open = {
  file: (path: string): Promise<UnzipperDirectory> =>
    unzipper.Open.file(path) as Promise<UnzipperDirectory>
}

/* Removing module declaration as it causes errors - will use direct casting instead
declare module 'unzipper' {
  export function Extract(options: { path: string }): NodeJS.WritableStream;
  export namespace Open {
    export function file(path: string): Promise<unzipper.Directory>;
  }
*/
import * as fs from 'fs'
import { appConfig } from './config'

// Create service-specific logger
const serviceLogger = createServiceLogger('trackerService')

interface DatasetSchema {
  datasets: Record<string, string>
  templates: Record<string, string>
}

interface AppFileTreeNode {
  name: string
  path: string
  type: 'file' | 'directory'
  children?: AppFileTreeNode[]
}

interface TrackerMetadata {
  name: string
  description: string
  version: string
  icon: string
  hasUpdate?: boolean
  tag?: string
  active?: boolean
}

interface TrackerPayload {
  trackers: {
    [key: string]: {
      tracker_description: string
      tracker_version: string
      tracker_documents: {
        datasets: Record<string, string>
        templates: Record<string, string>
      }
    }
  }
}

export class TrackerService {
  private appsBaseDir: string
  private apiEndpoint = 'http://localhost:4232/user/trackers'
  private scanIntervalId: NodeJS.Timeout | null = null
  private scanIntervalMs = 10000 // 10 seconds
  private trackerIdToFolderMap: Map<string, string> = new Map() // Maps numeric IDs to folder names
  private isTrackerServerAvailable = false // Flag to track availability of the tracker server

  constructor() {
    // We'll initialize this properly in startTrackerScan
    this.appsBaseDir = ''

    // Set the apps base directory from syftbox config if available
    if (syftboxConfig && syftboxConfig.data_dir) {
      this.appsBaseDir = join(syftboxConfig.data_dir, 'apps')
    }
  }

  /**
   * Check if the tracker server is available
   * @returns Promise<boolean> indicating if the server is available
   */
  public async checkTrackerServerAvailability(): Promise<boolean> {
    return new Promise((resolve) => {
      // Parse the server URL from config
      const serverUrl = new url.URL(appConfig.serverURL)
      const isHttps = serverUrl.protocol === 'https:'

      // Extract hostname and port from server URL
      const hostname = serverUrl.hostname
      const port = serverUrl.port ? parseInt(serverUrl.port) : isHttps ? 443 : 80

      // Build tracker endpoint path
      const trackerEndpoint = '/tracker-apps'

      serviceLogger.info(
        `Checking tracker server at: ${isHttps ? 'https' : 'http'}://${hostname}:${port}${trackerEndpoint}`
      )

      const options = {
        hostname: hostname,
        port: port,
        path: trackerEndpoint,
        method: 'GET',
        timeout: 3000, // 3 seconds timeout
        // Only disable certificate verification for localhost
        rejectUnauthorized: hostname !== 'localhost'
      }

      // Choose http or https based on the protocol
      const requestModule = isHttps ? https : http
      const req = requestModule.request(options, (res) => {
        if (res.statusCode === 200) {
          this.isTrackerServerAvailable = true
          serviceLogger.info('Tracker server is available')
          resolve(true)
        } else {
          this.isTrackerServerAvailable = false
          serviceLogger.warn(`Tracker server returned status: ${res.statusCode}`)
          resolve(false)
        }
      })

      req.on('error', (error) => {
        this.isTrackerServerAvailable = false
        serviceLogger.warn(`Tracker server is not available at ${hostname}:${port}:`, error.message)
        resolve(false)
      })

      req.on('timeout', () => {
        req.destroy()
        this.isTrackerServerAvailable = false
        serviceLogger.warn(`Tracker server connection at ${hostname}:${port} timed out`)
        resolve(false)
      })

      req.end()
    })
  }

  /**
   * Get tracker server availability status
   * @returns boolean indicating if the server is available
   */
  public isTrackerServerReady(): boolean {
    return this.isTrackerServerAvailable
  }

  public async startTrackerScan(): Promise<void> {
    if (this.scanIntervalId) {
      clearInterval(this.scanIntervalId)
    }

    // Check tracker server availability
    await this.checkTrackerServerAvailability()
    serviceLogger.info(
      `Tracker server is ${this.isTrackerServerAvailable ? 'available' : 'not available'}`
    )

    // Set the apps base directory from syftbox config
    if (syftboxConfig && syftboxConfig.data_dir) {
      this.appsBaseDir = join(syftboxConfig.data_dir, 'apps')
      serviceLogger.info(`Tracker scanning service configured to scan: ${this.appsBaseDir}`)
    } else {
      serviceLogger.error('SyftBox configuration not available. Cannot start tracker scanner.')
      return
    }

    // Immediately run first scan
    this.scanTrackers().catch((err) => {
      serviceLogger.error('Initial tracker scan failed:', err)
    })

    // Set up interval for future scans
    this.scanIntervalId = setInterval(() => {
      this.scanTrackers().catch((err) => {
        serviceLogger.error('Periodic tracker scan failed:', err)
      })
    }, this.scanIntervalMs)

    serviceLogger.info(`Tracker scanning service started with ${this.scanIntervalMs}ms interval`)
  }

  public async scanTrackers(): Promise<void> {
    // Update our tracker ID to folder mapping
    await this.updateTrackerIdMapping()
    return this.scanTrackersInternal()
  }

  public stopTrackerScan(): void {
    if (this.scanIntervalId) {
      clearInterval(this.scanIntervalId)
      this.scanIntervalId = null
      serviceLogger.info('Tracker scanning service stopped')
    }
  }

  private async scanTrackersInternal(): Promise<void> {
    try {
      // Make sure base directory exists before scanning
      try {
        await access(this.appsBaseDir, constants.R_OK)
      } catch (error) {
        serviceLogger.info(
          `Apps base directory ${this.appsBaseDir} does not exist or is not readable`
        )
        return
      }

      // Find all directories in the apps folder - each one is potentially a tracker
      const dirEntries = await readdir(this.appsBaseDir, { withFileTypes: true })

      // Filter to only include directories
      const appDirNames = dirEntries
        .filter((dirent) => dirent.isDirectory())
        .map((dirent) => dirent.name)

      if (appDirNames.length === 0) {
        serviceLogger.debug('No app directories found')
        return
      }

      const payload: TrackerPayload = { trackers: {} }

      // Process each app directory as a potential tracker
      for (const appName of appDirNames) {
        const appDir = join(this.appsBaseDir, appName)

        try {
          // Check for required files
          const metadataPath = join(appDir, 'metadata.json')
          const schemaPath = join(appDir, 'dataset_schema.json')

          try {
            await access(metadataPath, constants.R_OK)
            await access(schemaPath, constants.R_OK)
          } catch (error) {
            serviceLogger.debug(`App ${appName} missing required tracker files, skipping`)
            continue
          }

          // Read metadata and schema
          const metadata = JSON.parse(await readFile(metadataPath, 'utf-8')) as TrackerMetadata

          // Skip apps that are not active
          if (metadata.active !== true) {
            serviceLogger.debug(`App ${appName} is not active, skipping`)
            continue
          }

          const schema = JSON.parse(await readFile(schemaPath, 'utf-8')) as DatasetSchema

          // Add to payload
          payload.trackers[appName] = {
            tracker_description: metadata.description || `${appName} tracker`,
            tracker_version: metadata.version || '1.0.0',
            tracker_documents: {
              datasets: schema.datasets || {},
              templates: schema.templates || {}
            }
          }

          serviceLogger.debug(`Found valid active tracker in app: ${appName}`)
        } catch (error) {
          serviceLogger.error(`Error processing tracker ${appName}:`, error)
        }
      }

      // Only make request if we found valid trackers
      if (Object.keys(payload.trackers).length > 0) {
        await this.sendTrackerData(payload)
      } else {
        serviceLogger.debug('No valid trackers found, skipping API call')
      }
    } catch (error) {
      serviceLogger.error('Failed to scan trackers:', error)
      throw error // Re-throw for higher-level error handling
    }
  }

  private async sendTrackerData(payload: TrackerPayload): Promise<void> {
    try {
      serviceLogger.info(
        `Sending tracker data for ${Object.keys(payload.trackers).length} trackers`
      )

      const response = await axios.post(this.apiEndpoint, payload, {
        headers: {
          'Content-Type': 'application/json'
        }
      })

      serviceLogger.info(`Tracker data sent successfully. Status: ${response.status}`)
    } catch (error) {
      serviceLogger.error('Failed to send tracker data:', error)
      throw error
    }
  }

  /**
   * Get all app folder names
   * @returns A list of all app folder names in the apps directory
   */
  public async getAppFolderNames(): Promise<string[]> {
    try {
      if (!this.appsBaseDir) {
        if (syftboxConfig && syftboxConfig.data_dir) {
          this.appsBaseDir = join(syftboxConfig.data_dir, 'apps')
        } else {
          return []
        }
      }

      // Check if directory exists
      try {
        await access(this.appsBaseDir, constants.R_OK)
      } catch (error) {
        serviceLogger.error(`Apps directory not accessible: ${this.appsBaseDir}`, error)
        return []
      }

      // Read all directories
      const dirEntries = await readdir(this.appsBaseDir, { withFileTypes: true })

      // Filter to only include directories
      return dirEntries.filter((dirent) => dirent.isDirectory()).map((dirent) => dirent.name)
    } catch (error) {
      serviceLogger.error('Failed to get app folder names:', error)
      return []
    }
  }

  /**
   * Updates the ID to folder name mapping from the folder metadata
   */
  public async updateTrackerIdMapping(): Promise<void> {
    try {
      const folderNames = await this.getAppFolderNames()
      this.trackerIdToFolderMap.clear()

      for (const folderName of folderNames) {
        try {
          const metadataPath = join(this.appsBaseDir, folderName, 'metadata.json')

          // Skip if metadata doesn't exist
          try {
            await access(metadataPath, constants.R_OK)
          } catch {
            continue
          }

          // Read metadata
          const metadata = JSON.parse(await readFile(metadataPath, 'utf-8')) as TrackerMetadata

          // The database ID from the AppsSection is just the folder name
          // Map the folder name to itself (seems redundant but needed for consistency)
          this.trackerIdToFolderMap.set(folderName, folderName)
        } catch (error) {
          serviceLogger.debug(`Error processing folder ${folderName} for ID mapping:`, error)
        }
      }

      serviceLogger.info(
        `Updated tracker ID mapping with ${this.trackerIdToFolderMap.size} entries`
      )
    } catch (error) {
      serviceLogger.error('Failed to update tracker ID mapping:', error)
    }
  }

  /**
   * Get templates for a specific tracker from its dataset_schema.json file
   *
   * @param trackerId - The ID of the tracker from the UI
   * @returns Object containing template data with name and content
   */
  public async getTrackerTemplates(trackerId: string): Promise<{
    success: boolean
    templates?: Record<string, { name: string; content: string; filename: string }>
    error?: string
  }> {
    try {
      if (!this.appsBaseDir) {
        if (syftboxConfig && syftboxConfig.data_dir) {
          this.appsBaseDir = join(syftboxConfig.data_dir, 'apps')
        } else {
          return {
            success: false,
            error: 'Apps directory not configured'
          }
        }
      }

      // Make sure our ID to folder mapping is up to date
      if (this.trackerIdToFolderMap.size === 0) {
        await this.updateTrackerIdMapping()
      }

      // First try to use the ID directly as the folder name
      let folderName = trackerId

      // If that folder doesn't exist, scan all folders and read metadata to find a match
      try {
        await access(join(this.appsBaseDir, folderName), constants.R_OK)
      } catch {
        // Folder doesn't exist with the ID as name, so we need to scan
        const folders = await this.getAppFolderNames()

        // Just use the first folder we find as a fallback
        if (folders.length > 0) {
          folderName = folders[0]
          serviceLogger.warn(
            `Couldn't find folder for tracker ID ${trackerId}, using first available folder: ${folderName}`
          )
        } else {
          return {
            success: false,
            error: `No tracker folders found in ${this.appsBaseDir}`
          }
        }
      }

      // Build the path to the dataset_schema.json file
      const schemaPath = join(this.appsBaseDir, folderName, 'dataset_schema.json')

      // Check if file exists
      try {
        await access(schemaPath, constants.R_OK)
      } catch (error) {
        serviceLogger.error(
          `Schema file not accessible for tracker ${trackerId} (folder: ${folderName}):`,
          error
        )
        return {
          success: false,
          error: `Schema file not found for tracker in folder: ${folderName}`
        }
      }

      // Read and parse the schema file
      const schemaContent = await readFile(schemaPath, 'utf-8')
      const schema = JSON.parse(schemaContent) as DatasetSchema

      // Extract templates
      const templatesData: Record<string, { name: string; content: string; filename: string }> = {}

      // Check if templates exist in schema
      if (schema.templates) {
        // Process each template
        for (const [templateId, templateContent] of Object.entries(schema.templates)) {
          // Create a normalized filename for display, ensuring we don't duplicate .md extension
          const filename = templateId.toLowerCase().endsWith('.md')
            ? templateId.replace(/[^a-zA-Z0-9\.]/g, '_')
            : `${templateId.replace(/[^a-zA-Z0-9]/g, '_')}.md`

          templatesData[templateId] = {
            name: templateId,
            content: templateContent,
            filename
          }
        }

        return {
          success: true,
          templates: templatesData
        }
      } else {
        return {
          success: true,
          templates: {} // Return empty object if no templates found
        }
      }
    } catch (error) {
      serviceLogger.error(`Error getting templates for tracker ${trackerId}:`, error)
      const errorMessage = error instanceof Error ? error.message : String(error)
      return {
        success: false,
        error: `Failed to get templates: ${errorMessage}`
      }
    }
  }

  /**
   * Get datasets for a specific tracker from its dataset_schema.json file
   *
   * @param trackerId - The ID of the tracker from the UI
   * @returns Object containing dataset information with filename and templateId
   */
  public async getTrackerDatasets(trackerId: string): Promise<{
    success: boolean
    datasets?: Record<string, string>
    error?: string
  }> {
    try {
      if (!this.appsBaseDir) {
        if (syftboxConfig && syftboxConfig.data_dir) {
          this.appsBaseDir = join(syftboxConfig.data_dir, 'apps')
        } else {
          return {
            success: false,
            error: 'Apps directory not configured'
          }
        }
      }

      // Make sure our ID to folder mapping is up to date
      if (this.trackerIdToFolderMap.size === 0) {
        await this.updateTrackerIdMapping()
      }

      // First try to use the ID directly as the folder name
      let folderName = trackerId

      // If that folder doesn't exist, scan all folders and read metadata to find a match
      try {
        await access(join(this.appsBaseDir, folderName), constants.R_OK)
      } catch {
        // Folder doesn't exist with the ID as name, so we need to scan
        const folders = await this.getAppFolderNames()

        // Just use the first folder we find as a fallback
        if (folders.length > 0) {
          folderName = folders[0]
          serviceLogger.warn(
            `Couldn't find folder for tracker ID ${trackerId}, using first available folder: ${folderName}`
          )
        } else {
          return {
            success: false,
            error: `No tracker folders found in ${this.appsBaseDir}`
          }
        }
      }

      // Build the path to the dataset_schema.json file
      const schemaPath = join(this.appsBaseDir, folderName, 'dataset_schema.json')

      // Check if file exists
      try {
        await access(schemaPath, constants.R_OK)
      } catch (error) {
        serviceLogger.error(
          `Schema file not accessible for tracker ${trackerId} (folder: ${folderName}):`,
          error
        )
        return {
          success: false,
          error: `Schema file not found for tracker in folder: ${folderName}`
        }
      }

      // Read and parse the schema file
      const schemaContent = await readFile(schemaPath, 'utf-8')
      const schema = JSON.parse(schemaContent) as DatasetSchema

      // Check if datasets exist in schema
      if (schema.datasets) {
        return {
          success: true,
          datasets: schema.datasets
        }
      } else {
        return {
          success: true,
          datasets: {} // Return empty object if no datasets found
        }
      }
    } catch (error) {
      serviceLogger.error(`Error getting datasets for tracker ${trackerId}:`, error)
      const errorMessage = error instanceof Error ? error.message : String(error)
      return {
        success: false,
        error: `Failed to get datasets: ${errorMessage}`
      }
    }
  }

  /**
   * Get source files for a specific app folder
   *
   * @param trackerId - The ID of the tracker from the UI
   * @returns Object containing file tree structure
   */
  public async getAppSourceFiles(trackerId: string): Promise<{
    success: boolean
    files?: AppFileTreeNode[]
    error?: string
  }> {
    try {
      if (!this.appsBaseDir) {
        if (syftboxConfig && syftboxConfig.data_dir) {
          this.appsBaseDir = join(syftboxConfig.data_dir, 'apps')
        } else {
          return {
            success: false,
            error: 'Apps directory not configured'
          }
        }
      }

      // Make sure our ID to folder mapping is up to date
      if (this.trackerIdToFolderMap.size === 0) {
        await this.updateTrackerIdMapping()
      }

      // First try to use the ID directly as the folder name
      let folderName = trackerId

      // If that folder doesn't exist, scan all folders and read metadata to find a match
      try {
        await access(join(this.appsBaseDir, folderName), constants.R_OK)
      } catch {
        // Folder doesn't exist with the ID as name, so we need to scan
        const folders = await this.getAppFolderNames()

        // Just use the first folder we find as a fallback
        if (folders.length > 0) {
          folderName = folders[0]
          serviceLogger.warn(
            `Couldn't find folder for tracker ID ${trackerId}, using first available folder: ${folderName}`
          )
        } else {
          return {
            success: false,
            error: `No tracker folders found in ${this.appsBaseDir}`
          }
        }
      }

      // Build the path to the app folder
      const appPath = join(this.appsBaseDir, folderName)

      // Check if folder exists
      try {
        await access(appPath, constants.R_OK)
      } catch (error) {
        serviceLogger.error(`App folder not accessible for tracker ${trackerId}:`, error)
        return {
          success: false,
          error: `App folder not found for tracker: ${folderName}`
        }
      }

      // Get the file tree
      const fileTree = await this.buildFileTree(appPath, '')

      return {
        success: true,
        files: fileTree
      }
    } catch (error) {
      serviceLogger.error(`Error getting source files for tracker ${trackerId}:`, error)
      const errorMessage = error instanceof Error ? error.message : String(error)
      return {
        success: false,
        error: `Failed to get source files: ${errorMessage}`
      }
    }
  }

  /**
   * Download a tracker from the server and install it
   *
   * @param trackerId - The ID of the tracker to download
   * @returns Success status, message, and the installed tracker ID
   */
  public async downloadAndExtractTracker(trackerId: string): Promise<{
    success: boolean
    message?: string
    error?: string
    trackerId?: string
  }> {
    let zipPath = ''

    try {
      // 1. Validate parameters
      if (!trackerId || typeof trackerId !== 'string') {
        return {
          success: false,
          error: 'Invalid tracker ID provided'
        }
      }

      serviceLogger.info(`Starting download and installation for tracker ID: "${trackerId}"`)

      // 2. Ensure SyftBox is configured
      if (!syftboxConfig || !syftboxConfig.data_dir) {
        return {
          success: false,
          error: 'SyftBox configuration not available. Cannot download tracker.'
        }
      }

      // 3. Build paths for downloading and extraction
      const dataDir = syftboxConfig.data_dir
      zipPath = join(dataDir, `${trackerId}.zip`)
      const appsDir = join(dataDir, 'apps')

      // 4. Create apps directory if it doesn't exist
      if (!existsSync(appsDir)) {
        await mkdir(appsDir, { recursive: true })
        serviceLogger.info(`Created apps directory: ${appsDir}`)
      }

      // 5. Download the zip file
      try {
        serviceLogger.info(`Downloading tracker ${trackerId} from server...`)

        // Create parent directory for the zip file if needed
        const zipDir = dirname(zipPath)
        if (!existsSync(zipDir)) {
          await mkdir(zipDir, { recursive: true })
        }

        // Parse the server URL from config to build the download URL
        const serverUrl = new url.URL(appConfig.serverURL)
        const isHttps = serverUrl.protocol === 'https:'
        const hostname = serverUrl.hostname
        const port = serverUrl.port ? parseInt(serverUrl.port) : isHttps ? 443 : 80

        // Build the download URL using the server URL from config
        const downloadUrl = `${isHttps ? 'https' : 'http'}://${hostname}:${port}/tracker-folder/${trackerId}`
        serviceLogger.info(`Download URL: ${downloadUrl}`)

        // Create a write stream to save the zip file
        const fileWriter = createWriteStream(zipPath)

        try {
          // Download the file
          const response = await axios({
            method: 'GET',
            url: downloadUrl,
            responseType: 'stream',
            timeout: 30000, // 30 second timeout
            // Use appropriate agent based on protocol
            ...(isHttps
              ? {
                  httpsAgent: new https.Agent({
                    // Only disable certificate verification for localhost
                    rejectUnauthorized: hostname !== 'localhost'
                  })
                }
              : {})
          })

          // Check response status
          if (response.status !== 200) {
            throw new Error(`Server returned ${response.status} status code`)
          }

          // Pipe the response data to the file write stream
          await pipeline(response.data, fileWriter)

          serviceLogger.info(`Tracker zip downloaded to ${zipPath}`)
        } catch (axiosError) {
          if (axios.isAxiosError(axiosError)) {
            if (axiosError.code === 'ECONNREFUSED') {
              throw new Error('Connection refused. The tracker server may be offline.')
            } else if (axiosError.code === 'ETIMEDOUT') {
              throw new Error('Connection timed out. The tracker server may be unresponsive.')
            } else if (axiosError.response && axiosError.response.status === 404) {
              throw new Error(`Tracker with ID ${trackerId} not found on the server.`)
            } else {
              throw axiosError
            }
          } else {
            // For non-Axios errors
            throw new Error(`Download error: ${String(axiosError)}`)
          }
        }
      } catch (error) {
        serviceLogger.error(`Failed to download tracker ${trackerId}:`, error)
        const errorMessage = error instanceof Error ? error.message : String(error)
        return {
          success: false,
          error: `Failed to download tracker: ${errorMessage}`
        }
      }

      // 6. Extract the zip file
      try {
        serviceLogger.info(`Extracting tracker ${trackerId} to ${appsDir}...`)

        // Ensure the zip file exists and is readable
        if (!existsSync(zipPath)) {
          throw new Error(`Zip file not found at ${zipPath}`)
        }

        // Create a read stream from the zip file
        const fileReader = createReadStream(zipPath)

        // Extract the zip file to the apps directory
        await pipeline(fileReader, unzipper.Extract({ path: appsDir }))

        serviceLogger.info(`Tracker extracted successfully to ${appsDir}`)

        // Validate that the extraction succeeded by checking if files were created
        const trackerDir = join(appsDir, trackerId)
        const metadataPath = join(trackerDir, 'metadata.json')

        if (!existsSync(trackerDir)) {
          throw new Error(
            'Tracker files were not extracted correctly. Expected metadata file not found.'
          )
        }
      } catch (error) {
        serviceLogger.error(`Failed to extract tracker ${trackerId}:`, error)
        const errorMessage = error instanceof Error ? error.message : String(error)
        return {
          success: false,
          error: `Failed to extract tracker: ${errorMessage}`
        }
      }

      // 7. Return success
      return {
        success: true,
        message: `Tracker ${trackerId} downloaded and installed successfully.`,
        trackerId: trackerId
      }
    } catch (error) {
      serviceLogger.error(`Error downloading and extracting tracker ${trackerId}:`, error)
      const errorMessage = error instanceof Error ? error.message : String(error)
      return {
        success: false,
        error: `Failed to install tracker: ${errorMessage}`
      }
    } finally {
      // 8. Clean up temporary zip file if it exists
      try {
        if (zipPath && existsSync(zipPath)) {
          await unlink(zipPath)
          serviceLogger.info(`Deleted temporary zip file: ${zipPath}`)
        }
      } catch (cleanupError) {
        serviceLogger.warn(`Failed to delete temporary zip file ${zipPath}:`, cleanupError)
        // We don't fail the whole operation just because cleanup failed
      }
    }
  }

  /**
   * Read file content from the app directory
   *
   * @param trackerId - The ID of the tracker/app
   * @param filePath - The relative path to the file within the app folder
   * @returns Object containing file content
   */
  public async getAppFileContent(
    trackerId: string,
    filePath: string
  ): Promise<{
    success: boolean
    content?: string
    error?: string
  }> {
    try {
      if (!this.appsBaseDir) {
        if (syftboxConfig && syftboxConfig.data_dir) {
          this.appsBaseDir = join(syftboxConfig.data_dir, 'apps')
        } else {
          return {
            success: false,
            error: 'Apps directory not configured'
          }
        }
      }

      // Make sure our ID to folder mapping is up to date
      if (this.trackerIdToFolderMap.size === 0) {
        await this.updateTrackerIdMapping()
      }

      // First try to use the ID directly as the folder name
      let folderName = trackerId

      // If that folder doesn't exist, scan all folders and read metadata to find a match
      try {
        await access(join(this.appsBaseDir, folderName), constants.R_OK)
      } catch {
        // Folder doesn't exist with the ID as name, so we need to scan
        const folders = await this.getAppFolderNames()

        // Just use the first folder we find as a fallback
        if (folders.length > 0) {
          folderName = folders[0]
          serviceLogger.warn(
            `Couldn't find folder for tracker ID ${trackerId}, using first available folder: ${folderName}`
          )
        } else {
          return {
            success: false,
            error: `No tracker folders found in ${this.appsBaseDir}`
          }
        }
      }

      // Build the complete file path
      const fullPath = join(this.appsBaseDir, folderName, filePath)

      // Check if file exists
      try {
        await access(fullPath, constants.R_OK)
      } catch (error) {
        serviceLogger.error(`File not accessible: ${fullPath}`, error)
        return {
          success: false,
          error: `File not found: ${filePath}`
        }
      }

      // Read the file content
      const content = await readFile(fullPath, 'utf-8')

      return {
        success: true,
        content
      }
    } catch (error) {
      serviceLogger.error(`Error reading file content for ${filePath}:`, error)
      const errorMessage = error instanceof Error ? error.message : String(error)
      return {
        success: false,
        error: `Failed to read file: ${errorMessage}`
      }
    }
  }

  /**
   * Build a recursive file tree starting from the given directory
   *
   * @param rootPath - The absolute path to start from
   * @param relativePath - The relative path from the app root (used for display/traversal)
   * @returns An array of file tree nodes
   */
  private async buildFileTree(rootPath: string, relativePath: string): Promise<AppFileTreeNode[]> {
    const fullPath = join(rootPath, relativePath)
    const result: AppFileTreeNode[] = []

    // Skip node_modules and other common directories to exclude
    const excludeDirs = [
      'node_modules',
      '.git',
      'dist',
      'build',
      '.cache',
      'coverage',
      '__pycache__'
    ]
    const excludeFiles = ['.DS_Store', '.gitignore', '.npmrc', '*.pyc']

    try {
      const entries = await readdir(fullPath, { withFileTypes: true })

      // First add directories
      for (const entry of entries.filter((entry) => entry.isDirectory())) {
        // Skip excluded directories
        if (excludeDirs.includes(entry.name)) continue

        const dirRelativePath = join(relativePath, entry.name)
        const children = await this.buildFileTree(rootPath, dirRelativePath)

        // Only add directories that have content
        if (children.length > 0) {
          result.push({
            name: entry.name,
            path: dirRelativePath,
            type: 'directory',
            children
          })
        }
      }

      // Then add files
      for (const entry of entries.filter((entry) => entry.isFile())) {
        // Skip excluded files
        if (excludeFiles.includes(entry.name)) continue

        result.push({
          name: entry.name,
          path: join(relativePath, entry.name),
          type: 'file'
        })
      }

      // Sort by type (directories first), then name
      return result.sort((a, b) => {
        if (a.type !== b.type) {
          return a.type === 'directory' ? -1 : 1
        }
        return a.name.localeCompare(b.name)
      })
    } catch (error) {
      serviceLogger.error(`Error building file tree for ${fullPath}:`, error)
      return []
    }
  }
}

export const trackerService = new TrackerService()
