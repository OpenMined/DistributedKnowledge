package core

import (
	"context"
	"encoding/json"
	"fmt"
	openai "github.com/sashabaranov/go-openai"
	"os"
)

// OpenAIProvider implements the LLMProvider interface for OpenAI
type OpenAIProvider struct {
	client *openai.Client
	config ModelConfig
}

// NewOpenAIProvider creates a new OpenAI provider from a ModelConfig
func NewOpenAIProvider(config ModelConfig) (*OpenAIProvider, error) {
	cfg := openai.DefaultConfig(config.ApiKey)

	// Set custom base URL if provided
	if config.BaseURL != "" {
		cfg.BaseURL = config.BaseURL
	}

	if config.ApiKey == "" {
		config.ApiKey = os.Getenv("OPENAI_API_KEY")
		if config.ApiKey == "" {
			return nil, fmt.Errorf("no OpenAI API key provided")
		}
	}

	return &OpenAIProvider{
		client: openai.NewClientWithConfig(cfg),
		config: config,
	}, nil
}

// GenerateAnswer implements LLMProvider interface
func (p *OpenAIProvider) GenerateAnswer(ctx context.Context, question string, docs []Document) (string, error) {
	// Construct a prompt that includes the question and context from the documents.
	// prompt := "Question:" + question // fmt.Sprintf("You are an AI assistant that answers questions based on the context provided in the documents.\n\nQuestion: %s\n\nDocuments:\n", question)
	prompt := fmt.Sprintf("<QUESTION>%s<QUESTION>\n", question)
	prompt += "<CONTEXT>\n"
	for _, doc := range docs {
		prompt += fmt.Sprintf("%s", doc.Content)
	}
	prompt += "<CONTEXT>\n"

	// Default to GPT-3.5 if not specified
	model := p.config.Model
	if model == "" {
		model = openai.GPT3Dot5Turbo
	}

	// Use ChatCompletion for answer generation.
	chatReq := openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: GenerateAnswerPrompt},
			{Role: "user", Content: prompt},
		},
	}

	// Apply custom parameters if provided
	if p.config.Parameters != nil {
		if temp, ok := p.config.Parameters["temperature"].(float64); ok {
			chatReq.Temperature = float32(temp)
		}
		if maxTokens, ok := p.config.Parameters["max_tokens"].(float64); ok {
			chatReq.MaxTokens = int(maxTokens)
		}
	}

	chatResp, err := p.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return "", fmt.Errorf("failed to generate answer: %w", err)
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no answer returned")
	}
	answer := chatResp.Choices[0].Message.Content
	return answer, nil
}

// CheckAutomaticApproval implements LLMProvider interface
func (p *OpenAIProvider) CheckAutomaticApproval(ctx context.Context, answer string, query Query, conditions []string) (string, bool, error) {
	// Format the list as a pretty JSON string.
	formatted, err := json.MarshalIndent(conditions, "", "  ")
	if err != nil {
		return "Error formatting conditions as JSON", false, err
	}

	// Default to GPT-4o-mini if not specified
	model := p.config.Model
	if model == "" {
		model = openai.GPT4oMini
	}

	prompt := fmt.Sprintf("\n{'from': '%s', 'query': '%s', 'answer': '%s', 'conditions': %s}\n",
		query.From, query.Question, answer, string(formatted))

	os.WriteFile("prompt_text.txt", []byte(prompt), 0644)

	systemPrompt := CheckAutomaticApprovalPrompt

	// Use ChatCompletion for automatic approval check
	chatReq := openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{Type: "json_object"},
	}

	// Apply custom parameters if provided
	if p.config.Parameters != nil {
		if temp, ok := p.config.Parameters["temperature"].(float64); ok {
			chatReq.Temperature = float32(temp)
		}
	}

	chatResp, err := p.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return "Error generating response", false, err
	}
	if len(chatResp.Choices) == 0 {
		return "No response returned", false, fmt.Errorf("no response returned")
	}

	response := chatResp.Choices[0].Message.Content
	var result struct {
		Result bool   `json:"result"`
		Reason string `json:"reason"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return "Error parsing response", false, err
	}

	return result.Reason, result.Result, nil
}

func (p *OpenAIProvider) GenerateDescription(ctx context.Context, text string) (string, error) {
	userPrompt := fmt.Sprintf("---TEXT START---\n%s\n---TEXT END---", text)

	// Default to GPT-3.5 if not specified
	model := p.config.Model
	if model == "" {
		model = openai.GPT3Dot5Turbo
	}

	// Use ChatCompletion for answer generation.
	chatReq := openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{Role: "system", Content: GenerateDescriptionPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	// Apply custom parameters if provided
	if p.config.Parameters != nil {
		if temp, ok := p.config.Parameters["temperature"].(float64); ok {
			chatReq.Temperature = float32(temp)
		}
		if maxTokens, ok := p.config.Parameters["max_tokens"].(float64); ok {
			chatReq.MaxTokens = int(maxTokens)
		}
	}

	chatResp, err := p.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return "", fmt.Errorf("failed to generate answer: %w", err)
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no answer returned")
	}
	answer := chatResp.Choices[0].Message.Content
	return answer, nil
}
