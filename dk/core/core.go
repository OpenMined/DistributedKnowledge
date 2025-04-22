package core

import (
	"context"
	dk_client "dk/client"
	"dk/utils"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
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

	// Load existing queries
	queriesData, err := LoadQueries(*params.QueriesFile)
	if err != nil {
		return "", err
	}
	if queriesData.Queries == nil {
		queriesData.Queries = make(map[string]Query)
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

	if params.AutomaticApprovalFile != nil {
		var conditions []string

		if _, err := os.Stat(*params.AutomaticApprovalFile); !os.IsNotExist(err) {
			raw, err := os.ReadFile(*params.AutomaticApprovalFile)
			if err == nil {
				if err := json.Unmarshal(raw, &conditions); err == nil {
					if len(conditions) != 0 {
						reason, automaticApproval, err = llmProvider.CheckAutomaticApproval(ctx, answer, newQuery, conditions)
						if err != nil {
							reason = fmt.Sprintf("Error checking automatic approval: %v", err)
							automaticApproval = false
						}
					} else {
						reason = "There's not condition for automatic approval"
						automaticApproval = false
					}
				} else {
					reason = "Error unmarshaling automatic approval file"
					automaticApproval = false
				}
			} else {
				reason = "Error reading automatic approval file"
				automaticApproval = false
			}
		} else {
			reason = "No automatic approval file"
			automaticApproval = false
		}
	} else {
		reason = "No automatic approval file specified"
		automaticApproval = false
	}

	if automaticApproval {
		newQuery.Status = "accepted"
	}

	newQuery.Reason = reason
	queriesData.Queries[newID] = newQuery

	// Save updated queries
	if err := SaveQueries(*params.QueriesFile, queriesData); err != nil {
		return "", err
	}

	// If automatically approved, send the answer
	if automaticApproval {
		dkClient, err := utils.DkFromContext(ctx)
		if err == nil {
			answerMessage := utils.AnswerMessage{
				Query:  newQuery.Question,
				Answer: newQuery.Answer,
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
						To:        newQuery.From,
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
	params, err := utils.ParamsFromContext(ctx)
	if err != nil {
		return "", err
	}
	anwsersFile := *params.AnswersFile

	var remoteMsg utils.RemoteMessage
	err = json.Unmarshal([]byte(msg.Content), &remoteMsg)
	if err != nil || strings.TrimSpace(remoteMsg.Message) == "" {
		// return nil, fmt.Errorf("'question' parameter is required")
	}

	var answer utils.AnswerMessage
	err = json.Unmarshal([]byte(remoteMsg.Message), &answer)
	if err != nil {
		// return nil, fmt.Errorf("'question' parameter is required")
	}

	queryID := answer.Query
	from := msg.From

	// Load existing answers from the file.
	var answersData map[string]map[string]string
	if _, err := os.Stat(anwsersFile); os.IsNotExist(err) {
		// File doesn't exist; initialize a new map.
		answersData = make(map[string]map[string]string)
	} else {
		raw, err := os.ReadFile(anwsersFile)
		if err != nil {
			return "", fmt.Errorf("failed to read answers file: %w", err)
		}
		if err := json.Unmarshal(raw, &answersData); err != nil {
			return "", fmt.Errorf("failed to unmarshal answers file: %w", err)
		}
	}

	// Check if the query_id exists. If not, create a new entry.
	if answersData[queryID] == nil {
		answersData[queryID] = make(map[string]string)
	}

	// Append (or update) the new answer using both the 'from' and 'to' keys.
	answersData[queryID][from] = answer.Answer

	// Marshal the updated answersData back to JSON.
	updatedRaw, err := json.MarshalIndent(answersData, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal answers data: %w", err)
	}

	// Ensure the directory exists (using the same pattern as in saveQueries).
	answersDir := filepath.Dir(anwsersFile)
	if err := os.MkdirAll(answersDir, fs.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", answersDir, err)
	}

	// Write the updated JSON back to the file.
	if err := os.WriteFile(anwsersFile, updatedRaw, 0644); err != nil {
		return "", fmt.Errorf("failed to write answers file: %w", err)
	}
	return "", nil
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

	type AppRequest struct {
		RequestedBy string `json:"requested_by"`
		Description string `json:"description"`
		Status      string `json:"status"`
		Reason      string `json:"reason"`
		Safety      string `json:"safety"`
	}

	const defaultReason = "There is no automatic approval rule for this app"

	projectDir := filepath.Dir(*parameters.ModelConfigFile)
	appRequestPath := filepath.Join(projectDir, "app_request.json")
	appRequests := map[string]AppRequest{}

	if f, err := os.Open(appRequestPath); err == nil {
		// File exists → decode existing contents (best‑effort)
		dec := json.NewDecoder(f)
		if err := dec.Decode(&appRequests); err != nil {
			log.Printf("app_request.json is corrupt, starting fresh: %v", err)
			appRequests = map[string]AppRequest{}
		}
		_ = f.Close()
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("unable to open app_request.json: %w", err)
	}

	var firstKey string
	for k := range appRequest.Files {
		firstKey = k
		break
	}

	cleaned := filepath.Clean(firstKey)
	parts := strings.Split(cleaned, string(filepath.Separator))
	appName := parts[0]

	// Insert / overwrite this application:
	appRequests[appName] = AppRequest{
		RequestedBy: msg.From,
		Description: appRequest.Message,
		Status:      "pending",
		Reason:      defaultReason,
		Safety:      "Undefined",
	}

	//----------------------------------------------------------------------
	// Write the updated map back to disk (pretty‑printed for humans)
	//----------------------------------------------------------------------
	tmpFile := appRequestPath + ".tmp"
	data, err := json.MarshalIndent(appRequests, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal app requests: %w", err)
	}
	if err := os.WriteFile(tmpFile, data, 0o644); err != nil {
		return "", fmt.Errorf("failed to write temp app_request.json: %w", err)
	}
	// Atomic replace
	if err := os.Rename(tmpFile, appRequestPath); err != nil {
		return "", fmt.Errorf("failed to save app_request.json: %w", err)
	}

	return "", nil
}
