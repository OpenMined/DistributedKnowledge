import { ElectronAPI } from '@electron-toolkit/preload'

// Message attachment interface
interface MessageAttachment {
  id: string
  type: 'image' | 'video' | 'audio' | 'document' | 'other'
  url: string
  filename: string
  size?: number
  mimeType?: string
  thumbnailUrl?: string
  width?: number
  height?: number
  duration?: number
}

// Enhanced message interface
interface EnhancedMessage {
  id: number
  sender: {
    id: number | string
    name: string
    avatar: string
    online?: boolean
  }
  text: string
  timestamp: string | Date
  messageType?: string
  deliveryStatus?: 'sent' | 'delivered' | 'read' | 'failed'
  attachments?: MessageAttachment[]
  replyTo?: number
  metadata?: {
    clientId?: string
    mentions?: Array<string | number>
    isForwarded?: boolean
    originalMessageId?: number
    customData?: Record<string, any>
  }
}

// LLM types
interface ChatMessage {
  role: 'system' | 'user' | 'assistant'
  content: string
}

interface ChatCompletionRequest {
  messages: ChatMessage[]
  model?: string
  temperature?: number
  maxTokens?: number
  stream?: boolean
}

interface ChatCompletionResponse {
  id: string
  object: string
  created: number
  model: string
  message: ChatMessage
  usage?: {
    promptTokens: number
    completionTokens: number
    totalTokens: number
  }
}

interface StreamingChunk {
  id: string
  object: string
  created: number
  model: string
  delta: Partial<ChatMessage>
  finishReason: string | null
}

interface ProviderConfig {
  apiKey: string
  baseUrl?: string
  defaultModel: string
  models: string[]
}

enum LLMProvider {
  ANTHROPIC = 'anthropic',
  OPENAI = 'openai',
  OLLAMA = 'ollama',
  GEMINI = 'gemini'
}

declare global {
  interface Window {
    electron: ElectronAPI
    api: {
      window: {
        minimize: () => void
        maximize: () => void
        close: () => void
        isMaximized: () => Promise<boolean>
      }
      chat: {
        getMessages: (userId: number) => Promise<EnhancedMessage[]>
        getUserInfo: (userId: number) => Promise<{
          id: number
          name: string
          avatar: string
          online?: boolean
          status?: string
        } | null>
        sendMessage: (
          userId: number,
          text: string,
          attachments?: MessageAttachment[]
        ) => Promise<{
          success: boolean
          message?: EnhancedMessage
          error?: string
        }>
        markAsRead: (userId: number) => Promise<{
          success: boolean
          error?: string
        }>
        onNewMessage: (
          callback: (
            event: Electron.IpcRendererEvent,
            data: {
              cacheKey: string
              message: EnhancedMessage
            }
          ) => void
        ) => void
        removeNewMessageListener: (
          callback: (
            event: Electron.IpcRendererEvent,
            data: {
              cacheKey: string
              message: EnhancedMessage
            }
          ) => void
        ) => void
      }
      channel: {
        getMessages: (channelName: string) => Promise<
          Array<
            EnhancedMessage & {
              replies: EnhancedMessage[]
              replyCount: number
            }
          >
        >
        sendMessage: (
          channelName: string,
          text: string,
          attachments?: MessageAttachment[]
        ) => Promise<{
          success: boolean
          message?: EnhancedMessage & {
            replies: Array<any>
            replyCount: number
          }
          error?: string
        }>
        sendReply: (
          channelName: string,
          messageId: number,
          text: string,
          attachments?: MessageAttachment[]
        ) => Promise<{
          success: boolean
          reply?: EnhancedMessage
          error?: string
        }>
      }
      sidebar: {
        getUsers: () => Promise<
          Array<{
            id: number | string
            name: string
            online: boolean
            status?: string
          }>
        >
        getChannels: () => Promise<
          Array<{
            id: string
            name: string
            description?: string
          }>
        >
      }
      apps: {
        getAppTrackers: () => Promise<{
          success: boolean
          appTrackers?: Array<{
            id: number
            name: string
            description: string
            version: string
            enabled: boolean
            icon?: string
            hasUpdate?: boolean
            updateVersion?: string
          }>
          error?: string
        }>
        toggleAppTracker: (id: number) => Promise<{
          success: boolean
          appTracker?: {
            id: number
            name: string
            description: string
            version: string
            enabled: boolean
            icon?: string
            hasUpdate?: boolean
            updateVersion?: string
          }
          error?: string
        }>
        getDocumentCount: () => Promise<{
          success: boolean
          stats?: {
            count: number
          }
          error?: string
        }>
        cleanupDocuments: () => Promise<{
          success: boolean
          message?: string
          error?: string
        }>
        installAppTracker: (metadata: any) => Promise<{
          success: boolean
          message?: string
          error?: string
        }>
        updateAppTracker: (id: number) => Promise<{
          success: boolean
          appTracker?: {
            id: number
            name: string
            description: string
            version: string
            enabled: boolean
            icon?: string
            hasUpdate?: boolean
            updateVersion?: string
          }
          message?: string
          error?: string
        }>
        uninstallAppTracker: (id: number) => Promise<{
          success: boolean
          message?: string
          error?: string
        }>
        getAppIconPath: (appId: string) => Promise<string | null>
        searchRAGDocuments: (params: { query: string; numResults: number }) => Promise<{
          success: boolean
          results?: {
            documents: Array<{
              content: string
              file: string
              score: number
              metadata: Record<string, any>
            }>
          }
          error?: string
        }>
      }
      trackers: {
        getTemplates: (trackerId: string) => Promise<{
          success: boolean
          templates?: Record<string, { name: string; content: string; filename: string }>
          error?: string
        }>
        getDatasets: (trackerId: string) => Promise<{
          success: boolean
          datasets?: Record<string, string>
          error?: string
        }>
        getAppFolders: () => Promise<{
          success: boolean
          folders: string[]
          error?: string
        }>
        getTrackerForm: (trackerId: string) => Promise<{
          success: boolean
          form?: {
            formTitle: string
            fields: Array<{
              id: string
              variable_id: string
              type: string
              title?: string
              label?: string
              description?: string
              required?: boolean
              placeholder?: string
              min?: number
              max?: number
              accept?: string
              options?: Array<{
                value: string
                label: string
              }>
            }>
          }
          error?: string
        }>
        getTrackerConfig: (trackerId: string) => Promise<{
          success: boolean
          config?: Record<string, any>
          error?: string
        }>
        saveTrackerConfig: (
          trackerId: string,
          configData: Record<string, any>
        ) => Promise<{
          success: boolean
          message?: string
          error?: string
        }>
        getAppSourceFiles: (trackerId: string) => Promise<{
          success: boolean
          files?: Array<{
            name: string
            path: string
            type: 'file' | 'directory'
            children?: Array<{
              name: string
              path: string
              type: 'file' | 'directory'
              children?: Array<any>
            }>
          }>
          error?: string
        }>
        getAppFileContent: (
          trackerId: string,
          filePath: string
        ) => Promise<{
          success: boolean
          content?: string
          error?: string
        }>
        getAppConfig: () => Promise<{
          success: boolean
          config?: Record<string, any>
          error?: string
        }>
        updateAppConfig: (formValues: Record<string, any>) => Promise<{
          success: boolean
          message?: string
          error?: string
        }>
        uploadTrackerFile: (
          trackerId: string,
          filePath: string,
          variableId: string
        ) => Promise<{
          success: boolean
          filePath?: string
          fileName?: string
          message?: string
          error?: string
        }>
        showFileDialog: (
          trackerId: string,
          variableId: string,
          options?: { extensions?: string[] }
        ) => Promise<{
          success: boolean
          filePath?: string
          fileName?: string
          message?: string
          error?: string
          canceled?: boolean
        }>
      }
      config: {
        get: () => Promise<{
          serverURL: string
          userID: string
          isConnected: boolean
        }>
        save: (config: any) => Promise<{
          success: boolean
          error?: string
        }>
      }
      onboarding: {
        getStatus: () => Promise<{
          success: boolean
          status?: {
            isFirstRun: boolean
            currentStep: number
            totalSteps: number
            completed: boolean
          }
          error?: string
        }>
        setStep: (step: number) => Promise<{
          success: boolean
          currentStep: number
          error?: string
        }>
        complete: () => Promise<{
          success: boolean
          error?: string
        }>
        saveConfig: (config: {
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
        }) => Promise<{
          success: boolean
          error?: string
        }>
        generateAuthKeys: () => Promise<{
          success: boolean
          keys?: {
            private_key: string
            public_key: string
          }
          error?: string
        }>
        checkExternalServices: () => Promise<{
          success: boolean
          status?: {
            ollama: boolean
            syftbox: boolean
            nomicEmbedModel: boolean
          }
          error?: string
        }>
        pullNomicEmbedModel: () => Promise<{
          success: boolean
          message?: string
          error?: string
        }>
      }
      toast: {
        show: (
          message: string,
          options?: {
            type?: 'default' | 'success' | 'error' | 'warning' | 'info'
            title?: string
            duration?: number
            template?: 'simple' | 'action'
          }
        ) => void
      }
      trackerMarketplace: {
        getTrackerList: () => Promise<{
          success: boolean
          trackers?: Array<{
            id: string
            name: string
            version: string
            description: string
            iconPath: string
            developer: string
            verified: boolean
            featured: boolean
          }>
          error?: string
        }>
        installTracker: (trackerId: string) => Promise<{
          success: boolean
          message?: string
          error?: string
          trackerId?: string
        }>
      }
      llm: {
        // Provider management
        getProviders: () => Promise<LLMProvider[]>
        getActiveProvider: () => Promise<LLMProvider>
        setActiveProvider: (provider: LLMProvider) => Promise<boolean>

        // Model management
        getModels: () => Promise<string[]>
        getModelsForProvider: (provider: LLMProvider) => Promise<string[]>

        // Message sending
        sendMessage: (request: ChatCompletionRequest) => Promise<ChatCompletionResponse>

        // Streaming support
        streamMessage: (requestId: string, request: ChatCompletionRequest) => void
        onStreamChunk: (callback: (requestId: string, chunk: StreamingChunk) => void) => void
        onStreamComplete: (
          callback: (requestId: string, response: ChatCompletionResponse) => void
        ) => void
        onStreamError: (callback: (requestId: string, error: string) => void) => void
        removeStreamListeners: (
          chunkCallback: (requestId: string, chunk: StreamingChunk) => void,
          completeCallback: (requestId: string, response: ChatCompletionResponse) => void,
          errorCallback: (requestId: string, error: string) => void
        ) => void

        // Configuration
        getConfig: () => Promise<{
          activeProvider: LLMProvider
          providers: {
            [key in LLMProvider]?: ProviderConfig
          }
        }>
        updateProviderConfig: (
          provider: LLMProvider,
          config: Partial<ProviderConfig>
        ) => Promise<boolean>
      }
    }
  }
}
