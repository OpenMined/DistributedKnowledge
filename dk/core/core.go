package core

import (
	"context"
	dk_client "dk/client"
	"dk/db"
	"dk/utils"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func HandleRequests(ctx context.Context) {
	client, err := utils.DkFromContext(ctx)
	if err != nil {
		fmt.Println("Error getting client from context:", err)
		return
	}
	var query utils.RemoteMessage
	for msg := range client.Messages() {
		err := json.Unmarshal([]byte(msg.Content), &query)
		if err != nil || strings.TrimSpace(query.Message) == "" {
			fmt.Println("Error unmarshaling message content:", err, "skipping item")
			// return nil, fmt.Errorf("'question' parameter is required")
		}
		if query.Type == "query" {
			HandleQuery(ctx, msg)
		} else if query.Type == "app" {
			HandleApplicationRequest(ctx, msg)
		} else {
			HandleAnswer(ctx, msg)
		}
	}
}

func HandleQuery(ctx context.Context, msg dk_client.Message) (string, error) {
	var query utils.RemoteMessage
	err := json.Unmarshal([]byte(msg.Content), &query)
	if err != nil || strings.TrimSpace(query.Message) == "" {
		return "", fmt.Errorf("failed to parse message or empty question")
	}

	origin := msg.From

	// Get app parameters
	params, err := utils.ParamsFromContext(ctx)
	if err != nil {
		return "", err
	}

	// Get LLM provider
	llmProvider, err := LLMProviderFromContext(ctx)

	if err != nil {
		// If no LLM provider in context, try to load from config
		if params.ModelConfigFile != nil {
			modelConfig, err := LoadModelConfig(*params.ModelConfigFile)
			if err != nil {
				return "", fmt.Errorf("failed to load model config: %w", err)
			}

			llmProvider, err = CreateLLMProvider(modelConfig)
			if err != nil {
				return "", fmt.Errorf("failed to create LLM provider: %w", err)
			}

			// Store provider in context for future use
			ctx = WithLLMProvider(ctx, llmProvider)
		} else {
			return "", fmt.Errorf("no LLM provider found and no model config file specified")
		}
	}

	// Retrieve relevant documents
	docs, err := RetrieveDocuments(ctx, query.Message, 3)

	if err != nil {
		return "", fmt.Errorf("failed to retrieve documents: %v", err)
	}

	// Generate answer using the LLM provider
	answer, err := llmProvider.GenerateAnswer(ctx, query.Message, docs)
	if err != nil {
		return "", fmt.Errorf("failed to generate answer: %v", err)
	}

	// Generate new query ID
	newID, err := generateQueryID()
	if err != nil {
		return "", fmt.Errorf("failed to generate query ID: %w", err)
	}

	// Extract document filenames
	var docFilenames []string = []string{}
	for _, doc := range docs {
		docFilenames = append(docFilenames, doc.FileName)
	}

	// Create new query
	newQuery := Query{
		ID:               newID,
		From:             origin,
		Question:         query.Message,
		Answer:           answer,
		DocumentsRelated: docFilenames,
		Status:           "pending",
	}

	// Check for automatic approval
	var reason string
	var automaticApproval bool

	// ------------------------------------------------------------------
	//  ➤  Persist into SQLite instead of queries.json
	// ------------------------------------------------------------------
	dbInstance, err := utils.DatabaseFromContext(ctx)
	if err != nil {
		return "", err
	}

	docJSONNames := make([]string, len(docs))
	for i, d := range docs {
		docJSONNames[i] = d.FileName
	}

	newQueryItem := db.Query{
		ID:               newID,
		From:             origin,
		Question:         query.Message,
		Answer:           answer,
		DocumentsRelated: docJSONNames,
		Status:           "pending",
		Reason:           reason,
	}

	automaticApprovalRules, err := db.ListRules(ctx, dbInstance)

	if err == nil {
		if len(automaticApprovalRules) != 0 {
			reason, automaticApproval, err = llmProvider.CheckAutomaticApproval(ctx, answer, newQuery, automaticApprovalRules)
			if err != nil {
				reason = fmt.Sprintf("Error checking automatic approval: %v", err)
				automaticApproval = false
			}
		} else {
			reason = "There's not condition for automatic approval"
			automaticApproval = false
		}
	} else {
		reason = "Error recovering automatic approval rules from database."
		automaticApproval = false
	}

	if automaticApproval {
		newQueryItem.Status = "accepted"
	}
	newQueryItem.Reason = reason

	if err := db.InsertQuery(ctx, dbInstance, newQueryItem); err != nil {
		return "", err
	}

	// If automatically approved, send the answer
	if automaticApproval {
		dkClient, err := utils.DkFromContext(ctx)
		if err == nil {
			answerMessage := utils.AnswerMessage{
				Query:  newQueryItem.Question,
				Answer: newQueryItem.Answer,
				From:   dkClient.UserID,
			}

			jsonAnswer, err := json.Marshal(answerMessage)
			if err == nil {
				queryMsg := utils.RemoteMessage{
					Type:    "answer",
					Message: string(jsonAnswer),
				}

				jsonData, err := json.Marshal(queryMsg)
				if err == nil {
					dkClient.SendMessage(dk_client.Message{
						From:      dkClient.UserID,
						To:        newQueryItem.From,
						Content:   string(jsonData),
						Timestamp: time.Now(),
					})
				}
			}
		}
	}

	return answer, nil
}

func HandleAnswer(ctx context.Context, msg dk_client.Message) (string, error) {
	dbHandler, err := utils.DatabaseFromContext(ctx)
	if err != nil {
		return "", err
	}

	var remoteMsg utils.RemoteMessage
	if err := json.Unmarshal([]byte(msg.Content), &remoteMsg); err != nil ||
		strings.TrimSpace(remoteMsg.Message) == "" {
		return "", fmt.Errorf("invalid outer message: %w", err)
	}

	var answer utils.AnswerMessage
	if err := json.Unmarshal([]byte(remoteMsg.Message), &answer); err != nil {
		return "", fmt.Errorf("invalid answer payload: %w", err)
	}

	if err := db.InsertAnswer(ctx, dbHandler, db.Answer{
		Question: answer.Query,
		User:     msg.From,
		Text:     answer.Answer,
	}); err != nil {
		return "", err
	}
	return "", nil // no reply – same behaviour as before
}

func HandleApplicationRequest(ctx context.Context, msg dk_client.Message) (string, error) {
	var appRequest utils.RemoteMessage
	err := json.Unmarshal([]byte(msg.Content), &appRequest)
	if err != nil {
		return "", fmt.Errorf("failed to parse message or empty question")
	}

	parameters, err := utils.ParamsFromContext(ctx)
	if err != nil {
		return "", nil
	}

	file, err := os.ReadFile(*parameters.SyftboxConfig)
	if err != nil {
		// Wrap the result in a CallToolResult.
		return "", nil
	}

	var syftboxConfig struct {
		DataDir       string  `json:"data_dir"`
		ServerURL     string  `json:"server_url"`
		ClientURL     string  `json:"client_url"`
		Email         string  `json:"email"`
		Token         string  `json:"token"`
		AccessToken   string  `json:"access_token"`
		ClientTimeout float64 `json:"client_timeout"`
	}

	if err := json.Unmarshal(file, &syftboxConfig); err != nil {
		return "", nil
	}

	inboxPath := filepath.Join(syftboxConfig.DataDir, "datasites", syftboxConfig.Email, "inbox")

	err = WriteMapToDir(ctx, inboxPath, appRequest.Files)
	if err != nil {
		log.Println(err.Error())
	}

	const defaultReason = "There is no automatic approval rule for this app"

	dbConn, err := utils.DatabaseFromContext(ctx)
	if err != nil {
		return "", fmt.Errorf("db connection missing: %w", err)
	}

	var firstKey string
	for k := range appRequest.Files {
		firstKey = k
		break
	}

	cleaned := filepath.Clean(firstKey)
	parts := strings.Split(cleaned, string(filepath.Separator))
	appName := parts[0]

	ar := db.AppRequest{
		AppName:        appName,
		RequestedBy:    msg.From,
		AppDescription: appRequest.Message,
		Status:         "pending",
		Reason:         defaultReason,
		Safety:         "Undefined",
	}

	if err := db.InsertOrUpdateAppRequest(ctx, dbConn, ar); err != nil {
		return "", fmt.Errorf("saving app request: %w", err)
	}
	return "", nil
}
