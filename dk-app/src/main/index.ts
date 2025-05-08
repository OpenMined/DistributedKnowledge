import { app, BrowserWindow } from 'electron'
import { electronApp, optimizer } from '@electron-toolkit/utils'
import { createMainWindow } from './windows/mainWindow'
import { setupIpcHandlers, setWsClient } from './ipc/handlers'
import { initializeClient, disconnectClient } from './services/clientService'
import { closeDatabase } from './services/database'
import { initializeServices } from './services'
import logger from '../shared/logging'
import { spawn, ChildProcess, exec } from 'child_process'
import {
  appConfig,
  getOnboardingStatus,
  setOnboardingFirstRun,
  getConfigFilePath
} from './services/config'
import path from 'path'
import { existsSync, mkdirSync, readFileSync, unlinkSync } from 'fs'
import { join, dirname } from 'path'
import { promisify } from 'util'
import { chmod } from 'fs/promises'
import https from 'https'
import fs from 'fs'

// Variables to hold child process references
let childProcess: ChildProcess | null = null
let syftboxProcess: ChildProcess | null = null

// Function to determine current OS platform
function getOSPlatform(): string {
  const platform = process.platform

  if (platform === 'win32') {
    return 'windows'
  } else if (platform === 'darwin') {
    return 'mac'
  } else if (platform === 'linux') {
    return 'linux'
  } else {
    logger.error(`Unsupported platform: ${platform}`)
    return 'unsupported'
  }
}

// Function to ensure directory exists
function ensureDirectoryExists(filePath: string): void {
  const directory = dirname(filePath)
  if (!existsSync(directory)) {
    logger.info(`Creating directory: ${directory}`)
    mkdirSync(directory, { recursive: true })
  }
}

// Function to download the DK binary
async function downloadDKBinary(platform: string, targetPath: string): Promise<boolean> {
  return new Promise<boolean>((resolve) => {
    // Ensure directory exists
    ensureDirectoryExists(targetPath)

    const downloadUrl = `https://distributedknowledge.org/download/${platform}`
    logger.info(`Downloading DK binary from ${downloadUrl} to ${targetPath}`)

    // Create write stream
    const fileStream = fs.createWriteStream(targetPath)

    // Download file
    const request = https.get(downloadUrl, (response) => {
      if (response.statusCode !== 200) {
        logger.error(`Failed to download DK binary: HTTP ${response.statusCode}`)
        fileStream.close()
        fs.unlink(targetPath, () => {})
        resolve(false)
        return
      }

      response.pipe(fileStream)

      fileStream.on('finish', async () => {
        fileStream.close()

        try {
          // Make binary executable
          await chmod(targetPath, 0o755)
          logger.info(`Successfully downloaded DK binary to ${targetPath} and made it executable`)
          resolve(true)
        } catch (error) {
          logger.error(`Failed to make DK binary executable: ${error}`)
          resolve(false)
        }
      })
    })

    request.on('error', (error) => {
      logger.error(`Error downloading DK binary: ${error}`)
      fileStream.close()
      fs.unlink(targetPath, () => {})
      resolve(false)
    })

    fileStream.on('error', (error) => {
      logger.error(`Error writing DK binary to disk: ${error}`)
      fileStream.close()
      fs.unlink(targetPath, () => {})
      resolve(false)
    })
  })
}

// Function to check and download DK binary if needed
async function ensureDKBinaryExists(dkPath: string): Promise<boolean> {
  // Check if binary exists
  if (existsSync(dkPath)) {
    logger.info(`DK binary found at ${dkPath}`)
    return true
  }

  logger.info(`DK binary not found at ${dkPath}, downloading...`)

  // Get current platform
  const platform = getOSPlatform()
  if (platform === 'unsupported') {
    logger.error('Cannot download DK binary for unsupported platform')
    return false
  }

  // Download binary
  return await downloadDKBinary(platform, dkPath)
}

// Function to start external processes
export async function startExternalProcesses(): Promise<void> {
  // Start the DK binary if dk_config is available
  if (appConfig.dk_config) {
    try {
      // Don't start if already running
      if (childProcess) {
        return
      }

      const { dk_config } = appConfig

      // Resolve paths if they're not absolute
      const dkPath = path.isAbsolute(dk_config.dk)
        ? dk_config.dk
        : path.resolve(app.getAppPath(), dk_config.dk)

      const projectPath = path.isAbsolute(dk_config.project_path)
        ? dk_config.project_path
        : path.resolve(app.getAppPath(), dk_config.project_path)

      // Check if DK binary exists and download if needed
      const binaryExists = await ensureDKBinaryExists(dkPath)
      if (!binaryExists) {
        logger.error('DK binary could not be found or downloaded. DK process will not start.')
        return
      }

      // Build command arguments
      const args = [
        '-userId',
        appConfig.userID,
        '-private',
        appConfig.private_key || '',
        '-public',
        appConfig.public_key || '',
        '-project_path',
        projectPath,
        '-server',
        appConfig.serverURL,
        '-http_port',
        dk_config.http_port
      ]

      // Add syftbox_config if available
      if (appConfig.syftbox_config) {
        const syftboxConfigPath = path.isAbsolute(appConfig.syftbox_config)
          ? appConfig.syftbox_config
          : path.resolve(app.getAppPath(), appConfig.syftbox_config)

        args.push('-syftbox_config', syftboxConfigPath)
      }

      logger.info(`Starting DK binary: ${dkPath} with args:`, args)

      // Spawn the process
      childProcess = spawn(dkPath, args, {
        stdio: 'pipe',
        detached: false
      })

      logger.info('DK binary started with PID:', childProcess.pid)

      childProcess.stdout?.on('data', (data) => {
        logger.info(`DK binary stdout: ${data}`)
      })

      childProcess.stderr?.on('data', (data) => {
        logger.error(`DK binary stderr: ${data}`)
      })

      childProcess.on('error', (error) => {
        logger.error(`Failed to start DK binary: ${error}`)
        childProcess = null
      })

      childProcess.on('close', (code) => {
        logger.info(`DK binary exited with code ${code}`)
        childProcess = null
      })
    } catch (error) {
      logger.error('Failed to spawn DK binary:', error)
    }
  } else {
    logger.warn('No dk_config found, DK binary not started')
  }

  // Start the syftbox executable
  try {
    // Don't start if already running
    if (syftboxProcess) {
      return
    }

    const syftboxPath = '/home/ubuntu/.local/bin/syftbox'

    // Verify the syftbox binary exists before attempting to start it
    if (!existsSync(syftboxPath)) {
      logger.error(`SyftBox binary not found at path: ${syftboxPath}`)
      return
    }

    logger.info(`Starting syftbox binary: ${syftboxPath}`)

    // Spawn the syftbox process without arguments
    syftboxProcess = spawn(syftboxPath, [], {
      stdio: 'pipe',
      detached: false
    })

    logger.info('Syftbox binary started with PID:', syftboxProcess.pid)

    syftboxProcess.stdout?.on('data', (data) => {
      logger.info(`Syftbox binary stdout: ${data}`)
    })

    syftboxProcess.stderr?.on('data', (data) => {
      logger.error(`Syftbox binary stderr: ${data}`)
    })

    syftboxProcess.on('error', (error) => {
      logger.error(`Failed to start syftbox binary: ${error}`)
      syftboxProcess = null
    })

    syftboxProcess.on('close', (code) => {
      logger.info(`Syftbox binary exited with code ${code}`)
      syftboxProcess = null
    })
  } catch (error) {
    logger.error('Failed to spawn syftbox binary:', error)
  }
}

// Initialize all services in the correct order
logger.info('Application starting up...')

// Check if config.json exists before initializing services
const configPath = getConfigFilePath()
let configExists = existsSync(configPath)

// If config exists, check if it's a valid config or just a default one
if (configExists) {
  try {
    // Read the config file
    const configData = JSON.parse(readFileSync(configPath, 'utf8'))

    // Check if it has the default or empty userID
    if (!configData.userID || configData.userID === 'default-user' || configData.userID === '') {
      logger.info(
        `Found config with invalid/default userID "${configData.userID}" - removing this file`
      )

      // Delete the invalid config file
      unlinkSync(configPath)

      // Update existence flag
      configExists = false

      logger.info(`Removed invalid config file. App will now show onboarding.`)
    }
  } catch (error) {
    logger.error(`Error checking config validity:`, error)
  }
}

logger.info(`Valid config file exists at ${configPath}: ${configExists}`)

// Initialize services
initializeServices()

// Get onboarding status after services are initialized
const onboardingStatus = getOnboardingStatus()

// Always force onboarding if config file doesn't exist
if (!configExists) {
  // Make sure isFirstRun is set to true if config doesn't exist
  setOnboardingFirstRun(true)

  // Reset the completed status to ensure wizard is shown
  onboardingStatus.completed = false

  // Make sure these changes persist in memory across the entire process
  logger.info('Config file not found, forcing onboarding wizard')
  logger.info(
    `Updated onboarding status: isFirstRun=${onboardingStatus.isFirstRun}, completed=${onboardingStatus.completed}`
  )

  // Log a clear marker for debugging
  logger.info('ONBOARDING SHOULD BE DISPLAYED - NO CONFIG FILE EXISTS')
}

// This method will be called when Electron has finished
// initialization and is ready to create browser windows.
// Some APIs can only be used after this event occurs.
app.whenReady().then(async () => {
  // Set app user model id for windows
  electronApp.setAppUserModelId('com.electron')

  // Default open or close DevTools by F12 in development
  // and ignore CommandOrControl + R in production.
  // see https://github.com/alex8088/electron-toolkit/tree/master/packages/utils
  app.on('browser-window-created', (_, window) => {
    optimizer.watchWindowShortcuts(window)
  })

  // Initialize websocket client
  const client = await initializeClient()

  // Pass client to IPC handlers
  setWsClient(client)

  // Set up IPC handlers
  setupIpcHandlers()

  // Create main window
  createMainWindow()
  logger.info('Main window created')

  // Start the external programs only if config exists and onboarding is completed
  // Only start external processes when:
  // 1. Config file exists AND
  // 2. UserID is not empty or the default (to ensure it's a valid config) AND
  // 3. Onboarding is not in first run state
  const isValidConfig = configExists && appConfig.userID && appConfig.userID !== 'default-user'
  const shouldStartProcesses = isValidConfig && !onboardingStatus.isFirstRun

  logger.info(
    `Config exists: ${configExists}, Onboarding needed: ${onboardingStatus.isFirstRun}, Completed: ${onboardingStatus.completed}`
  )

  if (shouldStartProcesses) {
    // Start processes asynchronously
    startExternalProcesses().catch((error) => {
      logger.error('Failed to start external processes:', error)
    })
  } else {
    logger.info('Config not set up or onboarding not completed. External processes will not start.')
  }

  app.on('activate', function () {
    // On macOS it's common to re-create a window in the app when the
    // dock icon is clicked and there are no other windows open.
    if (BrowserWindow.getAllWindows().length === 0) createMainWindow()
  })
})

// Quit when all windows are closed, except on macOS. There, it's common
// for applications and their menu bar to stay active until the user quits
// explicitly with Cmd + Q.
app.on('window-all-closed', () => {
  // Disconnect the client if it exists
  disconnectClient()

  if (process.platform !== 'darwin') {
    app.quit()
  }
})

// Close the database and perform cleanup when quitting
app.on('will-quit', () => {
  logger.info('Application shutting down, cleaning up resources')

  // Terminate the DK process if it exists
  if (childProcess && childProcess.pid) {
    logger.info('Terminating DK binary with PID:', childProcess.pid)

    // Kill process and all of its children
    try {
      process.kill(-childProcess.pid, 'SIGTERM')
    } catch (error) {
      // If the process group kill fails, try killing just the process
      try {
        childProcess.kill('SIGTERM')
      } catch (innerError) {
        logger.error('Failed to terminate DK binary:', innerError)
      }
    }

    childProcess = null
  }

  // Terminate the syftbox process if it exists
  if (syftboxProcess && syftboxProcess.pid) {
    logger.info('Terminating syftbox binary with PID:', syftboxProcess.pid)

    // Kill process and all of its children
    try {
      process.kill(-syftboxProcess.pid, 'SIGTERM')
    } catch (error) {
      // If the process group kill fails, try killing just the process
      try {
        syftboxProcess.kill('SIGTERM')
      } catch (innerError) {
        logger.error('Failed to terminate syftbox binary:', innerError)
      }
    }

    syftboxProcess = null
  }

  closeDatabase()
})

// In this file you can include the rest of your app's specific main process
// code. You can also put them in separate files and require them here.
