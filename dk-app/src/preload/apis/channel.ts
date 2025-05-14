import { ipcRenderer } from 'electron'
import { ChannelAPI } from '../../shared/ipc'
import { ChannelChannels } from '../../shared/channels'
import { FileAttachment } from '../../shared/ipc'

export const channelAPI: ChannelAPI = {
  getMessages: (channelName: string) =>
    ipcRenderer.invoke(ChannelChannels.GetMessages, channelName),
  sendMessage: (channelName: string, text: string, attachments?: FileAttachment[]) =>
    ipcRenderer.invoke(ChannelChannels.SendMessage, {
      channelId: channelName,
      text,
      attachments
    }),
  sendReply: (
    channelName: string,
    messageId: number,
    text: string,
    attachments?: FileAttachment[]
  ) =>
    ipcRenderer.invoke(ChannelChannels.SendReply, {
      channelId: channelName,
      messageId,
      text,
      attachments
    }),
  receive: (channel: string, listener: (event: any, ...args: any[]) => void) => {
    ipcRenderer.on(channel, listener)
  },
  removeAllListeners: (channel: string) => {
    ipcRenderer.removeAllListeners(channel)
  }
}
