package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// LLMProviderKey is a context key for the LLM provider
type LLMProviderKey struct{}

// WithLLMProvider adds an LLM provider to the context
func WithLLMProvider(ctx context.Context, provider LLMProvider) context.Context {
	return context.WithValue(ctx, LLMProviderKey{}, provider)
}

// LLMProviderFromContext extracts the LLM provider from the context
func LLMProviderFromContext(ctx context.Context) (LLMProvider, error) {
	provider, ok := ctx.Value(LLMProviderKey{}).(LLMProvider)
	if !ok {
		return nil, fmt.Errorf("LLM provider not found in context")
	}
	return provider, nil
}

// LoadModelConfig loads LLM model configuration from a JSON file
func LoadModelConfig(configFile string) (ModelConfig, error) {
	var config ModelConfig

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return config, fmt.Errorf("model config file does not exist: %w", err)
	}

	raw, err := os.ReadFile(configFile)
	if err != nil {
		return config, fmt.Errorf("failed to read model config file: %w", err)
	}

	if err := json.Unmarshal(raw, &config); err != nil {
		return config, fmt.Errorf("failed to unmarshal model config: %w", err)
	}

	return config, nil
}

func LoadQueries(queriesFile string) (QueriesData, error) {
	var data QueriesData
	// If file doesn't exist, initialize an empty map.
	if _, err := os.Stat(queriesFile); os.IsNotExist(err) {
		data.Queries = make(map[string]Query)
		return data, nil
	}
	raw, err := os.ReadFile(queriesFile)
	if err != nil {
		return data, fmt.Errorf("failed to read queries file: %w", err)
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return data, fmt.Errorf("failed to unmarshal queries file: %w", err)
	}
	return data, nil
}

func SaveQueries(queriesFile string, data QueriesData) error {
	// Ensure directory exists.
	dir := filepath.Dir(queriesFile)
	if err := os.MkdirAll(dir, fs.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal queries data: %w", err)
	}
	if err := os.WriteFile(queriesFile, raw, 0644); err != nil {
		return fmt.Errorf("failed to write queries file: %w", err)
	}
	return nil
}

func generateQueryID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "qry-" + hex.EncodeToString(b), nil
}
