import { BrowserWindow, shell, app } from 'electron'
import { join } from 'path'
import { is } from '@electron-toolkit/utils'
import { existsSync } from 'fs'

// Create a type declaration to extend global object
declare global {
  var mainWindow: BrowserWindow | null
}

export function createMainWindow(): BrowserWindow {
  // Create the browser window.
  const mainWindow = new BrowserWindow({
    width: 900,
    height: 670,
    show: false,
    frame: false,
    autoHideMenuBar: true,
    webPreferences: {
      preload: join(__dirname, '../preload/index.js'),
      sandbox: false
    }
  })

  // Store a reference to the main window in the global object
  global.mainWindow = mainWindow

  mainWindow.on('ready-to-show', () => {
    mainWindow.show()
  })

  // Set Content Security Policy to allow connections to localhost:4232
  mainWindow.webContents.session.webRequest.onHeadersReceived((details, callback) => {
    callback({
      responseHeaders: {
        ...details.responseHeaders,
        'Content-Security-Policy': [
          "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; connect-src 'self' ws: wss: http://localhost:4232 https://localhost:4232; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:;"
        ]
      }
    })
  })

  mainWindow.webContents.setWindowOpenHandler((details) => {
    shell.openExternal(details.url)
    return { action: 'deny' }
  })

  // HMR for renderer base on electron-vite cli.
  // Load the remote URL for development or the local html file for production.
  if (is.dev && process.env['ELECTRON_RENDERER_URL']) {
    mainWindow.loadURL(process.env['ELECTRON_RENDERER_URL'])
  } else {
    // Try multiple possible paths for the renderer HTML file
    const possiblePaths = [
      join(__dirname, '../renderer/index.html'),
      join(__dirname, '../../renderer/index.html')
    ]

    // Try each path and use the first one that exists
    let rendererPath = possiblePaths[0] // Default to first path

    for (const path of possiblePaths) {
      try {
        if (existsSync(path)) {
          rendererPath = path
          break
        }
      } catch (error) {
        // Ignore errors checking for file existence
      }
    }

    mainWindow.loadFile(rendererPath)
  }

  return mainWindow
}
