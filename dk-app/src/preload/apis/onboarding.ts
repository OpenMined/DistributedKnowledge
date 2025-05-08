import { ipcRenderer } from 'electron'
import { OnboardingAPI } from '../../shared/ipc'
import { OnboardingChannels } from '../../shared/channels'

export const onboardingAPI: OnboardingAPI = {
  getStatus: () => ipcRenderer.invoke(OnboardingChannels.GetStatus),
  setStep: (step: number) => ipcRenderer.invoke(OnboardingChannels.SetStep, step),
  complete: () => ipcRenderer.invoke(OnboardingChannels.Complete),
  saveConfig: (config: Record<string, unknown>) =>
    ipcRenderer.invoke(OnboardingChannels.SaveConfig, config),
  generateAuthKeys: () => ipcRenderer.invoke(OnboardingChannels.GenerateAuthKeys),
  checkExternalServices: () => ipcRenderer.invoke(OnboardingChannels.CheckExternalServices),
  pullNomicEmbedModel: () => ipcRenderer.invoke(OnboardingChannels.PullNomicEmbedModel)
}
