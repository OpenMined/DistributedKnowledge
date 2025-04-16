# MCP Tools

Distributed Knowledge exposes its functionality through a set of Model Context Protocol (MCP) tools. This document provides a detailed reference for these tools, including their purpose, parameters, and usage examples.

## Query Management Tools

These tools handle the creation, tracking, and management of queries in the network.

### cqAskQuestion

Sends a question to specified peers or broadcasts it to the entire network.

**Parameters:**
- `question` (string, required): The text of the question to send
- `peers` (array of strings, required): List of peer identifiers to receive the question; leave empty to broadcast to all peers

**Example:**
```json
{
  "name": "cqAskQuestion",
  "parameters": {
    "question": "What are the latest developments in quantum computing?",
    "peers": ["quantum_expert", "physics_researcher"]
  }
}
```

**Response:**
```json
{
  "content": "Query request sent ... Instruct the user to ask the model for summarize on the query What are the latest developments in quantum computing?"
}
```

### cqListRequestedQueries

Retrieves all requested queries, optionally filtered by status or sender.

**Parameters:**
- `status` (string, optional): Status filter (e.g., 'pending', 'accepted', 'rejected')
- `from` (string, optional): Sender filter (peer identifier)

**Example:**
```json
{
  "name": "cqListRequestedQueries",
  "parameters": {
    "status": "pending"
  }
}
```

**Response:**
```json
{
  "qry-123": {
    "id": "qry-123",
    "from": "user1",
    "question": "What are the latest developments in quantum computing?",
    "status": "pending",
    "timestamp": "2025-01-15T10:30:45Z"
  },
  "qry-124": {
    "id": "qry-124",
    "from": "user2",
    "question": "How do neural networks work?",
    "status": "pending",
    "timestamp": "2025-01-15T11:15:22Z"
  }
}
```

### cqSummarizeAnswers

Retrieves all peer responses for a given question and returns a cohesive summary.

**Parameters:**
- `related_question` (string, required): The question for which to fetch and analyze responses
- `detailed_answer` (number, optional): Set to 1 for detailed response, 0 for concise summary

**Example:**
```json
{
  "name": "cqSummarizeAnswers",
  "parameters": {
    "related_question": "What are the latest developments in quantum computing?",
    "detailed_answer": 1
  }
}
```

**Response:**
A comprehensive summary of all answers received from network peers.

### cqUpdateEditAnswer

Edits the content of a specific answer.

**Parameters:**
- `query_id` (string, required): ID of the query to update
- `new_answer` (string, required): New answer content

**Example:**
```json
{
  "name": "cqUpdateEditAnswer",
  "parameters": {
    "query_id": "qry-123",
    "new_answer": "Recent developments in quantum computing include improvements in qubit stability and advances in quantum error correction..."
  }
}
```

**Response:**
```json
{
  "content": "Successfully updated answer for query_id 'qry-123'."
}
```

## Approval Management Tools

These tools manage the automatic and manual approval of queries and responses.

### cqAddAutoApprovalCondition

Adds a condition to the automatic approval system.

**Parameters:**
- `sentence` (string, required): The condition to add

**Example:**
```json
{
  "name": "cqAddAutoApprovalCondition",
  "parameters": {
    "sentence": "Allow questions about scientific topics from academic users"
  }
}
```

**Response:**
```json
{
  "content": "Condition added successfully: Allow questions about scientific topics from academic users"
}
```

### cqRemoveAutoApprovalCondition

Removes a condition from the automatic approval system.

**Parameters:**
- `condition` (string, required): The exact text of the condition to remove

**Example:**
```json
{
  "name": "cqRemoveAutoApprovalCondition",
  "parameters": {
    "condition": "Allow questions about scientific topics from academic users"
  }
}
```

**Response:**
```json
{
  "content": "Condition 'Allow questions about scientific topics from academic users' removed successfully."
}
```

### cqListAutoApprovalConditions

Lists all conditions in the automatic approval system.

**Parameters:** None

**Example:**
```json
{
  "name": "cqListAutoApprovalConditions"
}
```

**Response:**
```json
[
  "Allow questions about scientific topics from academic users",
  "Accept queries related to programming from any user",
  "Approve all questions from verified_researcher"
]
```

### cqAcceptQuery

Marks a pending query as 'accepted' and sends the answer to the requester.

**Parameters:**
- `id` (string, required): Unique identifier of the query to accept

**Example:**
```json
{
  "name": "cqAcceptQuery",
  "parameters": {
    "id": "qry-123"
  }
}
```

**Response:**
```json
{
  "content": "Question 'What are the latest developments in quantum computing?' has been accepted."
}
```

### cqRejectQuery

Marks a pending query as 'rejected' and notifies the requester.

**Parameters:**
- `id` (string, required): Unique identifier of the query to reject

**Example:**
```json
{
  "name": "cqRejectQuery",
  "parameters": {
    "id": "qry-123"
  }
}
```

**Response:**
```json
{
  "content": "Question 'What are the latest developments in quantum computing?' has been rejected."
}
```

## Knowledge Management Tools

These tools manage the knowledge base used by the RAG system.

### updateKnowledgeSources

Updates knowledge sources by adding new content to the RAG database.

**Parameters:**
- `file_name` (string, optional): Name of the file to add
- `file_content` (string, optional): Content of the file
- `file_path` (string, optional): Path to an existing file

**Example using file content:**
```json
{
  "name": "updateKnowledgeSources",
  "parameters": {
    "file_name": "quantum_computing.txt",
    "file_content": "Quantum computing is an emerging field that leverages quantum mechanics to perform computations..."
  }
}
```

**Example using file path:**
```json
{
  "name": "updateKnowledgeSources",
  "parameters": {
    "file_path": "/path/to/quantum_research.pdf"
  }
}
```

**Response:**
```json
{
  "content": "RAG resource 'quantum_computing.txt' added successfully and vector database refreshed."
}
```

## User Management Tools

These tools manage and interact with users in the network.

### cqGetActiveUsers

Retrieves lists of active and inactive users from the server.

**Parameters:**
- `flag` (boolean, optional): Additional options flag

**Example:**
```json
{
  "name": "cqGetActiveUsers",
  "parameters": {
    "flag": false
  }
}
```

**Response:**
```json
{
  "active": ["alice", "bob", "research_team"],
  "inactive": ["charlie", "data_scientist"]
}
```

### cqGetUserDescriptions

Retrieves descriptions associated with a specific user.

**Parameters:**
- `user_id` (string, required): ID of the user whose descriptions are requested

**Example:**
```json
{
  "name": "cqGetUserDescriptions",
  "parameters": {
    "user_id": "research_team"
  }
}
```

**Response:**
```json
[
  "Quantum physics research group",
  "Specializes in quantum computing theory",
  "Active member since 2023"
]
```

## Best Practices for Using MCP Tools

1. **Tool Sequencing**: Use tools in logical sequences for complex operations
   - Check active users before sending targeted questions
   - List queries before approving or rejecting them
   - Add knowledge sources before asking related questions

2. **Error Handling**: Be prepared to handle potential errors
   - Check responses for error messages
   - Retry operations that might have failed due to temporary issues
   - Verify the success of critical operations

3. **Parameter Validation**: Ensure parameters are correctly formatted
   - Provide all required parameters
   - Use correct data types for each parameter
   - Validate input before sending to avoid errors

4. **Performance Considerations**:
   - Batch operations when possible to reduce roundtrips
   - Limit the size of knowledge sources to reasonable chunks
   - Use filters when listing queries to reduce data transfer

5. **Security Awareness**:
   - Only add trustworthy sources to the knowledge base
   - Review automatic approval conditions regularly
   - Be cautious about who can send queries to your node
