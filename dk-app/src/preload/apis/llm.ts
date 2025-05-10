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
  getCommands: () => ipcRenderer.invoke(LLMChannels.GetCommands)
}
