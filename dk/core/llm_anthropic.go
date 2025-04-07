package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// AnthropicProvider implements the LLMProvider interface for Anthropic (Claude)
type AnthropicProvider struct {
	client *http.Client
	config ModelConfig
}

// AnthropicMessage represents a message in the Anthropic API format
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicRequest represents a request to the Anthropic API
type AnthropicRequest struct {
	Model       string             `json:"model"`
	Messages    []AnthropicMessage `json:"messages"`
	Temperature float64            `json:"temperature,omitempty"`
	MaxTokens   int                `json:"max_tokens,omitempty"`
	System      string             `json:"system,omitempty"`
}

// AnthropicResponse represents a response from the Anthropic API
type AnthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// NewAnthropicProvider creates a new Anthropic provider from a ModelConfig
func NewAnthropicProvider(config ModelConfig) (*AnthropicProvider, error) {
	return &AnthropicProvider{
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
		config: config,
	}, nil
}

// GenerateAnswer implements LLMProvider interface
func (p *AnthropicProvider) GenerateAnswer(ctx context.Context, question string, docs []Document) (string, error) {
	// Construct the system prompt and user prompt
	systemPrompt := "You are a helpful AI assistant. Your task is to answer questions based on the context provided in the documents. Answer in first person."

	// Construct a prompt that includes the question and context from the documents
	userPrompt := fmt.Sprintf("Question: %s\n\nDocuments:\n", question)
	for i, doc := range docs {
		userPrompt += fmt.Sprintf("Document %d - %s:\n%s\n\n", i+1, doc.FileName, doc.Content)
	}
	userPrompt += "Please provide a comprehensive answer based on the documents above."

	// Default to claude-3-sonnet-20240229 if not specified
	model := p.config.Model
	if model == "" {
		model = "claude-3-sonnet-20240229"
	}

	// Create the request
	apiURL := "https://api.anthropic.com/v1/messages"
	if p.config.BaseURL != "" {
		apiURL = p.config.BaseURL
	}

	req := AnthropicRequest{
		Model:    model,
		Messages: []AnthropicMessage{{Role: "user", Content: userPrompt}},
		System:   systemPrompt,
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
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(string(reqBody)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	httpReq.Header.Set("content-type", "application/json")
	httpReq.Header.Set("x-api-key", p.config.ApiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

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

	// Parse response
	var anthropicResp AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error: %s", anthropicResp.Error.Message)
	}

	// Extract the answer
	if len(anthropicResp.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return anthropicResp.Content[0].Text, nil
}

// CheckAutomaticApproval implements LLMProvider interface
func (p *AnthropicProvider) CheckAutomaticApproval(ctx context.Context, answer string, query Query, conditions []string) (string, bool, error) {
	// Format the list as a pretty JSON string
	formatted, err := json.MarshalIndent(conditions, "", "  ")
	if err != nil {
		return "Error formatting conditions as JSON", false, err
	}

	// Default to claude-3-haiku-20240307 if not specified (using a smaller model for this task)
	model := p.config.Model
	if model == "" {
		model = "claude-3-haiku-20240307"
	}

	// System prompt for evaluation
  systemPrompt := "You are an AI assistant responsible for verifying that if given fields=(query, queried from, and answer). Check if they satisfies all specified conditions with no tolerance for minor deviations. Evaluate the answer against each condition, and then return only a JSON object with a two keys, 'result' and 'reason', set to true if every condition is met, or false if any condition fails. The 'reason' key should contain a brief explanation of why the result is true or false. Do not include any additional text or formatting. If condition list is empty, return false."
	// User prompt with data to evaluate
	userPrompt := fmt.Sprintf("Query:'%s'\n\n'Queried From:'%s'\n\n My Answer: '%s'\n\nConditions: %s\n",
		query.Question, query.From, answer, string(formatted))

	// Create the request
	apiURL := "https://api.anthropic.com/v1/messages"
	if p.config.BaseURL != "" {
		apiURL = p.config.BaseURL
	}

	req := AnthropicRequest{
		Model:    model,
		Messages: []AnthropicMessage{{Role: "user", Content: userPrompt}},
		System:   systemPrompt,
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
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(string(reqBody)))
	if err != nil {
		return "Error creating request", false, err
	}

	// Add headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.config.ApiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

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

	// Parse response
	var anthropicResp AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return "Error decoding response", false, err
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		return "API error", false, fmt.Errorf("API error: %s", anthropicResp.Error.Message)
	}

	// Extract the answer
	if len(anthropicResp.Content) == 0 {
		return "No content in response", false, fmt.Errorf("no content in response")
	}

	// Parse the JSON response
	var result struct {
		Result bool   `json:"result"`
		Reason string `json:"reason"`
	}

	responseText := anthropicResp.Content[0].Text
	// Try to find JSON in the response
	jsonStart := strings.Index(responseText, "{")
	jsonEnd := strings.LastIndex(responseText, "}")

	if jsonStart >= 0 && jsonEnd > jsonStart {
		jsonStr := responseText[jsonStart : jsonEnd+1]
		if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
			log.Printf("Failed to parse JSON from response: %v", err)
			// Try to make a best effort determination
			return "Error parsing result JSON", strings.Contains(strings.ToLower(responseText), "true"), nil
		}
	} else {
		// Fallback if proper JSON wasn't returned
		return "Invalid response format", strings.Contains(strings.ToLower(responseText), "true"), nil
	}

	return result.Reason, result.Result, nil
}
