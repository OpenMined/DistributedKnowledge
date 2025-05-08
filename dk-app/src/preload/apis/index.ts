import { API } from '@shared/ipc'
import { windowAPI } from './window'
import { chatAPI } from './chat'
import { channelAPI } from './channel'
import { sidebarAPI } from './sidebar'
import { appsAPI } from './apps'
import { trackersAPI } from './trackers'
import { configAPI } from './config'
import { onboardingAPI } from './onboarding'
import { toastAPI } from './toast'
import { trackerMarketplaceAPI } from './trackerMarketplace'
import { llmAPI } from './llm'

export const api: API = {
  window: windowAPI,
  chat: chatAPI,
  channel: channelAPI,
  sidebar: sidebarAPI,
  apps: appsAPI,
  trackers: trackersAPI,
  config: configAPI,
  onboarding: onboardingAPI,
  toast: toastAPI,
  trackerMarketplace: trackerMarketplaceAPI,
  llm: llmAPI
}
