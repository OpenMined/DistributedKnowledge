import * as fs from 'fs'
import * as path from 'path'
import * as nacl from 'tweetnacl'
import { BrowserWindow, app } from 'electron'
import { createServiceLogger } from '../shared/logging'
import { ToastOptions } from '../shared/types'
import { homedir, userInfo } from 'os'

// Create a specific logger for utils
const logger = createServiceLogger('utils')

/**
 * Loads existing Ed25519 key pair from files or creates a new pair if files don't exist
 * @param privateKeyPath Path to store/load the private key in hex format
 * @param publicKeyPath Path to store/load the public key in hex format
 * @returns Promise resolving to an object containing public and private keys as Uint8Arrays
 */
export async function loadOrCreateKeys(
  privateKeyPath: string,
  publicKeyPath: string
): Promise<{ publicKey: Uint8Array; privateKey: Uint8Array }> {
  let privateKey: Uint8Array
  let publicKey: Uint8Array

  // Ensure directories exist
  fs.mkdirSync(path.dirname(privateKeyPath), { recursive: true })
  fs.mkdirSync(path.dirname(publicKeyPath), { recursive: true })

  if (fs.existsSync(privateKeyPath) && fs.existsSync(publicKeyPath)) {
    // Read keys from disk (hex)
    const privHex = fs.readFileSync(privateKeyPath, 'utf8').trim()
    const pubHex = fs.readFileSync(publicKeyPath, 'utf8').trim()
    privateKey = new Uint8Array(Buffer.from(privHex, 'hex'))
    publicKey = new Uint8Array(Buffer.from(pubHex, 'hex'))
  } else {
    // Generate a new Ed25519 key pair
    const keyPair = nacl.sign.keyPair()
    privateKey = keyPair.secretKey // 64 bytes (seed + pubkey)
    publicKey = keyPair.publicKey // 32 bytes

    // Write keys to disk in hex format
    fs.writeFileSync(privateKeyPath, Buffer.from(privateKey).toString('hex'), 'utf8')
    fs.writeFileSync(publicKeyPath, Buffer.from(publicKey).toString('hex'), 'utf8')
  }

  return { publicKey, privateKey }
}

/**
 * Debugging helper to log information about a key
 */
export function logKeyInfo(name: string, key: Uint8Array): void {
  logger.debug(`${name} key info:`, {
    type: key.constructor.name,
    length: key.length,
    firstBytes: Array.from(key.slice(0, 8))
      .map((b) => b.toString(16).padStart(2, '0'))
      .join(' ')
  })
}

/**
 * Shows a toast notification in all renderer processes
 * @param message The toast message
 * @param options Toast configuration options
 */
export function showToast(message: string, options: ToastOptions = {}): void {
  logger.debug('Showing toast:', { message, ...options })

  // Send the toast to all renderer processes
  BrowserWindow.getAllWindows().forEach((window) => {
    if (!window.isDestroyed()) {
      window.webContents.send('toast', message, options)
    }
  })
}

/**
 * Get platform-specific application paths
 * @returns An object containing various application-specific paths
 */
export function getAppPaths() {
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

  // Log paths for debugging
  logger.info(
    `App paths: userData=${userData}, appData=${appData}, userHomedir=${userHomedir}, username=${username}, basePath=${basePath}`
  )

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
