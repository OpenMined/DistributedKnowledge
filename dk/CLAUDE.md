# Distributed Knowledge (DK) Project Overview

## Core Architecture

Distributed Knowledge is a decentralized system for secure knowledge sharing and querying that combines:

1. **Vector Database Storage**: Uses Chromem for document retrieval through semantic search
2. **LLM Integration**: Leverages various AI providers (Anthropic, OpenAI, Ollama) for answering questions
3. **Secure Messaging**: Implements hybrid encryption and digital signatures for secure communication

## Key Components

### Core Engine (`core/`)
- **Message Processing**: Handles different message types (queries, answers, forwards)
- **RAG Implementation**: Retrieval-Augmented Generation with vector search
- **LLM Factory**: Modular integration with multiple AI providers
- **Context Management**: Smart context building for accurate AI responses

### Client Communication (`client/`)
- **WebSocket-based**: Real-time bidirectional communication
- **Cryptographic Security**: 
  - Ed25519 signatures for message authentication
  - Hybrid encryption (Ed25519/X25519 + AES-GCM) for direct messages
  - Public key management and caching

### Database Layer (`db/`)
- **SQLite Storage**: WAL mode for concurrent operations
- **Schema Design**:
  - `queries`: Stores user questions
  - `answers`: Tracks responses from users
  - `app_requests`: Manages application installation requests
  - `automatic_approval`: Rules for auto-approving answers

### API Interfaces
- **HTTP Server**: RESTful endpoints for document management
  - Document retrieval, addition, deletion, and updating
- **MCP Server**: Command interface with tools like:
  - Question asking and processing
  - Knowledge source management
  - User information retrieval
  - Automatic approval configuration

## Workflows

1. **Query Processing**:
   - Receive question → Retrieve relevant documents → Generate answer with LLM → Store in database → Apply approval rules → Return response

2. **Document Management**:
   - Add document → Generate vector embedding → Store in Chromem → Generate description with LLM → Make available for queries

3. **Secure Messaging**:
   - Generate symmetric key → Encrypt message → Encrypt key with recipient's public key → Sign package → Transmit → Verify signature → Decrypt

## Client Workflows and Edge Cases

### Client Communication Workflows

1. **Authentication Pipeline**:
   - **Registration**: New clients register by providing userID, username, and Ed25519 public key
   - **Login**: Challenge-response authentication mechanism:
     - Client requests challenge from server
     - Client signs challenge with private key
     - Server verifies signature and issues JWT token
     - Token is used for subsequent API calls and WebSocket connection

2. **Message Sending Pipeline**:
   - **Direct Messages**:
     - Generate hybrid encryption (AES-GCM + X25519)
     - Convert recipient's Ed25519 public key to X25519
     - Encrypt message content with AES-GCM
     - Encrypt AES key with recipient's public key
     - Sign the encrypted envelope with sender's Ed25519 private key
     - Transmit over WebSocket with signature

   - **Broadcast Messages**:
     - Sign message with sender's private key
     - Transmit in plaintext with signature

3. **Message Receiving Pipeline**:
   - Verify message signature using sender's cached public key
   - For direct messages:
     - Decrypt AES key using recipient's X25519 private key
     - Decrypt message content with AES-GCM
   - Add message status (verified, invalid_signature, unsigned, etc.)
   - Forward to application via message channel

4. **Connection Management**:
   - WebSocket connection with automatic reconnection
   - Exponential backoff retry mechanism (5s → 10s → 20s → ... → 60s max)
   - Ping/pong keepalive (54s ping interval, 60s read timeout)

### Key Security Features

- **Public Key Infrastructure**:
   - Client-side key caching to reduce server requests
   - Ed25519 for signatures, X25519 for asymmetric encryption
   - AES-GCM for symmetric encryption with random keys and nonces

- **Message Security Properties**:
   - Confidentiality: End-to-end encryption for direct messages
   - Integrity: Digital signatures verify message hasn't been tampered with
   - Authentication: Signatures verify sender identity
   - Non-repudiation: Signed messages can't be denied by sender

### Edge Cases and Error Handling

1. **Connection Disruptions**:
   - Automatic reconnection with exponential backoff
   - Graceful connection closure on client disconnect
   - Handling of read/write timeouts

2. **Cryptographic Failures**:
   - Failed signature verification: Message marked as "invalid_signature" but still delivered
   - Failed decryption: Message marked as "decryption_failed" but still delivered
   - Missing public key: Status set to "unverified" and fallback to server retrieval

3. **Message Validation**:
   - Timestamp verification to prevent replay attacks
   - Special handling for system messages and forwarded messages
   - Channel buffer limits (100 messages) to prevent memory exhaustion

4. **Error Recovery Strategies**:
   - Channel-based communication between components
   - Dead connection detection through ping/pong mechanism
   - Clean shutdown sequence to release resources
   - Write and send timeouts (10s) to prevent blocking operations

5. **Special Message Types**:
   - System messages bypass signature verification
   - Forwarded messages maintain original properties
   - Support for both broadcast and direct encrypted communication

## Security Architecture

- **Authentication**: Challenge-response protocol
- **Message Integrity**: Digital signatures with timestamp verification
- **End-to-End Encryption**: Hybrid cryptosystem for secure transmission
- **Access Control**: Rule-based approval system for information sharing

## Integration Points

- External WebSocket server for messaging backbone
- Multiple LLM provider APIs
- Vector database for semantic document storage
- SQLite for persistent data management

This system effectively combines decentralized communication, AI-powered knowledge processing, and secure data exchange to create a robust platform for distributed knowledge sharing with strong privacy guarantees.