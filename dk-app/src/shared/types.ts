// Types shared between main and renderer processes

// AI Chat History message interface for the renderer
// Ensure this is properly exported
export interface AIMessage {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  timestamp: Date
  isLoading?: boolean
}

// Message represents the structure of messages exchanged with the server.
export interface Message {
  id?: number
  from: string
  to: string
  timestamp?: Date
  content: string
  status?: string
  signature?: string // Base64-encoded signature of message content
}

// EncryptedMessage is the structure that will be marshaled into the Message.Content field
// for direct messages. It contains the envelope (asymmetrically encrypted symmetric key)
// and the symmetrically encrypted message content.
export interface EncryptedMessage {
  // Data to allow the receiver to recover the AES key.
  ephemeral_public_key: string
  key_nonce: string
  encrypted_key: string
  // Data for AES-GCM encryption of the message content.
  data_nonce: string
  encrypted_content: string
}

export interface User {
  id: number | string
  name: string
  avatar?: string
  online?: boolean
  status?: 'online' | 'away' | 'busy' | 'offline'
  lastSeen?: Date
}

export interface Channel {
  id: string
  name: string
  description?: string
  members?: Array<string | number>
  isPrivate?: boolean
  createdAt?: Date
}

// Enhanced ChatMessage interface to support all message types
export interface ChatMessage {
  id: number
  sender: User
  text: string
  timestamp: string | Date
  messageType?: 'text' | 'image' | 'file' | 'system'
  replyTo?: number // ID of the message this is replying to
  replies?: ChatMessage[]
  replyCount?: number
  attachments?: MessageAttachment[]
  isEdited?: boolean
  editHistory?: EditHistory[]
  deliveryStatus?: 'sent' | 'delivered' | 'read' | 'failed'
  metadata?: MessageMetadata
}

// File or image attachment
export interface MessageAttachment {
  id: string
  type: 'image' | 'video' | 'audio' | 'document' | 'other'
  url: string
  filename: string
  size?: number
  mimeType?: string
  thumbnailUrl?: string
  width?: number
  height?: number
  duration?: number // For audio/video
}

// History of edits to a message
export interface EditHistory {
  text: string
  timestamp: Date
}

// Additional metadata for messages
export interface MessageMetadata {
  clientId?: string // Client-generated message ID for deduplication
  mentions?: Array<string | number> // User IDs mentioned in the message
  tags?: string[]
  geolocation?: {
    latitude: number
    longitude: number
    locationName?: string
  }
  isForwarded?: boolean
  originalMessageId?: number
  customData?: Record<string, any> // For application-specific data
}

// Message content for wrapping in the Message interface
export interface MessageContent {
  text: string
  messageType?: 'text' | 'image' | 'file' | 'system'
  replyTo?: number
  attachments?: MessageAttachment[]
  metadata?: MessageMetadata
}

// Import from shared LLM types
import { LLMProvider, ProviderConfig } from './llmTypes'

// Configuration for all LLM providers
export interface LLMConfig {
  activeProvider: LLMProvider
  providers: {
    [key in LLMProvider]?: ProviderConfig
  }
}

export interface DKConfig {
  dk: string
  project_path: string
  http_port: string
}

export interface AppConfig {
  serverURL: string
  userID: string
  private_key?: string
  public_key?: string
  isConnected?: boolean
  llm?: LLMConfig
  database?: {
    path: string
  }
  syftbox_config?: string
  dk_config?: DKConfig
  dk_api?: string
}

// App tracker interface
export interface AppTracker {
  id: string // Changed from number to string (will be derived from app path)
  name: string
  description: string
  version: string
  enabled: boolean
  icon?: string
  hasUpdate?: boolean
  updateVersion?: string
  path?: string
}

// Document stats interface
export interface DocumentStats {
  count: number
  error?: string
}

// RAG Document interface that matches the server response
export interface RAGDocument {
  content: string
  file: string
  score: number
  metadata: Record<string, any> // Flexible metadata structure
}

export interface UserStatusResponse {
  active_users: string[]
  inactive_users: string[]
}

// Toast types for notifications
export interface ToastOptions {
  type?: 'default' | 'success' | 'error' | 'warning' | 'info'
  title?: string
  duration?: number
  template?: 'simple' | 'action'
  action?: {
    label: string
    onClick: () => void
  }
  onDismiss?: () => void
}

// Onboarding wizard types
export interface OnboardingStatus {
  isFirstRun: boolean
  currentStep: number
  totalSteps: number
  completed: boolean
}

export interface OnboardingConfig {
  serverURL: string
  userID: string
  private_key?: string
  public_key?: string
  llm?: {
    activeProvider: string
    providers: {
      [key: string]: {
        apiKey?: string
        baseUrl?: string
        defaultModel: string
        models: string[]
      }
    }
  }
}

// Type declarations for modules without proper type definitions
declare module 'ed2curve' {
  export function convertPublicKey(curve25519Key: Uint8Array): Uint8Array | null
  export function convertSecretKey(curve25519Key: Uint8Array): Uint8Array
}

// Tracker Marketplace Types
export interface TrackerListItem {
  id: string
  name: string
  version: string
  description: string
  iconPath: string
  developer: string
  verified: boolean
  featured: boolean
}

export interface TrackerListResponse {
  success: boolean
  trackers?: TrackerListItem[]
  error?: string
}

export interface TrackerInstallResponse {
  success: boolean
  message?: string
  error?: string
  trackerId?: string
}

// API response for getting onboarding status
export interface OnboardingStatusResponse {
  success: boolean
  status?: OnboardingStatus
  configExists?: boolean // Added flag to indicate if config.json exists
  error?: string
}
