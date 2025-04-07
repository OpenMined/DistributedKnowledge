package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// OllamaProvider implements the LLMProvider interface for Ollama
type OllamaProvider struct {
	client *http.Client
	config ModelConfig
}

// OllamaRequest represents a request to the Ollama API
type OllamaRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	System      string  `json:"system,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Format      string  `json:"format,omitempty"`
}

// OllamaResponse represents a response from the Ollama API
type OllamaResponse struct {
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
}

// NewOllamaProvider creates a new Ollama provider from a ModelConfig
func NewOllamaProvider(config ModelConfig) (*OllamaProvider, error) {
	return &OllamaProvider{
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
		config: config,
	}, nil
}

// GenerateAnswer implements LLMProvider interface
func (p *OllamaProvider) GenerateAnswer(ctx context.Context, question string, docs []Document) (string, error) {
	// Construct the system prompt and user prompt
	systemPrompt := "You are a helpful AI assistant. Your task is to answer questions based on the context provided in the documents. Answer in first person."

	// Construct a prompt that includes the question and context from the documents
	userPrompt := fmt.Sprintf("Question: %s\n\nDocuments:\n", question)
	for i, doc := range docs {
		userPrompt += fmt.Sprintf("Document %d - %s:\n%s\n\n", i+1, doc.FileName, doc.Content)
	}
	userPrompt += "Please provide a comprehensive answer based on the documents above."

	// Default to llama3 if not specified
	model := p.config.Model
	if model == "" {
		model = "llama3"
	}

	// Create the request
	baseURL := "http://localhost:11434/api/generate"
	if p.config.BaseURL != "" {
		baseURL = p.config.BaseURL
	}

	req := OllamaRequest{
		Model:  model,
		Prompt: userPrompt,
		System: systemPrompt,
	}

	// Apply custom parameters if provided
	if p.config.Parameters != nil {
		if temp, ok := p.config.Parameters["temperature"].(float64); ok {
			req.Temperature = temp
		}
		if maxTokens, ok := p.config.Parameters["max_tokens"].(float64); ok {
			req.MaxTokens = int(maxTokens)
		}
	}

	// Convert request to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Add custom headers if provided
	if p.config.Headers != nil {
		for key, value := range p.config.Headers {
			httpReq.Header.Set(key, value)
		}
	}

	// Send request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("Response: %v", resp)
	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error: %s", string(body))
	}

	// Parse the response - Ollama streams the response, so we might need to handle it differently
	var sb strings.Builder
	for _, line := range strings.Split(string(body), "\n") {
		if line == "" {
			continue
		}
		var ollamaResp OllamaResponse
		if err := json.Unmarshal([]byte(line), &ollamaResp); err != nil {
			continue // Skip lines that can't be parsed
		}
		if ollamaResp.Error != "" {
			return "", fmt.Errorf("API error: %s", ollamaResp.Error)
		}
		sb.WriteString(ollamaResp.Response)
	}

	return sb.String(), nil
}

// CheckAutomaticApproval implements LLMProvider interface
func (p *OllamaProvider) CheckAutomaticApproval(ctx context.Context, answer string, query Query, conditions []string) (string, bool, error) {
	// Format the list as a pretty JSON string
	formatted, err := json.MarshalIndent(conditions, "", "  ")
	if err != nil {
		return "Error formatting conditions as JSON", false, err
	}

	// System prompt for evaluation
	systemPrompt := "You are an AI assistant responsible for verifying that if given fields=(query, queried from, and answer). Check if they satisfies all specified conditions with no tolerance for minor deviations. Evaluate the answer against each condition, and then return only a JSON object with a two keys, 'result' and 'reason', set to true if every condition is met, or false if any condition fails. The 'reason' key should contain a brief explanation of why the result is true or false. Do not include any additional text or formatting. If condition list is empty, return false."

	// User prompt with data to evaluate
	userPrompt := fmt.Sprintf("Query:'%s'\n\n'Queried From:'%s'\n\n My Answer: '%s'\n\nConditions: %s\n",
		query.Question, query.From, answer, string(formatted))

	// Default to llama3 if not specified
	model := p.config.Model
	if model == "" {
		model = "llama3"
	}

	// Create the request
	baseURL := "http://localhost:11434/api/generate"
	if p.config.BaseURL != "" {
		baseURL = p.config.BaseURL
	}

	req := OllamaRequest{
		Model:  model,
		Prompt: userPrompt,
		System: systemPrompt,
		Format: "json", // Request JSON format if supported by the model
	}

	// Apply custom parameters if provided
	if p.config.Parameters != nil {
		if temp, ok := p.config.Parameters["temperature"].(float64); ok {
			req.Temperature = temp
		}
	}

	// Convert request to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return "Error marshaling request", false, err
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return "Error creating request", false, err
	}

	// Add headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Add custom headers if provided
	if p.config.Headers != nil {
		for key, value := range p.config.Headers {
			httpReq.Header.Set(key, value)
		}
	}

	// Send request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "Error sending request", false, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "Error reading response body", false, err
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		return "API error", false, fmt.Errorf("API error: %s", string(body))
	}

	// Parse the response - Ollama streams the response, so we need to collect it all
	var sb strings.Builder
	for _, line := range strings.Split(string(body), "\n") {
		if line == "" {
			continue
		}
		var ollamaResp OllamaResponse
		if err := json.Unmarshal([]byte(line), &ollamaResp); err != nil {
			continue // Skip lines that can't be parsed
		}
		if ollamaResp.Error != "" {
			return "API error", false, fmt.Errorf("API error: %s", ollamaResp.Error)
		}
		sb.WriteString(ollamaResp.Response)
	}

	// Extract the response text
	responseText := sb.String()

	// Parse the JSON response
	var result struct {
		Result bool   `json:"result"`
		Reason string `json:"reason"`
	}

	// Try to find JSON in the response
	jsonStart := strings.Index(responseText, "{")
	jsonEnd := strings.LastIndex(responseText, "}")

	if jsonStart >= 0 && jsonEnd > jsonStart {
		jsonStr := responseText[jsonStart : jsonEnd+1]
		if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
			// Try to make a best effort determination
			return "Error parsing JSON", strings.Contains(strings.ToLower(responseText), "true"), nil
		}
	} else {
		// Fallback if proper JSON wasn't returned
		return "Invalid response format", strings.Contains(strings.ToLower(responseText), "true"), nil
	}

	return result.Reason, result.Result, nil
}
