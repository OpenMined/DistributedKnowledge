import { ipcMain, BrowserWindow } from 'electron'
import { Channels } from '../../shared/constants'
import { Client } from '../services/client'
import { appConfig } from '../services/config'
import { mockUsers, mockMessages, mockChannelMessages, mockSidebarData } from '../services/mockData'
import {
  getClient,
  sendDirectMessage,
  sendChannelMessage,
  getMessages,
  markMessagesAsRead
} from '../services/clientService'
import { ChatMessage, MessageAttachment, ToastOptions } from '../../shared/types'
import { registerLLMHandlers } from './llmHandlers'
import { registerAppHandlers } from './appHandlers'
import { registerTrackerHandlers } from './trackerHandlers'
import { registerTrackerMarketplaceHandlers } from './trackerMarketplaceHandlers'
import { registerOnboardingHandlers } from './onboardingHandlers'
import { registerMCPHandlers } from './mcpHandlers'
import logger from '../../shared/logging'
import { showToast } from '../utils'

// Variable to hold the WebSocket client reference
let wsClient: Client | null = null

// Set the WebSocket client reference
export function setWsClient(client: Client | null): void {
  wsClient = client
}

// Setup IPC handlers
export function setupIpcHandlers(): void {
  // Register LLM handlers
  registerLLMHandlers()

  // Register App handlers
  registerAppHandlers()

  // Register Tracker scan handlers
  registerTrackerHandlers()

  // Register Tracker marketplace handlers
  registerTrackerMarketplaceHandlers()

  // Register Onboarding handlers
  registerOnboardingHandlers()

  // Register MCP handlers
  registerMCPHandlers()

  // IPC test
  ipcMain.on(Channels.Ping, () => logger.debug('pong'))

  // Window control IPC handlers
  ipcMain.on(Channels.WindowMinimize, () => {
    const win = BrowserWindow.getFocusedWindow()
    if (win) win.minimize()
  })

  ipcMain.on(Channels.WindowMaximize, () => {
    const win = BrowserWindow.getFocusedWindow()
    if (win) {
      win.isMaximized() ? win.unmaximize() : win.maximize()
    }
  })

  ipcMain.on(Channels.WindowClose, () => {
    const win = BrowserWindow.getFocusedWindow()
    if (win) win.close()
  })

  ipcMain.handle(Channels.WindowIsMaximized, () => {
    const win = BrowserWindow.getFocusedWindow()
    return win ? win.isMaximized() : false
  })

  // Chat data IPC handlers
  ipcMain.handle(Channels.GetChatMessages, (_, userId) => {
    // Try to get messages from the client service - this now uses the database first
    const clientMessages = getMessages(userId.toString())

    // If we have real messages, use those, otherwise fall back to mock data
    if (clientMessages && clientMessages.length > 0) {
      return clientMessages
    }

    // Fall back to mock data
    return mockMessages[userId] || []
  })

  ipcMain.handle(Channels.GetUserInfo, (_, userId) => {
    // Return user info for the requested user
    return mockUsers[userId] || null
  })

  ipcMain.handle(Channels.SendMessage, (_, { userId, text, attachments }) => {
    if (!userId || !text) return { success: false, error: 'Invalid message data' }

    try {
      // First, try to send via the client service (now persists to database)
      const client = wsClient || getClient()
      if (client) {
        sendDirectMessage(userId.toString(), text)

        // Return the last message from the service as confirmation
        const messages = getMessages(userId.toString())
        const lastMessage = messages[messages.length - 1]

        if (lastMessage) {
          return { success: true, message: lastMessage }
        }
      }

      // Fall back to mock implementation if client not available
      // Generate a new message id based on existing messages
      const userMessages = mockMessages[userId] || []
      const newId = userMessages.length > 0 ? Math.max(...userMessages.map((m) => m.id)) + 1 : 1

      // Create the new message
      const newMessage = {
        id: newId,
        sender: { id: 0, name: 'You', avatar: 'ME' },
        text,
        timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
      }

      // Add to messages
      if (!mockMessages[userId]) {
        mockMessages[userId] = []
      }

      mockMessages[userId].push(newMessage)
      return { success: true, message: newMessage }
    } catch (error) {
      logger.error('Failed to send message:', error)
      return { success: false, error: 'Failed to send message' }
    }
  })

  // Channel messages IPC handlers
  ipcMain.handle(Channels.GetChannelMessages, (_, channelId) => {
    // Try to get messages from the client service first (now uses database)
    const clientMessages = getMessages(channelId)

    // If we have real messages, use those, otherwise fall back to mock data
    if (clientMessages && clientMessages.length > 0) {
      return clientMessages
    }

    // Fall back to mock data
    return mockChannelMessages[channelId] || []
  })

  ipcMain.handle(Channels.SendChannelMessage, (_, { channelId, text, attachments }) => {
    if (!channelId || !text) return { success: false, error: 'Invalid message data' }

    try {
      // First, try to send via the client service (now persists to database)
      const client = wsClient || getClient()
      if (client) {
        sendChannelMessage(channelId, text)

        // Return the last message from the service as confirmation
        const messages = getMessages(channelId)
        const lastMessage = messages[messages.length - 1]

        if (lastMessage) {
          return { success: true, message: lastMessage }
        }
      }

      // Fall back to mock implementation if client not available
      // Generate a new message id based on existing messages
      const channelMsgs = mockChannelMessages[channelId] || []
      const newId = channelMsgs.length > 0 ? Math.max(...channelMsgs.map((m) => m.id)) + 1 : 1

      // Create the new message
      const newMessage = {
        id: newId,
        sender: { id: 0, name: 'You', avatar: 'ME' },
        text,
        timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
        replies: [],
        replyCount: 0
      }

      // Add to channel messages
      if (!mockChannelMessages[channelId]) {
        mockChannelMessages[channelId] = []
      }

      mockChannelMessages[channelId].push(newMessage)
      return { success: true, message: newMessage }
    } catch (error) {
      logger.error('Failed to send channel message:', error)
      return { success: false, error: 'Failed to send channel message' }
    }
  })

  ipcMain.handle(Channels.SendChannelReply, (_, { channelId, messageId, text }) => {
    if (!channelId || !messageId || !text) return { success: false, error: 'Invalid reply data' }

    try {
      // For now, we'll just use the mock implementation for replies
      // since our message infrastructure doesn't have a specific reply method yet

      // Find the message to reply to
      const channelMsgs = mockChannelMessages[channelId] || []
      const messageIndex = channelMsgs.findIndex((m) => m.id === messageId)

      if (messageIndex === -1) return { success: false, error: 'Message not found' }

      // Generate a new reply id
      const message = channelMsgs[messageIndex]
      const replies = message.replies || []
      const newReplyId =
        replies.length > 0 ? Math.max(...replies.map((r) => r.id)) + 1 : message.id * 100 + 1

      // Create the new reply
      const newReply = {
        id: newReplyId,
        sender: { id: 0, name: 'You', avatar: 'ME' },
        text,
        timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
      }

      // Add to replies
      if (!message.replies) {
        message.replies = []
      }

      message.replies.push(newReply)
      message.replyCount = message.replies.length

      return { success: true, reply: newReply }
    } catch (error) {
      logger.error('Failed to send channel reply:', error)
      return { success: false, error: 'Failed to send channel reply' }
    }
  })

  // Sidebar data IPC handlers
  ipcMain.handle(Channels.GetSidebarUsers, async () => {
    // Get the websocket client
    const client = wsClient || getClient()

    // Check if client is initialized and connected
    if (!client) {
      return mockSidebarData.users
    }

    try {
      // Get real active user status from the server
      const userStatus = await client.getUserActiveStatus()

      // Extract all unique user IDs from both active and inactive lists
      const allUserIds = [...new Set([...userStatus.active_users, ...userStatus.inactive_users])]

      // If no users returned, fall back to mock data
      if (allUserIds.length === 0) {
        return mockSidebarData.users
      }

      // Create a map for quick lookup of online status
      const onlineStatusMap = new Map()
      userStatus.active_users.forEach((userId) => onlineStatusMap.set(userId, true))
      userStatus.inactive_users.forEach((userId) => onlineStatusMap.set(userId, false))

      // Map all users to the format needed by the sidebar
      const users = allUserIds.map((userId) => {
        // Try to find matching user in mock data first for names
        const mockUser = mockSidebarData.users.find((u) => u.id.toString() === userId)
        const isOnline = onlineStatusMap.get(userId) || false

        // For the ID, try to parse as integer if possible, otherwise use string ID
        // For invalid numeric IDs, generate a unique ID based on the userId string
        let id: number | string = parseInt(userId, 10)
        if (isNaN(id)) {
          // If userId can't be parsed as a number, use it directly as string ID
          id = userId
        }

        // Format the user data - use userId directly as the name if no mock user found
        return {
          id: id,
          name: mockUser?.name || userId,
          online: isOnline,
          status: isOnline ? 'online' : 'offline'
        }
      })

      // Sort users: online first, then by name
      users.sort((a, b) => {
        if (a.online && !b.online) return -1
        if (!a.online && b.online) return 1
        return a.name.localeCompare(b.name)
      })

      return users
    } catch (error) {
      logger.error('Error in GetSidebarUsers handler:', error)
      // Fall back to mock data if there's an error
      return mockSidebarData.users
    }
  })

  ipcMain.handle(Channels.GetSidebarChannels, () => {
    return mockSidebarData.channels
  })

  // Message marking
  ipcMain.handle(Channels.MarkMessagesAsRead, (_, id) => {
    if (!id) return { success: false, error: 'Invalid ID' }

    try {
      markMessagesAsRead(id.toString())
      return { success: true }
    } catch (error) {
      logger.error('Failed to mark messages as read:', error)
      return { success: false, error: 'Failed to mark messages as read' }
    }
  })

  // Config data IPC handlers
  ipcMain.handle(Channels.GetConfig, () => {
    // Get the websocket client
    const client = wsClient || getClient()

    // Return a sanitized version of the config (without sensitive paths)
    return {
      serverURL: appConfig.serverURL,
      userID: appConfig.userID,
      // Don't send the actual private key paths to the renderer
      isConnected: client ? true : false
    }
  })

  // Toast handler
  ipcMain.on(Channels.ShowToast, (event, message: string, options: ToastOptions = {}) => {
    showToast(message, options)
  })
}
