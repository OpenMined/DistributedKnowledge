package mcp

import (
	"context"
	dk_client "dk/client"
	"dk/core"
	"dk/utils"
	"encoding/json"
	"fmt"
	mcp_lib "github.com/mark3labs/mcp-go/mcp"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Tool: Get Answers for Query
//
// This tool retrieves all answers associated with a given answer_id.
// The answers.json file is expected to have the following structure:
//
// Given an answer_id, this tool will load the file, check if the entry exists,
// and return the associated answers. In case of any error, the error message
// will be returned in the Text field of the CallToolResult.
func HandleAnswerListTool(ctx context.Context, request mcp_lib.CallToolRequest) (*mcp_lib.CallToolResult, error) {
	parameters, err := utils.ParamsFromContext(ctx)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't retrieve params from context: %s", err.Error()),
				},
			},
		}, nil
	}
	anwsersFile := *parameters.AnswersFile

	// Load existing answers from the file.
	var answersData map[string]map[string]string
	if _, err := os.Stat(anwsersFile); os.IsNotExist(err) {
		// File doesn't exist; therefore, no answers can be found.
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("No answers found "),
				},
			},
		}, nil
	} else {
		raw, err := os.ReadFile(anwsersFile)
		if err != nil {
			return &mcp_lib.CallToolResult{
				Content: []mcp_lib.Content{
					mcp_lib.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error reading answers file: %v", err),
					},
				},
			}, nil
		}
		if err := json.Unmarshal(raw, &answersData); err != nil {
			return &mcp_lib.CallToolResult{
				Content: []mcp_lib.Content{
					mcp_lib.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error unmarshalling answers file: %v", err),
					},
				},
			}, nil
		}
	}

	formatted, err := json.MarshalIndent(answersData, "", "  ")
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error formatting answer data: %v", err),
				},
			},
		}, nil
	}
	detailedStr := "general"
	args := request.Params.Arguments
	detailed, ok := args["detailed_answer"].(bool)

	if ok && detailed {
		detailedStr = "detailed"
	}

	related, ok := args["related_topic"].(string)
	if !ok {
		related = ""
	}
	return &mcp_lib.CallToolResult{
		Content: []mcp_lib.Content{
			mcp_lib.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Given the Answers: %s, and related topic: %s, provide a %s answer.", string(formatted), related, detailedStr),
			},
		},
	}, nil

}

// Tool: Get Answers for Query
//
// This tool retrieves all answers associated with a given answer_id.
// The answers.json file is expected to have the following structure:
//
// Given an answer_id, this tool will load the file, check if the entry exists,
// and return the associated answers. In case of any error, the error message
// will be returned in the Text field of the CallToolResult.
func HandleGetAnswerTool(ctx context.Context, request mcp_lib.CallToolRequest) (*mcp_lib.CallToolResult, error) {
	parameters, err := utils.ParamsFromContext(ctx)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't retrieve params from context: %s", err.Error()),
				},
			},
		}, nil
	}
	anwsersFile := *parameters.AnswersFile

	// Retrieve and validate input parameter.
	args := request.Params.Arguments
	queryId, ok := args["query"].(string)

	delay, ok := args["delay"].(int)
	if !ok {
		delay = 3
	}
	time.Sleep(time.Duration(delay) * time.Second)

	if !ok || strings.TrimSpace(queryId) == "" {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: "'query_id' parameter is required",
				},
			},
		}, nil
	}

	// Load existing answers from the file.
	var answersData map[string]map[string]string
	if _, err := os.Stat(anwsersFile); os.IsNotExist(err) {
		// File doesn't exist; therefore, no answers can be found.
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("No answers found for id: %s", queryId),
				},
			},
		}, nil
	} else {
		raw, err := os.ReadFile(anwsersFile)
		if err != nil {
			return &mcp_lib.CallToolResult{
				Content: []mcp_lib.Content{
					mcp_lib.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error reading answers file: %v", err),
					},
				},
			}, nil
		}
		if err := json.Unmarshal(raw, &answersData); err != nil {
			return &mcp_lib.CallToolResult{
				Content: []mcp_lib.Content{
					mcp_lib.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error unmarshalling answers file: %v", err),
					},
				},
			}, nil
		}
	}

	// Check if the answer_id exists in the loaded data.
	if answers, exists := answersData[queryId]; exists {
		// Format the answers as a pretty JSON string.
		formatted, err := json.MarshalIndent(answers, "", "  ")
		if err != nil {
			return &mcp_lib.CallToolResult{
				Content: []mcp_lib.Content{
					mcp_lib.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error formatting answer data: %v", err),
					},
				},
			}, nil
		}
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: string(formatted),
				},
			},
		}, nil
	}

	// If answer_id is not found, return a message indicating so.
	return &mcp_lib.CallToolResult{
		Content: []mcp_lib.Content{
			mcp_lib.TextContent{
				Type: "text",
				Text: fmt.Sprintf("No answers found for id: %s", queryId),
			},
		},
	}, nil
}

func HandleAskTool(
	ctx context.Context,
	request mcp_lib.CallToolRequest,
) (*mcp_lib.CallToolResult, error) {
	arguments := request.Params.Arguments
	message, ok := arguments["question"].(string)
	if !ok {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Some error happened at question casting \n\n arguments: %s\n\n, ok: %t", arguments, ok),
				},
			},
		}, nil
	}

	var peers []string
	if r, exists := arguments["peers"]; exists {
		for _, item := range r.([]any) {
			if str, ok := item.(string); ok {
				peers = append(peers, str)
			}
		}
	}
	dkClient, err := utils.DkFromContext(ctx)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't retrieve DK from context: %s", err.Error()),
				},
			},
		}, nil
	}
	query := utils.RemoteMessage{
		Type:    "query",
		Message: message,
	}
	jsonData, err := json.Marshal(query)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't marshal query: %s", err.Error()),
				},
			},
		}, nil
	}

	if len(peers) == 0 {
		err = dkClient.BroadcastMessage(string(jsonData))
		if err != nil {
			return &mcp_lib.CallToolResult{
				Content: []mcp_lib.Content{
					mcp_lib.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Couldn't send message: %s", err.Error()),
					},
				},
			}, nil
		}
	} else {
		for _, peer := range peers {
			err = dkClient.SendMessage(dk_client.Message{
				From:      dkClient.UserID,
				To:        peer,
				Content:   string(jsonData),
				Timestamp: time.Now(),
			})
			if err != nil {
				return &mcp_lib.CallToolResult{
					Content: []mcp_lib.Content{
						mcp_lib.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Couldn't send message: %s", err.Error()),
						},
					},
				}, nil
			}
		}
	}

	return &mcp_lib.CallToolResult{
		Content: []mcp_lib.Content{
			mcp_lib.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Query request sent ... Instruct the user to ask the model for summarize on the query %s", query.Message),
			},
		},
	}, nil
}

// Tool: List Queries
func HandleListQueriesTool(ctx context.Context, request mcp_lib.CallToolRequest) (*mcp_lib.CallToolResult, error) {
	params, err := utils.ParamsFromContext(ctx)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't retrieve params from context: %s", err.Error()),
				},
			},
		}, nil
	}

	// Load all queries from the file.
	queriesData, err := core.LoadQueries(*params.QueriesFile)
	if err != nil {
		return nil, err
	}

	// Retrieve optional filter parameters.
	args := request.Params.Arguments
	var statusFilter string
	var fromFilter string

	// Use "status" filter if provided.
	if statusVal, ok := args["status"].(string); ok && strings.TrimSpace(statusVal) != "" {
		statusFilter = strings.ToLower(strings.TrimSpace(statusVal))
	}
	// Use "from" filter if provided.
	if fromVal, ok := args["from"].(string); ok && strings.TrimSpace(fromVal) != "" {
		fromFilter = strings.TrimSpace(fromVal)
	}

	// If no filters are provided, return the complete list.
	if statusFilter == "" && fromFilter == "" {
		output, err := json.MarshalIndent(queriesData.Queries, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal queries: %w", err)
		}
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: string(output),
				},
			},
		}, nil
	}

	// Filter queries based on provided optional parameters.
	filtered := make(map[string]core.Query)
	for id, qry := range queriesData.Queries {
		// Apply status filter if set.
		if statusFilter != "" {
			if strings.ToLower(qry.Status) != statusFilter {
				continue
			}
		}
		// Apply from filter if set.
		if fromFilter != "" {
			if qry.From != fromFilter {
				continue
			}
		}
		filtered[id] = qry
	}

	// Marshal the filtered queries into a pretty JSON string.
	output, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal filtered queries: %w", err)
	}

	return &mcp_lib.CallToolResult{
		Content: []mcp_lib.Content{
			mcp_lib.TextContent{
				Type: "text",
				Text: string(output),
			},
		},
	}, nil
}

// Tool: Add Automatic Approval Condition
//
// This tool extracts a condition from a sentence and appends it to the automatic_approval.json file.
// The file is expected to store an array of condition strings.
// Input parameter: "sentence" (the sentence containing the condition).
func HandleAddApprovalConditionTool(ctx context.Context, request mcp_lib.CallToolRequest) (*mcp_lib.CallToolResult, error) {
	params, err := utils.ParamsFromContext(ctx)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't retrieve params from context: %s", err.Error()),
				},
			},
		}, nil
	}

	automaticApprovalFile := *params.AutomaticApprovalFile

	args := request.Params.Arguments
	sentence, ok := args["sentence"].(string)
	if !ok || strings.TrimSpace(sentence) == "" {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: "'sentence' parameter is required",
				},
			},
		}, nil
	}

	// Extract the condition (for now, simply trim the sentence).
	condition := strings.TrimSpace(sentence)

	// Load existing conditions (if file exists).
	var conditions []string
	if _, err := os.Stat(automaticApprovalFile); os.IsNotExist(err) {
		conditions = []string{}
	} else {
		raw, err := os.ReadFile(automaticApprovalFile)
		if err != nil {
			return &mcp_lib.CallToolResult{
				Content: []mcp_lib.Content{
					mcp_lib.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error reading automatic approval file: %v", err),
					},
				},
			}, nil
		}
		if err := json.Unmarshal(raw, &conditions); err != nil {
			return &mcp_lib.CallToolResult{
				Content: []mcp_lib.Content{
					mcp_lib.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Error unmarshalling automatic approval file: %v", err),
					},
				},
			}, nil
		}
	}

	// Append the new condition.
	conditions = append(conditions, condition)

	// Marshal the updated conditions.
	updatedRaw, err := json.MarshalIndent(conditions, "", "  ")
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error marshalling updated conditions: %v", err),
				},
			},
		}, nil
	}

	// Ensure the directory exists.
	approvalDir := filepath.Dir(automaticApprovalFile)
	if err := os.MkdirAll(approvalDir, fs.ModePerm); err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error creating directory for automatic approval file: %v", err),
				},
			},
		}, nil
	}

	// Write the updated file.
	if err := os.WriteFile(automaticApprovalFile, updatedRaw, 0644); err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error writing automatic approval file: %v", err),
				},
			},
		}, nil
	}

	return &mcp_lib.CallToolResult{
		Content: []mcp_lib.Content{
			mcp_lib.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Condition added successfully: %s", condition),
			},
		},
	}, nil
}

// Tool: Remove Automatic Approval Condition
//
// This tool removes a specific condition from the automatic_approval.json file.
// Input parameter: "condition" (the exact text of the condition to remove).
func HandleRemoveApprovalConditionTool(ctx context.Context, request mcp_lib.CallToolRequest) (*mcp_lib.CallToolResult, error) {
	params, err := utils.ParamsFromContext(ctx)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't retrieve params from context: %s", err.Error()),
				},
			},
		}, nil
	}

	automaticApprovalFile := *params.AutomaticApprovalFile

	args := request.Params.Arguments
	conditionToRemove, ok := args["condition"].(string)
	if !ok || strings.TrimSpace(conditionToRemove) == "" {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: "'condition' parameter is required",
				},
			},
		}, nil
	}
	conditionToRemove = strings.TrimSpace(conditionToRemove)
	var conditions []string

	if _, err := os.Stat(automaticApprovalFile); os.IsNotExist(err) {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: "No automatic approval conditions found.",
				},
			},
		}, nil
	}

	raw, err := os.ReadFile(automaticApprovalFile)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error reading automatic approval file: %v", err),
				},
			},
		}, nil
	}
	if err := json.Unmarshal(raw, &conditions); err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error unmarshalling automatic approval file: %v", err),
				},
			},
		}, nil
	}

	// Remove the specified condition.
	found := false
	newConditions := []string{}
	for _, cond := range conditions {
		if cond == conditionToRemove {
			found = true
			continue
		}
		newConditions = append(newConditions, cond)
	}

	if !found {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Condition '%s' not found.", conditionToRemove),
				},
			},
		}, nil
	}

	// Marshal and write the updated list back to the file.
	updatedRaw, err := json.MarshalIndent(newConditions, "", "  ")
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error marshalling updated conditions: %v", err),
				},
			},
		}, nil
	}
	approvalDir := filepath.Dir(automaticApprovalFile)
	if err := os.MkdirAll(approvalDir, fs.ModePerm); err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error creating directory for automatic approval file: %v", err),
				},
			},
		}, nil
	}
	if err := os.WriteFile(automaticApprovalFile, updatedRaw, 0644); err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error writing automatic approval file: %v", err),
				},
			},
		}, nil
	}

	return &mcp_lib.CallToolResult{
		Content: []mcp_lib.Content{
			mcp_lib.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Condition '%s' removed successfully.", conditionToRemove),
			},
		},
	}, nil
}

// Tool: List Automatic Approval Conditions
//
// This tool lists all the conditions in the automatic_approval.json file.
func HandleListApprovalConditionsTool(ctx context.Context, request mcp_lib.CallToolRequest) (*mcp_lib.CallToolResult, error) {

	params, err := utils.ParamsFromContext(ctx)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't retrieve params from context: %s", err.Error()),
				},
			},
		}, nil
	}

	automaticApprovalFile := *params.AutomaticApprovalFile
	var conditions []string

	if _, err := os.Stat(automaticApprovalFile); os.IsNotExist(err) {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: "No automatic approval conditions found.",
				},
			},
		}, nil
	}

	raw, err := os.ReadFile(automaticApprovalFile)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error reading automatic approval file: %v", err),
				},
			},
		}, nil
	}
	if err := json.Unmarshal(raw, &conditions); err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error unmarshalling automatic approval file: %v", err),
				},
			},
		}, nil
	}

	// Format the list as a pretty JSON string.
	formatted, err := json.MarshalIndent(conditions, "", "  ")
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error formatting conditions: %v", err),
				},
			},
		}, nil
	}
	return &mcp_lib.CallToolResult{
		Content: []mcp_lib.Content{
			mcp_lib.TextContent{
				Type: "text",
				Text: string(formatted),
			},
		},
	}, nil
}

func HandleUpdateRagSourcesTool(ctx context.Context, request mcp_lib.CallToolRequest) (*mcp_lib.CallToolResult, error) {
	args := request.Params.Arguments

	// Workflow 2: Check if file_name and file_content parameters are provided.
	// If either is provided we enforce both to be valid.
	fileName, hasFileName := args["file_name"].(string)
	fileContent, hasFileContent := args["file_content"].(string)
	if hasFileName || hasFileContent {
		// Check that both parameters are provided and are not empty.
		if !hasFileName || strings.TrimSpace(fileName) == "" {
			return &mcp_lib.CallToolResult{
				Content: []mcp_lib.Content{
					mcp_lib.TextContent{
						Type: "text",
						Text: "'file_name' parameter is required when using the file_name/file_content workflow",
					},
				},
			}, nil
		}
		if !hasFileContent || strings.TrimSpace(fileContent) == "" {
			return &mcp_lib.CallToolResult{
				Content: []mcp_lib.Content{
					mcp_lib.TextContent{
						Type: "text",
						Text: "'file_content' parameter is required when using the file_name/file_content workflow",
					},
				},
			}, nil
		}

		core.AddDocument(ctx, fileName, fileContent, true)

		// Return a success response.
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("RAG resource '%s' added successfully and vector database refreshed.", fileName),
				},
			},
		}, nil
	}

	// Workflow 1: Fallback to using the file_path parameter.
	filePath, ok := args["file_path"].(string)
	if !ok || strings.TrimSpace(filePath) == "" {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: "Either 'file_path' or both 'file_name' and 'file_content' parameters are required",
				},
			},
		}, nil
	}

	// Read the content from the file at the provided file_path.
	data, err := os.ReadFile(filePath)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error reading file at '%s': %v", filePath, err),
				},
			},
		}, nil
	}

	// Determine the base file name.
	baseFile := filepath.Base(filePath)

	core.AddDocument(ctx, baseFile, string(data), true)

	// Return a success response.
	return &mcp_lib.CallToolResult{
		Content: []mcp_lib.Content{
			mcp_lib.TextContent{
				Type: "text",
				Text: fmt.Sprintf("RAG resource '%s' added successfully and vector database refreshed.", baseFile),
			},
		},
	}, nil
}

func HandleRejectQuestionTool(ctx context.Context, request mcp_lib.CallToolRequest) (*mcp_lib.CallToolResult, error) {
	params, err := utils.ParamsFromContext(ctx)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't retrieve params from context: %s", err.Error()),
				},
			},
		}, nil
	}

	args := request.Params.Arguments

	id, ok := args["id"].(string)
	if !ok || strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("'id' parameter is required")
	}

	queriesData, err := core.LoadQueries(*params.QueriesFile)
	if err != nil {
		return nil, err
	}

	qry, exists := queriesData.Queries[id]
	if !exists {
		return nil, fmt.Errorf("query with ID '%s' not found", id)
	}

	// Update status to "rejected" if not already.
	qry.Status = "rejected"
	queriesData.Queries[id] = qry

	if err := core.SaveQueries(*params.QueriesFile, queriesData); err != nil {
		return nil, err
	}

	dkClient, err := utils.DkFromContext(ctx)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't retrieve DK from context: %s", err.Error()),
				},
			},
		}, nil
	}

	answerMessage := utils.AnswerMessage{
		Query:  qry.Question,
		Answer: "This query was rejected!",
		From:   dkClient.UserID,
	}

	jsonAnswer, err := json.Marshal(answerMessage)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't marshal answer: %s", err.Error()),
				},
			},
		}, nil
	}

	query := utils.RemoteMessage{
		Type:    "answer",
		Message: string(jsonAnswer),
	}

	jsonData, err := json.Marshal(query)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't marshal query: %s", err.Error()),
				},
			},
		}, nil
	}

	err = dkClient.SendMessage(dk_client.Message{
		From:      dkClient.UserID,
		To:        qry.From,
		Content:   string(jsonData),
		Timestamp: time.Now(),
	})
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't send message: %s", err.Error()),
				},
			},
		}, nil
	}

	return &mcp_lib.CallToolResult{
		Content: []mcp_lib.Content{
			mcp_lib.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Question '%s' has been rejected.\n", qry.Question),
			},
		},
	}, nil
}

func HandleAcceptQuestionTool(ctx context.Context, request mcp_lib.CallToolRequest) (*mcp_lib.CallToolResult, error) {
	params, err := utils.ParamsFromContext(ctx)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't retrieve params from context: %s", err.Error()),
				},
			},
		}, nil
	}

	args := request.Params.Arguments

	id, ok := args["id"].(string)
	if !ok || strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("'id' parameter is required")
	}

	queriesData, err := core.LoadQueries(*params.QueriesFile)
	if err != nil {
		return nil, err
	}

	qry, exists := queriesData.Queries[id]
	if !exists {
		return nil, fmt.Errorf("query with ID '%s' not found", id)
	}

	// Update status to "accepted" if not already.
	qry.Status = "accepted"
	queriesData.Queries[id] = qry

	if err := core.SaveQueries(*params.QueriesFile, queriesData); err != nil {
		return nil, err
	}

	dkClient, err := utils.DkFromContext(ctx)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't retrieve DK from context: %s", err.Error()),
				},
			},
		}, nil
	}

	answerMessage := utils.AnswerMessage{
		Query:  qry.Question,
		Answer: qry.Answer,
		From:   dkClient.UserID,
	}

	jsonAnswer, err := json.Marshal(answerMessage)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't marshal answer: %s", err.Error()),
				},
			},
		}, nil
	}

	query := utils.RemoteMessage{
		Type:    "answer",
		Message: string(jsonAnswer),
	}

	jsonData, err := json.Marshal(query)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't marshal query: %s", err.Error()),
				},
			},
		}, nil
	}

	err = dkClient.SendMessage(dk_client.Message{
		From:      dkClient.UserID,
		To:        qry.From,
		Content:   string(jsonData),
		Timestamp: time.Now(),
	})
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't send message: %s", err.Error()),
				},
			},
		}, nil
	}

	return &mcp_lib.CallToolResult{
		Content: []mcp_lib.Content{
			mcp_lib.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Question '%s' has been accepted.\n", qry.Question),
			},
		},
	}, nil
}

// HandleUpdateAnswerTool updates the answer associated with a given query_id in the queries JSON file.
//
// Input Parameters:
// - "query_id": the identifier for the query (string or integer)
// - "new_answer": the new answer content that will replace the existing answer
//
// The JSON file is expected to conform to this format:
//
//	{
//	  "queries": {
//	    "qry-xxx": {
//	      "id": "qry-xxx",
//	      "from": "UserName",
//	      "question": "...",
//	      "answer": "...",
//	      "documents_related": [...],
//	      "status": "...",
//	      "reason": "..."
//	    },
//	    ...
//	  }
//	}
//
// The function validates the inputs, loads the queries from the file defined in the context parameters,
// updates the answer for the specified query_id, saves the file back, and returns a success message or an error.
func HandleUpdateAnswerTool(ctx context.Context, request mcp_lib.CallToolRequest) (*mcp_lib.CallToolResult, error) {
	// Retrieve runtime parameters from the context.
	parameters, err := utils.ParamsFromContext(ctx)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't retrieve params from context: %s", err.Error()),
				},
			},
		}, nil
	}

	// Get the queries file path from the parameters.
	queriesFile := *parameters.QueriesFile

	// Retrieve input arguments
	args := request.Params.Arguments

	// Retrieve and validate the 'query_id' parameter.
	queryID, ok := args["query_id"].(string)
	if !ok || strings.TrimSpace(queryID) == "" {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: "'query_id' parameter is required",
				},
			},
		}, nil
	}

	// Retrieve and validate the 'new_answer' parameter.
	newAnswer, ok := args["new_answer"].(string)
	if !ok || strings.TrimSpace(newAnswer) == "" {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: "'new_answer' parameter is required",
				},
			},
		}, nil
	}

	// Load the existing queries data from the queries file.
	queriesData, err := core.LoadQueries(queriesFile)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error loading queries file: %v", err),
				},
			},
		}, nil
	}

	// Verify that a query exists with the provided queryID.
	query, exists := queriesData.Queries[queryID]
	if !exists {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("No query found for id: %s", queryID),
				},
			},
		}, nil
	}

	// Update the "answer" field with the new answer.
	query.Answer = newAnswer
	queriesData.Queries[queryID] = query

	// Save the updated queries data back to the queries file.
	if err := core.SaveQueries(queriesFile, queriesData); err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error saving updated queries file: %v", err),
				},
			},
		}, nil
	}

	// Return a success response.
	return &mcp_lib.CallToolResult{
		Content: []mcp_lib.Content{
			mcp_lib.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Successfully updated answer for query_id '%s'.", queryID),
			},
		},
	}, nil
}

// HandleGetActiveUsersTool retrieves the active/inactive users from the server
// and returns the information in a mcp_lib.CallToolResult.
func HandleGetActiveUsersTool(ctx context.Context, request mcp_lib.CallToolRequest) (*mcp_lib.CallToolResult, error) {
	// Retrieve the DK (client) from the context.
	dkClient, err := utils.DkFromContext(ctx)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error retrieving client from context: %s", err.Error()),
				},
			},
		}, nil
	}

	// Get the active users using the client method.
	userStatus, err := dkClient.GetActiveUsers()
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to get active users: %s", err.Error()),
				},
			},
		}, nil
	}

	// Format the result as JSON for a nice display.
	resultJSON, err := json.MarshalIndent(userStatus, "", "  ")
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error formatting result: %s", err.Error()),
				},
			},
		}, nil
	}

	// Return the active/inactive users wrapped in a CallToolResult.
	return &mcp_lib.CallToolResult{
		Content: []mcp_lib.Content{
			mcp_lib.TextContent{
				Type: "text",
				Text: string(resultJSON),
			},
		},
	}, nil
}

// Tool: Get User Descriptions
// This tool retrieves the list of descriptions for a given user by invoking dkclient.GetUserDescriptions.
func HandleGetUserDatasetsTool(ctx context.Context, request mcp_lib.CallToolRequest) (*mcp_lib.CallToolResult, error) {
	// Retrieve the tool arguments.
	args := request.Params.Arguments
	userID, ok := args["user_id"].(string)
	if !ok || strings.TrimSpace(userID) == "" {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: "'user_id' parameter is required",
				},
			},
		}, nil
	}

	// Retrieve the DK client from the context.
	dkClient, err := utils.DkFromContext(ctx)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to retrieve DK client from context: %s", err.Error()),
				},
			},
		}, nil
	}

	// Call the client's GetUserDescriptions method.
	descriptions, err := dkClient.GetUserDescriptions(userID)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to get user descriptions: %s", err.Error()),
				},
			},
		}, nil
	}

	// Format the descriptions list as a JSON string.
	formatted, err := json.MarshalIndent(descriptions, "", "  ")
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error formatting descriptions: %s", err.Error()),
				},
			},
		}, nil
	}

	// Wrap the result in a CallToolResult.
	return &mcp_lib.CallToolResult{
		Content: []mcp_lib.Content{
			mcp_lib.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Given the following list of items, represent it in a bullet list format %s", string(formatted)),
			},
		},
	}, nil
}

func HandleGetPendingApplicationsTool(ctx context.Context, request mcp_lib.CallToolRequest) (*mcp_lib.CallToolResult, error) {
	//----------------------------------------------------------------------
	// 0. Pull Syftbox parameters out of context (unchanged)
	//----------------------------------------------------------------------
	parameters, err := utils.ParamsFromContext(ctx)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{Type: "text", Text: fmt.Sprintf("Couldn't retrieve params from context: %s", err)},
			},
		}, nil
	}

	cfgBytes, err := os.ReadFile(*parameters.SyftboxConfig)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{Type: "text", Text: fmt.Sprintf("Couldn't read Syftbox config at %s", *parameters.SyftboxConfig)},
			},
		}, nil
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
	if err := json.Unmarshal(cfgBytes, &syftboxConfig); err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{Type: "text", Text: "Failed to parse syftbox config; please verify the file format."},
			},
		}, nil
	}

	//----------------------------------------------------------------------
	// 1. List entries in the inbox
	//----------------------------------------------------------------------
	inboxPath := filepath.Join(syftboxConfig.DataDir, "datasites", syftboxConfig.Email, "inbox")
	dirEntries, err := os.ReadDir(inboxPath)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{Type: "text", Text: fmt.Sprintf("Failed to read inbox directory: %s", err)},
			},
		}, nil
	}

	var inboxNames []string
	for _, de := range dirEntries {
		switch de.Name() {
		case "approved", "rejected", "syftperm.yaml":
			// Skip bookkeeping directories / files
		default:
			inboxNames = append(inboxNames, de.Name())
		}
	}

	//----------------------------------------------------------------------
	// 2. Load app_request.json (best‑effort)
	//----------------------------------------------------------------------
	type appRequest struct {
		RequestedBy string `json:"requested_by"`
		Description string `json:"description"`
		Status      string `json:"status"`
		Reason      string `json:"reason"`
		Safety      string `json:"safety"`
	}

	appRequestPath := filepath.Dir(*parameters.ModelConfigFile)
	appReqPath := filepath.Join(appRequestPath, "app_request.json")
	appReqMap := map[string]appRequest{}

	if data, err := os.ReadFile(appReqPath); err == nil {
		_ = json.Unmarshal(data, &appReqMap) // ignore errors → leave map empty on malformed JSON
	}

	//----------------------------------------------------------------------
	// 3. Merge inbox + app_request.json
	//----------------------------------------------------------------------
	const undef = "undefined"

	type summary struct {
		AppName     string `json:"app_name"`
		RequestedBy string `json:"requested_by"`
		Description string `json:"description"`
		Safety      string `json:"safety"`
		Reason      string `json:"reason"`
		Status      string `json:"status"`
	}

	var pending []summary
	for _, name := range inboxNames {
		if ar, ok := appReqMap[name]; ok {
			pending = append(pending, summary{
				AppName:     name,
				RequestedBy: ar.RequestedBy,
				Description: ar.Description,
				Safety:      ar.Safety,
				Reason:      ar.Reason,
				Status:      ar.Status,
			})
		} else {
			pending = append(pending, summary{
				AppName:     name,
				RequestedBy: undef,
				Description: undef,
				Safety:      undef,
				Reason:      undef,
				Status:      "pending",
			})
		}
	}

	//----------------------------------------------------------------------
	// 4. Pretty‑print & wrap in CallToolResult
	//----------------------------------------------------------------------
	out, err := json.MarshalIndent(pending, "", "  ")
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{Type: "text", Text: fmt.Sprintf("Error formatting pending list: %s", err)},
			},
		}, nil
	}

	return &mcp_lib.CallToolResult{
		Content: []mcp_lib.Content{
			mcp_lib.TextContent{
				Type: "text",
				// The agent that calls this tool can turn the JSON into a markdown table
				Text: fmt.Sprintf("Return the list of pending applications details in markdown tabular format. %s", string(out)),
				// Text: fmt.Sprintf("Pending application details (JSON):\n\n```json\n%s\n```", string(out)),
			},
		},
	}, nil
}

func HandleProcessApplicationRequestTool(ctx context.Context, request mcp_lib.CallToolRequest) (*mcp_lib.CallToolResult, error) {
	// Retrieve the tool arguments.
	args := request.Params.Arguments
	appName, ok := args["app_name"].(string)
	if !ok || strings.TrimSpace(appName) == "" {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: "'app_name' parameter is required",
				},
			},
		}, nil
	}

	approval, ok := args["approve"].(bool)
	if !ok || strings.TrimSpace(appName) == "" {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: "'approval' parameter is required",
				},
			},
		}, nil
	}

	parameters, err := utils.ParamsFromContext(ctx)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't retrieve params from context: %s", err.Error()),
				},
			},
		}, nil
	}

	file, err := os.ReadFile(*parameters.SyftboxConfig)
	if err != nil {
		// Wrap the result in a CallToolResult.
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't find Syftbox config file in path %s, please verify if this path exist", *parameters.SyftboxConfig),
				},
			},
		}, nil
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
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to parse the syftbox config file. Please check if your config file is set properly."),
				},
			},
		}, nil
	}

	appPath := filepath.Join(syftboxConfig.DataDir, "datasites", syftboxConfig.Email, "inbox", appName)

	prohibitedNames := appName == "approved" || appName == "rejected" || appName == "syftperm.yaml"
	if prohibitedNames {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("You can't approve the %s folder/file", appName),
				},
			},
		}, nil
	}

	_, err = os.Stat(appPath)
	if os.IsNotExist(err) {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("The app '%s' doesn't exist or isn't in pending state anymore. Please verify if you typed it properly.", appName),
				},
			},
		}, nil
	}

	approvalStatus := "approved"
	if approval {
		approvedPath := filepath.Join(syftboxConfig.DataDir, "apps", appName)
		os.Rename(appPath, approvedPath)
	} else {
		approvalStatus = "rejected"
		rejectedPath := filepath.Join(syftboxConfig.DataDir, "datasites", syftboxConfig.Email, "inbox", "rejected", appName)
		os.Rename(appPath, rejectedPath)
	}

	return &mcp_lib.CallToolResult{
		Content: []mcp_lib.Content{
			mcp_lib.TextContent{
				Type: "text",
				Text: fmt.Sprintf("The app '%s' has been %s successfully.", appName, approvalStatus),
			},
		},
	}, nil
}

func HandleSubmitAppFolderTool(ctx context.Context, request mcp_lib.CallToolRequest) (*mcp_lib.CallToolResult, error) {
	args := request.Params.Arguments
	appPath, ok := args["app_path"].(string)
	if !ok || strings.TrimSpace(appPath) == "" {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: "'app_path' parameter is required",
				},
			},
		}, nil
	}

	appDescription, ok := args["description"].(string)
	if !ok || strings.TrimSpace(appDescription) == "" {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: "'description' parameter is required",
				},
			},
		}, nil
	}

	var peers []string
	if r, exists := args["peers"]; exists {
		for _, item := range r.([]any) {
			if str, ok := item.(string); ok {
				peers = append(peers, str)
			}
		}
	}

	result, err := core.ScanDirToMap(ctx, appPath)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: "'app_path' parameter is required",
				},
			},
		}, nil
	}

	dkClient, err := utils.DkFromContext(ctx)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't retrieve DK from context: %s", err.Error()),
				},
			},
		}, nil
	}
	query := utils.RemoteMessage{
		Type:    "app",
		Message: appDescription,
		Files:   result,
	}
	jsonData, err := json.Marshal(query)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't marshal query: %s", err.Error()),
				},
			},
		}, nil
	}
	if len(peers) == 0 {
		err = dkClient.BroadcastMessage(string(jsonData))
		if err != nil {
			return &mcp_lib.CallToolResult{
				Content: []mcp_lib.Content{
					mcp_lib.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Couldn't send message: %s", err.Error()),
					},
				},
			}, nil
		}
	} else {
		for _, peer := range peers {
			err = dkClient.SendMessage(dk_client.Message{
				From:      dkClient.UserID,
				To:        peer,
				Content:   string(jsonData),
				Timestamp: time.Now(),
			})
			if err != nil {

				return &mcp_lib.CallToolResult{
					Content: []mcp_lib.Content{
						mcp_lib.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Couldn't send message: %s", err.Error()),
						},
					},
				}, nil
			}
		}
	}

	return &mcp_lib.CallToolResult{
		Content: []mcp_lib.Content{
			mcp_lib.TextContent{
				Type: "text",
				Text: "Application sent successfully!",
			},
		},
	}, nil

}
