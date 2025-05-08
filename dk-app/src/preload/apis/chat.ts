import { ipcRenderer } from 'electron'
import { ChatAPI } from '../../shared/ipc'
import { ChatChannels } from '../../shared/channels'
import { FileAttachment } from '../../shared/ipc'

export const chatAPI: ChatAPI = {
  getMessages: (userId: number) => ipcRenderer.invoke(ChatChannels.GetMessages, userId),
  getUserInfo: (userId: number) => ipcRenderer.invoke(ChatChannels.GetUserInfo, userId),
  sendMessage: (userId: number, text: string, attachments?: FileAttachment[]) =>
    ipcRenderer.invoke(ChatChannels.SendMessage, { userId, text, attachments }),
  markAsRead: (userId: number) => ipcRenderer.invoke(ChatChannels.MarkAsRead, userId),
  onNewMessage: (callback) => ipcRenderer.on(ChatChannels.NewMessage, callback),
  removeNewMessageListener: (callback) =>
    ipcRenderer.removeListener(ChatChannels.NewMessage, callback)
}
