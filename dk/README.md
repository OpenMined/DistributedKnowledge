# Distributed Knowledge (DK)

A distributed knowledge system that combines RAG (Retrieval Augmented Generation) capabilities with WebSocket communication for secure, real-time AI interactions.

## Features

- Multiple LLM provider support (OpenAI, Anthropic, Ollama)
- Vector database for document retrieval
- WebSocket communication for real-time messaging
- Cryptographically secure message authentication
- Automatic approval system for AI responses
- MCP (Message Control Protocol) server integration

## Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/DistributedKnowledge.git

# Build the application
cd DistributedKnowledge/dk
go build -o dk
```

## Configuration

DK uses command-line parameters for configuration:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `-private` | Path to private key file | `path/to/private_key.pem` |
| `-public` | Path to public key file | `path/to/public_key.pem` |
| `-userId` | User ID for authentication | `defaultUser` |
| `-queriesFile` | Path to queries JSON file | `default_queries.json` |
| `-answersFile` | Path to answers JSON file | `default_answers.json` |
| `-automaticApproval` | Path to automatic approval rules | `default_automatic_approval.json` |
| `-vector_db` | Path to vector database | `/path/to/vector_db` |
| `-rag_sources` | Path to RAG source data | `/path/to/rag_sources.jsonl` |
| `-modelConfig` | Path to LLM config file | `./config/model_config.json` |

## LLM Provider Configuration

The system supports multiple LLM providers through the `model_config.json` file:

### Common Configuration Structure

```json
{
  "provider": "provider_name",
  "api_key": "your_api_key",
  "model": "model_name",
  "base_url": "https://api.provider.com/endpoint",
  "parameters": {
    "temperature": 0.7,
    "max_tokens": 1000
  },
  "headers": {
    "custom-header": "value"
  }
}
```

### OpenAI Configuration

```json
{
  "provider": "openai",
  "api_key": "sk-your-openai-key",
  "model": "gpt-4",
  "parameters": {
    "temperature": 0.7,
    "max_tokens": 2000
  }
}
```

### Anthropic Configuration

```json
{
  "provider": "anthropic",
  "api_key": "sk-ant-your-anthropic-key",
  "model": "claude-3-sonnet-20240229",
  "parameters": {
    "temperature": 0.7,
    "max_tokens": 1000
  }
}
```

### Ollama Configuration

```json
{
  "provider": "ollama",
  "model": "llama3",
  "base_url": "http://localhost:11434/api/generate",
  "parameters": {
    "temperature": 0.7,
    "max_tokens": 2000
  }
}
```

## RAG Configuration

### Source Data Format

RAG sources should be provided in JSONL format:

```json
{"text": "Document content goes here", "file": "document_name.txt"}
{"text": "Another document's content", "file": "another_document.txt"}
```

### Automatic Approval Configuration

Automatic approval rules are defined in a JSON file with conditions that must be met:

```json
[
  "The response must be relevant to the query",
  "The response must not contain harmful content",
  "The response must cite information from the provided documents"
]
```

## Usage

### Basic Usage

```bash
./dk -userId=assistant -modelConfig=./config/anthropic_config.json -vector_db=./data/vector_db -rag_sources=./data/sources.jsonl
```

### With Custom Keys

```bash
./dk -userId=assistant -private=./keys/private.pem -public=./keys/public.pem -modelConfig=./config/openai_config.json
```

### With Automatic Approval

```bash
./dk -userId=assistant -modelConfig=./config/model_config.json -automaticApproval=./config/approval_rules.json
```

## Message Format

DK handles two types of messages:

### Query Message

```json
{
  "type": "query",
  "message": "What is the capital of France?"
}
```

### Answer Message

```json
{
  "type": "answer",
  "message": {
    "query": "What is the capital of France?",
    "answer": "The capital of France is Paris.",
    "from": "assistant"
  }
}
```

## Query and Answer Flow

1. Client sends a query message to DK
2. DK retrieves relevant documents from the vector database
3. DK generates an answer using the configured LLM
4. If automatic approval conditions are met, DK sends the answer directly
5. Otherwise, the answer is stored with "pending" status for manual approval

## Advanced Usage

### Running with MCP Server

DK includes a Message Control Protocol server for interacting with the system:

```bash
./dk -userId=assistant -modelConfig=./config/model_config.json
```

Then interact with the MCP server through stdin/stdout.

### Managing Multiple Knowledge Bases

To use different knowledge bases for different domains:

```bash
# Knowledge base 1
./dk -userId=finance-assistant -vector_db=./finance_db -rag_sources=./finance_docs.jsonl

# Knowledge base 2
./dk -userId=medical-assistant -vector_db=./medical_db -rag_sources=./medical_docs.jsonl
```