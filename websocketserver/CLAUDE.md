# DistributedKnowledge WebSocketServer - Detailed Technical Overview

## System Architecture

The DistributedKnowledge WebSocketServer is a Go-based real-time communication platform with focus on secure authentication, message delivery, and analytics tracking. Here's a comprehensive breakdown:

### Core Components

1. **Main Server**
   - HTTP server (port 80) redirecting to HTTPS
   - HTTPS server (default port 443) with TLS
   - Graceful shutdown handling with signal monitoring

2. **Database Layer**
   - SQLite with Write-Ahead Logging (WAL) mode
   - Schema with tables for users, messages, sessions, analytics, and user profiles
   - Optimized for concurrent access and data integrity

3. **Authentication System**
   - Ed25519 public-key cryptography for security
   - Challenge-response protocol preventing replay attacks
   - JWT tokens for session management

4. **WebSocket Communication**
   - Real-time message delivery with persistence
   - Support for direct messages and broadcasts
   - Rate limiting to prevent abuse
   - Client session management

5. **User Management**
   - User registration with custom IDs and public keys
   - User status tracking (online/offline)
   - User profile descriptions

6. **Analytics**
   - Session tracking with timestamps
   - Message activity recording
   - Key metrics: daily/weekly active users, message counts, session durations

7. **Client Integration**
   - Cross-platform client binaries (Linux, Mac, Windows)
   - Installation script for easy setup

## Key APIs and Workflows

### Authentication Workflow

1. **Registration**
   - Endpoint: `/auth/register` (POST)
   - Payload: User ID, username, Ed25519 public key
   - Stores user credentials in SQLite database

2. **Login (Challenge-based)**
   - Step 1: Client requests challenge
     - Endpoint: `/auth/login` (POST)
     - Server generates random 32-byte challenge
   - Step 2: Client signs challenge with private key
     - Endpoint: `/auth/login?verify=true` (POST)
     - Server verifies signature using stored public key
     - Issues JWT token valid for 24 hours

3. **User Info**
   - Endpoint: `/auth/users/{user_id}` (GET)
   - Returns user ID, username, and public key
   - Endpoint: `/auth/check-userid/{user_id}` (GET)
   - Checks if user ID exists

### WebSocket Communication

1. **Connection Establishment**
   - Endpoint: `/ws?token=JWT_TOKEN`
   - Performs JWT validation
   - Creates client session with read/write goroutines
   - Returns pending messages during connection

2. **Message Delivery**
   - Direct messages sent to specific users
   - Message persistence for offline users
   - Broadcast messages to all connected clients
   - Message signatures for verification

3. **User Status**
   - Endpoint: `/active-users` (GET)
   - Returns lists of online and offline users
   - Real-time connection status

4. **Direct Message API**
   - Endpoint: `/direct-message/` (POST)
   - Allows HTTP applications to send messages to WebSocket clients
   - JWT authentication required
   - Messages marked with `IsForwardMessage` flag

### Security Features

1. **Authentication**
   - Ed25519 public key cryptography
   - Challenge-response protocol
   - JWT tokens with 24-hour expiration

2. **Rate Limiting**
   - Token bucket algorithm implementation
   - Configurable rate and burst parameters
   - Per-user rate limiting buckets

3. **Database Security**
   - SQL injection protection with parameterized queries
   - WAL mode for ACID compliance
   - Foreign key constraints for data integrity

4. **Transport Security**
   - HTTPS with TLS
   - HTTP-to-HTTPS redirection

## Data Models

### User
```go
type User struct {
    UserID    string
    Username  string
    PublicKey string    // Ed25519 public key
    CreatedAt time.Time
}
```

### Message
```go
type Message struct {
    ID               int
    From             string
    To               string    // "broadcast" for broadcast messages
    Timestamp        time.Time
    Content          string
    Status           string    // "pending", "delivered", "verified"
    IsBroadcast      bool
    Signature        string    // Optional base64-encoded signature
    IsForwardMessage bool      // True for messages from HTTP API
}
```

## Database Schema

1. **users** - User accounts with public keys
2. **messages** - User communication records
3. **broadcast_deliveries** - Tracking delivery status
4. **sessions** - User connection history
5. **message_events** - Message activity for analytics
6. **user_descriptions** - User profile information

## System Features

### Real-time Communication
- WebSocket-based messaging with persistent connections
- Efficient broadcast delivery with delivery tracking
- Non-blocking channel operations for performance
- Message persistence for offline recipients

### Analytics System
- Session duration tracking
- Message volume analytics
- User engagement metrics (daily/weekly active users)
- Persistent metrics storage

### Client Distribution
- Binary distribution for multiple platforms
- Installation script with automatic setup
- Download page with platform detection

## Technical Implementation Details

### Thread Safety
- Mutex protection for shared resources
- Context-based goroutine lifecycle management
- Safe concurrent access to client maps

### Performance Optimization
- SQLite WAL mode for concurrent database access
- Non-blocking channel operations
- Rate limiting to prevent resource exhaustion
- Buffered send channels (256 message capacity)

### Error Handling
- Graceful degradation under failures
- Detailed error logging
- Context cancelation for proper cleanup
- Connection timeout handling

### Networking
- WebSocket ping/pong for connection health
- 60-second read deadlines
- 10-second write deadlines
- Controlled WebSocket upgrades

## Workflows

1. **Client Connection**
   - JWT token validation
   - Client registration in server map
   - Start read/write goroutines
   - Session recording for analytics
   - Retrieve pending messages

2. **Message Flow**
   - Rate limit checking
   - Message persistence to database
   - Real-time delivery attempt
   - Status tracking ("pending", "delivered")
   - Broadcast handling with recipient tracking

3. **Authentication Flow**
   - Random challenge generation
   - Public key signature verification
   - JWT token issuance with 24-hour validity
   - Server-side challenge storage with sync.Map

4. **Shutdown Process**
   - Signal capture (SIGINT, SIGTERM)
   - Context timeout for graceful termination
   - Connection cleanup
   - Proper database closure

## Configuration

The system supports configuration via environment variables:
- `SERVER_ADDR` - Server address (default ":443")
- `MESSAGE_RATE_LIMIT` - Rate limit for messages per second (default 5.0)
- `MESSAGE_BURST_LIMIT` - Maximum burst size for rate limiting (default 10)