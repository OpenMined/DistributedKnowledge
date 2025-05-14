import { app, BrowserWindow } from 'electron'
import { electronApp, optimizer, is } from '@electron-toolkit/utils'
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
import os from 'os'
import { getAppPaths } from './getAppPaths'

// Variables to hold child process references
let childProcess: ChildProcess | null = null
let syftboxProcess: ChildProcess | null = null

// Function to determine current OS platform
function getOSPlatform(): string {
  const platform = process.platform

  if (platform === 'win32') {
    return 'windows'
  } else if (platform === 'darwin') {
    return 'darwin'
  } else if (platform === 'linux') {
    return 'linux'
  } else {
    logger.error(`Unsupported platform: ${platform}`)
    return 'unsupported'
  }
}

// Function to determine CPU architecture
function getCPUArchitecture(): string {
  const arch = process.arch

  if (arch === 'x64') {
    return 'amd64'
  } else if (arch === 'arm64') {
    return 'arm64'
  } else {
    logger.error(`Unsupported architecture: ${arch}`)
    return 'unsupported'
  }
}

// Function to ensure directory exists
function ensureDirectoryExists(filePath: string): void {
  const directory = dirname(filePath)
  if (!existsSync(directory)) {
    logger.debug(`Creating directory: ${directory}`)
    mkdirSync(directory, { recursive: true })
  }
}

// Function to download the DK binary
async function downloadDKBinary(platform: string, targetPath: string): Promise<boolean> {
  return new Promise<boolean>((resolve) => {
    // Ensure directory exists
    ensureDirectoryExists(targetPath)

    const downloadUrl = `https://distributedknowledge.org/download/${platform}`
    logger.debug(`Downloading DK binary from ${downloadUrl} to ${targetPath}`)

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
          logger.debug(`Successfully downloaded DK binary to ${targetPath} and made it executable`)
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

// Function to download the SyftBox binary
async function downloadSyftBoxBinary(
  os: string,
  arch: string,
  targetPath: string
): Promise<boolean> {
  return new Promise<boolean>((resolve) => {
    const targetDir = path.dirname(targetPath)

    // Ensure directory exists
    ensureDirectoryExists(targetPath)

    const downloadUrl = `https://syftboxdev.openmined.org/releases/syftbox_client_${os}_${arch}.tar.gz`
    logger.debug(`Downloading SyftBox binary from ${downloadUrl} to ${targetPath}`)

    // Prepare the tarball filename and path
    const tarName = `syftbox_client_${os}_${arch}.tar.gz`
    const tarPath = path.join(targetDir, tarName)

    // Create write stream
    const fileStream = fs.createWriteStream(tarPath)

    // Download file
    const request = https.get(downloadUrl, (response) => {
      if (response.statusCode !== 200) {
        logger.error(`Failed to download SyftBox binary: HTTP ${response.statusCode}`)
        logger.error(`Download URL used: ${downloadUrl}`)
        fileStream.close()
        fs.unlink(tarPath, () => {})
        resolve(false)
        return
      }

      response.pipe(fileStream)

      fileStream.on('finish', async () => {
        fileStream.close()

        try {
          // Extract the tarball
          logger.debug(`Extracting SyftBox binary from ${tarPath} to ${targetDir}`)
          const extractCommand = `tar -xzf "${tarPath}" -C "${targetDir}"`

          // Use promisify to create a promise-based version of exec
          const execPromise = promisify(exec)

          await execPromise(extractCommand)

          // The extraction creates a directory like syftbox_client_linux_amd64/
          // with the binary inside it
          const extractedDir = path.join(targetDir, `syftbox_client_${os}_${arch}`)
          const extractedBinary = path.join(extractedDir, 'syftbox')

          logger.debug(`Looking for extracted binary at ${extractedBinary}`)

          if (!existsSync(extractedBinary)) {
            logger.error(`Extracted binary not found at expected path: ${extractedBinary}`)
            resolve(false)
            return
          }

          // Move the extracted binary to the target path
          logger.debug(`Moving binary from ${extractedBinary} to ${targetPath}`)

          // If target already exists, remove it first
          if (existsSync(targetPath)) {
            fs.unlinkSync(targetPath)
          }

          // Copy the binary to its final destination
          fs.copyFileSync(extractedBinary, targetPath)

          // Make the binary executable
          await chmod(targetPath, 0o755)

          // Clean up the extracted directory and tarball
          logger.debug(`Cleaning up temporary files in ${extractedDir}`)
          fs.unlinkSync(tarPath)

          // Remove the entire extracted directory
          const rmCommand = `rm -rf "${extractedDir}"`
          await execPromise(rmCommand)

          logger.debug(
            `Successfully downloaded and extracted SyftBox binary to ${targetPath} and made it executable`
          )
          resolve(true)
        } catch (error) {
          logger.error(`Failed to extract or make SyftBox binary executable: ${error}`)
          resolve(false)
        }
      })
    })

    request.on('error', (error) => {
      logger.error(`Error downloading SyftBox binary: ${error}`)
      fileStream.close()
      fs.unlink(tarPath, () => {})
      resolve(false)
    })

    fileStream.on('error', (error) => {
      logger.error(`Error writing SyftBox binary to disk: ${error}`)
      fileStream.close()
      fs.unlink(tarPath, () => {})
      resolve(false)
    })
  })
}

// Function to check and download DK binary if needed
async function ensureDKBinaryExists(dkPath: string): Promise<boolean> {
  // Check if binary exists
  if (existsSync(dkPath)) {
    logger.debug(`DK binary found at ${dkPath}`)
    return true
  }

  logger.debug(`DK binary not found at ${dkPath}, downloading...`)

  // Get current platform
  const platform = getOSPlatform()
  if (platform === 'unsupported') {
    logger.error('Cannot download DK binary for unsupported platform')
    return false
  }

  // Download binary
  return await downloadDKBinary(platform, dkPath)
}

// Function to check and download SyftBox binary if needed
async function ensureSyftBoxBinaryExists(syftboxPath: string): Promise<boolean> {
  // Check if binary exists
  if (existsSync(syftboxPath)) {
    logger.debug(`SyftBox binary found at ${syftboxPath}`)
    return true
  }

  logger.debug(`SyftBox binary not found at ${syftboxPath}, downloading...`)

  // Get current platform and architecture
  const os = getOSPlatform()
  const arch = getCPUArchitecture()

  if (os === 'unsupported' || arch === 'unsupported') {
    logger.error(
      `Cannot download SyftBox binary for unsupported platform/architecture: ${os}/${arch}`
    )
    return false
  }

  // Download binary
  return await downloadSyftBoxBinary(os, arch, syftboxPath)
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

      logger.debug(`Starting DK binary: ${dkPath} with args:`, args)

      // Spawn the process
      childProcess = spawn(dkPath, args, {
        stdio: 'pipe',
        detached: false
      })

      logger.debug('DK binary started with PID:', childProcess.pid)

      childProcess.stdout?.on('data', (data) => {
        logger.debug(`DK binary stdout: ${data}`)
      })

      childProcess.stderr?.on('data', (data) => {
        logger.error(`DK binary stderr: ${data}`)
      })

      childProcess.on('error', (error) => {
        logger.error(`Failed to start DK binary: ${error}`)
        childProcess = null
      })

      childProcess.on('close', (code) => {
        logger.debug(`DK binary exited with code ${code}`)
        childProcess = null
      })
    } catch (error) {
      logger.error('Failed to spawn DK binary:', error)
    }
  } else {
    logger.debug('No dk_config found, DK binary not started')
  }

  // Start the syftbox executable
  try {
    // Don't start if already running
    if (syftboxProcess) {
      return
    }

    // Determine the base directory for storing syftbox binary
    const baseDir = app.getPath('userData')
    const binDir = path.join(baseDir, 'bin')

    // Ensure bin directory exists
    ensureDirectoryExists(path.join(binDir, 'dummy.txt'))

    // Determine the path to the syftbox executable based on platform
    let syftboxPath = ''
    const osPlatform = getOSPlatform()

    if (osPlatform === 'linux') {
      // First check in our app's bin directory
      syftboxPath = path.join(binDir, 'syftbox')

      // Add snap-specific logging
      if (process.env.SNAP) {
        logger.debug('Running in Snap environment, make sure interfaces are connected')
      }
    } else if (osPlatform === 'darwin') {
      // On macOS, try our app's bin directory first
      syftboxPath = path.join(binDir, 'syftbox')
    } else if (osPlatform === 'windows') {
      // On Windows, use our app's bin directory with .exe extension
      syftboxPath = path.join(binDir, 'syftbox.exe')
    } else {
      logger.error(`Unsupported platform: ${osPlatform}`)
      return
    }

    logger.debug(`Using syftbox path for ${osPlatform}: ${syftboxPath}`)

    // Check if SyftBox binary exists and download if needed
    const binaryExists = await ensureSyftBoxBinaryExists(syftboxPath)
    if (!binaryExists) {
      logger.error(
        'SyftBox binary could not be found or downloaded. SyftBox process will not start.'
      )

      if (osPlatform === 'linux' && process.env.SNAP) {
        logger.error(
          'If running in Snap environment, make sure system-files interface is connected'
        )
        logger.error('Run: sudo snap connect dk:syftbox-exec-plug :system-files')
      } else if (osPlatform === 'darwin') {
        logger.error('You may need to grant additional permissions to this application')
      }

      return
    }


    // Get the syftbox config path from the utility function
    // that matches what's used in the onboarding process
    const appPaths = getAppPaths()
    const syftboxConfigPath = appPaths.syftboxConfig


    // Ensure the syftbox config directory exists
    const syftboxConfigDir = path.dirname(syftboxConfigPath || '')
    
    if (!existsSync(syftboxConfigDir)) {
      mkdirSync(syftboxConfigDir, { recursive: true })
    }


    // Spawn the syftbox process with config parameter
    try {
      // Ensure PATH includes common binary locations for all platforms
      const commonPaths = '/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin:/snap/bin'
      const platformPaths =
        process.platform === 'win32'
          ? ';C:\\Windows\\System32;C:\\Windows;C:\\Windows\\System32\\Wbem'
          : ''

      // Include user's ~/.local/bin in PATH for additional binaries
      // Using userInfo().homedir for more reliable path resolution in confined environments
      const userHomeDir = os.userInfo().homedir
      const userLocalBin = path.join(userHomeDir, '.local/bin')
      const newPath = `${process.env.PATH || ''}${platformPaths}:${commonPaths}:${userLocalBin}`



      syftboxProcess = spawn(syftboxPath, ['--config', syftboxConfigPath || ''], {
        stdio: 'pipe',
        detached: false,
        env: {
          ...process.env,
          // Ensure PATH is properly set for all platforms to find system binaries like 'uv'
          // Also include user's ~/.local/bin directory
          PATH: newPath
        }
      })

    } catch (error) {
      logger.error(`Failed to spawn syftbox process: ${error}`)
      logger.error(
        `Spawn error details - Type: ${typeof error}, Message: ${
          error instanceof Error ? error.message : 'Unknown error'
        }`
      )
      if (error instanceof Error && error.stack) {
        logger.error(`Error stack trace: ${error.stack}`)
      }
      return
    }

    if (syftboxProcess) {

      syftboxProcess.stdout?.on('data', (data) => {
        const output = data.toString().trim()
        // Only log errors
        if (output.includes('Error') || output.includes('error')) {
          logger.error(`Syftbox stdout error: ${output}`)
        }
      })

      syftboxProcess.stderr?.on('data', (data) => {
        const errorOutput = data.toString().trim()
        logger.error(`Syftbox stderr: ${errorOutput}`)
        
        // Keep essential error categorization for troubleshooting
        if (errorOutput.includes('permission denied')) {
          logger.error(`Permission error detected: ${errorOutput}`)
        } else if (errorOutput.includes('not found')) {
          logger.error(`Binary or dependency not found: ${errorOutput}`)
        }
      })

      syftboxProcess.on('error', (error) => {
        logger.error(`Failed to start syftbox binary: ${error}`)
        logger.error(`Process error details: ${error instanceof Error ? error.message : 'Unknown error'}`)
        syftboxProcess = null
      })

      syftboxProcess.on('close', (code) => {
        if (code !== 0) {
          logger.error(`Syftbox terminated with non-zero exit code: ${code}`)
        }
        syftboxProcess = null
      })
    }
  } catch (error) {
    logger.error('Failed to spawn syftbox binary:', error)
  }
}

// Initialize all services in the correct order
logger.debug('Application starting up...')

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
      logger.debug(
        `Found config with invalid/default userID "${configData.userID}" - removing this file`
      )

      // Delete the invalid config file
      unlinkSync(configPath)

      // Update existence flag
      configExists = false

      logger.debug(`Removed invalid config file. App will now show onboarding.`)
    }
  } catch (error) {
    logger.error(`Error checking config validity:`, error)
  }
}

logger.debug(`Valid config file exists at ${configPath}: ${configExists}`)

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
  logger.debug('Config file not found, forcing onboarding wizard')
  logger.debug(
    `Updated onboarding status: isFirstRun=${onboardingStatus.isFirstRun}, completed=${onboardingStatus.completed}`
  )

  // Log a clear marker for debugging
  logger.debug('ONBOARDING SHOULD BE DISPLAYED - NO CONFIG FILE EXISTS')
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
  const mainWindow = createMainWindow()
  logger.debug('Main window created')

  // Start the external programs only if config exists and onboarding is completed
  // Only start external processes when:
  // 1. Config file exists AND
  // 2. UserID is not empty or the default (to ensure it's a valid config) AND
  // 3. Onboarding is not in first run state
  const isValidConfig = configExists && appConfig.userID && appConfig.userID !== 'default-user'
  const shouldStartProcesses = isValidConfig && !onboardingStatus.isFirstRun

  logger.debug(
    `Config exists: ${configExists}, Onboarding needed: ${onboardingStatus.isFirstRun}, Completed: ${onboardingStatus.completed}`
  )

  if (shouldStartProcesses) {
    // Start processes asynchronously
    startExternalProcesses().catch((error) => {
      logger.error('Failed to start external processes:', error)
    })
  } else {
    logger.debug(
      'Config not set up or onboarding not completed. External processes will not start.'
    )
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
  logger.debug('Application shutting down, cleaning up resources')

  // Terminate the DK process if it exists
  if (childProcess && childProcess.pid) {
    logger.debug('Terminating DK binary with PID:', childProcess.pid)

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
    logger.debug('Terminating syftbox binary with PID:', syftboxProcess.pid)

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
