import { join, isAbsolute, basename, dirname } from 'path'
import { readFileSync, existsSync, readdirSync, mkdirSync, writeFileSync, rmSync } from 'fs'
import { app } from 'electron'
import { createHash } from 'crypto'
import { AppTracker, DocumentStats } from '../../shared/types'
import { createServiceLogger } from '../../shared/logging'
import { appConfig } from './config'
import axios from 'axios'

// Create a specific logger for app service
const logger = createServiceLogger('appService')

interface SyftboxConfig {
  data_dir: string
  server_url: string
  client_url: string
  email: string
  token: string
  access_token: string
  client_timeout: number
}

// Define interface for app metadata from metadata.json
interface AppMetadata {
  name: string
  description: string
  version: string
  icon: string
  hasUpdate?: boolean
  updateVersion?: string
  active?: boolean
  tag?: string // Optional tag property used for filtering
}

export let syftboxConfig: SyftboxConfig | null = null

// Mock document count - this would typically come from a RAG system or other source
// We're keeping this in memory rather than in the database
let documentCount = 24

// Sample app metadata for creating default apps
const sampleApps: AppMetadata[] = [
  {
    name: 'Slack Tracker',
    description: 'Monitor messages and activity across Slack workspaces',
    version: '1.3.2',
    icon: 'MessageSquare',
    hasUpdate: true,
    updateVersion: '1.4.0'
  },
  {
    name: 'GitHub Tracker',
    description: 'Track issues, pull requests, and repository activity',
    version: '2.1.0',
    icon: 'Github',
    hasUpdate: false
  },
  {
    name: 'Gmail Tracker',
    description: 'Monitor inbox activity and important email threads',
    version: '1.5.3',
    icon: 'Mail',
    hasUpdate: true,
    updateVersion: '2.0.0'
  },
  {
    name: 'Notion Tracker',
    description: 'Track changes to pages, databases, and workspace activity',
    version: '0.9.7',
    icon: 'FileText',
    hasUpdate: false
  },
  {
    name: 'Discord Tracker',
    description: 'Monitor messages and activity across Discord servers',
    version: '1.2.1',
    icon: 'Headphones',
    hasUpdate: false
  }
]

/**
 * Initialize app trackers directory
 * Called during application initialization
 */
export function initializeAppTrackers(): void {
  // Check if we have SyftBox configuration with data_dir
  if (syftboxConfig && syftboxConfig.data_dir) {
    try {
      // First check if we need to migrate from old directory structure
      migrateAppDirectories()

      // Make sure the apps directory exists
      const appsDir = getAppsDir()
      if (!existsSync(appsDir)) {
        logger.debug(`Creating apps directory: ${appsDir}`)
        mkdirSync(appsDir, { recursive: true })

        // Create sample app if no apps exist yet
        createSampleApp(appsDir)
      }
    } catch (error) {
      logger.error('Error initializing app tracker filesystem structure:', error)
    }
  } else {
    logger.debug('SyftBox configuration not available. Skipping filesystem app initialization.')
  }
}

/**
 * Migrate apps from old directory structure (installed/active) to new single directory
 * This function is called during initialization to handle the transition
 */
function migrateAppDirectories(): void {
  if (!syftboxConfig || !syftboxConfig.data_dir) {
    return
  }

  try {
    // Check if old directories exist
    const oldInstalledDir = join(syftboxConfig.data_dir, 'installed')
    const oldActiveDir = join(syftboxConfig.data_dir, 'apps')
    const newAppsDir = getAppsDir() // This is the same as oldActiveDir

    // Skip migration if old installed directory doesn't exist
    if (!existsSync(oldInstalledDir)) {
      logger.debug('No migration needed - old installed directory not found')
      return
    }

    logger.debug('Starting migration of apps to new directory structure')

    // Ensure the new apps directory exists
    if (!existsSync(newAppsDir)) {
      mkdirSync(newAppsDir, { recursive: true })
    }

    // First, migrate apps from installed directory
    if (existsSync(oldInstalledDir)) {
      try {
        const appFolders = readdirSync(oldInstalledDir, { withFileTypes: true })
          .filter((dirent) => dirent.isDirectory())
          .map((dirent) => dirent.name)

        logger.debug(`Found ${appFolders.length} apps to migrate from installed directory`)

        for (const folder of appFolders) {
          const sourcePath = join(oldInstalledDir, folder)
          const destPath = join(newAppsDir, folder)

          // Check if app already exists in destination
          if (existsSync(destPath)) {
            logger.debug(`App ${folder} already exists in apps directory, skipping migration`)
            continue
          }

          // Read the app metadata
          const metadata = readAppMetadata(sourcePath)
          if (metadata) {
            // Add active field (false for apps from installed directory)
            metadata.active = false

            // Create destination directory
            mkdirSync(destPath, { recursive: true })

            // Copy all files from source to destination
            const files = readdirSync(sourcePath)
            for (const file of files) {
              const sourceFilePath = join(sourcePath, file)
              const destFilePath = join(destPath, file)

              // If it's the metadata.json file, write the updated version
              if (file === 'metadata.json') {
                writeFileSync(destFilePath, JSON.stringify(metadata, null, 2))
              } else {
                // Otherwise copy the file directly
                const content = readFileSync(sourceFilePath)
                writeFileSync(destFilePath, content)
              }
            }

            logger.debug(`Migrated app ${folder} from installed directory with active=false`)
          }
        }

        logger.debug('Migration from installed directory completed')
      } catch (error) {
        logger.error('Error migrating apps from installed directory:', error)
      }
    }

    // Update metadata for apps in the active directory to ensure they have active=true
    if (existsSync(oldActiveDir)) {
      try {
        const appFolders = readdirSync(oldActiveDir, { withFileTypes: true })
          .filter((dirent) => dirent.isDirectory())
          .map((dirent) => dirent.name)

        logger.debug(`Found ${appFolders.length} apps in active directory to update`)

        for (const folder of appFolders) {
          const appPath = join(oldActiveDir, folder)
          const metadata = readAppMetadata(appPath)

          if (metadata) {
            // Only update if active field is not already set
            if (metadata.active === undefined) {
              // Set active to true for apps in the active directory
              metadata.active = true

              // Write updated metadata
              const metadataPath = join(appPath, 'metadata.json')
              writeFileSync(metadataPath, JSON.stringify(metadata, null, 2))

              logger.debug(`Updated app ${folder} metadata with active=true`)
            }
          }
        }

        logger.debug('Active directory apps metadata update completed')
      } catch (error) {
        logger.error('Error updating active directory apps metadata:', error)
      }
    }

    // Once migration is complete, we can delete the old installed directory
    if (existsSync(oldInstalledDir)) {
      try {
        // First check if there are any apps left in the directory
        const remainingFiles = readdirSync(oldInstalledDir)
        if (remainingFiles.length === 0) {
          // Safe to remove the directory
          rmSync(oldInstalledDir, { recursive: true, force: true })
          logger.debug('Old installed directory removed after successful migration')
        } else {
          logger.debug(
            'Old installed directory not empty after migration, manual cleanup may be needed'
          )
        }
      } catch (error) {
        logger.error('Error removing old installed directory:', error)
      }
    }

    logger.debug('App directory migration completed successfully')
  } catch (error) {
    logger.error('Error during app directory migration:', error)
  }
}

/**
 * Create a sample app if no apps exist yet
 */
function createSampleApp(appsDir: string): void {
  try {
    // Create sample apps from the sampleApps array
    for (const app of sampleApps) {
      // Create folder name based on app name
      const folderName = app.name.toLowerCase().replace(/\s+/g, '-')
      const appPath = join(appsDir, folderName)

      if (!existsSync(appPath)) {
        logger.debug(`Creating sample app: ${app.name}...`)
        mkdirSync(appPath, { recursive: true })

        // Add active field to metadata (set to false by default)
        const appMetadata = { ...app, active: false }

        // Create metadata.json
        const metadataPath = join(appPath, 'metadata.json')
        writeFileSync(metadataPath, JSON.stringify(appMetadata, null, 2))
      }
    }

    logger.debug('Sample apps created successfully.')
  } catch (error) {
    logger.error('Error creating sample apps:', error)
  }
}

/**
 * Get apps directory path
 */
export function getAppsDir(): string {
  if (!syftboxConfig || !syftboxConfig.data_dir) {
    throw new Error('syftboxConfig or data_dir not defined')
  }
  return join(syftboxConfig.data_dir, 'apps')
}

/**
 * Scan for installed apps in the apps directory
 */
export function scanInstalledApps(): Record<string, AppMetadata> {
  try {
    const appMetadata: Record<string, AppMetadata> = {}

    // Ensure apps directory exists
    const appsDir = getAppsDir()
    if (!existsSync(appsDir)) {
      mkdirSync(appsDir, { recursive: true })
    }

    // Scan apps directory
    try {
      const appFolders = readdirSync(appsDir, { withFileTypes: true })
        .filter((dirent) => dirent.isDirectory())
        .map((dirent) => dirent.name)

      for (const folder of appFolders) {
        const folderPath = join(appsDir, folder)
        const metadata = readAppMetadata(folderPath)
        if (metadata) {
          appMetadata[folderPath] = metadata
        }
      }
    } catch (error) {
      // Error handled by returning empty object
    }

    return appMetadata
  } catch (error) {
    return {}
  }
}

/**
 * Read app metadata from metadata.json
 */
export function readAppMetadata(folderPath: string): AppMetadata | null {
  try {
    const metadataPath = join(folderPath, 'metadata.json')
    if (!existsSync(metadataPath)) {
      return null
    }

    const metadataContent = readFileSync(metadataPath, 'utf8')
    const metadata = JSON.parse(metadataContent)

    // Validate required fields
    if (!metadata.name || !metadata.description || !metadata.version || !metadata.icon) {
      return null
    }

    // Ensure hasUpdate and updateVersion are defined (optional fields)
    if (metadata.hasUpdate === undefined) {
      metadata.hasUpdate = false
    }
    if (metadata.hasUpdate && !metadata.updateVersion) {
      metadata.hasUpdate = false
    }

    // Ensure active field is defined, set to false by default if not present
    if (metadata.active === undefined) {
      metadata.active = false

      // Update the metadata.json file with active field
      writeFileSync(metadataPath, JSON.stringify(metadata, null, 2))
    }

    return metadata
  } catch (error) {
    return null
  }
}

/**
 * Generate a unique ID for an app based on its path
 * Used to replace database IDs with filesystem-based IDs
 */
function generateAppId(path: string): string {
  // Create a hash of the path to use as ID
  return createHash('md5').update(path).digest('hex').substring(0, 8)
}

/**
 * Get all available app trackers (filesystem-only approach)
 * Uses the filesystem as the source of truth for app metadata
 */
export function getAppTrackers(): AppTracker[] {
  try {
    // Early return if SyftBox is not configured
    if (!syftboxConfig || !syftboxConfig.data_dir) {
      logger.debug('SyftBox not configured. Returning empty list.')
      return []
    }

    // Get apps directory path
    const appsDir = getAppsDir()

    // Ensure apps directory exists
    if (!existsSync(appsDir)) {
      mkdirSync(appsDir, { recursive: true })
      return []
    }

    // Scan the filesystem directly to get all apps
    const appTrackers: AppTracker[] = []

    try {
      const appFolders = readdirSync(appsDir, { withFileTypes: true })
        .filter((dirent) => dirent.isDirectory())
        .map((dirent) => dirent.name)

      for (const folder of appFolders) {
        const folderPath = join(appsDir, folder)
        const metadata = readAppMetadata(folderPath)

        if (metadata) {
          // Create app tracker object directly from metadata
          appTrackers.push({
            id: generateAppId(folderPath), // Generate ID from path
            name: metadata.name,
            description: metadata.description,
            version: metadata.version,
            enabled: metadata.active || false,
            icon: metadata.icon,
            hasUpdate: metadata.hasUpdate || false,
            updateVersion: metadata.updateVersion || undefined,
            path: folderPath
          })
        }
      }
    } catch (error) {
      logger.error(`Error scanning apps directory ${appsDir}:`, error)
    }

    logger.debug(`Returning ${appTrackers.length} app trackers from filesystem`)
    return appTrackers
  } catch (error) {
    logger.error('Error loading app trackers from filesystem:', error)
    return []
  }
}

/**
 * @deprecated Use getAppsDir() instead
 * Get active apps directory path (kept for backward compatibility)
 */
export function getActiveAppsDir(): string {
  return getAppsDir()
}

/**
 * Move a folder from source to destination
 * @param source Source folder path
 * @param destination Destination folder path (directory where the folder should be moved to)
 * @returns True if successful, false otherwise
 */
function moveFolder(source: string, destination: string): boolean {
  try {
    // Create destination directory if it doesn't exist
    if (!existsSync(destination)) {
      mkdirSync(destination, { recursive: true })
    }

    // Get folder name from source
    const folderName = basename(source)
    const destPath = join(destination, folderName)

    // If destination already exists, remove it first
    if (existsSync(destPath)) {
      // In a full implementation, you would use fs.rmSync recursively here
      // For now, we'll just continue
    }

    // Perform the move operation
    // Use node's built-in require for child_process
    // Note: In a production app, you should import this at the top of the file
    const childProcess = require('child_process')
    childProcess.execSync(`mv "${source}" "${destination}"`)
    return true
  } catch (error) {
    return false
  }
}

/**
 * Toggle the enabled state of an app tracker by updating the metadata.active field
 * @param id App tracker ID
 * @returns Updated app tracker
 */
export async function toggleAppTracker(id: string): Promise<AppTracker | null> {
  logger.debug(`Toggling app tracker with ID ${id}`)

  // Get all apps from filesystem
  const allApps = getAppTrackers()
  const appTracker = allApps.find((app) => app.id === id)

  if (!appTracker) {
    logger.error(`App tracker with ID ${id} not found`)
    return null
  }

  if (!appTracker.path) {
    logger.error(`App tracker with ID ${id} has no path information`)
    return null
  }

  try {
    // Check if app exists in filesystem
    if (!existsSync(appTracker.path)) {
      logger.error(`App ${appTracker.name} not found at path ${appTracker.path}`)
      return null
    }

    // Read app metadata
    const metadata = readAppMetadata(appTracker.path)
    if (!metadata) {
      logger.error(`Failed to read metadata for app ${appTracker.name}`)
      return null
    }

    // Toggle the active state in metadata
    const newActiveState = !metadata.active
    metadata.active = newActiveState

    // Update metadata.json file
    const metadataPath = join(appTracker.path, 'metadata.json')
    writeFileSync(metadataPath, JSON.stringify(metadata, null, 2))

    // Send request to the dk_api endpoint to toggle metadata active state
    if (appConfig && appConfig.dk_api) {
      try {
        // Get the tag field from metadata if it exists
        const tagField = metadata.tag || appTracker.name.toLowerCase()

        // Make request to toggle active state in the backend using axios
        logger.debug(
          `Sending metadata toggle request to ${appConfig.dk_api}/rag/toggle-active-metadata for app ${appTracker.name}`
        )

        // Use async/await with a try/catch block instead of promise chains
        try {
          const response = await axios.post(
            `${appConfig.dk_api}/rag/toggle-active-metadata`,
            {
              filter_field: 'app',
              filter_value: tagField
            },
            {
              headers: {
                'Content-Type': 'application/json'
              },
              timeout: 5000 // 5 second timeout
            }
          )

          logger.debug(
            `Successfully sent metadata toggle request to dk_api for app ${appTracker.name}`
          )
          logger.debug(`Response from toggle-active-metadata: ${JSON.stringify(response.data)}`)
        } catch (requestError) {
          logger.error(
            `Error sending metadata toggle request to dk_api for app ${appTracker.name}:`,
            requestError
          )
        }
      } catch (error) {
        logger.error(`Error preparing metadata toggle request for app ${appTracker.name}:`, error)
      }
    }

    // Create updated tracker with new state
    const updatedTracker = {
      ...appTracker,
      enabled: newActiveState // Update state based on metadata.active
    }

    logger.debug(
      `Successfully toggled app ${appTracker.name} to ${newActiveState ? 'active' : 'inactive'}`
    )
    return updatedTracker
  } catch (error) {
    logger.error(`Error toggling app tracker with ID ${id}:`, error)
    return null
  }
}

/**
 * Uninstall an app tracker
 * Deletes the app folder
 * @param id App tracker ID
 * @returns Success status and message
 */
export function uninstallAppTracker(id: string): { success: boolean; message: string } {
  logger.debug(`Uninstalling app tracker with ID ${id}`)

  // Get all apps from filesystem
  const allApps = getAppTrackers()
  const appTracker = allApps.find((app) => app.id === id)

  if (!appTracker) {
    return {
      success: false,
      message: `App tracker with ID ${id} not found`
    }
  }

  if (!appTracker.path || !existsSync(appTracker.path)) {
    return {
      success: true,
      message: `App tracker ${appTracker.name} folder not found, considering it already uninstalled`
    }
  }

  try {
    // Only allow uninstallation for disabled apps
    if (appTracker.enabled) {
      return {
        success: false,
        message: `Cannot uninstall enabled app ${appTracker.name}. Please disable it first.`
      }
    }

    // Delete the app folder
    logger.debug(`Deleting app folder at ${appTracker.path}`)
    rmSync(appTracker.path, { recursive: true, force: true })

    return {
      success: true,
      message: `App ${appTracker.name} uninstalled successfully`
    }
  } catch (error) {
    logger.error(`Error uninstalling app tracker with ID ${id}:`, error)
    const errorMessage = error instanceof Error ? error.message : String(error)
    return {
      success: false,
      message: `Error uninstalling app: ${errorMessage}`
    }
  }
}

/**
 * Get document count statistics
 * Fetches the count from the dk_api endpoint
 */
export async function getDocumentCount(): Promise<DocumentStats & { error?: string }> {
  try {
    // Import the config from '../services/config' to avoid circular dependency
    const { appConfig } = await import('../services/config')

    if (appConfig.dk_api) {
      try {
        // Use axios for consistent API handling across the app
        const response = await axios.get(`${appConfig.dk_api}/rag/count`, {
          timeout: 5000 // 5 second timeout
        })

        // Check if response data has the expected structure
        if (response.data && typeof response.data.count === 'number') {
          logger.debug(`Successfully retrieved document count: ${response.data.count}`)
          return { count: response.data.count }
        } else {
          logger.debug('Invalid response format from RAG document count endpoint')
          return {
            count: documentCount,
            error: 'Invalid response format from server. Using cached data.'
          }
        }
      } catch (fetchError) {
        logger.error('Failed to fetch document count from API:', fetchError)
        // Return error message along with fallback count
        return {
          count: documentCount,
          error: `Failed to connect to ${appConfig.dk_api}/rag/count. Using cached data.`
        }
      }
    } else {
      // Fallback to in-memory value if dk_api is not configured
      logger.debug('No dk_api configured in app settings. Using mock document count.')
      return {
        count: documentCount,
        error: 'No API endpoint configured (dk_api missing from config). Using mock data.'
      }
    }
  } catch (error) {
    logger.error('Failed to fetch document count from API:', error)
    // Fallback to in-memory value in case of error
    return {
      count: documentCount,
      error: 'Unexpected error while fetching document count. Using mock data.'
    }
  }
}

/**
 * Cleanup Documents
 * Sends a DELETE request to dk_api + '/rag/all' to remove all documents
 * @returns Success status and message
 */
export async function cleanupDocuments(): Promise<{ success: boolean; message: string }> {
  try {
    // Import the config to avoid circular dependency
    const { appConfig } = await import('../services/config')

    if (!appConfig.dk_api) {
      logger.debug('Cannot clean up documents: No dk_api endpoint configured')
      return {
        success: false,
        message: 'No API endpoint configured (dk_api missing from config).'
      }
    }

    try {
      logger.debug(`Sending DELETE request to ${appConfig.dk_api}/rag/all`)

      // Send DELETE request to dk_api/rag/all using axios for consistency
      const response = await axios.delete(`${appConfig.dk_api}/rag/all`, {
        headers: {
          'Content-Type': 'application/json'
        },
        timeout: 10000 // 10 second timeout
      })

      // Check if request was successful (status 200-299)
      if (response.status >= 200 && response.status < 300) {
        // Reset in-memory document count to 0
        documentCount = 0

        logger.debug('Successfully cleaned up all documents')
        return {
          success: true,
          message: 'All documents have been successfully removed.'
        }
      } else {
        // This branch is less likely to execute with axios as it throws errors for non-2xx responses
        const errorData = response.data || 'Unknown error'
        logger.error(`Failed to cleanup documents. Status: ${response.status}. Error: ${errorData}`)
        return {
          success: false,
          message: `Failed to cleanup documents. Server returned: ${response.status} ${errorData}`
        }
      }
    } catch (axiosError) {
      logger.error('Failed to connect to cleanup API:', axiosError)

      // Extract message from axios error
      let errorMessage = 'Unknown error'
      if (axios.isAxiosError(axiosError)) {
        if (axiosError.response) {
          // The request was made and the server responded with a status outside 2xx
          errorMessage = `Server returned ${axiosError.response.status}: ${axiosError.response.data || axiosError.message}`
        } else if (axiosError.request) {
          // The request was made but no response received
          errorMessage = 'No response received from server (timeout or connection error)'
        } else {
          // Something happened in setting up the request
          errorMessage = axiosError.message
        }
      } else if (axiosError instanceof Error) {
        errorMessage = axiosError.message
      }

      return {
        success: false,
        message: `Failed to connect to ${appConfig.dk_api}/rag/all: ${errorMessage}`
      }
    }
  } catch (error) {
    logger.error('Error in cleanupDocuments function:', error)
    const errorMessage = error instanceof Error ? error.message : String(error)
    return {
      success: false,
      message: `Unexpected error while cleaning up documents: ${errorMessage}`
    }
  }
}

/**
 * Search RAG documents from the server
 * @param query Search query
 * @param numResults Number of results to return
 * @returns Array of RAG documents or empty array if failed
 */
export async function searchRAGDocuments(query: string, numResults: number = 5): Promise<any> {
  try {
    // Use dk_api as the preferred endpoint
    if (!appConfig.dk_api) {
      logger.error('dk_api URL not defined in config')
      return { documents: [] }
    }

    // Use dk_api for all RAG endpoints
    const ragServerBaseUrl = appConfig.dk_api

    let response

    // For empty query, get all active documents
    if (!query.trim()) {
      logger.debug(`Getting all active documents from ${ragServerBaseUrl}/rag/active/true`)

      // Simple GET request without any parameters
      response = await axios.get(`${ragServerBaseUrl}/rag/active/true`, {
        timeout: 10000 // 10 second timeout
      })
    } else {
      // Normal search with query
      logger.debug(`Searching RAG documents with query: "${query}", numResults: ${numResults}`)

      response = await axios.get(`${ragServerBaseUrl}/rag`, {
        params: {
          query,
          num_results: numResults
        },
        timeout: 10000 // 10 second timeout
      })
    }

    // Process the response to ensure required fields exist
    if (response.data && response.data.documents) {
      response.data.documents.forEach((doc: any) => {
        // If the document doesn't have metadata, create an empty one
        if (!doc.metadata) {
          doc.metadata = { date: new Date().toLocaleString() }
        }
      })
    }

    // Return the documents from the response
    return response.data
  } catch (error) {
    logger.error(`Failed to search or get RAG documents:`, error)
    // Return empty results on error
    return { documents: [] }
  }
}

/**
 * Update document count (affects in-memory state only)
 */
export function updateDocumentCount(count: number): void {
  documentCount = count
}

/**
 * Install a new app tracker
 * Creates the app directory and metadata.json file in the apps directory
 * @param metadata App metadata to install
 * @param sourcePath Optional file path to source files for the app
 * @returns Success status, message, and the new app tracker
 */
export function installAppTracker(
  metadata?: AppMetadata,
  sourcePath?: string
): { success: boolean; message: string; appTracker?: AppTracker } {
  try {
    // If no metadata provided, this is a placeholder/mock installation
    if (!metadata) {
      return {
        success: true,
        message:
          'App installation requires metadata. Use the app store or provide metadata directly.'
      }
    }

    // Ensure syftbox is configured
    if (!syftboxConfig || !syftboxConfig.data_dir) {
      return {
        success: false,
        message: 'SyftBox configuration not available. Cannot install app.'
      }
    }

    const appsDir = getAppsDir()

    // Create folder name based on app name
    const folderName = metadata.name.toLowerCase().replace(/\s+/g, '-')
    const appPath = join(appsDir, folderName)

    // Create app directory if it doesn't exist
    if (!existsSync(appPath)) {
      mkdirSync(appPath, { recursive: true })
    }

    // Ensure the active field is set (default to false for new installations)
    if (metadata.active === undefined) {
      metadata.active = false
    }

    // Write metadata.json
    const metadataPath = join(appPath, 'metadata.json')
    writeFileSync(metadataPath, JSON.stringify(metadata, null, 2))

    // Copy app files if sourcePath is provided
    if (sourcePath && existsSync(sourcePath)) {
      // Implementation for copying app files from source
      logger.debug(`Source path provided: ${sourcePath}. File copying would happen here.`)
    }

    // Create app tracker object directly
    const appTracker: AppTracker = {
      id: generateAppId(appPath),
      name: metadata.name,
      description: metadata.description,
      version: metadata.version,
      enabled: metadata.active || false, // Use metadata.active for enabled state
      icon: metadata.icon,
      hasUpdate: false,
      path: appPath
    }

    return {
      success: true,
      message: 'App installed successfully',
      appTracker: appTracker
    }
  } catch (error) {
    logger.error('Error installing app tracker:', error)
    const errorMessage = error instanceof Error ? error.message : String(error)
    return {
      success: false,
      message: `Error installing app: ${errorMessage}`
    }
  }
}

/**
 * Update an app tracker
 * Updates the metadata.json file with the new version
 * @param id App tracker ID
 * @returns Updated app tracker or null if not found
 */
export function updateAppTracker(id: string): {
  success: boolean
  appTracker?: AppTracker
  message?: string
} {
  // Get all apps from filesystem
  const allApps = getAppTrackers()
  const appTracker = allApps.find((app) => app.id === id)

  if (!appTracker) {
    return {
      success: false,
      message: 'App tracker not found'
    }
  }

  try {
    // Check the metadata.json file (source of truth)
    if (appTracker.path && existsSync(appTracker.path)) {
      const metadataPath = join(appTracker.path, 'metadata.json')

      if (existsSync(metadataPath)) {
        // Read existing metadata
        const existingMetadata = readAppMetadata(appTracker.path)

        if (existingMetadata) {
          // Check if update is available from metadata
          if (!existingMetadata.hasUpdate || !existingMetadata.updateVersion) {
            return {
              success: false,
              message: 'No update available for this app'
            }
          }

          // Update the version in the metadata
          const updatedMetadata: AppMetadata = {
            ...existingMetadata,
            version: existingMetadata.updateVersion,
            hasUpdate: false,
            updateVersion: undefined
          }

          // Write updated metadata back to file
          writeFileSync(metadataPath, JSON.stringify(updatedMetadata, null, 2))
          logger.debug(`Updated metadata.json for app at ${appTracker.path}`)

          // Create updated tracker object
          const updatedAppTracker = {
            ...appTracker,
            version: existingMetadata.updateVersion,
            hasUpdate: false,
            updateVersion: undefined
          }

          return {
            success: true,
            appTracker: updatedAppTracker
          }
        }
      }
    }

    return {
      success: false,
      message: 'Could not update app: metadata.json not found or no update available'
    }
  } catch (error) {
    logger.error('Error updating app:', error)
    const errorMessage = error instanceof Error ? error.message : String(error)
    return {
      success: false,
      message: `Error updating app: ${errorMessage}`
    }
  }
}

/**
 * Load SyftBox configuration from syftbox_config.json or custom path
 * @returns boolean indicating if loading was successful
 */
export function loadSyftboxConfig(): boolean {
  // Use syftbox_config from appConfig directly rather than environment variable
  const configPath = appConfig.syftbox_config
    ? isAbsolute(appConfig.syftbox_config)
      ? appConfig.syftbox_config
      : join(process.cwd(), appConfig.syftbox_config)
    : join(app.getAppPath(), 'syftbox_config.json')

  try {
    // Check if file exists
    if (!existsSync(configPath)) {
      // SyftBox integration will be disabled
      syftboxConfig = null
      return false
    }

    // Read and parse config file
    const configFile = readFileSync(configPath, 'utf8')
    syftboxConfig = JSON.parse(configFile)

    // Verify data_dir exists in the filesystem
    if (syftboxConfig && syftboxConfig.data_dir && !existsSync(syftboxConfig.data_dir)) {
      // Data directory does not exist - continue anyway
    }

    return true
  } catch (error) {
    syftboxConfig = null
    return false
  }
}

/**
 * Get the current SyftBox configuration
 */
export function getSyftboxConfig(): SyftboxConfig | null {
  return syftboxConfig
}
