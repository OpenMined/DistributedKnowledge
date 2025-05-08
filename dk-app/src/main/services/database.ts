import { app } from 'electron'
import { existsSync, mkdirSync } from 'fs'
import { dirname } from 'path'
import BetterSqlite3 from 'better-sqlite3'
// Define a Database type to avoid namespace issues
type Database = BetterSqlite3.Database
import { appConfig } from './config'
import { ChatMessage, Message, MessageContent, DocumentStats } from '../../shared/types'
import logger, { createServiceLogger } from '../../shared/logging'

// Create a specific logger for database service
const dbLogger = createServiceLogger('database')

// Define a type for database message records
export interface DbMessage {
  id: number
  from_user: string
  to_user: string
  content: string
  timestamp: string
  status?: string
  signature?: string
  message_type?: string
  attachment_data?: string
}

let db: BetterSqlite3.Database | null = null

export function initDatabaseService(): void {
  try {
    dbLogger.info('Initializing database service...')

    if (!appConfig.database?.path) {
      dbLogger.warn(
        'Database path not provided in config. Database service will not be initialized.'
      )
      return
    }

    dbLogger.info(`Using database path: ${appConfig.database.path}`)

    // Check if database file already exists
    const dbFileExists = existsSync(appConfig.database.path)
    logger.debug(`Database file exists: ${dbFileExists}`)

    // Ensure directory exists
    const dbDir = dirname(appConfig.database.path)
    if (!existsSync(dbDir)) {
      logger.info(`Creating database directory: ${dbDir}`)
      mkdirSync(dbDir, { recursive: true })
    }

    // Connect to SQLite database with verbose logging
    db = new BetterSqlite3(appConfig.database.path, {
      verbose: function (message) {
        logger.debug(`[SQLite]: ${message}`)
        return logger
      }
    })
    logger.info(`Connected to SQLite database at ${appConfig.database.path}`)

    // Initialize database schema if needed
    initSchema()

    // Close database when app quits
    app.on('quit', () => {
      logger.info('Application quitting, closing database...')
      closeDatabase()
    })

    // Also close database on will-quit event as a backup
    app.on('will-quit', () => {
      logger.info('Application will quit, closing database...')
      closeDatabase()
    })

    // Register a 'before-quit' handler to ensure we close the database
    app.on('before-quit', (event) => {
      logger.info('Application before-quit, closing database...')
      closeDatabase()
    })

    logger.info('Database service initialization complete')
  } catch (error) {
    logger.error('Failed to initialize database:', error)
  }
}

// Initialize schema (tables, indices, etc.)
function initSchema(): void {
  if (!db) return

  try {
    // Create the initial tables if they don't exist
    db.exec(`
      -- Messages table
      CREATE TABLE IF NOT EXISTS messages (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        from_user TEXT NOT NULL,
        to_user TEXT NOT NULL,
        content TEXT NOT NULL,
        timestamp TEXT NOT NULL,
        status TEXT,
        signature TEXT
      );

      -- Create indices for faster queries if they don't exist
      CREATE INDEX IF NOT EXISTS idx_messages_from_user ON messages(from_user);
      CREATE INDEX IF NOT EXISTS idx_messages_to_user ON messages(to_user);
      CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);
    `)

    // Check existing columns
    const tableInfo = db.prepare('PRAGMA table_info(messages)').all() as { name: string }[]
    const columnNames = tableInfo.map((col) => col.name)
    dbLogger.debug('Existing columns:', { columns: columnNames.join(', ') })

    // Add message_type column if it doesn't exist
    if (!columnNames.includes('message_type')) {
      dbLogger.info('Adding message_type column to messages table...')
      db!.exec("ALTER TABLE messages ADD COLUMN message_type TEXT DEFAULT 'text'")
    }

    // Add attachment_data column if it doesn't exist
    if (!columnNames.includes('attachment_data')) {
      dbLogger.info('Adding attachment_data column to messages table...')
      db!.exec('ALTER TABLE messages ADD COLUMN attachment_data TEXT')
    }

    // Log table sizes after init for debugging
    const messageCount = db.prepare('SELECT COUNT(*) as count FROM messages').get() as {
      count: number
    }

    logger.debug(`Database schema initialized. Messages table has ${messageCount.count} records.`)
  } catch (error) {
    logger.error('Error initializing database schema:', error)
  }
}

// Close the database connection
export function closeDatabase(): void {
  if (db) {
    try {
      // First check if there are any pending transactions
      const pragmaStmt = db.prepare(`PRAGMA wal_checkpoint;`)
      pragmaStmt.run()

      // Close the database
      db.close()
      db = null
      logger.info('Database connection closed successfully')
    } catch (error) {
      logger.error('Error closing database connection:', error)
      // Still set db to null to avoid repeated close attempts
      db = null
    }
  } else {
    logger.info('No active database connection to close')
  }
}

// Get the database instance
export function getDb(): BetterSqlite3.Database | null {
  return db
}

// Save a message to the database
export function saveMessage(
  fromUser: string,
  toUser: string,
  content: string,
  timestamp: string,
  status?: string,
  signature?: string,
  messageType: string = 'text',
  attachmentData?: string
): number | null {
  if (!db) {
    logger.error('Cannot save message: Database connection not initialized')
    return null
  }

  // Non-null assertion: db is checked above, so we know it's not null in the rest of the function

  try {
    // Begin a transaction to ensure data integrity
    // DB is non-null at this point (checked above)
    const transaction = db!.transaction(() => {
      const stmt = db!.prepare(`
        INSERT INTO messages (from_user, to_user, content, timestamp, status, signature, message_type, attachment_data)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
      `)

      const result = stmt.run(
        fromUser,
        toUser,
        content,
        timestamp,
        status,
        signature,
        messageType,
        attachmentData
      )

      // Log the saved message for debugging
      dbLogger.info(
        `Message saved to database: ID=${result.lastInsertRowid}, from=${fromUser}, to=${toUser}`
      )

      return result.lastInsertRowid as number
    })

    // Execute the transaction
    const messageId = transaction()
    return messageId
  } catch (error) {
    console.error('Error saving message to database:', error)
    return null
  }
}

// Get messages between two users
export function getConversationMessages(
  user1: string,
  user2: string,
  limit = 100,
  offset = 0
): DbMessage[] {
  if (!db) {
    dbLogger.error('Cannot get conversation messages: Database connection not initialized')
    return []
  }

  try {
    // First log what we're trying to fetch (for debugging)
    dbLogger.debug(
      `Fetching conversation between ${user1} and ${user2} (limit: ${limit}, offset: ${offset})`
    )

    // Check how many messages we expect to find
    const countStmt = db.prepare(`
      SELECT COUNT(*) as count FROM messages 
      WHERE (from_user = ? AND to_user = ?) OR (from_user = ? AND to_user = ?)
    `)
    const { count } = countStmt.get(user1, user2, user2, user1) as { count: number }
    dbLogger.debug(
      `Found ${count} messages in database for conversation between ${user1} and ${user2}`
    )

    // Get the actual messages
    const stmt = db.prepare(`
      SELECT * FROM messages 
      WHERE (from_user = ? AND to_user = ?) OR (from_user = ? AND to_user = ?)
      ORDER BY timestamp ASC
      LIMIT ? OFFSET ?
    `)

    const results = stmt.all(user1, user2, user2, user1, limit, offset) as DbMessage[]
    dbLogger.debug(`Retrieved ${results.length} messages from database`)
    return results
  } catch (error) {
    dbLogger.error('Error fetching messages from database:', error)
    return []
  }
}

// Get all messages for a user
export function getUserMessages(userId: string, limit = 100, offset = 0): DbMessage[] {
  if (!db) {
    dbLogger.error('Cannot get user messages: Database connection not initialized')
    return []
  }

  try {
    dbLogger.debug(`Fetching messages for user ${userId} (limit: ${limit}, offset: ${offset})`)

    // Check how many messages we expect to find
    const countStmt = db.prepare(`
      SELECT COUNT(*) as count FROM messages 
      WHERE from_user = ? OR to_user = ?
    `)
    const { count } = countStmt.get(userId, userId) as { count: number }
    dbLogger.debug(`Found ${count} messages in database for user ${userId}`)

    const stmt = db.prepare(`
      SELECT * FROM messages 
      WHERE from_user = ? OR to_user = ?
      ORDER BY timestamp DESC
      LIMIT ? OFFSET ?
    `)

    const results = stmt.all(userId, userId, limit, offset) as DbMessage[]
    dbLogger.debug(`Retrieved ${results.length} messages from database for user ${userId}`)
    return results
  } catch (error) {
    dbLogger.error('Error fetching user messages from database:', error)
    return []
  }
}

// Update message status
export function updateMessageStatus(messageId: number, status: string): boolean {
  if (!db) {
    dbLogger.error('Cannot update message status: Database connection not initialized')
    return false
  }

  try {
    dbLogger.debug(`Updating message status for ID ${messageId} to "${status}"`)

    // DB is non-null at this point (checked above)
    const transaction = db!.transaction(() => {
      const stmt = db!.prepare(`
        UPDATE messages
        SET status = ?
        WHERE id = ?
      `)

      const result = stmt.run(status, messageId)
      dbLogger.debug(`Updated status for message ID ${messageId}: ${result.changes} rows affected`)
      return result.changes > 0
    })

    return transaction()
  } catch (error) {
    dbLogger.error('Error updating message status:', error)
    return false
  }
}

// Update status for all messages from a specific user
export function updateUserMessagesStatus(
  fromUser: string,
  toUser: string,
  status: string
): boolean {
  if (!db) {
    dbLogger.error('Cannot update user messages status: Database connection not initialized')
    return false
  }

  try {
    dbLogger.debug(`Updating status to "${status}" for messages from ${fromUser} to ${toUser}`)

    // First check how many messages we'll be updating
    const countStmt = db.prepare(`
      SELECT COUNT(*) as count FROM messages
      WHERE from_user = ? AND to_user = ? AND status != ?
    `)
    const { count } = countStmt.get(fromUser, toUser, status) as { count: number }
    dbLogger.debug(`Found ${count} messages to update from ${fromUser} to ${toUser}`)

    // DB is non-null at this point (checked above)
    const transaction = db!.transaction(() => {
      const stmt = db!.prepare(`
        UPDATE messages
        SET status = ?
        WHERE from_user = ? AND to_user = ? AND status != ?
      `)

      const result = stmt.run(status, fromUser, toUser, status)
      dbLogger.debug(`Updated ${result.changes} messages from ${fromUser} to ${toUser}`)
      return result.changes > 0
    })

    return transaction()
  } catch (error) {
    dbLogger.error('Error updating user messages status:', error)
    return false
  }
}

// Convert DbMessage to ChatMessage for frontend use
export function dbMessageToChatMessage(dbMessage: DbMessage, currentUserId: string): ChatMessage {
  // Parse attachment data if it exists
  let attachments = []
  if (dbMessage.attachment_data) {
    try {
      attachments = JSON.parse(dbMessage.attachment_data)
    } catch (error) {
      dbLogger.error('Error parsing attachment data:', error)
    }
  }

  // Determine if message is from the current user
  const isFromCurrentUser = dbMessage.from_user === currentUserId

  // Try to extract the actual text content from JSON if possible
  let textContent = dbMessage.content
  try {
    const contentObj = JSON.parse(dbMessage.content)
    if (contentObj.text) {
      textContent = contentObj.text
    }
  } catch (error) {
    // If not valid JSON, use content as-is
    dbLogger.debug('Content is not valid JSON, using as plain text:', {
      content: dbMessage.content
    })
  }

  return {
    id: dbMessage.id,
    sender: {
      id: dbMessage.from_user,
      name: isFromCurrentUser ? 'You' : dbMessage.from_user,
      avatar: dbMessage.from_user.substring(0, 2).toUpperCase()
    },
    text: textContent,
    timestamp: dbMessage.timestamp,
    messageType: dbMessage.message_type as 'text' | 'image' | 'file' | 'system',
    deliveryStatus: dbMessage.status as 'sent' | 'delivered' | 'read' | 'failed',
    attachments: attachments
  }
}

// Convert Message to DbMessage for storage
export function messageToDbMessage(message: Message, status?: string): Omit<DbMessage, 'id'> {
  let messageType = 'text'
  let attachmentData = undefined

  // Check if content is JSON and might contain additional metadata
  try {
    const contentObj = JSON.parse(message.content)
    if (contentObj.messageType) {
      messageType = contentObj.messageType
    }
    if (contentObj.attachments && contentObj.attachments.length > 0) {
      attachmentData = JSON.stringify(contentObj.attachments)
    }
  } catch (error) {
    // Content is not JSON, treat as plain text
  }

  return {
    from_user: message.from,
    to_user: message.to,
    content: message.content,
    timestamp: message.timestamp
      ? new Date(message.timestamp).toISOString()
      : new Date().toISOString(),
    status: status || 'sent',
    signature: message.signature,
    message_type: messageType,
    attachment_data: attachmentData
  }
}
