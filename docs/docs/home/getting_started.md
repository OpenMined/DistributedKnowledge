# Getting Started with Distributed Knowledge

This guide will help you set up and run your own instance of Distributed Knowledge, connecting to the network and leveraging collective intelligence for your AI applications.

## Prerequisites

Before getting started, make sure you have the following installed:

- **Go 1.x**: The system is built with Go, so you'll need a recent version installed
- **Access to LLM providers**: You'll need API access to at least one of:
  - Anthropic (Claude)
  - OpenAI (GPT models)
  - Ollama (for local LLM hosting)
- **Ollama with `nomic-embed-text` model**: Required for local RAG vector embeddings

## Installation

### Option 1: Download Prebuilt Binaries

Visit [distributedknowledge.org](https://distributedknowledge.org) to download prebuilt binaries for your platform.

### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/OpenMined/DistributedKnowledge.git
cd DistributedKnowledge

# Build the Distributed Knowledge client
go build -o dk
```

## Configuration 

### 1. Model Configuration

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

### 2. Set Up RAG Sources

RAG (Retrieval Augmented Generation) allows your system to reference specific knowledge sources. Create a JSONL file with your knowledge sources:

```json
{"text": "This is a document about quantum computing", "file": "quantum.txt"}
{"text": "Information about climate change impacts", "file": "climate.txt"}
```

Save this as `rag_sources.jsonl` or use the example in `dk/examples/rag_source_example.jsonl`.

### 3. Generate Authentication Keys (Optional)

For secure communications, you may want to generate your own key pair:

```bash
# Generate private key
openssl genpkey -algorithm RSA -out private_key.pem -pkeyopt rsa_keygen_bits:2048

# Extract public key
openssl rsa -pubout -in private_key.pem -out public_key.pem
```

## Running Distributed Knowledge

### Basic Usage

Run the Distributed Knowledge client with your configurations:

```bash
./dk -userId="YourUsername" \
     -modelConfig="path/to/model_config.json" \
     -rag_sources="path/to/rag_sources.jsonl" \
     -server="https://distributedknowledge.org"
```

### With Custom Keys

```bash
./dk -userId="YourUsername" \
     -private="path/to/private_key.pem" \
     -public="path/to/public_key.pem" \
     -modelConfig="path/to/model_config.json" \
     -server="https://distributedknowledge.org"
```

## Running Your Own Network Server

If you want to host your own Distributed Knowledge network:

```bash
# Navigate to the websocket server directory
cd websocketserver

# Build the server
go build

# Run the server
./websocketserver
```

The server will be available at `https://localhost:8080` by default.

## Integrating with MCP (Model Context Protocol)

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

## Next Steps

- Learn about [Architecture Concepts](../architecture/overview.md)
- Explore [Advanced Configuration](../configuration/advanced.md)
- Check out our [Tutorials](../tutorials/basic_usage.md) for common use cases
- Join the community on [GitHub](https://github.com/OpenMined/DistributedKnowledge)
