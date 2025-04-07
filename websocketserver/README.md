# WebSocket Server

A secure WebSocket server implementation in Go, featuring authentication, real-time messaging, and rate limiting.

## Features

- Token-based authentication using JWT
- Public/private key cryptography for secure message signing
- Secure WebSocket connections (WSS)
- Persistent message storage with SQLite
- Direct and broadcast messaging
- Message delivery status tracking
- Cryptographic message verification
- Graceful server shutdown
- Rate limiting to prevent abuse

## Configuration

The server can be configured using environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| SERVER_ADDR | Server address and port | :8080 |
| MESSAGE_RATE_LIMIT | Maximum messages per second per user | 5.0 |
| MESSAGE_BURST_LIMIT | Maximum message burst size | 10 |

## Rate Limiting

The server implements a token bucket algorithm for rate limiting messages:

- Each user gets a separate rate limit bucket
- Default rate: 5 messages per second
- Default burst size: 10 messages
- When rate limit is exceeded, a system message is sent to the client
- Connection is maintained even when rate limited

## API Endpoints

### Authentication
- `POST /auth/register` - Register a new user
  ```json
  {
    "user_id": "user123",
    "username": "John Doe",
    "public_key": "base64_encoded_public_key"
  }
  ```

- `POST /auth/login` - Request an authentication challenge
  ```json
  {
    "user_id": "user123"
  }
  ```
  Response:
  ```json
  {
    "challenge": "random_challenge_string"
  }
  ```

- `POST /auth/login?verify=true` - Verify the challenge and get a JWT token
  ```json
  {
    "user_id": "user123",
    "signature": "base64_encoded_signature_of_challenge"
  }
  ```
  Response:
  ```json
  {
    "token": "jwt_token"
  }
  ```

- `GET /auth/users/{user_id}` - Get user public key
  Response:
  ```json
  {
    "user_id": "user123",
    "public_key": "base64_encoded_public_key"
  }
  ```

### WebSocket
- `GET /ws?token=<jwt_token>` - Connect to the WebSocket server

## Building and Running

```bash
# Build the server
go build -o websocketserver .

# Run the server
./websocketserver
```

## Message Format

Messages are JSON objects with the following structure:

```json
{
  "id": 123,                           // Automatically set by the server
  "from": "user_id",                   // Sender user ID
  "to": "recipient_id",                // Recipient user ID or "broadcast"
  "content": "message content",        // Message content
  "timestamp": "2023-04-05T12:34:56Z", // Automatically set by the server
  "status": "pending",                 // Message status (pending, delivered, etc.)
  "signature": "base64_signature"      // Cryptographic signature of the message
}
```

### Message Status Values
- `pending` - Message received by server but not yet delivered
- `delivered` - Message delivered to recipient
- `verified` - Message signature verified
- `invalid_signature` - Message signature verification failed
- `unverified` - Couldn't verify signature (missing public key)
- `unsigned` - No signature provided

## Security Features

- Challenge-response authentication using Ed25519 signatures
- Message signing and verification
- HTTPS/WSS encrypted connections
- JWT tokens for session management