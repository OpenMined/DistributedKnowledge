package core

import (
	"fmt"
)

// CreateLLMProvider creates an LLM provider based on the provided configuration
func CreateLLMProvider(config ModelConfig) (LLMProvider, error) {
	switch config.Provider {
	case "openai":
		return NewOpenAIProvider(config)
	case "anthropic":
		return NewAnthropicProvider(config)
	case "ollama":
		return NewOllamaProvider(config)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", config.Provider)
	}
}
