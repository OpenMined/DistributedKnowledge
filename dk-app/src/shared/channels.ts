// Define all IPC channel names as constants to avoid string literals and typos

export enum WindowChannels {
  Minimize = 'window-minimize',
  Maximize = 'window-maximize',
  Close = 'window-close',
  IsMaximized = 'window-is-maximized'
}

export enum ChatChannels {
  GetMessages = 'get-chat-messages',
  GetUserInfo = 'get-user-info',
  SendMessage = 'send-message',
  MarkAsRead = 'mark-messages-as-read',
  NewMessage = 'new-message'
}

export enum ChannelChannels {
  GetMessages = 'get-channel-messages',
  SendMessage = 'send-channel-message',
  SendReply = 'send-channel-reply'
}

export enum SidebarChannels {
  GetUsers = 'get-sidebar-users',
  GetChannels = 'get-sidebar-channels'
}

export enum AppsChannels {
  GetAppTrackers = 'get-app-trackers',
  ToggleAppTracker = 'toggle-app-tracker',
  GetDocumentCount = 'get-document-count',
  CleanupDocuments = 'cleanup-documents',
  InstallAppTracker = 'install-app-tracker',
  UpdateAppTracker = 'update-app-tracker',
  UninstallAppTracker = 'uninstall-app-tracker',
  GetAppIconPath = 'get-app-icon-path',
  SearchRAGDocuments = 'search-rag-documents',
  DeleteDocument = 'delete-document'
}

export enum TrackerChannels {
  GetTemplates = 'get-tracker-templates',
  GetDatasets = 'get-tracker-datasets',
  GetAppFolders = 'get-app-folders',
  GetTrackerForm = 'get-tracker-form',
  GetTrackerConfig = 'get-tracker-config',
  SaveTrackerConfig = 'save-tracker-config',
  GetAppSourceFiles = 'get-app-source-files',
  GetAppFileContent = 'get-app-file-content',
  GetAppConfig = 'get-app-config',
  UpdateAppConfig = 'update-app-config',
  UploadTrackerFile = 'upload-tracker-file',
  ShowFileDialog = 'show-file-dialog'
}

export enum ConfigChannels {
  Get = 'get-config',
  Save = 'save-config'
}

export enum OnboardingChannels {
  GetStatus = 'onboarding:getStatus',
  SetStep = 'onboarding:setStep',
  Complete = 'onboarding:complete',
  SaveConfig = 'onboarding:saveConfig',
  GenerateAuthKeys = 'onboarding:generateAuthKeys',
  CheckExternalServices = 'onboarding:checkExternalServices',
  PullNomicEmbedModel = 'onboarding:pullNomicEmbedModel'
}

export enum ToastChannels {
  Show = 'show-toast'
}

export enum TrackerMarketplaceChannels {
  GetTrackerList = 'tracker-marketplace:get-tracker-list',
  InstallTracker = 'tracker-marketplace:install-tracker'
}

export enum LLMChannels {
  GetProviders = 'llm:getProviders',
  GetActiveProvider = 'llm:getActiveProvider',
  SetActiveProvider = 'llm:setActiveProvider',
  GetModels = 'llm:getModels',
  GetModelsForProvider = 'llm:getModelsForProvider',
  SendMessage = 'llm:sendMessage',
  StreamMessage = 'llm:streamMessage',
  StreamChunk = 'llm:streamChunk',
  StreamComplete = 'llm:streamComplete',
  StreamError = 'llm:streamError',
  GetConfig = 'llm:getConfig',
  UpdateProviderConfig = 'llm:updateProviderConfig',
  SaveAIChatHistory = 'llm:saveAIChatHistory',
  GetAIChatHistory = 'llm:getAIChatHistory',
  ClearAIChatHistory = 'llm:clearAIChatHistory',
  ProcessCommand = 'llm:processCommand',
  GetCommands = 'llm:getCommands'
}
