import { LLMProvider, ProviderConfig } from '@shared/llmTypes'
import * as SharedTypes from '@shared/types'

// Reuse LLMConfig from shared types
type LLMConfig = SharedTypes.LLMConfig

// Default configuration
export const defaultLLMConfig: LLMConfig = {
  activeProvider: LLMProvider.ANTHROPIC,
  providers: {
    [LLMProvider.ANTHROPIC]: {
      apiKey: '',
      defaultModel: 'claude-3-opus-20240229',
      models: ['claude-3-opus-20240229', 'claude-3-sonnet-20240229', 'claude-3-haiku-20240307']
    },
    [LLMProvider.OPENAI]: {
      apiKey: '',
      defaultModel: 'gpt-4o',
      models: ['gpt-4.1-nano', 'gpt-4.1-mini', 'gpt-4.1', 'gpt-4o', 'gpt-4o-mini']
    },
    [LLMProvider.GEMINI]: {
      apiKey: '',
      defaultModel: 'gemini-2.5-pro-preview',
      models: ['gemini-2.5-pro-preview', 'gemini-2.5-flash-preview']
    },
    [LLMProvider.OLLAMA]: {
      apiKey: '', // Not used for Ollama, but kept for consistency
      baseUrl: 'http://localhost:11434',
      defaultModel: 'gemma3:4b',
      models: ['gemma3:4b', 'gemma:2b', 'qwen2.5:latest']
    },
    [LLMProvider.OPENROUTER]: {
      apiKey: '',
      baseUrl: 'https://openrouter.ai/api',
      defaultModel: 'anthropic/claude-3-opus',
      models: [
        'anthropic/claude-3.7-sonnet',
        'anthropic/claude-3.5-sonnet',
        'anthropic/claude-3-opus',
        'anthropic/claude-3-sonnet',
        'anthropic/claude-3-haiku',
        'openai/gpt-4o',
        'openai/gpt-4.1',
        'mistralai/mistral-large',
        'google/gemini-2.5-pro-preview',
        'google/gemini-2.5-flash-preview',
        'meta/llama-3-70b'
      ]
    }
  }
}
