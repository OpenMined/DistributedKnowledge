import { ipcRenderer } from 'electron'
import { LLMAPI } from '@shared/ipc'
import { LLMChannels } from '@shared/channels'
import * as LLMTypes from '@shared/llmTypes'
// Alias types for easier use
type ChatCompletionRequest = LLMTypes.ChatCompletionRequest
type StreamingChunk = LLMTypes.StreamingChunk
type ChatCompletionResponse = LLMTypes.ChatCompletionResponse
type LLMProvider = LLMTypes.LLMProvider
type ProviderConfig = LLMTypes.ProviderConfig
import * as SharedTypes from '@shared/types'
type AIMessage = SharedTypes.AIMessage

export const llmAPI: LLMAPI = {
  // Provider management
  getProviders: () => ipcRenderer.invoke(LLMChannels.GetProviders),
  getActiveProvider: () => ipcRenderer.invoke(LLMChannels.GetActiveProvider),
  setActiveProvider: (provider: LLMTypes.LLMProvider) =>
    ipcRenderer.invoke(LLMChannels.SetActiveProvider, provider),

  // Model management
  getModels: () => ipcRenderer.invoke(LLMChannels.GetModels),
  getModelsForProvider: (provider: LLMTypes.LLMProvider) =>
    ipcRenderer.invoke(LLMChannels.GetModelsForProvider, provider),

  // Message sending
  sendMessage: (request: LLMTypes.ChatCompletionRequest) =>
    ipcRenderer.invoke(LLMChannels.SendMessage, request),

  // Streaming support
  streamMessage: (requestId: string, request: LLMTypes.ChatCompletionRequest) =>
    ipcRenderer.send(LLMChannels.StreamMessage, requestId, request),
  onStreamChunk: (
    callback: (
      event: Electron.IpcRendererEvent,
      requestId: string,
      chunk: LLMTypes.StreamingChunk
    ) => void
  ) =>
    ipcRenderer.on(
      LLMChannels.StreamChunk,
      (event: Electron.IpcRendererEvent, requestId: string, chunk: LLMTypes.StreamingChunk) =>
        callback(event, requestId, chunk)
    ),
  onStreamComplete: (
    callback: (
      event: Electron.IpcRendererEvent,
      requestId: string,
      response: LLMTypes.ChatCompletionResponse
    ) => void
  ) =>
    ipcRenderer.on(
      LLMChannels.StreamComplete,
      (
        event: Electron.IpcRendererEvent,
        requestId: string,
        response: LLMTypes.ChatCompletionResponse
      ) => callback(event, requestId, response)
    ),
  onStreamError: (
    callback: (event: Electron.IpcRendererEvent, requestId: string, error: string) => void
  ) =>
    ipcRenderer.on(
      LLMChannels.StreamError,
      (event: Electron.IpcRendererEvent, requestId: string, error: string) =>
        callback(event, requestId, error)
    ),
  removeStreamListeners: (
    chunkCallback: (
      event: Electron.IpcRendererEvent,
      requestId: string,
      chunk: LLMTypes.StreamingChunk
    ) => void,
    completeCallback: (
      event: Electron.IpcRendererEvent,
      requestId: string,
      response: LLMTypes.ChatCompletionResponse
    ) => void,
    errorCallback: (event: Electron.IpcRendererEvent, requestId: string, error: string) => void
  ) => {
    ipcRenderer.removeListener(LLMChannels.StreamChunk, chunkCallback)
    ipcRenderer.removeListener(LLMChannels.StreamComplete, completeCallback)
    ipcRenderer.removeListener(LLMChannels.StreamError, errorCallback)
  },

  // Configuration
  getConfig: () => ipcRenderer.invoke(LLMChannels.GetConfig),
  updateProviderConfig: (
    provider: LLMTypes.LLMProvider,
    config: Partial<LLMTypes.ProviderConfig>
  ) => ipcRenderer.invoke(LLMChannels.UpdateProviderConfig, provider, config),

  // AI Chat History management
  saveAIChatHistory: (messages: AIMessage[]) =>
    ipcRenderer.invoke(LLMChannels.SaveAIChatHistory, messages),
  getAIChatHistory: () => ipcRenderer.invoke(LLMChannels.GetAIChatHistory),
  clearAIChatHistory: () => ipcRenderer.invoke(LLMChannels.ClearAIChatHistory),

  // Slash command support
  processCommand: (request: { prompt: string; userId: string }) =>
    ipcRenderer.invoke(LLMChannels.ProcessCommand, request),
  // User mention support
  processMentions: (request: { prompt: string; userId: string }) => {
    const timestamp = new Date().toISOString()
    console.log(`[MENTIONS_PRELOAD] ${timestamp} - START: Sending processMentions request`)
    console.log(
      `[MENTIONS_PRELOAD] Request details - prompt: "${request.prompt}", userId: "${request.userId}"`
    )

    return ipcRenderer
      .invoke(LLMChannels.ProcessMentions, request)
      .then((result) => {
        const responseTime = new Date().toISOString()
        console.log(`[MENTIONS_PRELOAD] ${responseTime} - SUCCESS: Received processMentions result`)
        console.log(`[MENTIONS_PRELOAD] Result type: ${typeof result}`)
        console.log(`[MENTIONS_PRELOAD] Result value: ${JSON.stringify(result)}`)

        // Additional debug info about the payload
        if (result && typeof result === 'object') {
          if (result.payload) {
            console.log(`[MENTIONS_PRELOAD] Payload found: "${result.payload}"`)
          } else {
            console.log(`[MENTIONS_PRELOAD] No payload property in result object`)
          }
        }

        return result
      })
      .catch((err) => {
        const errorTime = new Date().toISOString()
        console.error(`[MENTIONS_PRELOAD] ${errorTime} - ERROR: processMentions failed`)
        console.error(`[MENTIONS_PRELOAD] Error message: ${err.message || 'Unknown error'}`)
        console.error(`[MENTIONS_PRELOAD] Error stack: ${err.stack || 'No stack trace'}`)

        return {
          payload: `Error in mention processing: ${err.message || 'Unknown error'}`,
          error: true,
          timestamp: errorTime
        }
      })
      .finally(() => {
        console.log(
          `[MENTIONS_PRELOAD] ${new Date().toISOString()} - END: processMentions request completed`
        )
      })
  },
  // Fetch answers for a mention query using IPC instead of direct fetch
  fetchAnswers: (request: { query: string }) => {
    console.log(`[MENTIONS_FETCH_IPC] Requesting answers for query: "${request.query}"`)

    return ipcRenderer
      .invoke(LLMChannels.FetchAnswers, request)
      .then((result) => {
        console.log(`[MENTIONS_FETCH_IPC] Success - received answers:`, result)
        return result
      })
      .catch((err) => {
        console.error(`[MENTIONS_FETCH_IPC] Error fetching answers:`, err)
        throw err
      })
  },
  getCommands: () => ipcRenderer.invoke(LLMChannels.GetCommands)
}
