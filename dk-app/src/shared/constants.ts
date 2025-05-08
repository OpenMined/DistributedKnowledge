// Constants shared between main and renderer processes

export enum Channels {
  // Window management
  WindowMinimize = 'window-minimize',
  WindowMaximize = 'window-maximize',
  WindowClose = 'window-close',
  WindowIsMaximized = 'window-is-maximized',

  // General
  Ping = 'ping',

  // Tracker Marketplace
  TrackerMarketplaceGetTrackerList = 'tracker-marketplace:get-tracker-list',
  TrackerMarketplaceInstallTracker = 'tracker-marketplace:install-tracker',

  // Chat
  GetChatMessages = 'get-chat-messages',
  GetUserInfo = 'get-user-info',
  SendMessage = 'send-message',
  MarkMessagesAsRead = 'mark-messages-as-read',
  NewMessage = 'new-message',

  // Channels
  GetChannelMessages = 'get-channel-messages',
  SendChannelMessage = 'send-channel-message',
  SendChannelReply = 'send-channel-reply',

  // Sidebar
  GetSidebarUsers = 'get-sidebar-users',
  GetSidebarChannels = 'get-sidebar-channels',

  // Apps Section
  GetAppTrackers = 'get-app-trackers',
  ToggleAppTracker = 'toggle-app-tracker',
  GetDocumentCount = 'get-document-count',
  CleanupDocuments = 'cleanup-documents',
  InstallAppTracker = 'install-app-tracker',
  UpdateAppTracker = 'update-app-tracker',
  UninstallAppTracker = 'uninstall-app-tracker',
  GetAppIconPath = 'get-app-icon-path',
  SearchRAGDocuments = 'search-rag-documents',
  DeleteDocument = 'delete-document',

  // Tracker Scanning
  TriggerTrackerScan = 'trigger-tracker-scan',
  StartTrackerService = 'start-tracker-service',
  StopTrackerService = 'stop-tracker-service',
  GetTrackerTemplates = 'get-tracker-templates',
  GetTrackerDatasets = 'get-tracker-datasets',
  GetTrackerForm = 'get-tracker-form',
  GetTrackerConfig = 'get-tracker-config',
  SaveTrackerConfig = 'save-tracker-config',
  GetAppFolders = 'get-app-folders',
  GetAppSourceFiles = 'get-app-source-files',
  GetAppFileContent = 'get-app-file-content',
  GetAppConfig = 'get-app-config',
  UpdateAppConfig = 'update-app-config',
  UploadTrackerFile = 'upload-tracker-file',
  ShowFileDialog = 'show-file-dialog',

  // Config
  GetConfig = 'get-config',
  SaveConfig = 'save-config',

  // Toast
  ShowToast = 'show-toast',

  // Onboarding
  GetOnboardingStatus = 'onboarding:getStatus',
  SetOnboardingStep = 'onboarding:setStep',
  CompleteOnboarding = 'onboarding:complete',
  SaveOnboardingConfig = 'onboarding:saveConfig',
  GenerateAuthKeys = 'onboarding:generateAuthKeys',
  CheckExternalServices = 'onboarding:checkExternalServices',
  PullNomicEmbedModel = 'onboarding:pullNomicEmbedModel',

  // LLM Services
  LLMGetProviders = 'llm:getProviders',
  LLMGetActiveProvider = 'llm:getActiveProvider',
  LLMSetActiveProvider = 'llm:setActiveProvider',
  LLMGetModels = 'llm:getModels',
  LLMGetModelsForProvider = 'llm:getModelsForProvider',
  LLMSendMessage = 'llm:sendMessage',
  LLMStreamMessage = 'llm:streamMessage',
  LLMStreamChunk = 'llm:streamChunk',
  LLMStreamComplete = 'llm:streamComplete',
  LLMStreamError = 'llm:streamError',
  LLMGetConfig = 'llm:getConfig',
  LLMUpdateProviderConfig = 'llm:updateProviderConfig'
}

export enum MessageType {
  Text = 'text',
  Image = 'image',
  File = 'file',
  System = 'system'
}

export enum DeliveryStatus {
  Sent = 'sent',
  Delivered = 'delivered',
  Read = 'read',
  Failed = 'failed'
}
