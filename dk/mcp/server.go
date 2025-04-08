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

	// Tool: Ask New Query (Remote)
	// mcpServer.AddTool(
	// 	mcp_lib.NewTool("cqAskQuestion",
	// 		mcp_lib.WithDescription("Creates a new query item to be sent to a peer/group of peers, identified by the '@' (eg. @peer1, @peer2, @peer3) or to the network (if no peer mentioned)."),
	// 		mcp_lib.WithString("question", mcp_lib.Description("The question for the query"), mcp_lib.Required()),
	//      mcp_lib.WithArray(
	//        "peers",
	//        mcp_lib.Description("The list of peers to send the query to. Remove the '@' from the peer name. If no peers are mentioned, peers list is empty and query."), mcp_lib.Required(),
	// 			mcp_lib.Items(map[string]any{"type": "string"}),
	//      ),
	// 	),
	// 	HandleAskTool,
	// )

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
		mcp_lib.NewTool("cqAcceptQuery",
			mcp_lib.WithDescription("Mark a pending query as 'accepted'."),
			mcp_lib.WithString(
				"id",
				mcp_lib.Description("Unique identifier of the query to accept."),
				mcp_lib.Required(),
			),
		),
		HandleAcceptQuestionTool,
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

	// Tool: Get Answer
	// mcpServer.AddTool(
	//     mcp_lib.NewTool("cqGetAnswer",
	//         mcp_lib.WithDescription("Fetch answers for a specified query."),
	//         mcp_lib.WithString(
	//             "query",
	//             mcp_lib.Description("The text of the query to retrieve answers for."),
	//             mcp_lib.Required(),
	//         ),
	//         mcp_lib.WithNumber(
	//           "delay",
	//           mcp_lib.Description("Delay in seconds before fetching answers."),
	//           mcp_lib.DefaultNumber(0),
	//        ),
	//     ),
	//     HandleGetAnswerTool,
	// )

	// Tool: Update RAG Knowledge Base
	mcpServer.AddTool(
		mcp_lib.NewTool("cqUpdateRagKnowledgeBase",
			mcp_lib.WithDescription("Add a document to the RAG knowledge base and refresh the vector database."),
			mcp_lib.WithString(
				"file_path",
				mcp_lib.Description("Filesystem path to the document to add."),
				mcp_lib.Required(),
			),
		),
		HandleUpdateRagSourcesTool,
	)

	// Tool: List Queries
	//  mcpServer.AddTool(
	//    mcp_lib.NewTool("cqListQueries",
	//      mcp_lib.WithDescription("Lists all queries."),
	//      mcp_lib.WithString("status", mcp_lib.Description("Filter by status (optional)")),
	//      mcp_lib.WithString("from", mcp_lib.Description("Filter by sender (optional)")),
	//    ),
	//    HandleListQueriesTool,
	//  )
	//
	// mcpServer.AddTool(
	// 	mcp_lib.NewTool("cqAddAutoApprovalCondition",
	// 		mcp_lib.WithDescription("Extracts a condition from a sentence and appends it to automatic_approval.json."),
	// 		mcp_lib.WithString("sentence", mcp_lib.Description("Sentence containing the condition"), mcp_lib.Required()),
	// 	),
	// 	HandleAddApprovalConditionTool,
	// )
	//
	//
	// mcpServer.AddTool(
	// 	mcp_lib.NewTool("cqRemoveAutoApprovalCondition",
	// 		mcp_lib.WithDescription("Removes a specific condition from automatic_approval.json."),
	// 		mcp_lib.WithString("condition", mcp_lib.Description("The condition text to remove"), mcp_lib.Required()),
	// 	),
	// 	HandleRemoveApprovalConditionTool,
	// )
	//
	// mcpServer.AddTool(
	// 	mcp_lib.NewTool("cqListAutoApprovalConditions",
	// 		mcp_lib.WithDescription("Lists all automatic approval conditions from automatic_approval.json."),
	// 		mcp_lib.WithBoolean("flag", mcp_lib.DefaultBool(false), mcp_lib.Description("Ignore this parameter")),
	// 	),
	// 	HandleListApprovalConditionsTool,
	// )
	//
	// mcpServer.AddTool(
	// 	mcp_lib.NewTool("cqAcceptQuery",
	// 		mcp_lib.WithDescription("Accepts a pending query by updating its status to 'accepted'."),
	// 		mcp_lib.WithString("id", mcp_lib.Description("ID of the query to accept"), mcp_lib.Required()),
	// 	),
	// 	HandleAcceptQuestionTool,
	// )
	//
	// // Tool: Get Answers
	// mcpServer.AddTool(
	// 	mcp_lib.NewTool("cqGetAnswer",
	// 		mcp_lib.WithDescription("Retrieves answer for a given query question."),
	// 		mcp_lib.WithString("query", mcp_lib.Description("Question of the query to retrieve answers for"), mcp_lib.Required()),
	// 	),
	// 	HandleGetAnswersTool,
	// )

	// -------------------------------------------------------------------------
	// New Tool Registration: Add RAG Resource
	// -------------------------------------------------------------------------
	// mcpServer.AddTool(
	// 	mcp_lib.NewTool("cqUpdateRagKnowledgeBase",
	// 		mcp_lib.WithDescription("Adds a new RAG resource by loading a file's content, appending it to the rag_sources file, and refreshing the vector database."),
	// 		mcp_lib.WithString("file_path", mcp_lib.Description("Path to the file to add as a RAG resource"), mcp_lib.Required()),
	// 	),
	// 	HandleUpdateRagSourcesTool,
	// )

	return mcpServer
}
