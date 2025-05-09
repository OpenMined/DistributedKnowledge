package mcp

import (
	mcp_lib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func NewMCPServer() *server.MCPServer {
	mcpServer := server.NewMCPServer(
		"openmined/dk-server",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
		server.WithLogging(),
	)

	// Tool: Ask Question
	mcpServer.AddTool(
		mcp_lib.NewTool("cqAskQuestion",
			mcp_lib.WithDescription("Send a question to specified peers (identified by their '@' prefix) or broadcast to the entire network."),
			mcp_lib.WithString(
				"question",
				mcp_lib.Description("The text of the question to send."),
				mcp_lib.Required(),
			),
			mcp_lib.WithArray(
				"peers",
				mcp_lib.Description("List of peer identifiers (without '@') to receive the question. Leave empty to broadcast to all peers."),
				mcp_lib.Items(map[string]any{"type": "string"}),
				mcp_lib.Required(),
			),
		),
		HandleAskTool,
	)

	// Tool: List Queries
	mcpServer.AddTool(
		mcp_lib.NewTool("cqListRequestedQueries",
			mcp_lib.WithDescription("Retrieve all requested queries, optionally filtered by status or sender."),
			mcp_lib.WithString(
				"status",
				mcp_lib.Description("Optional status filter (e.g., 'pending', 'accepted')."),
			),
			mcp_lib.WithString(
				"from",
				mcp_lib.Description("Optional sender filter (peer identifier)."),
			),
		),
		HandleListQueriesTool,
	)

	// Tool: Add Auto Approval Condition
	mcpServer.AddTool(
		mcp_lib.NewTool("cqAddAutoApprovalCondition",
			mcp_lib.WithDescription("Extract a conditional rule from a sentence and append it to automatic_approval.json."),
			mcp_lib.WithString(
				"sentence",
				mcp_lib.Description("Sentence containing the condition to add."),
				mcp_lib.Required(),
			),
		),
		HandleAddApprovalConditionTool,
	)

	// Tool: Remove Auto Approval Condition
	mcpServer.AddTool(
		mcp_lib.NewTool("cqRemoveAutoApprovalCondition",
			mcp_lib.WithDescription("Remove a rule from automatic_approval.json by its exact text."),
			mcp_lib.WithString(
				"condition",
				mcp_lib.Description("Exact text of the condition to remove."),
				mcp_lib.Required(),
			),
		),
		HandleRemoveApprovalConditionTool,
	)

	// Tool: List Auto Approval Conditions
	mcpServer.AddTool(
		mcp_lib.NewTool("cqListAutoApprovalConditions",
			mcp_lib.WithDescription("List all automatic approval conditions stored in automatic_approval.json."),
		),
		HandleListApprovalConditionsTool,
	)

	// Tool: Accept Query
	mcpServer.AddTool(
		mcp_lib.NewTool("cqProcessQuery",
			mcp_lib.WithDescription("Mark a pending query as 'accepted' or 'rejected'."),
			mcp_lib.WithString(
				"id",
				mcp_lib.Description("Unique identifier of the query to accept."),
				mcp_lib.Required(),
			),
			mcp_lib.WithBoolean(
				"approve",
				mcp_lib.Description("A boolean flag to identify if the pending query is accepted or rejected."),
				mcp_lib.Required(),
			),
		),
		HandleProcessQuestionTool,
	)

	mcpServer.AddTool(
		mcp_lib.NewTool("cqSummarizeAnswers",
			// What this tool does, in one precise sentence
			mcp_lib.WithDescription(
				"Retrieve all peer responses for a given question, evaluate each answer, and return a cohesive summary that highlights the key insights.",
			),

			// The question to look up
			mcp_lib.WithString(
				"related_question",
				mcp_lib.Description(
					"The exact question or topic for which peer responses should be fetched and analyzed.",
				),
				mcp_lib.Required(),
			),

			// Whether to go deep or stay brief
			mcp_lib.WithNumber(
				"detailed_answer",
				mcp_lib.Description(
					"Detail level flag: set to 1 to receive an in‑depth, comprehensive answer; set to 0 for a concise, high‑level summary.",
				),
				mcp_lib.DefaultBool(false),
			),
		),
		HandleAnswerListTool,
	)

	// Tool: Update RAG Knowledge Base
	mcpServer.AddTool(mcp_lib.NewTool("updateKnowledgeSources",
		mcp_lib.WithDescription("Updates knowledge sources by saving provided file name and content or file path, then refreshing the vector database."),
		// Two string parameters: file_name and file_content.
		mcp_lib.WithString("file_name", mcp_lib.Description("The name of the file to add (e.g., mydocument.pdf)")),
		mcp_lib.WithString("file_content", mcp_lib.Description("The content of the file")),
		mcp_lib.WithString("file_path", mcp_lib.Description("The content of the file")),
	), HandleUpdateRagSourcesTool)

	// Tool: Update Answer Content
	mcpServer.AddTool(
		mcp_lib.NewTool("cqUpdateEditAnswer",
			mcp_lib.WithDescription("Edit an specific answer content with a new content."),
			mcp_lib.WithString(
				"query_id",
				mcp_lib.Description("Query ID of the answer that will get its content updated."),
				mcp_lib.Required(),
			),
			mcp_lib.WithString(
				"new_answer",
				mcp_lib.Description("New Answer to be updated."),
				mcp_lib.Required(),
			),
		),
		HandleUpdateAnswerTool,
	)

	// Tool: Get Active Users
	mcpServer.AddTool(
		mcp_lib.NewTool("cqGetUsers",
			mcp_lib.WithDescription("Retrieve active and inactive user lists from the network."),
			mcp_lib.WithBoolean(
				"flag",
				mcp_lib.DefaultBool(false),
			),
		),
		HandleGetActiveUsersTool,
	)

	// Tool: Get User Descriptions
	mcpServer.AddTool(
		mcp_lib.NewTool("cqGetUserDatasets",
			mcp_lib.WithDescription("Retrieve list of descriptions for a user."),
			mcp_lib.WithString("user_id",
				mcp_lib.Description("The ID of the user whose descriptions are requested."),
				mcp_lib.Required(),
			),
		),
		HandleGetUserDatasetsTool,
	)

	// Tool: Get Pending Application Requests
	mcpServer.AddTool(
		mcp_lib.NewTool("cqGetPendingApplications",
			mcp_lib.WithDescription("Retrieve a list of pending application requests in the network."),
			mcp_lib.WithBoolean(
				"flag",
				mcp_lib.DefaultBool(false),
			),
		),
		HandleGetPendingApplicationsTool,
	)

	// Tool: Accept or Deny Pending Application
	mcpServer.AddTool(
		mcp_lib.NewTool("cqProcessApplicationRequest",
			mcp_lib.WithDescription("Accept or deny a pending application request by its application name."),
			mcp_lib.WithString(
				"app_name",
				mcp_lib.Description("The name of the application to process."),
				mcp_lib.Required(),
			),
			mcp_lib.WithBoolean(
				"approve",
				mcp_lib.Description("Set to true to approve the application, false to deny."),
				mcp_lib.Required(),
			),
		),
		HandleProcessApplicationRequestTool,
	)

	// Tool: Submit App Folder
	mcpServer.AddTool(
		mcp_lib.NewTool("cqSubmitAppFolder",
			mcp_lib.WithDescription("Submit an application folder to specified peers or broadcast to the entire network."),
			mcp_lib.WithString(
				"app_path",
				mcp_lib.Description("The full path to the application folder to submit."),
				mcp_lib.Required(),
			),
			mcp_lib.WithString(
				"description",
				mcp_lib.Description("App description"),
				mcp_lib.Required(),
			),
			mcp_lib.WithArray(
				"peers",
				mcp_lib.Description("List of peer identifiers (without '@') to receive the app folder. Leave empty to broadcast to all peers."),
				mcp_lib.Items(map[string]any{"type": "string"}),
				mcp_lib.Required(),
			),
		),
		HandleSubmitAppFolderTool,
	)

	// Tool: Get Client Token
	mcpServer.AddTool(
		mcp_lib.NewTool("cqGetToken",
			mcp_lib.WithDescription("Retrieves the current JWT token used by the client for authentication."),
			mcp_lib.WithBoolean(
				"flag",
				mcp_lib.Description("Ignore this parameter"),
				mcp_lib.DefaultBool(false),
			),
		),
		HandleGetTokenTool,
	)

	return mcpServer
}
