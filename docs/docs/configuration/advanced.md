# Advanced Configuration

This document covers advanced configuration options for Distributed Knowledge, including tuning parameters, optimization strategies, and specialized setups.

## RAG System Configuration

The Retrieval Augmented Generation (RAG) system can be fine-tuned for different use cases:

### Vector Database Tuning

```bash
# Specify vector database location and options
./dk -vector_db="/path/to/vector_db" \
     -vector_dimensions=384 \
     -vector_similarity_threshold=0.75 \
     -max_context_chunks=10
```

| Parameter | Description | Default |
|-----------|-------------|---------|
| `-vector_dimensions` | Dimensionality of embeddings | 384 |
| `-vector_similarity_threshold` | Minimum similarity score (0-1) | 0.65 |
| `-max_context_chunks` | Maximum chunks to include in context | 5 |

### Document Processing

Advanced options for processing documents:

```bash
# Configure document chunking and processing
./dk -chunk_size=512 \
     -chunk_overlap=50 \
     -min_chunk_size=100 \
     -embedding_batch_size=32
```

| Parameter | Description | Default |
|-----------|-------------|---------|
| `-chunk_size` | Target size for document chunks | 512 |
| `-chunk_overlap` | Overlap between adjacent chunks | 50 |
| `-min_chunk_size` | Minimum chunk size to process | 100 |
| `-embedding_batch_size` | Batch size for embedding generation | 16 |

## LLM Parameters

Fine-tune the language model behavior:

```bash
# Advanced LLM settings via environment variables
export DK_SYSTEM_PROMPT="You are a knowledgeable assistant specialized in physics."
export DK_MAX_RETRY_ATTEMPTS=3
export DK_TIMEOUT_SECONDS=30
./dk -modelConfig="./config/model_config.json"
```

| Environment Variable | Description | Default |
|-----------|-------------|---------|
| `DK_SYSTEM_PROMPT` | System prompt for the LLM | Generic helpful assistant prompt |
| `DK_MAX_RETRY_ATTEMPTS` | Number of LLM API retries | 2 |
| `DK_TIMEOUT_SECONDS` | Timeout for LLM API calls | 60 |
| `DK_DEFAULT_TEMPERATURE` | Default temperature if not in config | 0.7 |

### Model Configuration Overrides

You can override specific model configuration values via command line:

```bash
./dk -modelConfig="./config/base_config.json" \
     -model_override="claude-3-opus-20240229" \
     -temperature_override=0.3
```

## Network Configuration

Advanced networking options:

```bash
# Configure WebSocket behavior
./dk -ws_ping_interval=30 \
     -ws_timeout=10 \
     -max_message_size=1048576 \
     -reconnect_delay=5
```

| Parameter | Description | Default |
|-----------|-------------|---------|
| `-ws_ping_interval` | WebSocket ping interval (seconds) | 30 |
| `-ws_timeout` | WebSocket operation timeout (seconds) | 10 |
| `-max_message_size` | Maximum WebSocket message size (bytes) | 1048576 |
| `-reconnect_delay` | Delay between reconnection attempts (seconds) | 5 |

### TLS Configuration

For secure connections with custom certificates:

```bash
./dk -tls_cert="/path/to/client.crt" \
     -tls_key="/path/to/client.key" \
     -tls_ca="/path/to/ca.crt" \
     -skip_tls_verify=false
```

| Parameter | Description | Default |
|-----------|-------------|---------|
| `-tls_cert` | Path to TLS certificate | None |
| `-tls_key` | Path to TLS private key | None |
| `-tls_ca` | Path to CA certificate | None |
| `-skip_tls_verify` | Skip TLS certificate verification | false |

## Query and Answer Management

Configure how queries and responses are handled:

```bash
./dk -queriesFile="/data/queries.json" \
     -answersFile="/data/answers.json" \
     -query_retention_days=30 \
     -max_pending_queries=100
```

| Parameter | Description | Default |
|-----------|-------------|---------|
| `-query_retention_days` | Days to retain query history | 90 |
| `-max_pending_queries` | Maximum pending queries to store | 50 |
| `-answer_cache_size` | Number of answers to cache in memory | 100 |
| `-query_batch_size` | Number of queries to process in batch | 10 |

## Logging Configuration

Detailed logging configuration:

```bash
./dk -log_level="debug" \
     -log_file="/var/log/dk.log" \
     -log_format="json" \
     -log_max_size=100 \
     -log_max_files=5 \
     -log_compress=true
```

| Parameter | Description | Default |
|-----------|-------------|---------|
| `-log_level` | Logging level (debug, info, warn, error) | info |
| `-log_file` | Path to log file | stdout |
| `-log_format` | Log format (text or json) | text |
| `-log_max_size` | Maximum log file size in MB before rotation | 100 |
| `-log_max_files` | Maximum number of rotated log files to keep | 5 |
| `-log_compress` | Compress rotated log files | false |

## Performance Optimization

Settings to optimize performance for different environments:

```bash
./dk -concurrent_queries=4 \
     -max_workers=8 \
     -memory_limit=1024 \
     -cache_size=256
```

| Parameter | Description | Default |
|-----------|-------------|---------|
| `-concurrent_queries` | Maximum concurrent queries to process | 2 |
| `-max_workers` | Maximum worker goroutines | 4 |
| `-memory_limit` | Maximum memory usage in MB | Unlimited |
| `-cache_size` | Size of in-memory cache in MB | 128 |

## Specialized Deployment Scenarios

### High-Volume Server

For systems handling many requests:

```bash
./dk -userId="high_volume_node" \
     -concurrent_queries=16 \
     -max_workers=32 \
     -vector_db="/fast/ssd/vector_db" \
     -ws_ping_interval=15 \
     -embedding_batch_size=64
```

### Low-Resource Environment

For systems with limited resources:

```bash
./dk -userId="edge_device" \
     -concurrent_queries=1 \
     -max_workers=2 \
     -memory_limit=512 \
     -chunk_size=256 \
     -max_context_chunks=3 \
     -vector_dimensions=128
```

### Specific Domain Expert

For a node specializing in a particular knowledge domain:

```bash
export DK_SYSTEM_PROMPT="You are a specialized assistant with expertise in quantum physics."
./dk -userId="quantum_expert" \
     -rag_sources="/data/quantum_physics.jsonl" \
     -vector_similarity_threshold=0.8 \
     -modelConfig="/config/specialized_model.json"
```

## Configuration File

Instead of using many command-line parameters, you can create a configuration file:

```json
{
  "user_id": "research_node",
  "server": "wss://distributedknowledge.org",
  "model_config": "./config/anthropic_config.json",
  "rag_sources": "./data/knowledge_base.jsonl",
  "vector_db": "./data/vector_database",
  "private_key": "./keys/private.pem",
  "public_key": "./keys/public.pem",
  "network": {
    "ws_ping_interval": 30,
    "ws_timeout": 10,
    "max_message_size": 1048576,
    "reconnect_delay": 5
  },
  "rag": {
    "vector_dimensions": 384,
    "vector_similarity_threshold": 0.75,
    "max_context_chunks": 10,
    "chunk_size": 512,
    "chunk_overlap": 50
  },
  "performance": {
    "concurrent_queries": 4,
    "max_workers": 8,
    "cache_size": 256
  },
  "logging": {
    "level": "info",
    "file": "./logs/dk.log",
    "format": "json",
    "max_size": 100,
    "max_files": 5,
    "compress": true
  }
}
```

Use the configuration file with:

```bash
./dk -config="./config/dk_config.json"
```

## Environment Variables

All configuration parameters can also be set via environment variables with the `DK_` prefix:

```bash
export DK_USER_ID="research_node"
export DK_SERVER="wss://distributedknowledge.org"
export DK_MODEL_CONFIG="./config/anthropic_config.json"
export DK_RAG_SOURCES="./data/knowledge_base.jsonl"
export DK_VECTOR_DB="./data/vector_database"
export DK_LOG_LEVEL="debug"
export DK_CONCURRENT_QUERIES="4"
./dk
```

## Best Practices

1. **Start Simple**: Begin with basic configuration and add advanced options as needed
2. **Monitor Performance**: Use logging to identify bottlenecks before adjusting parameters
3. **Test Changes**: Validate configuration changes in a development environment first
4. **Document Custom Settings**: Maintain documentation of your custom configuration
5. **Use Environment-Specific Configs**: Create different configurations for development, testing, and production
6. **Secure Sensitive Values**: Use environment variables for API keys and other sensitive data