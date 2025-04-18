# Build from Source

This guide walks you through the process of building the client from source, configuring model providers, setting up Retrieval Augmented Generation (RAG) sources, and integrating with various AI platforms.

Whether you're using Anthropic's Claude, OpenAI's GPT models, or local Ollama instances, this documentation provides comprehensive instructions to get your Distributed Knowledge client up and running quickly and securely.

```bash
# Clone the repository
git clone https://github.com/OpenMined/DistributedKnowledge.git
cd DistributedKnowledge

# Build the Distributed Knowledge client
go build -o dk
```

# Configuration
If you downloaded the prebuilt binary or installed via installation script, you're ready to go. For source builds, continue with the configuration steps below.

## 1. Model Configuration

Create a model configuration file based on your preferred LLM provider. Examples are available in `dk/examples/model_config/`.

**Anthropic (Claude) Example:**

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

**OpenAI Example:**

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

**Ollama Example:**

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

## 2. Set Up RAG Sources

RAG (Retrieval Augmented Generation) allows your system to reference specific knowledge sources. Create a JSONL file with your knowledge sources:

```json
{"text": "This is a document about quantum computing", "file": "quantum.txt"}
{"text": "Information about climate change impacts", "file": "climate.txt"}
```

Save this as `rag_sources.jsonl` or use the example in `dk/examples/rag_source_example.jsonl`.

## 3. Generate Authentication Keys (Optional)

For secure communications, you may want to generate your own key pair:

```bash
# Generate private key
openssl genpkey -algorithm RSA -out private_key.pem -pkeyopt rsa_keygen_bits:2048

# Extract public key
openssl rsa -pubout -in private_key.pem -out public_key.pem
```

## Integrating with MCP Host (Claude Desktop App,etc) 

You can add Distributed Knowledge as an MCP server in your workflow:

```json
{
    "mcpServers": {
        "DistributedKnowledge": {
            "command": "dk",
            "args": [
                "-userId", "YourUsername",
                "-private", "/path/to/private_key",
                "-public", "/path/to/public_key",
                "-project_path", "/path/to/project",
                "-rag_sources", "/path/to/rag_sources.jsonl",
                "-server", "https://distributedknowledge.org"
            ]
        }
    }
}
```


