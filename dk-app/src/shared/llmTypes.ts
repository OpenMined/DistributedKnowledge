// Message types for LLM conversations

export interface ChatMessage {
  role: 'system' | 'user' | 'assistant' | 'tool';
  content: string;
  tool_call_id?: string;
  tool_calls?: ToolCall[];
}

export interface ToolCall {
  id: string;
  type: 'function';
  function: {
    name: string;
    arguments: string;
  }
}


export interface ChatCompletionRequest {
  messages: ChatMessage[]
  model?: string
  temperature?: number
  maxTokens?: number
  stream?: boolean
}

export interface ChatCompletionResponse {
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

export interface StreamingChunk {
  id: string
  object: string
  created: number
  model: string
  delta: Partial<ChatMessage>
  finishReason: string | null
}

export interface ProviderConfig {
  apiKey: string
  baseUrl?: string
  defaultModel: string
  models: string[]
}

export enum LLMProvider {
  ANTHROPIC = 'anthropic',
  OPENAI = 'openai',
  OLLAMA = 'ollama',
  GEMINI = 'gemini'
}

export interface LLMProviderInterface {
  provider: LLMProvider
  getModels(): Promise<string[]>
  sendMessage(request: ChatCompletionRequest): Promise<ChatCompletionResponse>
  streamMessage(
    request: ChatCompletionRequest,
    onChunk: (chunk: StreamingChunk) => void,
    onComplete: (fullResponse: ChatCompletionResponse) => void,
    onError: (error: Error) => void,
    requestId?: string
  ): Promise<void>
}
