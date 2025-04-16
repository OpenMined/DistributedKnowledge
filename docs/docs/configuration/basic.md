# Basic Configuration

Distributed Knowledge can be configured through command-line parameters and configuration files. This document covers the basic configuration options to get started with the system.

## Command Line Parameters

The Distributed Knowledge client (`dk`) accepts several command-line parameters:

| Parameter | Description | Default | Required |
|-----------|-------------|---------|----------|
| `-userId` | User identifier in the network | None | Yes |
| `-server` | WebSocket server URL | `wss://distributedknowledge.org` | Yes |
| `-modelConfig` | Path to LLM configuration file | `./model_config.json` | Yes |
| `-rag_sources` | Path to RAG source file (JSONL) | None | No |
| `-vector_db` | Path to vector database directory | `/tmp/vector_db` | No |
| `-private` | Path to private key file | None | No |
| `-public` | Path to public key file | None | No |
| `-project_path` | Root path for project files | Current directory | No |
| `-queriesFile` | Path to queries storage file | `./queries.json` | No |
| `-answersFile` | Path to answers storage file | `./answers.json` | No |
| `-automaticApproval` | Path to approval rules file | `./automatic_approval.json` | No |

### Example Usage

A basic command to start the Distributed Knowledge client:

```bash
./dk -userId="alice" \
     -server="wss://distributedknowledge.org" \
     -modelConfig="./config/anthropic_config.json" \
     -rag_sources="./data/knowledge_base.jsonl"
```

With additional authentication options:

```bash
./dk -userId="research_team" \
     -private="./keys/private.pem" \
     -public="./keys/public.pem" \
     -server="wss://distributedknowledge.org" \
     -modelConfig="./config/openai_config.json" \
     -vector_db="./data/vector_database"
```

## LLM Configuration

The LLM configuration file specifies which model provider and settings to use. It should be in JSON format:

### Example: Anthropic Configuration

```json
{
  "provider": "anthropic",
  "api_key": "sk-ant-your-anthropic-api-key",
  "model": "claude-3-sonnet-20240229",
  "parameters": {
    "temperature": 0.7,
    "max_tokens": 1000
  }
}
```

### Example: OpenAI Configuration

```json
{
  "provider": "openai",
  "api_key": "sk-your-openai-api-key",
  "model": "gpt-4",
  "parameters": {
    "temperature": 0.7,
    "max_tokens": 2000
  }
}
```

### Example: Ollama Configuration

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

## RAG Sources Configuration

The RAG (Retrieval Augmented Generation) system uses a JSONL file to define knowledge sources. Each line in this file represents a document in JSON format:

```json
{"text": "The capital of France is Paris.", "file": "geography.txt"}
{"text": "Water boils at 100 degrees Celsius at sea level.", "file": "science.txt"}
```

Each entry should include:
- `text`: The content of the document
- `file`: A name or identifier for the document

## Authentication Keys

For secure communication, Distributed Knowledge uses RSA key pairs. If not provided, temporary keys will be generated, but for production use, you should create and specify permanent keys.

### Generating Keys

Generate a key pair using OpenSSL:

```bash
# Generate private key
openssl genpkey -algorithm RSA -out private_key.pem -pkeyopt rsa_keygen_bits:2048

# Extract public key
openssl rsa -pubout -in private_key.pem -out public_key.pem
```

### Key Security

Important security considerations:
- Keep your private key secure and never share it
- The public key can be shared with others for verification
- Use appropriate file permissions (e.g., `chmod 600 private_key.pem`)

## Automatic Approval Configuration

The automatic approval system uses a JSON file containing an array of condition strings:

```json
[
  "Accept all questions about public information",
  "Allow queries from trusted peers",
  "Reject questions about personal data"
]
```

These conditions are used to determine which incoming queries should be automatically accepted or rejected.

## Directory Structure

A recommended directory structure for your Distributed Knowledge setup:

```
dk/
├── config/
│   ├── model_config.json
│   └── automatic_approval.json
├── data/
│   ├── knowledge_base.jsonl
│   └── vector_database/
├── keys/
│   ├── private_key.pem
│   └── public_key.pem
├── storage/
│   ├── queries.json
│   └── answers.json
└── dk  # executable
```

## Environment Variables

Distributed Knowledge also supports configuration through environment variables:

| Variable | Equivalent Parameter | Description |
|----------|---------------------|-------------|
| `DK_USER_ID` | `-userId` | User identifier |
| `DK_SERVER` | `-server` | WebSocket server URL |
| `DK_MODEL_CONFIG` | `-modelConfig` | Path to model config |
| `DK_RAG_SOURCES` | `-rag_sources` | Path to RAG sources |
| `DK_PRIVATE_KEY` | `-private` | Path to private key |
| `DK_PUBLIC_KEY` | `-public` | Path to public key |

Example using environment variables:

```bash
export DK_USER_ID="alice"
export DK_SERVER="wss://distributedknowledge.org"
export DK_MODEL_CONFIG="./config/anthropic_config.json"
./dk
```

## Next Steps

After completing basic configuration:

1. Learn about [advanced configuration options](advanced.md)
2. Explore [network configuration](network.md)
3. Set up [automatic approval rules](approval_system.md)
4. Configure [LLM parameters](llm_parameters.md)