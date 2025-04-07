# WebSocket Client

A secure WebSocket client implementation in Go with cryptographic message signing, automatic reconnection, and robust message handling.

## Features

- Ed25519 public/private key cryptography for security
- Automatic WebSocket reconnection with exponential backoff
- Message signing and verification
- Support for direct and broadcast messages
- JWT authentication support
- Secure connections with TLS

## Installation

```bash
go get github.com/yourusername/websocketclient
```

## Usage

### Basic Setup

```go
package main

import (
    "crypto/ed25519"
    "log"
    "time"
    "websocketclient/lib"
)

func main() {
    // Create or load keys
    publicKey, privateKey, _ := loadOrCreateKeys("private_key", "public_key")
    
    // Create new client
    client := lib.NewClient("https://your-server.com:8080", "your-user-id", privateKey, publicKey)
    
    // For testing with self-signed certificates
    client.SetInsecure(true)
    
    // Register with the server
    if err := client.Register("Your Display Name"); err != nil {
        log.Printf("Registration failed: %v", err)
    }
    
    // Login to get JWT token
    if err := client.Login(); err != nil {
        log.Fatalf("Login failed: %v", err)
    }
    
    // Connect to WebSocket server
    if err := client.Connect(); err != nil {
        log.Fatalf("WebSocket connection failed: %v", err)
    }
    
    // Set message size limit (optional)
    client.SetReadLimit(1024 * 1024) // 1MB
    
    // Start receiving messages
    go receiveMessages(client)
    
    // Send a direct message
    err := client.SendMessage(lib.Message{
        To:      "recipient-user-id",
        Content: "Hello, recipient!",
    })
    
    // Send a broadcast message
    err = client.BroadcastMessage("Hello, everyone!")
    
    // Graceful disconnect
    client.Disconnect()
}

func receiveMessages(client *lib.Client) {
    for msg := range client.Messages() {
        log.Printf("From: %s, Status: %s, Content: %s", 
            msg.From, msg.Status, msg.Content)
    }
}
```

### Helper Function for Loading Keys

```go
// loadOrCreateKeys loads or creates new Ed25519 keys
func loadOrCreateKeys(privateKeyPath, publicKeyPath string) (ed25519.PublicKey, ed25519.PrivateKey, error) {
    // Check if keys exist
    if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
        // Generate new keys
        publicKey, privateKey, err := ed25519.GenerateKey(nil)
        if err != nil {
            return nil, nil, err
        }
        
        // Save keys to files
        if err := os.WriteFile(privateKeyPath, []byte(hex.EncodeToString(privateKey)), 0600); err != nil {
            return nil, nil, err
        }
        if err := os.WriteFile(publicKeyPath, []byte(hex.EncodeToString(publicKey)), 0600); err != nil {
            return nil, nil, err
        }
        
        return publicKey, privateKey, nil
    }
    
    // Load existing keys
    privateKeyHex, err := os.ReadFile(privateKeyPath)
    if err != nil {
        return nil, nil, err
    }
    publicKeyHex, err := os.ReadFile(publicKeyPath)
    if err != nil {
        return nil, nil, err
    }
    
    privateKey, err := hex.DecodeString(string(privateKeyHex))
    if err != nil {
        return nil, nil, err
    }
    publicKey, err := hex.DecodeString(string(publicKeyHex))
    if err != nil {
        return nil, nil, err
    }
    
    return ed25519.PublicKey(publicKey), ed25519.PrivateKey(privateKey), nil
}
```

## Message Verification

The client automatically verifies signatures on received messages and adds a verification status:

- `verified` - Message signature successfully verified
- `invalid_signature` - Message signature verification failed
- `unverified` - Couldn't verify signature (missing public key)
- `unsigned` - No signature provided

## Advanced Usage

### Verifying Message Signatures

```go
// Manual signature verification
senderPubKey, err := client.GetUserPublicKey("sender-id")
isValid := client.VerifyMessageSignature(message, senderPubKey)
```

### Handling WebSocket Disconnections

The client automatically handles reconnections with exponential backoff:

```go
// Customize reconnection interval
client.reconnectInterval = 10 * time.Second
```

### Direct Access to Channels

For advanced use cases, you can access the channels directly:

```go
// Get the receive channel
recvCh := client.Messages()

// Get the send channel (for testing)
sendCh := client.SendCh()
```

## Command Line Usage

The package includes a sample command-line client:

```bash
go run main.go -userId=YourUserID -private=path/to/private -public=path/to/public
```