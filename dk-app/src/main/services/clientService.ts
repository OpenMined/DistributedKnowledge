import { Client } from './client'
import { appConfig } from './config'
import {
  loadOrCreateKeys,
  messageToChat,
  createDirectMessage,
  createChannelMessage
} from '../../shared/utils'
import { Message, ChatMessage, User, Channel } from '../../shared/types'
import { mockUsers } from './mockData'
import { Channels } from '../../shared/constants'
import {
  saveMessage,
  getConversationMessages,
  updateMessageStatus,
  updateUserMessagesStatus,
  dbMessageToChatMessage,
  messageToDbMessage
} from './database'
import { createServiceLogger } from '../../shared/logging'

// Create a specific logger for client service
const logger = createServiceLogger('clientService')

// Client instance
let wsClient: Client | null = null

// Cache of messages (used as a supplementary cache for performance)
const messageCache: Record<string, ChatMessage[]> = {}

// Function to initialize the client
export async function initializeClient(): Promise<Client | null> {
  try {
    // Log the server URL for debugging
    logger.debug(`Connecting to server: ${appConfig.serverURL}`)

    // For development/testing, set a localhost URL if none specified
    if (!appConfig.serverURL || appConfig.serverURL === 'http://localhost:3000') {
      // Use testing data since we're defaulting to localhost
      logger.debug('Using default localhost URL with test data')
      // Don't attempt to establish a real connection since it will fail
      return null
    }

    // Load or create keys using the proper utility function
    const { privateKey, publicKey } = await loadOrCreateKeys(
      appConfig.private_key!,
      appConfig.public_key!
    )

    // Keys loaded successfully

    // Initialize the WebSocket client
    wsClient = new Client(appConfig.serverURL, appConfig.userID, privateKey, publicKey)

    // Set up message handler
    wsClient.onMessage((msg) => {
      handleIncomingMessage(msg)
    })

    // For development testing, always allow insecure connections
    wsClient.setInsecure(true)

    try {
      // Try to register first (in case this is a new user)
      await wsClient.register(appConfig.userID)
      logger.debug('User registered successfully')
    } catch (error) {
      logger.debug(`Registration failed, likely user already exists: ${error}`)
    }

    try {
      // Then log in
      await wsClient.login()
      logger.debug('User authenticated successfully')

      // Once logged in, establish WebSocket connection
      await wsClient.connect()
    } catch (error) {
      logger.error('Authentication or connection failed:', error)
      return null
    }

    return wsClient
  } catch (error) {
    logger.error('Failed to initialize client:', error)
    return null
  }
}

// Handle incoming messages from the chat server
function handleIncomingMessage(message: Message): void {
  // Convert the server Message to our app's ChatMessage format
  const chatMessage = messageToChat(message, mockUsers as Record<string | number, User>)

  // Determine the cache key (channel ID or user ID)
  const cacheKey =
    message.to === 'broadcast' || message.to.startsWith('#')
      ? message.to
      : message.from === appConfig.userID
        ? message.to // Message sent by us to someone else
        : message.from // Message sent by someone else to us

  // Store in message cache
  if (!messageCache[cacheKey]) {
    messageCache[cacheKey] = []
  }
  messageCache[cacheKey].push(chatMessage)

  // Sort messages by timestamp
  messageCache[cacheKey].sort((a, b) => {
    const timestampA =
      typeof a.timestamp === 'string'
        ? new Date(a.timestamp).getTime()
        : (a.timestamp as Date).getTime()
    const timestampB =
      typeof b.timestamp === 'string'
        ? new Date(b.timestamp).getTime()
        : (b.timestamp as Date).getTime()
    return timestampA - timestampB
  })

  // Save message to database with "delivered" status for received messages
  const dbMessage = messageToDbMessage(message, 'delivered')
  const messageId = saveMessage(
    dbMessage.from_user,
    dbMessage.to_user,
    dbMessage.content,
    dbMessage.timestamp,
    dbMessage.status,
    dbMessage.signature,
    dbMessage.message_type,
    dbMessage.attachment_data
  )

  // If the message was saved successfully, update the chatMessage with the database ID
  if (messageId) {
    chatMessage.id = messageId
  }

  // Notify renderer process of new message
  // Import the main window to send IPC messages
  const { BrowserWindow } = require('electron')
  const mainWindow = BrowserWindow.getAllWindows()[0]
  if (mainWindow) {
    mainWindow.webContents.send(Channels.NewMessage, { cacheKey, message: chatMessage })
    logger.debug(`Sent new message notification for ${cacheKey}`)
  }
}

// Get the client instance
export function getClient(): Client | null {
  return wsClient
}

// Disconnect the client
export function disconnectClient(): void {
  if (wsClient) {
    wsClient.disconnect()
    wsClient = null
  }
}

// Send a direct message to another user
export function sendDirectMessage(recipientId: string, text: string): void {
  if (!wsClient) {
    logger.error('Client not initialized')
    return
  }

  // Create a User object for the current user
  const currentUser: User = {
    id: appConfig.userID,
    name: appConfig.userID,
    avatar: appConfig.userID.substring(0, 2).toUpperCase()
  }

  // Create and send the message
  const message = createDirectMessage(currentUser, recipientId, text)
  wsClient.sendMessage(message)

  // Save message to database with "sent" status
  const dbMessage = messageToDbMessage(message, 'sent')
  const messageId = saveMessage(
    dbMessage.from_user,
    dbMessage.to_user,
    dbMessage.content,
    dbMessage.timestamp,
    dbMessage.status,
    dbMessage.signature,
    dbMessage.message_type,
    dbMessage.attachment_data
  )

  // Also add to our local cache as a sent message
  const chatMessage = messageToChat(message, { [appConfig.userID]: currentUser })
  chatMessage.deliveryStatus = 'sent'

  // If the message was saved successfully, update the chatMessage with the database ID
  if (messageId) {
    chatMessage.id = messageId
  }

  if (!messageCache[recipientId]) {
    messageCache[recipientId] = []
  }
  messageCache[recipientId].push(chatMessage)
}

// Send a message to a channel
export function sendChannelMessage(channelId: string, text: string): void {
  if (!wsClient) {
    logger.error('Client not initialized')
    return
  }

  // Create a User object for the current user
  const currentUser: User = {
    id: appConfig.userID,
    name: appConfig.userID,
    avatar: appConfig.userID.substring(0, 2).toUpperCase()
  }

  // Create and send the message
  const message = createChannelMessage(currentUser, channelId, text)
  wsClient.sendMessage(message)

  // Save message to database with "sent" status
  const dbMessage = messageToDbMessage(message, 'sent')
  const messageId = saveMessage(
    dbMessage.from_user,
    dbMessage.to_user,
    dbMessage.content,
    dbMessage.timestamp,
    dbMessage.status,
    dbMessage.signature,
    dbMessage.message_type,
    dbMessage.attachment_data
  )

  // Also add to our local cache as a sent message
  const chatMessage = messageToChat(message, { [appConfig.userID]: currentUser })
  chatMessage.deliveryStatus = 'sent'

  // If the message was saved successfully, update the chatMessage with the database ID
  if (messageId) {
    chatMessage.id = messageId
  }

  if (!messageCache[channelId]) {
    messageCache[channelId] = []
  }
  messageCache[channelId].push(chatMessage)
}

// Get messages for a specific channel or direct conversation
export function getMessages(id: string): ChatMessage[] {
  // Special case for own user ID - return empty for now
  if (id === appConfig.userID) {
    return []
  }

  // For direct messages, get conversation between current user and specified user
  const dbMessages = getConversationMessages(appConfig.userID, id)

  // Convert database messages to chat messages (even if empty)
  const chatMessages = dbMessages.map((msg) => dbMessageToChatMessage(msg, appConfig.userID))

  // Update cache for future quick access if we have messages
  if (chatMessages.length > 0) {
    messageCache[id] = chatMessages
  } else if (messageCache[id] && messageCache[id].length > 0) {
    // If no messages in database but we have cached messages, return cached ones
    return messageCache[id]
  }

  // Return the messages from database (even if empty)
  return chatMessages
}

// Mark messages as read
export function markMessagesAsRead(id: string): void {
  // Update message status in database
  updateUserMessagesStatus(id, appConfig.userID, 'read')

  // Also update cache
  if (!messageCache[id]) return

  messageCache[id].forEach((message) => {
    if (message.deliveryStatus === 'delivered') {
      message.deliveryStatus = 'read'

      // Update database record if we have an ID
      if (typeof message.id === 'number') {
        updateMessageStatus(message.id, 'read')
      }
    }
  })

  // Notify the server if needed (implementation depends on server API)
  if (wsClient) {
    const readReceipt = {
      from: appConfig.userID,
      to: id,
      content: JSON.stringify({ messageType: 'read_receipt', id: id }),
      timestamp: new Date()
    }
    wsClient.sendMessage(readReceipt)
  }
}
