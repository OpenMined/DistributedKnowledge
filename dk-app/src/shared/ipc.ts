// IPC Interface types for type-safe communication between main and renderer processes
import {
  LLMConfig,
  AppConfig,
  OnboardingStatus,
  OnboardingConfig,
  AppTracker,
  TrackerListResponse,
  TrackerInstallResponse,
  RAGDocument,
  DocumentStats,
  User,
  ChatMessage,
  AIMessage
} from './types'
import * as LLMTypes from './llmTypes'
// Alias types for easier use
type LLMProvider = LLMTypes.LLMProvider
type ChatCompletionRequest = LLMTypes.ChatCompletionRequest
type ChatCompletionResponse = LLMTypes.ChatCompletionResponse
type StreamingChunk = LLMTypes.StreamingChunk
type ProviderConfig = LLMTypes.ProviderConfig

// File attachment interface
export interface FileAttachment {
  path: string
  name: string
  size: number
  type: string
}

// Window IPC interface
export interface WindowAPI {
  minimize: () => void
  maximize: () => void
  close: () => void
  isMaximized: () => Promise<boolean>
}

// Chat IPC interface
export interface ChatAPI {
  getMessages: (userId: number) => Promise<ChatMessage[]>
  getUserInfo: (userId: number) => Promise<User>
  sendMessage: (userId: number, text: string, attachments?: FileAttachment[]) => Promise<boolean>
  markAsRead: (userId: number) => Promise<boolean>
  onNewMessage: (callback: (event: Electron.IpcRendererEvent, message: ChatMessage) => void) => void
  removeNewMessageListener: (
    callback: (event: Electron.IpcRendererEvent, message: ChatMessage) => void
  ) => void
}

// Channel IPC interface
export interface ChannelAPI {
  getMessages: (channelName: string) => Promise<ChatMessage[]>
  sendMessage: (
    channelName: string,
    text: string,
    attachments?: FileAttachment[]
  ) => Promise<boolean>
  sendReply: (
    channelName: string,
    messageId: number,
    text: string,
    attachments?: FileAttachment[]
  ) => Promise<boolean>
  receive: (channel: string, listener: (event: any, ...args: any[]) => void) => void
  removeAllListeners: (channel: string) => void
}

// Sidebar IPC interface
export interface SidebarAPI {
  getUsers: () => Promise<User[]>
  getChannels: () => Promise<string[]>
}

// API Management related interfaces
export interface ApiUser {
  id: string
  name: string
  avatar: string
}

export interface ApiDocument {
  id: string
  name: string
  type: string
}

export interface ApiPolicy {
  rateLimit: string
  dailyQuota: string
}

export interface ApiData {
  id: string
  name: string
  description: string
  users: ApiUser[]
  documents: ApiDocument[]
  policy: ApiPolicy
  active: boolean
}

export interface ApiRequest {
  id: string
  apiName: string
  description: string
  user: ApiUser
  submittedDate: string
  documents: ApiDocument[]
  requiredTrackers: { id: string; name: string }[]
  deniedDate?: string
  denialReason?: string
}

export interface ApiManagement {
  activeApis: ApiData[]
  pendingRequests: ApiRequest[]
  deniedRequests: ApiRequest[]
}

// Apps IPC interface
// Policy interfaces
export interface PolicyRule {
  type: string
  limit: number
  period?: string
  action: string
}

export interface Policy {
  id: string
  name: string
  type: string
  rules: PolicyRule[]
}

export interface AppsAPI {
  getAppTrackers: () => Promise<AppTracker[]>
  toggleAppTracker: (id: string) => Promise<{ success: boolean; appTracker?: AppTracker }>
  getDocumentCount: () => Promise<DocumentStats>
  getDocuments: () => Promise<{ success: boolean; data: ApiDocument[] }>
  cleanupDocuments: () => Promise<boolean>
  installAppTracker: (metadata: Record<string, unknown>) => Promise<boolean>
  updateAppTracker: (id: string) => Promise<boolean>
  uninstallAppTracker: (id: string) => Promise<boolean>
  getAppIconPath: (appId: string, appPath?: string) => Promise<string>
  searchRAGDocuments: (params: { query: string; numResults: number }) => Promise<RAGDocument[]>
  deleteDocument: (filename: string) => Promise<{ success: boolean; message: string }>
  getApiManagement: () => Promise<{ success: boolean; data: ApiManagement }>
  updateApiStatus: (params: {
    id: string
    active: boolean
  }) => Promise<{ success: boolean; message: string }>
  approveApiRequest: (requestId: string) => Promise<{ success: boolean; message: string }>
  denyApiRequest: (params: {
    requestId: string
    reason?: string
  }) => Promise<{ success: boolean; message: string }>
  getPolicies: (params?: {
    type?: string
    active?: boolean
  }) => Promise<{ success: boolean; data: { policies: Policy[] } }>
  getPolicy: (id: string) => Promise<{ success: boolean; data: Policy }>
  getAPIsByPolicy: (
    policyId: string,
    params?: { limit?: number; offset?: number; sort?: string; order?: string }
  ) => Promise<{
    success: boolean
    data: { total: number; limit: number; offset: number; apis: ApiData[] }
  }>
  createPolicy: (policy: {
    name: string
    description: string
    type: string
    rules?: PolicyRule[]
  }) => Promise<{ success: boolean; data?: Policy; message?: string }>
  updatePolicy: (
    id: string,
    updates: { name?: string; description?: string; isActive?: boolean; rules?: PolicyRule[] }
  ) => Promise<{ success: boolean; data?: Policy; message?: string }>
  deletePolicy: (id: string) => Promise<{ success: boolean; message?: string }>
  changeAPIPolicy: (
    apiId: string,
    params: {
      policyId: string
      effectiveImmediately: boolean
      scheduledDate?: Date
      changeReason: string
    }
  ) => Promise<{ success: boolean; message: string }>
  createApi: (apiData: {
    name: string
    description: string
    policyId: string
    documentIds: string[]
    externalUsers: { userId: string; accessLevel: string }[]
    isActive: boolean
  }) => Promise<{ success: boolean; data?: { id: string }; error?: string }>
  deleteApi: (id: string) => Promise<{ success: boolean; message: string }>
}

// Tracker form field
export interface TrackerFormField {
  id: string
  label: string
  type: 'text' | 'password' | 'select' | 'file' | 'checkbox'
  required: boolean
  options?: string[]
  defaultValue?: string | boolean
  description?: string
}

// Trackers IPC interface
export interface TrackersAPI {
  getTemplates: (trackerId: string) => Promise<string[]>
  getDatasets: (trackerId: string) => Promise<string[]>
  getAppFolders: () => Promise<string[]>
  getTrackerForm: (trackerId: string) => Promise<TrackerFormField[]>
  getTrackerConfig: (trackerId: string) => Promise<Record<string, unknown>>
  saveTrackerConfig: (trackerId: string, configData: Record<string, unknown>) => Promise<boolean>
  getAppSourceFiles: (trackerId: string) => Promise<string[]>
  getAppFileContent: (trackerId: string, filePath: string) => Promise<string>
  getAppConfig: () => Promise<AppConfig>
  updateAppConfig: (formValues: Record<string, unknown>) => Promise<boolean>
  uploadTrackerFile: (trackerId: string, filePath: string, variableId: string) => Promise<boolean>
  showFileDialog: (
    trackerId: string,
    variableId: string,
    options?: { extensions?: string[] }
  ) => Promise<string>
}

// Config IPC interface
export interface ConfigAPI {
  get: () => Promise<AppConfig>
  save: (config: Partial<AppConfig>) => Promise<boolean>
  getMCPConfig: () => Promise<{
    mcpServers: {
      [key: string]: {
        command: string
        args: string[]
      }
    }
  }>
  saveMCPConfig: (config: Record<string, unknown>) => Promise<boolean>
}

// Onboarding IPC interface
export interface OnboardingAPI {
  getStatus: () => Promise<OnboardingStatus>
  setStep: (step: number) => Promise<boolean>
  complete: () => Promise<boolean>
  saveConfig: (config: Partial<OnboardingConfig>) => Promise<boolean>
  generateAuthKeys: () => Promise<{ privateKey: string; publicKey: string }>
  checkExternalServices: () => Promise<Record<string, boolean>>
  pullNomicEmbedModel: () => Promise<boolean>
}

// Toast options interface
export interface ToastOptions {
  type?: 'default' | 'success' | 'error' | 'warning' | 'info'
  duration?: number
  dismissable?: boolean
}

// Toast IPC interface
export interface ToastAPI {
  show: (message: string, options?: ToastOptions) => void
}

// Tracker marketplace IPC interface
export interface TrackerMarketplaceAPI {
  getTrackerList: () => Promise<TrackerListResponse>
  installTracker: (trackerId: string) => Promise<TrackerInstallResponse>
}

// LLM IPC interface
export interface LLMAPI {
  // Provider management
  getProviders: () => Promise<string[]>
  getActiveProvider: () => Promise<LLMProvider>
  setActiveProvider: (provider: LLMProvider) => Promise<boolean>

  // Model management
  getModels: () => Promise<string[]>
  getModelsForProvider: (provider: LLMProvider) => Promise<string[]>

  // Message sending
  sendMessage: (request: ChatCompletionRequest) => Promise<ChatCompletionResponse>

  // Streaming support
  streamMessage: (requestId: string, request: ChatCompletionRequest) => void
  onStreamChunk: (
    callback: (event: Electron.IpcRendererEvent, requestId: string, chunk: StreamingChunk) => void
  ) => void
  onStreamComplete: (
    callback: (
      event: Electron.IpcRendererEvent,
      requestId: string,
      response: ChatCompletionResponse
    ) => void
  ) => void
  onStreamError: (
    callback: (event: Electron.IpcRendererEvent, requestId: string, error: string) => void
  ) => void
  removeStreamListeners: (
    chunkCallback: (
      event: Electron.IpcRendererEvent,
      requestId: string,
      chunk: StreamingChunk
    ) => void,
    completeCallback: (
      event: Electron.IpcRendererEvent,
      requestId: string,
      response: ChatCompletionResponse
    ) => void,
    errorCallback: (event: Electron.IpcRendererEvent, requestId: string, error: string) => void
  ) => void

  // Configuration
  getConfig: () => Promise<LLMConfig>
  updateProviderConfig: (provider: LLMProvider, config: Partial<ProviderConfig>) => Promise<boolean>

  // AI Chat History management
  saveAIChatHistory: (messages: AIMessage[]) => Promise<boolean>
  getAIChatHistory: () => Promise<AIMessage[]>
  clearAIChatHistory: () => Promise<boolean>

  // Slash command support
  processCommand: (request: {
    prompt: string
    userId: string
  }) => Promise<{ passthrough: boolean; payload: string }>
  // User mention support
  processMentions: (request: { prompt: string; userId: string }) => Promise<{ payload: string }>
  // Fetch answers for a query string
  fetchAnswers: (request: {
    query: string
  }) => Promise<{ query: string; answers: Record<string, string> }>
  getCommands: () => Promise<{ name: string; summary: string }[]>
}

// Complete API interface for preload
export interface API {
  window: WindowAPI
  chat: ChatAPI
  channel: ChannelAPI
  sidebar: SidebarAPI
  apps: AppsAPI
  trackers: TrackersAPI
  config: ConfigAPI
  onboarding: OnboardingAPI
  toast: ToastAPI
  trackerMarketplace: TrackerMarketplaceAPI
  llm: LLMAPI
}

// ProviderConfig interface is imported from llmTypes.ts
