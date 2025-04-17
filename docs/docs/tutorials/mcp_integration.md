# MCP Integration Tutorial

This tutorial explains how to integrate Distributed Knowledge with Model Context Protocol (MCP) enabled systems. You'll learn how to set up Distributed Knowledge as an MCP server and use it with compatible LLM applications.

## What is MCP?

Model Context Protocol (MCP) is a standardized way for LLM systems to access external tools and capabilities. By integrating Distributed Knowledge with MCP, you can:

- Access the collective intelligence of the network through standardized tools
- Incorporate network knowledge into LLM workflows
- Enable LLMs to query and interact with network peers
- Maintain a consistent interface across different systems

## Prerequisites

Before starting, ensure you have:

- Distributed Knowledge installed (see [Getting Started](../home/getting_started.md))
- An MCP-compatible host application (like Claude or another MCP-aware LLM interface)
- Basic familiarity with JSON configuration

## Step 1: Configure Distributed Knowledge

First, ensure you have a properly configured Distributed Knowledge installation:

```bash
./dk -userId="mcp_user" \
     -modelConfig="./config/model_config.json" \
     -rag_sources="./data/knowledge_base.jsonl" \
     -server="wss://distributedknowledge.org"
```

Test that it's working correctly by running a simple command:

```
> {"name": "cqGetActiveUsers"}
```

You should receive a list of active users, confirming that your setup is working.

## Step 2: Create MCP Configuration

Create an MCP configuration file that includes Distributed Knowledge as a server. Save this as `mcp_config.json`:

```json
{
  "mcpServers": {
    "DistributedKnowledge": {
      "command": "/path/to/dk",
      "args": [
        "-userId", "mcp_user",
        "-private", "/path/to/private_key.pem",
        "-public", "/path/to/public_key.pem",
        "-modelConfig", "/path/to/model_config.json",
        "-rag_sources", "/path/to/rag_sources.jsonl",
        "-server", "wss://distributedknowledge.org"
      ]
    }
  }
}
```

Adjust the paths and parameters to match your specific setup. This configuration tells the MCP host where to find the Distributed Knowledge executable and what parameters to use when launching it.

## Step 3: Configure Your MCP Host

The exact steps depend on which MCP host you're using. Here are configurations for common hosts:

=== "Claude.ai"

    In your Claude.ai settings, add the MCP configuration:
    
    1. Go to Settings > Experimental Features
    2. Enable "Model Context Protocol"
    3. Upload your `mcp_config.json` file
    4. Restart Claude if necessary

=== "Custom MCP Host"

    For a custom MCP host, follow that system's specific instructions for adding MCP servers, referencing your `mcp_config.json` file.

## Step 4: Test the Integration

Now that you've configured your MCP host to use Distributed Knowledge, test the integration:

1. Open your MCP host application
2. Start a new conversation
3. Ask the LLM to use the Distributed Knowledge tools

Example prompt:

```
Can you help me check which users are active on the Distributed Knowledge network?
```

The LLM should:

1. Recognize this as a task for Distributed Knowledge
2. Use the `cqGetActiveUsers` tool
3. Return the results to you

## Step 5: Use Advanced MCP Functions

Now try more advanced functions:

### Ask a Network Question

Prompt:

```
Please ask the Distributed Knowledge network about recent advances in quantum computing.
```

The LLM should:

1. Use the `cqAskQuestion` tool
2. Broadcast the question to the network
3. Inform you that the question has been sent

### Retrieve and Summarize Answers

After waiting for responses, ask:

```
Can you check if there are any responses to my quantum computing question and summarize them?
```

The LLM should:

1. Use the `cqListRequestedQueries` tool to check for responses
2. Use the `cqSummarizeAnswers` tool to create a summary
3. Present the results to you

### Update Knowledge Base

You can also add new information to your knowledge base:

```
Please add this information to the Distributed Knowledge base: "In 2025, researchers achieved a quantum advantage for the first time in protein folding simulations, using a 300-qubit quantum computer."
```

The LLM should:

1. Use the `updateKnowledgeSources` tool
2. Add the information to your RAG sources
3. Confirm the addition

## Step 6: Create Tool Sequences

The real power of MCP integration comes from combining tools into useful sequences. Here's an example workflow:

```
I'd like to do research on climate adaptation strategies. Can you:
1. Check which experts on climate are active on the network
2. Send them a specific question about regional adaptation approaches
3. Wait for responses and then summarize the different perspectives
```

The LLM should execute this multi-step process using the appropriate sequence of tools.

## Advanced MCP Integration

### Custom Tool Selection

You can specifically ask the LLM to use certain Distributed Knowledge tools:

```
Use the Distributed Knowledge tool cqGetUserDescriptions to get information about the user "climate_scientist".
```

### Tool-Specific Parameters

Provide specific parameters for tool calls:

```
Please use the cqSummarizeAnswers tool with detailed_answer set to 1 to get comprehensive information about climate adaptation strategies.
```

### Approval System Management

Manage your automatic approval system:

```
Please add an automatic approval rule to accept all questions related to climate science from verified researchers.
```

## Integration Best Practices

1. **Start Simple**: Begin with basic tool calls before complex sequences
2. **Be Specific**: Clearly state what you want the LLM to do with the tools
3. **Provide Context**: Give enough background so the LLM knows which tools to use
4. **Verify Results**: Double-check important information returned via the tools
5. **Use Appropriate Tools**: Learn which tools are best for specific tasks

## Troubleshooting

### Common Issues

| Issue | Solution |
|-------|----------|
| "Tool not found" errors | Verify your MCP configuration file and paths |
| Connection failures | Check that Distributed Knowledge is properly connected to the server |
| Permission errors | Ensure file permissions are correct for all paths in your configuration |
| Tool timeouts | For complex operations, break them into smaller steps |

### Diagnostic Steps

If you encounter problems:

1. Check that Distributed Knowledge runs properly outside of MCP
2. Verify your MCP configuration syntax
3. Look for error messages in your MCP host's logs
4. Try restarting both Distributed Knowledge and your MCP host
5. Simplify your requests until you identify the failing component

## Next Steps

Now that you've integrated Distributed Knowledge with MCP, consider:

1. Exploring all available [MCP tools](../features/mcp_tools.md)
2. Setting up [specialized knowledge domains](domain_expert.md)
3. Creating [custom workflows](../how-to-guides/mcp_workflows.md) for common tasks
4. Integrating with [multiple LLM systems](../how-to-guides/multi_llm_integration.md)

MCP integration makes Distributed Knowledge's capabilities available in a standardized way across different LLM platforms, enabling powerful collaborative intelligence workflows.
