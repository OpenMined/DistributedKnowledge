# MCP Server

The Distributed Knowledge MCP (Model Context Protocol) server provides a structured interface for interacting with the network. This document explains how the MCP server works and the tools it provides.

## What is MCP?

MCP (Model Context Protocol) is a standardized way for LLM systems to access external tools and capabilities. The Distributed Knowledge MCP server integrates the network's functionality into this protocol, making it accessible to compatible clients.

## Server Configuration

The MCP server is initialized in `dk/mcp/server.go` with specific capabilities:

```go
mcpServer := server.NewMCPServer(
    "openmined/dk-server",
    "1.0.0",
    server.WithResourceCapabilities(true, true),
    server.WithPromptCapabilities(true),
    server.WithLogging(),
)
```

The server is configured with:
- Resource capabilities for file access
- Prompt capabilities for text generation
- Logging for diagnostic purposes

## Available Tools

The MCP server exposes several tools that clients can use to interact with the Distributed Knowledge network:

### Query Management Tools

1. **cqAskQuestion**
   - Sends a question to specified peers or broadcasts to the entire network
   - Parameters:
     - `question`: The text of the question to send (required)
     - `peers`: List of peer identifiers to receive the question (required)

2. **cqListRequestedQueries**
   - Retrieves all requested queries, optionally filtered by status or sender
   - Parameters:
     - `status`: Optional status filter (e.g., 'pending', 'accepted')
     - `from`: Optional sender filter (peer identifier)

3. **cqSummarizeAnswers**
   - Retrieves all peer responses for a given question and returns a summary
   - Parameters:
     - `related_question`: The question for which to fetch responses (required)
     - `detailed_answer`: Flag for detail level (0 for concise, 1 for in-depth)

4. **cqUpdateEditAnswer**
   - Edits the content of a specific answer
   - Parameters:
     - `query_id`: ID of the query to update (required)
     - `new_answer`: New answer content (required)

### Approval Management Tools

1. **cqAddAutoApprovalCondition**
   - Adds a condition to the automatic approval system
   - Parameters:
     - `sentence`: The condition to add (required)

2. **cqRemoveAutoApprovalCondition**
   - Removes a condition from the automatic approval system
   - Parameters:
     - `condition`: The exact text of the condition to remove (required)

3. **cqListAutoApprovalConditions**
   - Lists all conditions in the automatic approval system
   - No parameters required

4. **cqAcceptQuery**
   - Marks a pending query as 'accepted'
   - Parameters:
     - `id`: Unique identifier of the query to accept (required)

5. **cqRejectQuery**
   - Marks a pending query as 'rejected'
   - Parameters:
     - `id`: Unique identifier of the query to reject (required)

### Knowledge Management Tools

1. **updateKnowledgeSources**
   - Updates knowledge sources by adding new content to the RAG database
   - Parameters:
     - `file_name`: Name of the file to add
     - `file_content`: Content of the file
     - `file_path`: Path to an existing file

### User Management Tools

1. **cqGetActiveUsers**
   - Retrieves lists of active and inactive users from the server
   - Parameters:
     - `flag`: Boolean flag for additional options

2. **cqGetUserDescriptions**
   - Retrieves descriptions associated with a specific user
   - Parameters:
     - `user_id`: ID of the user whose descriptions are requested (required)

## Tool Implementation

Each tool is implemented as a Go function in `dk/mcp/tools.go` that:

1. Receives a context and request parameters
2. Performs the requested action
3. Returns a result or error

Here's a simplified example of a tool handler:

```go
func HandleAskTool(ctx context.Context, request mcp_lib.CallToolRequest) (*mcp_lib.CallToolResult, error) {
    // Extract arguments
    arguments := request.Params.Arguments
    message := arguments["question"].(string)
    
    // Process the request
    // ...
    
    // Return the result
    return &mcp_lib.CallToolResult{
        Content: []mcp_lib.Content{
            mcp_lib.TextContent{
                Type: "text",
                Text: "Query request sent successfully",
            },
        },
    }, nil
}
```

## Using the MCP Server

To use the Distributed Knowledge MCP server with compatible clients:

1. Configure the MCP client to connect to the server:
   ```json
   {
     "mcpServers": {
       "DistributedKnowledge": {
         "command": "dk",
         "args": [
           "-userId", "YourUsername",
           "-private", "/path/to/private_key",
           "-public", "/path/to/public_key",
           "-rag_sources", "/path/to/rag_sources.jsonl",
           "-server", "https://distributedknowledge.org"
         ]
       }
     }
   }
   ```

2. The client can then call tools using the MCP protocol:
   ```json
   {
     "name": "cqAskQuestion",
     "parameters": {
       "question": "What is quantum computing?",
       "peers": ["expertUser", "quantumResearcher"]
     }
   }
   ```

## Example Workflow

A typical workflow using the MCP server might look like:

1. Check which users are active using `cqGetActiveUsers`
2. Send a question to relevant peers using `cqAskQuestion`
3. Wait for responses to arrive
4. Retrieve and summarize the answers using `cqSummarizeAnswers`
5. Optionally edit answers using `cqUpdateEditAnswer`

## Security and Permissions

The MCP server respects the security model of the Distributed Knowledge network:

- Tool calls are authenticated based on the user's credentials
- Access to certain tools may be restricted based on permissions
- Sensitive operations require proper authorization
- All tool calls are logged for audit purposes

## Error Handling

The MCP server provides detailed error information:

- Invalid parameters result in clear error messages
- Network or system failures are reported with context
- Permission issues are explained with appropriate detail
- Unexpected errors are logged for troubleshooting
