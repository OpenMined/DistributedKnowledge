# Network Communication

Distributed Knowledge employs a WebSocket-based communication system to facilitate real-time interactions among network nodes. This document details the architecture and functionality of this communication layer.

## WebSocket Protocol

The system uses WebSockets to establish persistent, bidirectional connections between nodes:

- **Persistent Connections**: Unlike HTTP, WebSockets maintain an open connection
- **Low Latency**: Minimizes overhead for real-time communication
- **Bidirectional**: Both server and client can initiate messages
- **Text and Binary Support**: Handles both formats for different needs

## Message Types

Distributed Knowledge uses a structured message system with two primary types:


- **Direct Messages**: Encrypted and signed to a specific peer.
- **Broadcast Messages**: Plain-text, signed and delivered to everyone in the network.


## Message Routing

Messages can be delivered in two ways:

1. **Direct Messages**: Sent to a specific node identified by user ID

   ```go
   err = dkClient.SendMessage(dk_client.Message{
     From:      dkClient.UserID,
     To:        targetPeer,
     Content:   messageContent,
     Timestamp: time.Now(),
   })
   ```

2. **Broadcast Messages**: Sent to all nodes in the network

   ```go
   err = dkClient.BroadcastMessage(messageContent)
   ```

## Authentication System

All network communications are authenticated using:

1. **Public/Private Key Pairs**: Nodes identify themselves using Ed25519 key pairs
2. **Message Signing**: Outgoing messages are signed with the sender's private key
3. **Signature Verification**: Recipients verify authenticity using the sender's public key

This ensures that:

- Messages can't be forged
- Identities can't be impersonated
- Message content can't be altered in transit

## Rate Limiting

To prevent abuse, the communication system implements rate limiting:

- **Request Quotas**: Limits the number of messages within time periods
- **Dynamic Throttling**: Adjusts limits based on network conditions
- **Priority System**: Certain message types may have different limits

## Error Handling

The communication layer includes robust error handling:

- **Automatic Reconnection**: Attempts to re-establish dropped connections
- **Message Delivery Confirmation**: Acknowledgment system for critical messages
- **Failure Notification**: Informs senders when delivery fails

## Implementation Details

The network communication is implemented in the `dk/client/client.go` file and uses:

- **WebSocket Protocol**: For real-time bidirectional communication
- **JSON Encoding**: For message serialization
- **Hybrid Cryptography**: Peers use their assymetric keys to encrypt/decrypt messages with an exchanged symmetric key. 
- **Connection Pools**: For managing multiple concurrent connections

## Message Flow Example

1. User A formulates a query about quantum computing
2. The query is encoded as a message and signed with User A's private key
3. User A's client broadcasts the message to all active nodes
4. Each node receives the message and verifies the signature
5. Nodes with relevant knowledge process the query
6. Responding nodes generate answers and sign them
7. Answer messages are sent directly back to User A
8. User A verifies each response signature and processes the answers

## Security Considerations

- All WebSocket connections use TLS/SSL encryption
- Message signing prevents man-in-the-middle attacks
- Connection credentials are never shared or stored insecurely
- Node identities are cryptographically verified

## Best Practices

When implementing clients that connect to the Distributed Knowledge network:

- Always verify message signatures before processing content
- Implement exponential backoff for reconnection attempts
- Handle network partitions gracefully
- Monitor connection health and quality
- Cache important messages for potential redelivery
