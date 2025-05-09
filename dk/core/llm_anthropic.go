package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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

	if config.ApiKey == "" {
		config.ApiKey = os.Getenv("ANTHROPIC_API_KEY")
		if config.ApiKey == "" {
			return nil, fmt.Errorf("no Anthropic API key provided")
		}
	}

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
	systemPrompt := GenerateAnswerPrompt

	// Construct a prompt that includes the question and context from the documents
	prompt := fmt.Sprintf("<QUESTION>%s<QUESTION>\n", question)
	prompt += "<CONTEXT>\n"
	for _, doc := range docs {
		prompt += fmt.Sprintf("%s", doc.Content)
	}
	prompt += "<CONTEXT>\n"

	// userPrompt := fmt.Sprintf("Question: %s\n\nDocuments:\n", question)
	// for i, doc := range docs {
	// 	userPrompt += fmt.Sprintf("Document %d - %s:\n%s\n\n", i+1, doc.FileName, doc.Content)
	// }
	// userPrompt += "Please provide a comprehensive answer based on the documents above."

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
		Messages: []AnthropicMessage{{Role: "user", Content: prompt}},
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
	systemPrompt := CheckAutomaticApprovalPrompt

	// User prompt with data to evaluate
	userPrompt := fmt.Sprintf("\n{'from': '%s', 'query': '%s', 'answer': '%s', 'conditions': %s}\n",
		query.From, query.Question, answer, string(formatted))

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

func (p *AnthropicProvider) GenerateDescription(ctx context.Context, text string) (string, error) {
	// System prompt for evaluation
	systemPrompt := GenerateDescriptionPrompt

	// User prompt with data to evaluate
	// userPrompt := fmt.Sprintf("Query:'%s'\n\n'Queried From:'%s'\n\n My Answer: '%s'\n\nConditions: %s\n",
	// 	query.Question, query.From, answer, string(formatted))
	userPrompt := fmt.Sprintf("---TEXT START---\n%s\n---TEXT END---", text)

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
		return "", fmt.Errorf("Error marshaling request")
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("Error creating request")
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
		return "", fmt.Errorf("Error sending request")
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Error reading response body")
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error")
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
			return "", fmt.Errorf("API error")
		}
		sb.WriteString(ollamaResp.Response)
	}

	// Extract the response text
	responseText := sb.String()

	return responseText, nil
}
