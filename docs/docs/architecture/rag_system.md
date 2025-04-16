# RAG System

Distributed Knowledge incorporates a Retrieval Augmented Generation (RAG) system that enhances LLM responses with relevant information from a knowledge base. This document explains how the RAG system works in the Distributed Knowledge architecture.

## What is RAG?

Retrieval Augmented Generation (RAG) is a technique that combines:

1. **Information Retrieval**: Finding relevant documents or information from a knowledge base
2. **Content Generation**: Using an LLM to generate responses based on the retrieved information

This approach addresses limitations of traditional LLMs by providing:
- Access to knowledge beyond the model's training data
- More up-to-date information
- Source-based responses that can be verified
- Domain-specific expertise

## RAG Implementation in Distributed Knowledge

The Distributed Knowledge RAG system is implemented in `dk/core/rag.go` and consists of the following components:

### 1. Vector Database

The system uses a vector database to store and retrieve embeddings:

- **Document Embeddings**: Text is converted into numerical vectors
- **Semantic Search**: Finds documents based on meaning rather than keywords
- **Efficient Retrieval**: Quickly identifies the most relevant information
- **Persistent Storage**: Maintains the knowledge base between sessions

### 2. Document Ingestion

The system processes documents from various sources:

- **Source Format**: Documents are stored in JSONL format:
  ```json
  {"text": "Document content goes here", "file": "document_name.txt"}
  ```
- **Chunking**: Long documents are divided into manageable segments
- **Metadata Preservation**: Each chunk maintains its source information
- **Embedding Generation**: Text chunks are converted to vector embeddings

### 3. Query Processing

When a query is received:

- **Query Embedding**: The question is converted to a vector embedding
- **Similarity Search**: The system finds chunks most similar to the query
- **Context Assembly**: Relevant chunks are compiled into a context package
- **Prompt Construction**: The query and context are formatted for the LLM

### 4. Response Generation

The system generates responses based on retrieved information:

- **Context Integration**: The LLM receives both query and retrieved context
- **Citation Generation**: Responses reference source documents
- **Confidence Assessment**: The system evaluates response reliability
- **Format Standardization**: Responses follow a consistent structure

## Using the RAG System

### Configuring RAG Sources

RAG sources are defined in a JSONL file with the following format:

```json
{"text": "Einstein published the theory of relativity in 1905.", "file": "physics.txt"}
{"text": "Machine learning models use mathematical algorithms to improve through experience.", "file": "ai.txt"}
```

This file is specified using the `-rag_sources` parameter:

```bash
./dk -rag_sources=./data/knowledge_base.jsonl
```

### Updating the Knowledge Base

New documents can be added to the RAG system using the `updateKnowledgeSources` MCP tool:

```json
{
  "name": "updateKnowledgeSources",
  "parameters": {
    "file_name": "new_research.txt",
    "file_content": "The latest research shows significant progress in quantum computing..."
  }
}
```

Alternatively, you can specify a file path:

```json
{
  "name": "updateKnowledgeSources",
  "parameters": {
    "file_path": "/path/to/document.pdf"
  }
}
```

### Vector Database Configuration

The vector database location is configured with the `-vector_db` parameter:

```bash
./dk -vector_db=./data/vector_database
```

## Federated RAG Capabilities

Distributed Knowledge extends traditional RAG with network awareness:

- **Distributed Knowledge Sources**: Access information across multiple nodes
- **Expertise Routing**: Direct queries to nodes with relevant knowledge
- **Collaborative Response Generation**: Combine insights from multiple sources
- **Dynamic Knowledge Updates**: Incorporate new information in real-time

## Performance Considerations

Several factors affect RAG performance:

- **Vector Dimensions**: Higher dimensions provide more precise matching but use more memory
- **Database Size**: Larger knowledge bases require more resources but offer broader coverage
- **Chunk Size**: Smaller chunks enable more precise retrieval but increase database size
- **Similarity Threshold**: Higher thresholds improve relevance but may miss useful information

## Best Practices

To get the most from the RAG system:

1. **Organize Knowledge**: Group related information in well-structured documents
2. **Provide Context**: Include sufficient background in each document
3. **Update Regularly**: Keep the knowledge base current with new information
4. **Balance Coverage**: Include both broad and specialized knowledge
5. **Monitor Performance**: Track which sources are frequently retrieved

## Technical Implementation

The RAG system uses:
- **Vector Embeddings**: Generated using the Ollama `nomic-embed-text` model
- **Vector Search**: Employs cosine similarity for matching
- **Document Chunking**: Uses overlapping segments to preserve context
- **Context Selection**: Chooses top-k most relevant chunks
- **Memory Management**: Efficiently handles large knowledge bases