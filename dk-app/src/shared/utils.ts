import * as fs from 'fs'
import * as path from 'path'
import * as nacl from 'tweetnacl'
import { Message, MessageContent, ChatMessage, User, Channel, MessageAttachment } from './types'

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
  console.log(
    `${name} key info:\n` +
      `  - Type: ${key.constructor.name}\n` +
      `  - Length: ${key.length}\n` +
      `  - First 8 bytes: ${Array.from(key.slice(0, 8))
        .map((b) => b.toString(16).padStart(2, '0'))
        .join(' ')}\n`
  )
}

// Convert a Date object to a readable string format
export function formatMessageTimestamp(date: Date | string): string {
  // Convert string dates to Date objects for consistent formatting
  let dateObj: Date

  if (typeof date === 'string') {
    // Check if it's already a formatted string (not an ISO date)
    if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/.test(date)) {
      return date // Return already formatted strings
    }
    // Parse ISO date string into Date object
    dateObj = new Date(date)
  } else {
    dateObj = date
  }

  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const yesterday = new Date(today)
  yesterday.setDate(yesterday.getDate() - 1)

  // Date is today
  if (dateObj >= today) {
    return dateObj.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }

  // Date is yesterday
  if (dateObj >= yesterday) {
    return 'Yesterday'
  }

  // Date is within last 7 days
  const lastWeek = new Date(today)
  lastWeek.setDate(lastWeek.getDate() - 6)
  if (dateObj >= lastWeek) {
    const options: Intl.DateTimeFormatOptions = { weekday: 'long' }
    return dateObj.toLocaleDateString(undefined, options)
  }

  // Date is older than a week but in the current year
  if (dateObj.getFullYear() === now.getFullYear()) {
    const options: Intl.DateTimeFormatOptions = { month: 'short', day: 'numeric' }
    return dateObj.toLocaleDateString(undefined, options)
  }

  // Date is from a different year
  const options: Intl.DateTimeFormatOptions = { year: 'numeric', month: 'short', day: 'numeric' }
  return dateObj.toLocaleDateString(undefined, options)
}

// Convert ChatMessage to Message for sending through the client
export function chatMessageToMessage(chatMessage: ChatMessage, targetId: string): Message {
  // Create the message content structure
  const messageContent: MessageContent = {
    text: chatMessage.text,
    messageType: chatMessage.messageType || 'text',
    attachments: chatMessage.attachments || [],
    metadata: {
      clientId: `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
      mentions: extractMentions(chatMessage.text),
      isForwarded: chatMessage.metadata?.isForwarded || false
    }
  }

  // Add replyTo if present
  if (chatMessage.replyTo) {
    messageContent.replyTo = chatMessage.replyTo
  }

  // Convert the message content to a string for the Message interface
  const contentString = JSON.stringify(messageContent)

  // Create the Message object
  const message: Message = {
    from:
      typeof chatMessage.sender.id === 'string'
        ? chatMessage.sender.id
        : chatMessage.sender.id.toString(),
    to: targetId, // Channel ID or user ID
    content: contentString,
    timestamp:
      typeof chatMessage.timestamp === 'string' ? new Date() : (chatMessage.timestamp as Date)
  }

  return message
}

// Convert Message back to ChatMessage for display
export function messageToChat(message: Message, users: Record<string | number, User>): ChatMessage {
  try {
    // Parse the message content
    let messageContent: MessageContent
    try {
      // First try parsing as MessageContent
      messageContent = JSON.parse(message.content) as MessageContent

      // Check if we have a valid MessageContent object with text property
      if (!messageContent || typeof messageContent !== 'object' || !('text' in messageContent)) {
        // This might be a plain JSON string, try to interpret it directly
        if (typeof message.content === 'string') {
          messageContent = {
            text: message.content,
            messageType: 'text'
          }
        } else {
          // Fallback to empty text
          messageContent = {
            text: '',
            messageType: 'text'
          }
        }
      }
    } catch (e) {
      // If not in JSON format, treat as simple text message
      messageContent = {
        text: message.content || '',
        messageType: 'text'
      }
    }

    // Add debugging for message content
    console.log('Parsed message content:', JSON.stringify(messageContent))

    // Find the sender in the users map or create a placeholder
    const senderId = message.from
    const sender = users[senderId] || {
      id: senderId,
      name: senderId,
      avatar: senderId.substring(0, 2).toUpperCase()
    }

    // Create a timestamp string for display
    const timestamp = message.timestamp
      ? formatMessageTimestamp(message.timestamp)
      : formatMessageTimestamp(new Date())

    // Build the ChatMessage
    const chatMessage: ChatMessage = {
      id: message.id || Date.now(),
      sender: sender,
      text: messageContent.text || '', // Ensure text is never undefined
      timestamp: timestamp,
      messageType: messageContent.messageType || 'text',
      deliveryStatus: mapMessageStatus(message.status)
    }

    // Add optional fields if present
    if (messageContent.replyTo) {
      chatMessage.replyTo = messageContent.replyTo
    }

    if (messageContent.attachments && messageContent.attachments.length > 0) {
      chatMessage.attachments = messageContent.attachments
    }

    if (messageContent.metadata) {
      chatMessage.metadata = messageContent.metadata
    }

    return chatMessage
  } catch (error) {
    console.error('Error converting message to chat format:', error)

    // Fallback to basic message format
    return {
      id: message.id || Date.now(),
      sender: {
        id: message.from,
        name: message.from,
        avatar: message.from.substring(0, 2).toUpperCase()
      },
      text: message.content || '', // Ensure text is never undefined
      timestamp: formatMessageTimestamp(message.timestamp || new Date())
    }
  }
}

// Extract user mentions from message text
function extractMentions(text: string): string[] {
  const mentionRegex = /@(\w+)/g
  const mentions: string[] = []
  let match

  while ((match = mentionRegex.exec(text)) !== null) {
    mentions.push(match[1])
  }

  return mentions
}

// Map message status to delivery status
function mapMessageStatus(status?: string): ChatMessage['deliveryStatus'] {
  if (!status) return 'sent'

  switch (status) {
    case 'verified':
      return 'delivered'
    case 'read':
      return 'read'
    case 'invalid_signature':
    case 'decryption_failed':
      return 'failed'
    default:
      return 'sent'
  }
}

// Create a channel message
export function createChannelMessage(
  sender: User,
  channelId: string,
  text: string,
  attachments?: MessageAttachment[]
): Message {
  const chatMessage: ChatMessage = {
    id: Date.now(),
    sender: sender,
    text: text,
    timestamp: new Date(),
    messageType: 'text'
  }

  if (attachments && attachments.length > 0) {
    chatMessage.attachments = attachments
  }

  return chatMessageToMessage(chatMessage, channelId)
}

// Create a direct message
export function createDirectMessage(
  sender: User,
  recipientId: string | number,
  text: string,
  attachments?: MessageAttachment[]
): Message {
  const chatMessage: ChatMessage = {
    id: Date.now(),
    sender: sender,
    text: text,
    timestamp: new Date(),
    messageType: 'text'
  }

  if (attachments && attachments.length > 0) {
    chatMessage.attachments = attachments
  }

  return chatMessageToMessage(
    chatMessage,
    typeof recipientId === 'string' ? recipientId : recipientId.toString()
  )
}

// Create a reply message
export function createReplyMessage(
  sender: User,
  targetId: string,
  text: string,
  replyToMessageId: number,
  attachments?: MessageAttachment[]
): Message {
  const chatMessage: ChatMessage = {
    id: Date.now(),
    sender: sender,
    text: text,
    timestamp: new Date(),
    messageType: 'text',
    replyTo: replyToMessageId
  }

  if (attachments && attachments.length > 0) {
    chatMessage.attachments = attachments
  }

  return chatMessageToMessage(chatMessage, targetId)
}
