<div align="center">
  <a href="https://distributedknowledge.org"> 
    <img src="websocketserver/static/images/dk_logo.png" alt="Distributed Knowledge Logo" style="display: block; margin: 0 auto; width: 200px;"/>
  </a>
  <h1>A decentralized, network-aware LLM system for collaborative intelligence.<h1>
  <div>
    <img src="https://badge.mcpx.dev/?type=server" />
     <img src="https://img.shields.io/badge/license-Apache%202.0-brightgreen?style=flat"/>
     <img src="https://img.shields.io/badge/PRs-welcome-brightgreen.svg"/>
  </div>
   <!-- <img src="https://gifyu.com/image/bp984"> -->
  <a href="https://gifyu.com/image/bp984"><img src="https://s6.gifyu.com/images/bp984.gif" alt="demo" border="0" /></a>
</div>


## Overview


Distributed Knowledge is an innovative approach to AI that turns the entire network into a unified LLM model. Rather than relying on a single monolithic model, this system creates a federated network where every node contributes to collective intelligence without central control.

## Key Features

- **Federated Architecture**: Decentralized by design, with no central control point.
- **Hybrid Privacy Model**: Public when it matters, private when it counts. Intelligence adapts to your privacy needs in real-time.
- **Dynamic Knowledge**: No more retraining. The model evolves with every interaction and stays current through network contributions.
- **Autonomous Operation**: Self-organizing systems that adapt, respond, and evolve without human intervention.
- **Open Ecosystem**: Not owned or controlled by any single entity. A public good with transparent governance and collective ownership.
- **Lightweight Design**: Access the web's collective knowledge without massive computing resources.

## Technical Features

- **Privacy by Design**: Your data is not shared with the network without your consent.
- **Real-time Synchronization**: Unlock access to your network's data in real-time.
- **Unified Context**: Grants AI access to a network-wide contextual knowledge base.
- **Ollama Compatible**: Easily connect and run your favorite Ollama models.
- **End-to-End Encryption**: Network peers are authenticated. Direct messages are signed and encrypted.
- **MCP Compatible**: Fully compatible with regular MCP Hosts.

## Getting Started

### Prerequisites

- Go 1.x
- Access to LLM providers (Anthropic, OpenAI, or Ollama)
- Ollama `nomic-embed-text` model (to generate local RAG)

### Installation

```bash
# Clone the repository
git clone https://github.com/OpenMined/DistributedKnowledge.git
cd DistributedKnowledge

# Build the MCP Project
go build -o dk
```

### Configuration

1. Create a model configuration file similar to the examples in `dk/examples/model_config/`.
2. Set up your RAG sources by following the example in `dk/examples/rag_source_example.jsonl`.
3. For MCP configuration, refer to `dk/examples/mcp_config_example.json`.

### Running your own Network Server

```bash
# Start the websocket server
cd websocketserver
go build
./websocketserver
```

### Adding the DK mcp Server into your LLM Workflow

```json
{
  "mcpServers": {
    "DistributedKnowledge": {
      "command": "dk",
      "args": [
        "-userId", "Bob",
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

## Contributing

We welcome contributions to the Distributed Knowledge project. Please see our contributing guidelines for details on how to get involved.

## License

This project is licensed under the terms of the LICENSE file included in the repository.

## Learn More

Visit [Distributed Knowledge](https://distributedknowledge.org) to learn more about the project.

## Project Maintainers

- OpenMined Organization

---
