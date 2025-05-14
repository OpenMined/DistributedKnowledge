import * as path from 'path'
import { homedir, userInfo } from 'os'

/**
 * Get platform-specific application paths
 * Shared utility to prevent circular dependencies
 * @returns An object containing various application-specific paths
 */
export function getAppPaths() {
  try {
    // We need to load electron dynamically to avoid circular dependencies
    // Only import electron when we're in the main process
    if (process.type === 'browser') {
      const { app } = require('electron')

      // Base directories
      const userData = app.getPath('userData')
      const appData = app.getPath('appData')

      // Using os.userInfo() to get accurate user homedir even in confined environments like Snap
      const userHomedir = userInfo().homedir
      const username = userInfo().username

      // Config base directory
      const configDir = path.join(userData, 'config')

      // Calculate base directory for all app data (parent of config directory)
      const basePath = path.dirname(configDir)

      return {
        // Base path (parent directory of config)
        basePath,

        // Config paths
        configDir,
        configFile: path.join(configDir, 'config.json'),

        // Data paths
        dataDir: path.join(userData, 'data'),

        // Binary paths
        binDir: path.join(userData, 'bin'),
        dkBinary: path.join(userData, 'bin', 'dk'),

        // Resource paths for resources and logs (using base path)
        resourcesDir: path.join(basePath, 'resources'),
        logsDir: path.join(basePath, 'logs'),

        // SyftBox paths (platform-specific)
        syftboxConfig:
          process.platform === 'win32'
            ? path.join(appData, 'syftbox', 'config.json')
            : path.join(userHomedir, '.syftbox', 'config.json')
      }
    }
  } catch (error) {
    console.error('Error getting app paths:', error)
  }

  // Return minimal fallback paths
  return {
    logsDir: path.join(process.cwd(), 'logs')
  }
}
