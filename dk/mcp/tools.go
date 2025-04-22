package mcp

import (
	"context"
	"database/sql"
	dk_client "dk/client"
	"dk/core"
	"dk/db"
	"dk/utils"
	"encoding/json"
	"errors"
	"fmt"
	mcp_lib "github.com/mark3labs/mcp-go/mcp"
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
func HandleAnswerListTool(ctx context.Context, req mcp_lib.CallToolRequest) (*mcp_lib.CallToolResult, error) {
	dbHandler, err := utils.DatabaseFromContext(ctx)
	if err != nil {
		return &mcp_lib.CallToolResult{Content: []mcp_lib.Content{
			mcp_lib.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Couldn't retrieve database instace. %v", err.Error()),
			},
		}}, nil
	}

	all, err := db.AllAnswers(ctx, dbHandler)
	if err != nil {
		return &mcp_lib.CallToolResult{Content: []mcp_lib.Content{
			mcp_lib.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Couldn't retrieve all answers: %v", err.Error()),
			},
		}}, nil
	}
	raw, _ := json.MarshalIndent(all, "", "  ")

	args := req.Params.Arguments
	detail := "general"
	if d, ok := args["detailed_answer"].(bool); ok && d {
		detail = "detailed"
	}
	related, _ := args["related_topic"].(string)

	return &mcp_lib.CallToolResult{Content: []mcp_lib.Content{
		mcp_lib.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Given the Answers: %s, and related topic: %s, provide a %s answer.",
				string(raw), related, detail),
		},
	}}, nil
}

// Tool: Get Answers for Query
//
// This tool retrieves all answers associated with a given answer_id.
// The answers.json file is expected to have the following structure:
//
// Given an answer_id, this tool will load the file, check if the entry exists,
// and return the associated answers. In case of any error, the error message
// will be returned in the Text field of the CallToolResult.
func HandleGetAnswerTool(ctx context.Context, req mcp_lib.CallToolRequest) (*mcp_lib.CallToolResult, error) {
	dbInstance, err := utils.DatabaseFromContext(ctx)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't retrieve database instance %v", err.Error()),
				},
			},
		}, nil
	}

	args := req.Params.Arguments
	qID, _ := args["query"].(string)
	if strings.TrimSpace(qID) == "" {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("'query' parameter is required"),
				},
			},
		}, nil
	}

	// optional delay
	if d, ok := args["delay"].(float64); ok && d > 0 {
		time.Sleep(time.Duration(int(d)) * time.Second)
	}

	ans, err := db.AnswersForQuestion(ctx, dbInstance, qID)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error while trying to get the answers for question %s : %v", qID, err.Error()),
				},
			},
		}, nil
	}

	if len(ans) == 0 {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("No answers found for id: %s", qID),
				},
			},
		}, nil
	}
	raw, _ := json.MarshalIndent(ans, "", "  ")
	return &mcp_lib.CallToolResult{
		Content: []mcp_lib.Content{
			mcp_lib.TextContent{
				Type: "text",
				Text: fmt.Sprintf("%s", string(raw)),
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
	args := request.Params.Arguments
	statusFilter, _ := args["status"].(string)
	fromFilter, _ := args["from"].(string)

	dbInstance, err := utils.DatabaseFromContext(ctx)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't access the databse instance: %s", err.Error()),
				},
			},
		}, nil
	}

	list, err := db.ListQueries(ctx, dbInstance, statusFilter, fromFilter)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't retrieve the list of queries.: %s", err.Error()),
				},
			},
		}, nil
	}

	out, _ := json.MarshalIndent(list, "", "  ")
	return &mcp_lib.CallToolResult{Content: []mcp_lib.Content{
		mcp_lib.TextContent{Type: "text", Text: string(out)},
	}}, nil
}

// Tool: Add Automatic Approval Condition
//
// This tool extracts a condition from a sentence and appends it to the automatic_approval.json file.
// The file is expected to store an array of condition strings.
// Input parameter: "sentence" (the sentence containing the condition).
func HandleAddApprovalConditionTool(ctx context.Context, req mcp_lib.CallToolRequest) (*mcp_lib.CallToolResult, error) {
	dbHandle, err := utils.DatabaseFromContext(ctx)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't retrieve databse instance : %v'", err.Error()),
				},
			},
		}, nil
	}

	ruleRaw, ok := req.Params.Arguments["sentence"].(string)
	rule := strings.TrimSpace(ruleRaw)
	if !ok || rule == "" {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("'sentence' parameter is required", err.Error()),
				},
			},
		}, nil
	}

	if err := db.InsertRule(ctx, dbHandle, rule); err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't add the new rule into the automatic approval register : %v", err.Error()),
				},
			},
		}, nil
	}
	return &mcp_lib.CallToolResult{
		Content: []mcp_lib.Content{
			mcp_lib.TextContent{
				Type: "text",
				Text: fmt.Sprintf("New automatic approval rule '%s' added successfully.", rule),
			},
		},
	}, nil
}

// Tool: Remove Automatic Approval Condition
//
// This tool removes a specific condition from the automatic_approval.json file.
// Input parameter: "condition" (the exact text of the condition to remove).
func HandleRemoveApprovalConditionTool(ctx context.Context, req mcp_lib.CallToolRequest) (*mcp_lib.CallToolResult, error) {
	dbHandle, err := utils.DatabaseFromContext(ctx)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("DB unavailable: %v", err),
				},
			},
		}, nil
	}

	ruleRaw, ok := req.Params.Arguments["condition"].(string)
	rule := strings.TrimSpace(ruleRaw)
	if !ok || rule == "" {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("'condition' parameter is required"),
				},
			},
		}, nil
	}

	deleted, err := db.DeleteRule(ctx, dbHandle, rule)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Could not remove rule: %v", err.Error()),
				},
			},
		}, nil
		// return errorResult(fmt.Sprintf("Could not remove rule: %v", err)), nil
	}
	if !deleted {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Condition '%s' not found.", rule),
				},
			},
		}, nil
	}
	return &mcp_lib.CallToolResult{
		Content: []mcp_lib.Content{
			mcp_lib.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Condition '%s' removed successfully.", rule),
			},
		},
	}, nil
}

// Tool: List Automatic Approval Conditions
//
// This tool lists all the conditions in the automatic_approval.json file.
func HandleListApprovalConditionsTool(ctx context.Context, _ mcp_lib.CallToolRequest) (*mcp_lib.CallToolResult, error) {
	dbHandle, err := utils.DatabaseFromContext(ctx)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("DB unavailable: %v", err),
				},
			},
		}, nil
	}
	rules, err := db.ListRules(ctx, dbHandle)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Could not list rules: %v", err),
				},
			},
		}, nil
	}
	// pretty print like before
	blob, _ := json.MarshalIndent(rules, "", "  ")
	return &mcp_lib.CallToolResult{
		Content: []mcp_lib.Content{
			mcp_lib.TextContent{
				Type: "text",
				Text: fmt.Sprintf(string(blob)),
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

func HandleProcessQuestionTool(ctx context.Context, request mcp_lib.CallToolRequest) (*mcp_lib.CallToolResult, error) {
	id, _ := request.Params.Arguments["id"].(string)
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("'id' parameter is required")
	}

	approved, _ := request.Params.Arguments["approve"].(bool)

	dbInstance, err := utils.DatabaseFromContext(ctx)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error while trying to get db instance : %s", err.Error()),
				},
			},
		}, nil
	}

	var newStatus = "accepted"
	if !approved {
		newStatus = "rejected"
	}

	if err := db.UpdateQueryStatus(ctx, dbInstance, id, newStatus); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("query with ID '%s' not found", id)
		}
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error while trying to update the query status: %s", err.Error()),
				},
			},
		}, nil
	}

	qry, err := db.GetQuery(ctx, dbInstance, id)
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error while trying to get the query by its ID: %s", err.Error()),
				},
			},
		}, nil
	}

	if approved {
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
	}

	return &mcp_lib.CallToolResult{
		Content: []mcp_lib.Content{
			mcp_lib.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Question '%s' has been %s.\n", qry.Question, newStatus),
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
func HandleUpdateAnswerTool(
	ctx context.Context,
	request mcp_lib.CallToolRequest,
) (*mcp_lib.CallToolResult, error) {

	//----------------------------------------------------------------------
	// 1.  Grab the database handle from the context
	//----------------------------------------------------------------------
	// db, ok := ctx.Value("db").(*sql.DB) // replace if you use another helper
	dbHandler, err := utils.DatabaseFromContext(ctx)
	if err != nil {
		return &mcp_lib.CallToolResult{Content: []mcp_lib.Content{
			mcp_lib.TextContent{Type: "text", Text: "internal error: DB handle missing"},
		}}, nil
	}

	//----------------------------------------------------------------------
	// 2.  Read & validate input arguments
	//----------------------------------------------------------------------
	args := request.Params.Arguments

	queryID, _ := args["query_id"].(string)
	if strings.TrimSpace(queryID) == "" {
		return &mcp_lib.CallToolResult{Content: []mcp_lib.Content{
			mcp_lib.TextContent{Type: "text", Text: "'query_id' parameter is required"},
		}}, nil
	}

	newAnswer, _ := args["new_answer"].(string)
	if strings.TrimSpace(newAnswer) == "" {
		return &mcp_lib.CallToolResult{Content: []mcp_lib.Content{
			mcp_lib.TextContent{Type: "text", Text: "'new_answer' parameter is required"},
		}}, nil
	}

	//----------------------------------------------------------------------
	// 3.  Perform the UPDATE … SET answer = ? WHERE id = ?
	//     The query table was created in db.RunMigrations (see db.go).
	//----------------------------------------------------------------------
	res, err := dbHandler.ExecContext(ctx,
		`UPDATE queries SET answer = ? WHERE id = ?`, newAnswer, queryID)
	if err != nil {
		return &mcp_lib.CallToolResult{Content: []mcp_lib.Content{
			mcp_lib.TextContent{Type: "text", Text: fmt.Sprintf("database error: %v", err)},
		}}, nil
	}

	//----------------------------------------------------------------------
	// 4.  Check whether the row actually existed
	//----------------------------------------------------------------------
	if n, _ := res.RowsAffected(); n == 0 {
		return &mcp_lib.CallToolResult{Content: []mcp_lib.Content{
			mcp_lib.TextContent{Type: "text", Text: fmt.Sprintf("No query found for id: %s", queryID)},
		}}, nil
	}

	//----------------------------------------------------------------------
	// 5.  Success
	//----------------------------------------------------------------------
	return &mcp_lib.CallToolResult{Content: []mcp_lib.Content{
		mcp_lib.TextContent{
			Type: "text",
			Text: fmt.Sprintf("Successfully updated answer for query_id '%s'.", queryID),
		},
	}}, nil
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

	dbConn, err := utils.DatabaseFromContext(ctx)
	if err != nil {
		// fall back or error out
	}

	type summary struct {
		AppName     string `json:"app_name"`
		RequestedBy string `json:"requested_by"`
		Description string `json:"description"`
		Safety      string `json:"safety"`
		Reason      string `json:"reason"`
		Status      string `json:"status"`
	}

	var pending []summary
	var undef = "Undefined"
	for _, name := range inboxNames {
		ar, err := db.GetAppRequest(ctx, dbConn, name)
		if err == sql.ErrNoRows {
			pending = append(pending, summary{
				AppName:     name,
				RequestedBy: undef,
				Description: undef,
				Safety:      undef,
				Reason:      undef,
				Status:      "pending",
			})
		} else if err != nil {
			fmt.Printf("error loading app_request %q: %v", name, err)
			continue
		} else {
			pending = append(pending, summary{
				AppName:     ar.AppName,
				RequestedBy: ar.RequestedBy,
				Description: ar.AppDescription,
				Safety:      ar.Safety,
				Reason:      ar.Reason,
				Status:      ar.Status,
			})
		}
	}

	out, err := json.MarshalIndent(pending, "", "  ")
	if err != nil {
		return &mcp_lib.CallToolResult{
			Content: []mcp_lib.Content{
				mcp_lib.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Couldn't marshal the output result %v", err.Error()),
				},
			},
		}, nil
	}

	return &mcp_lib.CallToolResult{
		Content: []mcp_lib.Content{
			mcp_lib.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Return the list of pending applications details in markdown tabular format. %s", out),
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
