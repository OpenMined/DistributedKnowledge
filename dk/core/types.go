package core

import "context"

// Query represents a question from a user and its associated data
type Query struct {
	ID               string   `json:"id"`
	From             string   `json:"from"`
	Question         string   `json:"question"`
	Answer           string   `json:"answer"`
	DocumentsRelated []string `json:"documents_related"`
	Status           string   `json:"status"`
	Reason           string   `json:"reason"`
}

// QueriesData represents a collection of queries
type QueriesData struct {
	Queries map[string]Query `json:"queries"`
}

// Document represents a content document with its filename
type Document struct {
	Content  string `json:"content"`
	FileName string `json:"file"`
}

// LLMProvider defines the interface that all LLM providers must implement
type LLMProvider interface {
	GenerateAnswer(ctx context.Context, question string, docs []Document) (string, error)
	CheckAutomaticApproval(ctx context.Context, answer string, query Query, conditions []string) (string, bool, error)
}

// ModelConfig stores configuration for an LLM model
type ModelConfig struct {
	Provider   string            `json:"provider"`   // e.g., "openai", "anthropic", "ollama", etc.
	ApiKey     string            `json:"api_key"`    // API key for the service
	Model      string            `json:"model"`      // Model name to use
	BaseURL    string            `json:"base_url"`   // Optional base URL for the API
	Parameters map[string]any    `json:"parameters"` // Additional parameters like temperature, max_tokens, etc.
	Headers    map[string]string `json:"headers"`    // Additional headers for API requests
}
