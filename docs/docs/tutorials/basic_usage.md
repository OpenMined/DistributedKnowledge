# Basic Usage Tutorial

This tutorial will guide you through the basic usage of Distributed Knowledge, from setup to sending your first query and receiving answers from the network.

## Prerequisites

Before starting, ensure you have:

- Installed Distributed Knowledge (see [Getting Started](../home/getting_started.md))
- Access to at least one LLM provider (Anthropic, OpenAI, or Ollama)
- Basic knowledge of terminal/command line usage

## Step 1: Create Configuration Files

First, let's create the necessary configuration files.

### Create a Model Configuration File

Create a file named `model_config.json` with your preferred LLM provider settings:

=== "Anthropic (Claude)"
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

=== "OpenAI (GPT)"
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

=== "Ollama"
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

### Create a RAG Sources File

Create a file named `rag_sources.jsonl` with some sample knowledge:

```json
{"text": "Distributed Knowledge is a decentralized, network-aware LLM system that enables collaborative intelligence across a network of nodes.", "file": "overview.txt"}
{"text": "The RAG system in Distributed Knowledge enables queries to be answered based on information stored in the vector database.", "file": "rag_system.txt"}
{"text": "Secure communication in Distributed Knowledge uses public key cryptography to verify the identity of network participants.", "file": "security.txt"}
```

## Step 2: Start Distributed Knowledge

Run the Distributed Knowledge client with basic configuration:

```bash
./dk -userId="tutorial_user" \
     -modelConfig="./model_config.json" \
     -rag_sources="./rag_sources.jsonl" \
     -server="wss://distributedknowledge.org"
```

You should see output indicating that the client has started and connected to the server.

## Step 3: Check Active Users

Let's see who's available on the network. You'll interact with the DK client using the MCP protocol:

```
> {"name": "cqGetActiveUsers"}
```

You should see a response like:

```json
{
  "active": ["alice", "bob", "tutorial_user"],
  "inactive": ["charlie", "research_team"]
}
```

This shows that you ("tutorial_user") and two other users ("alice" and "bob") are currently active.

## Step 4: Ask a Question

Now, let's ask a question to the network:

```
> {"name": "cqAskQuestion", "parameters": {"question": "How does the RAG system in Distributed Knowledge work?", "peers": ["alice"]}}
```

This sends your question specifically to the user "alice". You could also broadcast to all users by providing an empty peers array:

```
> {"name": "cqAskQuestion", "parameters": {"question": "How does the RAG system in Distributed Knowledge work?", "peers": []}}
```

After sending, you'll receive a confirmation that your query was sent.

## Step 5: Check for Responses

After a short while, you can check if any responses have arrived:

```
> {"name": "cqListRequestedQueries"}
```

You might see a response like:

```json
{
  "qry-456": {
    "id": "qry-456",
    "from": "alice",
    "question": "How does the RAG system in Distributed Knowledge work?",
    "answer": "The RAG (Retrieval Augmented Generation) system in Distributed Knowledge works by storing document embeddings in a vector database. When a query is received, it converts the query to an embedding and finds semantically similar documents. These documents are then used as context for the LLM to generate a comprehensive response. This allows the system to provide answers based on specific knowledge sources rather than just the LLM's training data.",
    "status": "accepted",
    "timestamp": "2025-04-16T14:30:22Z"
  }
}
```

## Step 6: Summarize Multiple Answers

If you broadcasted your question and received multiple responses, you can summarize them:

```
> {"name": "cqSummarizeAnswers", "parameters": {"related_question": "How does the RAG system in Distributed Knowledge work?", "detailed_answer": 1}}
```

This will give you a comprehensive summary combining insights from all responses.

## Step 7: Add to Your Knowledge Base

Let's add some new information to your RAG knowledge base:

```
> {"name": "updateKnowledgeSources", "parameters": {"file_name": "new_info.txt", "file_content": "Distributed Knowledge supports multiple LLM providers including Anthropic Claude, OpenAI GPT, and locally-hosted Ollama models."}}
```

This adds the new information to your vector database, making it available for future queries.

## Step 8: Set Up Automatic Approval

To streamline handling of incoming queries, set up some automatic approval rules:

```
> {"name": "cqAddAutoApprovalCondition", "parameters": {"sentence": "Accept questions about Distributed Knowledge architecture"}}
```

Add another rule:

```
> {"name": "cqAddAutoApprovalCondition", "parameters": {"sentence": "Accept questions from user alice"}}
```

Now, let's check our rules:

```
> {"name": "cqListAutoApprovalConditions"}
```

You should see:

```json
[
  "Accept questions about Distributed Knowledge architecture",
  "Accept questions from user alice"
]
```

## Step 9: Handle Incoming Queries

When someone asks you a question, it will either be automatically approved based on your rules or placed in a pending queue.

To check pending queries:

```
> {"name": "cqListRequestedQueries", "parameters": {"status": "pending"}}
```

If you have pending queries, you can accept one:

```
> {"name": "cqAcceptQuery", "parameters": {"id": "qry-789"}}
```

Or reject it:

```
> {"name": "cqRejectQuery", "parameters": {"id": "qry-789"}}
```

## Step 10: Update an Answer

If you want to modify a previous answer:

```
> {"name": "cqUpdateEditAnswer", "parameters": {"query_id": "qry-456", "new_answer": "The RAG (Retrieval Augmented Generation) system in Distributed Knowledge works by storing document embeddings in a vector database for efficient semantic retrieval. When a query is received, the system converts it to an embedding vector and performs a similarity search to find relevant documents. These documents are then used as context for the LLM, allowing it to generate answers grounded in specific knowledge sources. This approach combines the strengths of retrieval systems and generative AI."}}
```

## Next Steps

Congratulations! You've completed the basic usage tutorial for Distributed Knowledge. Here are some next steps to explore:

1. Learn about [advanced configuration options](../configuration/advanced.md)
2. Set up [your own network server](running_server.md)
3. Explore [integration with other systems](integration.md)
4. Experiment with [specialized knowledge domains](domain_expert.md)

Remember that Distributed Knowledge becomes more powerful as the network grows. Consider connecting with other users or running multiple nodes to experience the full potential of collaborative intelligence!
