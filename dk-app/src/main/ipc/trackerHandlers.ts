import { ipcMain, dialog, BrowserWindow, app } from 'electron'
import { trackerService } from '../services/trackerService'
import { Channels } from '../../shared/constants'
import logger from '../../shared/logging'
import { readdir } from 'fs/promises'
import { join, resolve as pathResolve, basename, extname } from 'path'
import * as path from 'path'
import { syftboxConfig } from '../services/appService'
import * as fs from 'fs'
import { getOnboardingStatus, getConfigFilePath } from '../services/config'

/**
 * Register IPC handlers for tracker scanning service
 */
export function registerTrackerHandlers(): void {
  // Handler to manually trigger a tracker scan
  ipcMain.handle(Channels.TriggerTrackerScan, async () => {
    try {
      logger.info('Manual tracker scan triggered via IPC')
      await trackerService.scanTrackers()
      return { success: true, message: 'Tracker scan completed successfully' }
    } catch (error) {
      logger.error('Manual tracker scan failed:', error)
      return { success: false, message: 'Tracker scan failed', error }
    }
  })

  // Handler to start tracker scanning service
  ipcMain.handle(Channels.StartTrackerService, () => {
    try {
      // Check if config exists and onboarding is complete before starting
      const configPath = getConfigFilePath()
      const isConfigExists = fs.existsSync(configPath)

      // Get onboarding status from config service
      const onboardingStatus = getOnboardingStatus()
      const shouldStartTracker = isConfigExists && !onboardingStatus.isFirstRun

      if (shouldStartTracker) {
        logger.info('Starting tracker scanning service - config exists and onboarding is complete')
        trackerService.startTrackerScan()
        return { success: true, message: 'Tracker scanning service started' }
      } else {
        logger.warn('Cannot start tracker service - config missing or onboarding incomplete')
        return {
          success: false,
          message: 'Tracker service not started - configuration not complete',
          needsConfig: true
        }
      }
    } catch (error) {
      logger.error('Failed to start tracker service:', error)
      return { success: false, message: 'Failed to start tracker service', error }
    }
  })

  // Handler to stop tracker scanning service
  ipcMain.handle(Channels.StopTrackerService, () => {
    try {
      trackerService.stopTrackerScan()
      return { success: true, message: 'Tracker scanning service stopped' }
    } catch (error) {
      logger.error('Failed to stop tracker service:', error)
      return { success: false, message: 'Failed to stop tracker service', error }
    }
  })

  // Helper function to get all app folders
  async function getAvailableAppFolders(): Promise<string[]> {
    try {
      if (!syftboxConfig || !syftboxConfig.data_dir) {
        return []
      }

      const appsDir = join(syftboxConfig.data_dir, 'apps')
      const entries = await readdir(appsDir, { withFileTypes: true })
      return entries.filter((entry) => entry.isDirectory()).map((entry) => entry.name)
    } catch (error) {
      logger.error('Error getting available app folders:', error)
      return []
    }
  }

  // IPC handler to get app folders
  ipcMain.handle(Channels.GetAppFolders, async () => {
    try {
      const folders = await getAvailableAppFolders()
      logger.info(`Retrieved ${folders.length} app folders`)
      return { success: true, folders }
    } catch (error) {
      logger.error('Failed to get app folders:', error)
      const errorMessage = error instanceof Error ? error.message : String(error)
      return { success: false, error: errorMessage, folders: [] }
    }
  })

  // Handler to get templates for a specific tracker
  ipcMain.handle(Channels.GetTrackerTemplates, async (_, trackerId: string) => {
    try {
      logger.info(`Getting templates for tracker with ID: ${trackerId}`)

      const appFolders = await getAvailableAppFolders()

      // Force update of the tracker ID mapping to make sure we have fresh data
      await trackerService.updateTrackerIdMapping()

      const result = await trackerService.getTrackerTemplates(trackerId)

      if (result.success) {
        const templateCount = result.templates ? Object.keys(result.templates).length : 0
        logger.info(`Successfully fetched ${templateCount} templates for tracker ID: ${trackerId}`)
      } else {
        logger.warn(`Failed to get templates: ${result.error}`)
      }

      return result
    } catch (error) {
      logger.error(`Failed to get templates for tracker ${trackerId}:`, error)
      const errorMessage = error instanceof Error ? error.message : String(error)
      return {
        success: false,
        error: errorMessage || 'Failed to get templates',
        templates: {}
      }
    }
  })

  // Handler to get datasets for a specific tracker
  ipcMain.handle(Channels.GetTrackerDatasets, async (_, trackerId: string) => {
    try {
      logger.info(`Getting datasets for tracker with ID: ${trackerId}`)

      const appFolders = await getAvailableAppFolders()

      // Force update of the tracker ID mapping to make sure we have fresh data
      await trackerService.updateTrackerIdMapping()

      const result = await trackerService.getTrackerDatasets(trackerId)

      if (result.success) {
        const datasetCount = result.datasets ? Object.keys(result.datasets).length : 0
        logger.info(`Successfully fetched ${datasetCount} datasets for tracker ID: ${trackerId}`)
      } else {
        logger.warn(`Failed to get datasets: ${result.error}`)
      }

      return result
    } catch (error) {
      logger.error(`Failed to get datasets for tracker ${trackerId}:`, error)
      const errorMessage = error instanceof Error ? error.message : String(error)
      return {
        success: false,
        error: errorMessage || 'Failed to get datasets',
        datasets: {}
      }
    }
  })

  // Handler to get source files for a specific app/tracker
  ipcMain.handle(Channels.GetAppSourceFiles, async (_, trackerId: string) => {
    try {
      logger.info(`Getting source files for app/tracker with ID: ${trackerId}`)

      // Force update of the tracker ID mapping to make sure we have fresh data
      await trackerService.updateTrackerIdMapping()

      const result = await trackerService.getAppSourceFiles(trackerId)

      if (result.success) {
        const fileCount = result.files ? result.files.length : 0
        logger.info(
          `Successfully fetched ${fileCount} top-level file entries for app ID: ${trackerId}`
        )
      } else {
        logger.warn(`Failed to get source files: ${result.error}`)
      }

      return result
    } catch (error) {
      logger.error(`Failed to get source files for app ${trackerId}:`, error)
      const errorMessage = error instanceof Error ? error.message : String(error)
      return {
        success: false,
        error: errorMessage || 'Failed to get source files',
        files: []
      }
    }
  })

  // Handler to get file content for a specific app/tracker
  ipcMain.handle(Channels.GetAppFileContent, async (_, trackerId: string, filePath: string) => {
    try {
      logger.info(`Getting file content for "${filePath}" in app/tracker: ${trackerId}`)

      const result = await trackerService.getAppFileContent(trackerId, filePath)

      if (result.success) {
        const contentLength = result.content ? result.content.length : 0
        logger.info(`Successfully fetched content (${contentLength} bytes) for file: ${filePath}`)
      } else {
        logger.warn(`Failed to get file content: ${result.error}`)
      }

      return result
    } catch (error) {
      logger.error(`Failed to get file content for ${filePath}:`, error)
      const errorMessage = error instanceof Error ? error.message : String(error)
      return {
        success: false,
        error: errorMessage || 'Failed to get file content',
        content: ''
      }
    }
  })

  // Handler to get the form.json file from a tracker folder
  ipcMain.handle(Channels.GetTrackerForm, async (_, trackerId: string) => {
    try {
      logger.info(`Getting form.json for tracker with ID: ${trackerId}`)

      if (!trackerId) {
        return {
          success: false,
          error: 'No tracker ID provided'
        }
      }

      // Get available app folders to map trackerId to folder name
      const appFolders = await getAvailableAppFolders()

      // Find matching folder for this trackerId or use first available
      const folderMatch = appFolders.find((folder) => folder === trackerId)

      if (!folderMatch) {
        logger.warn(`No matching folder found for tracker ID: ${trackerId}`)
        return {
          success: false,
          error: 'No matching tracker folder found'
        }
      }

      // Get the path to the tracker's folder using the matched folder name
      if (!syftboxConfig || !syftboxConfig.data_dir) {
        return {
          success: false,
          error: 'SyftBox configuration is not available or missing data_dir'
        }
      }
      const trackerDir = pathResolve(join(syftboxConfig.data_dir, 'apps', folderMatch))
      const formPath = join(trackerDir, 'form.json')

      logger.info(`Looking for form.json at: ${formPath}`)

      if (fs.existsSync(formPath)) {
        const formContent = fs.readFileSync(formPath, 'utf8')
        const formData = JSON.parse(formContent)

        logger.info(`Successfully loaded form for tracker ${folderMatch}`)

        return {
          success: true,
          form: formData
        }
      } else {
        logger.warn(`No form.json found for tracker ${folderMatch} at ${formPath}`)

        return {
          success: false,
          error: 'No form.json found for this tracker'
        }
      }
    } catch (error) {
      logger.error(`Error loading form for tracker ${trackerId}:`, error)
      const errorMessage = error instanceof Error ? error.message : String(error)

      return {
        success: false,
        error: `Failed to load form: ${errorMessage || 'Unknown error'}`
      }
    }
  })

  // Handler to get the current tracker configuration
  ipcMain.handle(Channels.GetTrackerConfig, async (_, trackerId: string) => {
    try {
      logger.info(`Getting config for tracker with ID: ${trackerId}`)

      if (!trackerId) {
        return {
          success: false,
          error: 'No tracker ID provided'
        }
      }

      // Get available app folders to map trackerId to folder name
      const appFolders = await getAvailableAppFolders()

      // Find matching folder for this trackerId or use first available
      const folderMatch = appFolders.find((folder) => folder === trackerId)

      if (!folderMatch) {
        logger.warn(`No matching folder found for tracker ID: ${trackerId}`)
        return {
          success: false,
          error: 'No matching tracker folder found'
        }
      }

      // Get the path to the tracker's folder using the matched folder name
      if (!syftboxConfig || !syftboxConfig.data_dir) {
        return {
          success: false,
          error: 'SyftBox configuration is not available or missing data_dir'
        }
      }
      const trackerDir = pathResolve(join(syftboxConfig.data_dir, 'apps', folderMatch))
      const configPath = join(trackerDir, 'config.json')

      logger.info(`Looking for config.json at: ${configPath}`)

      if (fs.existsSync(configPath)) {
        const configContent = fs.readFileSync(configPath, 'utf8')
        const configData = JSON.parse(configContent)

        logger.info(`Successfully loaded config for tracker ${folderMatch}`)

        return {
          success: true,
          config: configData
        }
      } else {
        logger.warn(`No config.json found for tracker ${folderMatch} at ${configPath}`)

        // If no config.json exists yet, return an empty config
        return {
          success: true,
          config: {}
        }
      }
    } catch (error) {
      logger.error(`Error loading config for tracker ${trackerId}:`, error)
      const errorMessage = error instanceof Error ? error.message : String(error)

      return {
        success: false,
        error: `Failed to load config: ${errorMessage || 'Unknown error'}`
      }
    }
  })

  // Handler to save the tracker configuration
  ipcMain.handle(
    Channels.SaveTrackerConfig,
    async (_, trackerId: string, configData: Record<string, any>) => {
      try {
        logger.info(`Saving config for tracker with ID: ${trackerId}`)

        if (!trackerId) {
          return {
            success: false,
            error: 'No tracker ID provided'
          }
        }

        // Get available app folders to map trackerId to folder name
        const appFolders = await getAvailableAppFolders()
        logger.info(`Available app folders: ${JSON.stringify(appFolders)}`)

        // Find matching folder for this trackerId or use first available
        const folderMatch = appFolders.find((folder) => folder === trackerId)

        if (!folderMatch) {
          logger.warn(`No matching folder found for tracker ID: ${trackerId}`)
          return {
            success: false,
            error: 'No matching tracker folder found'
          }
        }

        // Get the path to the tracker's folder using the matched folder name
        if (!syftboxConfig || !syftboxConfig.data_dir) {
          return {
            success: false,
            error: 'SyftBox configuration is not available or missing data_dir'
          }
        }
        const trackerDir = pathResolve(join(syftboxConfig.data_dir, 'apps', folderMatch))
        const configPath = join(trackerDir, 'config.json')

        // Ensure directory exists
        if (!fs.existsSync(trackerDir)) {
          logger.error(`Tracker directory does not exist: ${trackerDir}`)
          return {
            success: false,
            error: 'Tracker directory does not exist'
          }
        }

        // Write the updated config to the file
        fs.writeFileSync(configPath, JSON.stringify(configData, null, 2), 'utf8')

        logger.info(`Successfully saved config for tracker ${folderMatch}`)

        return {
          success: true,
          message: 'Configuration saved successfully'
        }
      } catch (error) {
        logger.error(`Error saving config for tracker ${trackerId}:`, error)
        const errorMessage = error instanceof Error ? error.message : String(error)

        return {
          success: false,
          error: `Failed to save config: ${errorMessage || 'Unknown error'}`
        }
      }
    }
  )

  // Handler to get the app-level config.json
  ipcMain.handle(Channels.GetAppConfig, async () => {
    try {
      logger.info('Getting app-level config.json')

      // Determine the app's main config file location
      const configPath = getConfigFilePath()
      const appDir = app.getAppPath()
      // For backward compatibility, also check the old location
      const defaultConfigPath = pathResolve(appDir, 'config.json')

      logger.info(`Looking for config.json at: ${configPath} or ${defaultConfigPath}`)

      let configData = {}

      // First try to read from the user data directory
      if (fs.existsSync(configPath)) {
        const configContent = fs.readFileSync(configPath, 'utf8')
        configData = JSON.parse(configContent)
        logger.info('Successfully loaded config.json from user data directory')
      }
      // If not found, try to read from the app directory
      else if (fs.existsSync(defaultConfigPath)) {
        const configContent = fs.readFileSync(defaultConfigPath, 'utf8')
        configData = JSON.parse(configContent)
        logger.info('Successfully loaded config.json from app directory')
      } else {
        logger.warn('No config.json found in user data or app directory')
      }

      return {
        success: true,
        config: configData
      }
    } catch (error) {
      logger.error('Error loading app-level config.json:', error)
      const errorMessage = error instanceof Error ? error.message : String(error)

      return {
        success: false,
        error: `Failed to load app config: ${errorMessage || 'Unknown error'}`
      }
    }
  })

  // Handler to update the app-level config.json with form values
  ipcMain.handle(Channels.UpdateAppConfig, async (_, formValues: Record<string, unknown>) => {
    try {
      logger.info('Updating app-level config.json with form values')

      if (!formValues || Object.keys(formValues).length === 0) {
        logger.warn('No form values provided to update app config')
        return {
          success: false,
          error: 'No form values provided'
        }
      }

      // Determine the app's main config file location
      const configPath = getConfigFilePath()
      const appDir = app.getAppPath()
      // For backward compatibility, also check the old location
      const defaultConfigPath = pathResolve(appDir, 'config.json')

      // Determine which config file to update
      let targetConfigPath = ''
      let currentConfig = {}

      // First try to update the user data directory config
      if (fs.existsSync(configPath)) {
        targetConfigPath = configPath
        const configContent = fs.readFileSync(configPath, 'utf8')
        currentConfig = JSON.parse(configContent)
        logger.info('Using config.json from user data directory')
      }
      // If not found, try to update the app directory config
      else if (fs.existsSync(defaultConfigPath)) {
        targetConfigPath = defaultConfigPath
        const configContent = fs.readFileSync(defaultConfigPath, 'utf8')
        currentConfig = JSON.parse(configContent)
        logger.info('Using config.json from app directory')
      } else {
        // If no config exists, create one in the user data directory
        targetConfigPath = configPath
        logger.info('No existing config.json found, will create new one in user data directory')

        // Ensure the directory exists
        const configDir = pathResolve(app.getPath('userData'), '../dk')
        if (!fs.existsSync(configDir)) {
          fs.mkdirSync(configDir, { recursive: true })
        }
      }

      // Update the current config with form values
      const updatedConfig: Record<string, unknown> = { ...currentConfig }

      // Merge the form values into the current config
      Object.keys(formValues).forEach((key) => {
        updatedConfig[key] = formValues[key]
      })

      // Write the updated config to the file
      fs.writeFileSync(targetConfigPath, JSON.stringify(updatedConfig, null, 2), 'utf8')

      logger.info(`Successfully updated app config at ${targetConfigPath}`)

      return {
        success: true,
        message: 'App configuration updated successfully'
      }
    } catch (error) {
      logger.error('Error updating app-level config.json:', error)

      const errorMessage = error instanceof Error ? error.message : String(error)
      return {
        success: false,
        error: `Failed to update app config: ${errorMessage}`
      }
    }
  })

  // Handler to upload file for tracker configuration
  ipcMain.handle(
    Channels.UploadTrackerFile,
    async (_, trackerId: string, filePath: string, variableId: string) => {
      try {
        logger.info(`Uploading file for tracker with ID: ${trackerId}, variable: ${variableId}`)

        if (!trackerId || !filePath || !variableId) {
          logger.warn('Missing required parameters for file upload')
          return {
            success: false,
            error: 'Missing required parameters'
          }
        }

        // Get available app folders to map trackerId to folder name
        const appFolders = await getAvailableAppFolders()
        logger.info(`Available app folders: ${JSON.stringify(appFolders)}`)

        // Find matching folder for this trackerId
        const folderMatch = appFolders.find((folder) => folder === trackerId)

        if (!folderMatch) {
          logger.warn(`No matching folder found for tracker ID: ${trackerId}`)
          return {
            success: false,
            error: 'No matching tracker folder found'
          }
        }

        // Make sure the source file exists
        if (!fs.existsSync(filePath)) {
          logger.error(`Source file does not exist: ${filePath}`)
          return {
            success: false,
            error: 'Source file does not exist'
          }
        }

        // Get the file name from the path
        const fileName = basename(filePath)

        // Use syftboxConfig.data_dir + "/apps" + /<tracker_app_name> path as requested
        let filesDir
        if (syftboxConfig && syftboxConfig.data_dir) {
          // Use the data_dir + "/apps" + /tracker_app_name from syftbox configuration
          filesDir = join(syftboxConfig.data_dir, 'apps', trackerId)
        } else {
          // Fallback to app directory if syftboxConfig is not available
          const appDir = app.getAppPath()
          filesDir = pathResolve(appDir, 'files')
          logger.warn('syftboxConfig.data_dir not available, using app directory for file storage')
        }

        // Create files directory if it doesn't exist
        if (!fs.existsSync(filesDir)) {
          fs.mkdirSync(filesDir, { recursive: true })
        }

        // Generate a unique filename to avoid conflicts
        const fileExt = extname(filePath)
        const fileNameWithoutExt = basename(filePath, fileExt)
        const uniqueFileName = `${fileNameWithoutExt}_${Date.now()}${fileExt}`
        const destinationPath = join(filesDir, uniqueFileName)

        // Copy the file to the destination
        fs.copyFileSync(filePath, destinationPath)
        logger.info(`Successfully copied file to: ${destinationPath}`)

        // Create a path for storing in config
        let relativePath
        if (syftboxConfig && syftboxConfig.data_dir) {
          // For syftboxConfig paths, store the full path to ensure consistency
          relativePath = join(filesDir, uniqueFileName)
        } else {
          // For app directory paths, use relative path
          relativePath = join('files', trackerId, uniqueFileName)
        }

        // Update the app-level config.json with the file path
        const configPath = getConfigFilePath()
        const appDir = app.getAppPath() // Ensure appDir is defined for defaultConfigPath
        // For backward compatibility, also check the old location
        const defaultConfigPath = pathResolve(appDir, 'config.json')

        let targetConfigPath = ''
        let currentConfig = {}

        // First try to update the user data directory config
        if (fs.existsSync(configPath)) {
          targetConfigPath = configPath
          const configContent = fs.readFileSync(configPath, 'utf8')
          currentConfig = JSON.parse(configContent)
        }
        // If not found, try to update the app directory config
        else if (fs.existsSync(defaultConfigPath)) {
          targetConfigPath = defaultConfigPath
          const configContent = fs.readFileSync(defaultConfigPath, 'utf8')
          currentConfig = JSON.parse(configContent)
        } else {
          // If no config exists, create one in the user data directory
          targetConfigPath = configPath

          // Ensure the directory exists
          const configDir = pathResolve(app.getPath('userData'), '../dk')
          if (!fs.existsSync(configDir)) {
            fs.mkdirSync(configDir, { recursive: true })
          }
        }

        // Update the configuration with the file path
        const updatedConfig: Record<string, unknown> = { ...currentConfig }
        updatedConfig[variableId] = relativePath

        // Write the updated config to the file
        fs.writeFileSync(targetConfigPath, JSON.stringify(updatedConfig, null, 2), 'utf8')
        logger.info(`Updated config at ${targetConfigPath} with file path ${relativePath}`)

        return {
          success: true,
          filePath: relativePath,
          fileName: uniqueFileName,
          message: 'File uploaded successfully'
        }
      } catch (error) {
        logger.error(`Error uploading file for tracker ${trackerId}:`, error)
        const errorMessage = error instanceof Error ? error.message : String(error)
        return {
          success: false,
          error: `Failed to upload file: ${errorMessage || 'Unknown error'}`
        }
      }
    }
  )

  // Handler to show native file dialog
  ipcMain.handle(
    Channels.ShowFileDialog,
    async (event, trackerId: string, variableId: string, options?: { extensions?: string[] }) => {
      try {
        logger.info(
          `Opening file dialog for tracker with ID: ${trackerId}, variable: ${variableId}`
        )

        // Get source window
        const sourceWindow = BrowserWindow.fromWebContents(event.sender)
        if (!sourceWindow) {
          logger.error('Could not determine source window for file dialog')
          return {
            success: false,
            error: 'Could not determine source window'
          }
        }

        // Configure dialog options
        const dialogOptions: Electron.OpenDialogOptions = {
          properties: ['openFile'] as (
            | 'openFile'
            | 'openDirectory'
            | 'multiSelections'
            | 'showHiddenFiles'
            | 'createDirectory'
            | 'promptToCreate'
            | 'noResolveAliases'
            | 'treatPackageAsDirectory'
            | 'dontAddToRecent'
          )[],
          filters: [] // Initialize as empty array
        }

        // Ensure filters is defined
        if (!dialogOptions.filters) {
          dialogOptions.filters = []
        }

        // Add file filters if extensions were provided
        if (options?.extensions && options.extensions.length > 0) {
          // Convert extensions to dialog filter format
          // Group by type (e.g., ['jpg', 'png'] -> 'Images (*.jpg, *.png)')
          const extensionMap = {
            jpg: 'Images',
            jpeg: 'Images',
            png: 'Images',
            gif: 'Images',
            bmp: 'Images',
            svg: 'Images',

            mp3: 'Audio',
            wav: 'Audio',
            ogg: 'Audio',

            mp4: 'Video',
            avi: 'Video',
            mov: 'Video',

            json: 'Data Files',
            csv: 'Data Files',
            xml: 'Data Files',

            txt: 'Text Files',
            md: 'Text Files',

            pdf: 'Documents',
            doc: 'Documents',
            docx: 'Documents',

            // Default group for other types
            other: 'Other Files'
          }

          // Group extensions by type
          const extensionsByType: Record<string, string[]> = {}

          options.extensions.forEach((ext) => {
            // Remove the dot if present
            const cleanExt = ext.startsWith('.') ? ext.substring(1) : ext
            // Get the type or use 'other' as fallback
            const type =
              extensionMap[cleanExt.toLowerCase() as keyof typeof extensionMap] ||
              extensionMap.other

            if (!extensionsByType[type]) {
              extensionsByType[type] = []
            }
            extensionsByType[type].push(cleanExt)
          })

          // Convert grouped extensions to dialog filters
          for (const [type, exts] of Object.entries(extensionsByType)) {
            dialogOptions.filters.push({
              name: `${type} (*.${exts.join(', *.')})`,
              extensions: exts
            })
          }

          // Always add "All Files" option
          dialogOptions.filters.push({ name: 'All Files', extensions: ['*'] })
        }

        // Show the dialog
        const result = await dialog.showOpenDialog(sourceWindow, dialogOptions)

        if (result.canceled || result.filePaths.length === 0) {
          logger.info('File dialog canceled or no file selected')
          return {
            success: false,
            canceled: true
          }
        }

        const selectedFilePath = result.filePaths[0]
        logger.info(`File selected: ${selectedFilePath}`)

        // Now process the selected file directly (using folderName instead of trackerId)
        // Note: we're now using the dataPath directly rather than looking it up from trackerId
        return await uploadFile(trackerId, selectedFilePath, variableId)
      } catch (error) {
        logger.error(`Error showing file dialog for tracker ${trackerId}:`, error)
        const errorMessage = error instanceof Error ? error.message : String(error)
        return {
          success: false,
          error: `Failed to show file dialog: ${errorMessage}`
        }
      }
    }
  )

  // Helper function to upload a file
  async function uploadFile(trackerId: string, filePath: string, variableId: string) {
    try {
      logger.info(`Uploading file for tracker with ID: ${trackerId}, variable: ${variableId}`)

      if (!trackerId || !filePath || !variableId) {
        logger.warn('Missing required parameters for file upload')
        return {
          success: false,
          error: 'Missing required parameters'
        }
      }

      // Get available app folders to map trackerId to folder name
      const appFolders = await getAvailableAppFolders()

      // Find matching folder for this trackerId
      const folderMatch = appFolders.find((folder) => folder === trackerId)

      if (!folderMatch) {
        logger.warn(`No matching folder found for tracker ID: ${trackerId}`)
        return {
          success: false,
          error: 'No matching tracker folder found'
        }
      }

      // Make sure the source file exists
      if (!fs.existsSync(filePath)) {
        logger.error(`Source file does not exist: ${filePath}`)
        return {
          success: false,
          error: 'Source file does not exist'
        }
      }

      // Get the file name from the path
      const fileName = basename(filePath)

      // Use syftboxConfig.data_dir + "/apps" + /<tracker_app_name> path as requested
      let filesDir
      if (syftboxConfig && syftboxConfig.data_dir) {
        // Use the data_dir + "/apps" + /tracker_app_name from syftbox configuration
        filesDir = join(syftboxConfig.data_dir, 'apps', trackerId)
      } else {
        // Fallback to app directory if syftboxConfig is not available
        const appDir = app.getAppPath()
        filesDir = pathResolve(appDir, 'files')
        logger.warn('syftboxConfig.data_dir not available, using app directory for file storage')
      }

      // Create files directory if it doesn't exist
      if (!fs.existsSync(filesDir)) {
        fs.mkdirSync(filesDir, { recursive: true })
      }

      // Generate a unique filename to avoid conflicts
      const fileExt = extname(filePath)
      const fileNameWithoutExt = basename(filePath, fileExt)
      const uniqueFileName = `${fileNameWithoutExt}_${Date.now()}${fileExt}`
      const destinationPath = join(filesDir, uniqueFileName)

      // Copy the file to the destination
      fs.copyFileSync(filePath, destinationPath)
      logger.info(`Successfully copied file to: ${destinationPath}`)

      // Create a path for storing in config
      let relativePath
      if (syftboxConfig && syftboxConfig.data_dir) {
        // For syftboxConfig paths, store the full path to ensure consistency
        relativePath = join(filesDir, uniqueFileName)
      } else {
        // For app directory paths, use relative path
        relativePath = join('files', trackerId, uniqueFileName)
      }

      // Update the app-level config.json with the file path
      const configPath = getConfigFilePath()
      const appDir = app.getAppPath() // Ensure appDir is defined for defaultConfigPath
      // For backward compatibility, also check the old location
      const defaultConfigPath = pathResolve(appDir, 'config.json')

      let targetConfigPath = ''
      let currentConfig = {}

      // First try to update the user data directory config
      if (fs.existsSync(configPath)) {
        targetConfigPath = configPath
        const configContent = fs.readFileSync(configPath, 'utf8')
        currentConfig = JSON.parse(configContent)
      }
      // If not found, try to update the app directory config
      else if (fs.existsSync(defaultConfigPath)) {
        targetConfigPath = defaultConfigPath
        const configContent = fs.readFileSync(defaultConfigPath, 'utf8')
        currentConfig = JSON.parse(configContent)
      } else {
        // If no config exists, create one in the user data directory
        targetConfigPath = configPath

        // Ensure the directory exists
        const configDir = pathResolve(app.getPath('userData'), '../dk')
        if (!fs.existsSync(configDir)) {
          fs.mkdirSync(configDir, { recursive: true })
        }
      }

      // Update the configuration with the file path
      const updatedConfig: Record<string, unknown> = { ...currentConfig }
      updatedConfig[variableId] = relativePath

      // Write the updated config to the file
      fs.writeFileSync(targetConfigPath, JSON.stringify(updatedConfig, null, 2), 'utf8')
      logger.info(`Updated config at ${targetConfigPath} with file path ${relativePath}`)

      return {
        success: true,
        filePath: relativePath,
        fileName: uniqueFileName,
        message: 'File uploaded successfully'
      }
    } catch (error) {
      logger.error(`Error uploading file for tracker ${trackerId}:`, error)
      const errorMessage = error instanceof Error ? error.message : String(error)
      return {
        success: false,
        error: `Failed to upload file: ${errorMessage || 'Unknown error'}`
      }
    }
  }
}
