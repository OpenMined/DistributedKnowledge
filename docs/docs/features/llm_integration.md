# LLM Integration

Distributed Knowledge supports multiple Large Language Model (LLM) providers, enabling flexibility in choosing the right model for your needs. This document explains how LLM integration works and how to configure different providers.

## Supported Providers

The system currently supports three major LLM providers:

1. **Anthropic (Claude)**: Advanced reasoning and conversation capabilities
2. **OpenAI (GPT)**: Versatile models with broad knowledge
3. **Ollama**: Local, open-source models for privacy and cost efficiency

## Provider Integration Architecture

The LLM integration is implemented through a provider abstraction layer:

- **Factory Pattern**: `llm_factory.go` creates appropriate client based on configuration
- **Provider-Specific Clients**: Separate implementations for each provider
- **Common Interface**: Unified method for generating responses
- **Configuration Management**: JSON-based configuration for each provider

## Configuration Format

All LLM providers are configured using a standardized JSON format:

```json
{
  "provider": "provider_name",
  "api_key": "your_api_key",
  "model": "model_name",
  "base_url": "https://api.provider.com/endpoint",
  "parameters": {
    "temperature": 0.7,
    "max_tokens": 1000
  },
  "headers": {
    "custom-header": "value"
  }
}
```

### Common Parameters

Some parameters are common across all providers:

- `provider`: Specifies which provider to use
- `model`: The specific model to use from the provider
- `parameters`: Model-specific settings like temperature and token limit

### Provider-Specific Parameters

Each provider may require additional configuration:

- `api_key`: Authentication key for API access (Anthropic, OpenAI)
- `base_url`: Custom endpoint URL (useful for proxies or self-hosted models)
- `headers`: Additional HTTP headers for API requests

## Provider-Specific Configuration

### Anthropic (Claude)

```json
{
  "provider": "anthropic",
  "api_key": "sk-ant-your-anthropic-key",
  "model": "claude-3-sonnet-20240229",
  "parameters": {
    "temperature": 0.7,
    "max_tokens": 1000
  }
}
```

**Available Models:**
- `claude-3-opus-20240229`: Highest capability model
- `claude-3-sonnet-20240229`: Balanced capability and performance
- `claude-3-haiku-20240307`: Fastest, most efficient model

### OpenAI (GPT)

```json
{
  "provider": "openai",
  "api_key": "sk-your-openai-key",
  "model": "gpt-4",
  "parameters": {
    "temperature": 0.7,
    "max_tokens": 2000
  }
}
```

**Available Models:**
- `gpt-4`: High-capability model
- `gpt-4-turbo`: Faster version with slight quality tradeoffs
- `gpt-3.5-turbo`: Balanced performance and cost

### Ollama

```json
{
  "provider": "ollama",
  "model": "llama3",
  "base_url": "http://localhost:11434/api/generate",
  "parameters": {
    "temperature": 0.7,
    "max_tokens": 2000
  }
}
```

**Available Models:**
- Depends on models installed in your Ollama instance
- Common options include: `llama3`, `mistral`, `vicuna`

## Implementation Details

### Provider Factory

The system uses a factory pattern to create the appropriate LLM client:

```go
// In llm_factory.go
func NewLLMClient(config LLMConfig) (LLMClient, error) {
    switch config.Provider {
    case "anthropic":
        return NewAnthropicClient(config)
    case "openai":
        return NewOpenAIClient(config)
    case "ollama":
        return NewOllamaClient(config)
    default:
        return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
    }
}
```

### Anthropic Implementation

The Anthropic integration uses Claude's API:

```go
// In llm_anthropic.go
func (c *AnthropicClient) GenerateResponse(prompt string) (string, error) {
    // Create API request to Anthropic
    // Process response
    // Return generated text
}
```

### OpenAI Implementation

The OpenAI integration uses the OpenAI API:

```go
// In llm_openai.go
func (c *OpenAIClient) GenerateResponse(prompt string) (string, error) {
    // Create API request to OpenAI
    // Process response
    // Return generated text
}
```

### Ollama Implementation

The Ollama integration connects to a local Ollama server:

```go
// In llm_ollama.go
func (c *OllamaClient) GenerateResponse(prompt string) (string, error) {
    // Create API request to Ollama
    // Process response
    // Return generated text
}
```

## Prompt Construction

The system constructs prompts for LLMs that include:

1. **System Instructions**: Defines the assistant's role and capabilities
2. **Retrieved Context**: Information from the RAG system
3. **User Query**: The specific question being asked
4. **Response Format**: Guidelines for how to structure the answer

A typical prompt structure:

```
System: You are a helpful assistant with access to the following information. 
Answer questions based on this information.

Context:
[Retrieved documents from RAG system]

User Question: [User's query]

Please provide a clear, accurate answer based on the context provided.
Include citations to the relevant sources in your response.
```

## Response Processing

After receiving responses from the LLM:

1. **Validation**: Checks for completeness and relevance
2. **Formatting**: Ensures consistent structure
3. **Citation Extraction**: Identifies and validates references
4. **Quality Assessment**: Evaluates answer quality

## Best Practices

### Model Selection

- **Anthropic Claude**: Best for complex reasoning, nuanced understanding
- **OpenAI GPT**: Good general purpose option with strong coding abilities
- **Ollama**: Ideal for privacy-sensitive applications or offline use

### Parameter Tuning

- **Temperature**: Lower (0.1-0.4) for factual responses, higher (0.7-1.0) for creative ones
- **Max Tokens**: Set high enough to accommodate complete answers (typically 1000-2000)
- **Top P/Top K**: Can be adjusted to control response diversity

### Cost Management

- Use smaller, more efficient models for simple queries
- Implement caching for common questions
- Set appropriate token limits to prevent runaway costs

### Fallback Mechanisms

For robust operation, implement fallbacks:

```go
func generateWithFallback(query string, context string) (string, error) {
    // Try primary provider
    response, err := primaryLLM.GenerateResponse(query, context)
    if err == nil {
        return response, nil
    }
    
    // Log failure and try fallback
    log.Printf("Primary LLM failed: %v, trying fallback", err)
    return fallbackLLM.GenerateResponse(query, context)
}
```