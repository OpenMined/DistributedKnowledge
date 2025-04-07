package lib

import (
	"bytes"
	"crypto/ed25519"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Message represents the structure of messages exchanged with the server.
type Message struct {
	ID        int       `json:"id,omitempty"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Timestamp time.Time `json:"timestamp,omitempty"`
	Content   string    `json:"content"`
	Status    string    `json:"status,omitempty"`
	Signature string    `json:"signature,omitempty"` // Base64-encoded signature of message content
}

// Client encapsulates the connection and authentication information for the WebSocket client.
type Client struct {
	UserID     string
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey

	serverURL string
	jwtToken  string

	// The WebSocket connection is protected by a read–write mutex.
	wsConn *websocket.Conn
	connMu sync.RWMutex

	recvCh chan Message // Channel for incoming messages.
	sendCh chan Message // Channel for outgoing messages.
	doneCh chan struct{}

	// Cache of user public keys for signature verification
	pubKeyCache   map[string]ed25519.PublicKey
	pubKeyCacheMu sync.RWMutex

	reconnectInterval time.Duration
	insecure          bool
}

// NewClient creates a new Client instance.
func NewClient(serverURL, userID string, privateKey ed25519.PrivateKey, publicKey ed25519.PublicKey) *Client {
	// Create client with public key cache
	client := &Client{
		serverURL:         serverURL,
		UserID:            userID,
		privateKey:        privateKey,
		publicKey:         publicKey,
		recvCh:            make(chan Message, 100),
		sendCh:            make(chan Message, 100),
		doneCh:            make(chan struct{}),
		pubKeyCache:       make(map[string]ed25519.PublicKey),
		reconnectInterval: 5 * time.Second,
	}

	// Add own public key to cache
	client.pubKeyCache[userID] = publicKey

	return client
}

// signMessage generates a cryptographic signature of the message content
// The signature covers all critical fields: From, To, Content, and Timestamp
func (c *Client) signMessage(msg *Message) error {
	// Ensure timestamp exists
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	// Create a canonical representation of the message for signing
	// Format: from|to|timestamp|content
	canonicalMsg := fmt.Sprintf("%s|%s|%d|%s",
		msg.From,
		msg.To,
		msg.Timestamp.UnixNano(),
		msg.Content)

	// Sign the canonical message
	signature := ed25519.Sign(c.privateKey, []byte(canonicalMsg))

	// Store base64-encoded signature
	msg.Signature = base64.StdEncoding.EncodeToString(signature)

	return nil
}

// verifyMessageSignature verifies that a message was signed by the claimed sender
// Returns true if signature is valid, false otherwise
func (c *Client) verifyMessageSignature(msg Message, senderPubKey ed25519.PublicKey) bool {
	// Skip verification for messages without signatures
	if msg.Signature == "" {
		return false
	}

	// Get timestamp to use for verification
	timestampValue := msg.Timestamp.UnixNano()
	log.Printf("Using timestamp for signature verification: %d", timestampValue)

	// Create the same canonical representation as used for signing
	canonicalMsg := fmt.Sprintf("%s|%s|%d|%s",
		msg.From,
		msg.To,
		timestampValue,
		msg.Content)

	// Decode signature
	signature, err := base64.StdEncoding.DecodeString(msg.Signature)
	if err != nil {
		log.Printf("Failed to decode signature: %v", err)
		return false
	}

	// Verify signature using sender's public key
	return ed25519.Verify(senderPubKey, []byte(canonicalMsg), signature)
}

// GetUserPublicKey fetches a user's public key for verification
// It will first check the local cache, and if not found, it will query the server
func (c *Client) GetUserPublicKey(userID string) (ed25519.PublicKey, error) {
	// Check cache first (read lock)
	c.pubKeyCacheMu.RLock()
	pubKey, found := c.pubKeyCache[userID]
	c.pubKeyCacheMu.RUnlock()

	if found {
		return pubKey, nil
	}

	// Not in cache, need to fetch from server
	endpoint := fmt.Sprintf("%s/auth/users/%s", c.serverURL, userID)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	// Add authorization header
	if c.jwtToken != "" {
		req.Header.Add("Authorization", "Bearer "+c.jwtToken)
	}

	// Send request
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user public key: %s", string(body))
	}

	// Parse response
	var userInfo struct {
		UserID    string `json:"user_id"`
		PublicKey string `json:"public_key"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	// Decode base64 public key
	pubKeyBytes, err := base64.StdEncoding.DecodeString(userInfo.PublicKey)
	if err != nil {
		return nil, err
	}

	// Cache the public key (write lock)
	c.pubKeyCacheMu.Lock()
	c.pubKeyCache[userID] = pubKeyBytes
	c.pubKeyCacheMu.Unlock()

	return pubKeyBytes, nil
}

// SetInsecure configures the client to skip TLS verification (for testing only).
func (c *Client) SetInsecure(insecure bool) {
	c.insecure = insecure
}
func (c *Client) SetReadLimit(limit int) {
	c.wsConn.SetReadLimit(int64(limit))
}

// httpClient returns a custom HTTP client.
func (c *Client) httpClient() *http.Client {
	if c.insecure {
		return &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	}
	return http.DefaultClient
}

// Register calls the /auth/register endpoint.
func (c *Client) Register(username string) error {
	endpoint := fmt.Sprintf("%s/auth/register", c.serverURL)
	payload := map[string]string{
		"user_id":    c.UserID,
		"username":   username,
		"public_key": base64.StdEncoding.EncodeToString(c.publicKey),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := c.httpClient().Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		b, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("registration failed: %s", string(b))
	}
	return nil
}

// Login performs challenge–response authentication using /auth/login.
func (c *Client) Login() error {
	// Step 1: Get challenge.
	loginURL := fmt.Sprintf("%s/auth/login", c.serverURL)
	payload := map[string]string{"user_id": c.UserID}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := c.httpClient().Post(loginURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("login challenge failed: %s", string(b))
	}

	var challengeResp map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&challengeResp); err != nil {
		return err
	}
	challenge, ok := challengeResp["challenge"]
	if !ok {
		return errors.New("challenge not found in response")
	}

	// Step 2: Sign challenge and verify.
	signature := ed25519.Sign(c.privateKey, []byte(challenge))
	sigB64 := base64.StdEncoding.EncodeToString(signature)
	verifyURL := fmt.Sprintf("%s/auth/login?verify=true", c.serverURL)
	payloadVerify := map[string]string{
		"user_id":   c.UserID,
		"signature": sigB64,
	}
	body2, err := json.Marshal(payloadVerify)
	if err != nil {
		return err
	}

	resp2, err := c.httpClient().Post(verifyURL, "application/json", bytes.NewReader(body2))
	if err != nil {
		return err
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		b, _ := ioutil.ReadAll(resp2.Body)
		return fmt.Errorf("login verification failed: %s", string(b))
	}

	var tokenResp map[string]string
	if err := json.NewDecoder(resp2.Body).Decode(&tokenResp); err != nil {
		return err
	}
	token, ok := tokenResp["token"]
	if !ok {
		return errors.New("token not found in response")
	}
	c.jwtToken = token
	return nil
}

// Connect opens a WebSocket connection and launches the read and write pumps.
func (c *Client) Connect() error {
	wsURL := fmt.Sprintf("%s/ws?token=%s", c.serverURL, c.jwtToken)
	parsedURL, err := url.Parse(wsURL)
	if err != nil {
		return err
	}
	// Convert HTTP(S) to WS(S) accordingly.
	switch parsedURL.Scheme {
	case "https":
		parsedURL.Scheme = "wss"
	case "http":
		parsedURL.Scheme = "ws"
	}
	dialer := websocket.DefaultDialer
	if parsedURL.Scheme == "wss" {
		dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: c.insecure}
	}

	conn, _, err := dialer.Dial(parsedURL.String(), nil)
	if err != nil {
		return err
	}

	c.connMu.Lock()
	c.wsConn = conn
	c.connMu.Unlock()

	// Set pong handler for keep–alive.
	c.wsConn.SetPongHandler(func(appData string) error {
		c.wsConn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Launch read and write pumps.
	go c.readPump()
	go c.writePump()
	return nil
}

// readPump continuously reads messages from the WebSocket.
func (c *Client) readPump() {
	defer close(c.recvCh)
	for {
		select {
		case <-c.doneCh:
			return
		default:
			c.connMu.RLock()
			conn := c.wsConn
			c.connMu.RUnlock()
			if conn == nil {
				return
			}
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			_, msgBytes, err := conn.ReadMessage()
			if err != nil {
				log.Printf("WebSocket read error: %v", err)
				go c.handleReconnect()
				return
			}
			var msg Message
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				log.Printf("Failed to unmarshal message: %v", err)
				continue
			}

			// Skip signature verification for system messages
			if msg.From == "system" {
				c.recvCh <- msg
				continue
			}

			// Verify the message signature if present
			if msg.Signature != "" {
				// Get sender's public key
				senderPubKey, err := c.GetUserPublicKey(msg.From)
				if err != nil {
					log.Printf("Failed to get public key for user %s: %v", msg.From, err)
					// We still deliver the message but add a warning about unverified signature
					msg.Status = "unverified"
					c.recvCh <- msg
					continue
				}

				// Verify signature
				if !c.verifyMessageSignature(msg, senderPubKey) {
					log.Printf("WARNING: Invalid signature for message from %s", msg.From)
					// We still deliver the message but mark it as having an invalid signature
					msg.Status = "invalid_signature"
					c.recvCh <- msg
					continue
				}

				// Signature valid, add verified status
				if msg.Status == "" || msg.Status == "pending" {
					msg.Status = "verified"
				}
			} else {
				// No signature present
				if msg.Status == "" {
					msg.Status = "unsigned"
				}
			}

			c.recvCh <- msg
		}
	}
}

// writePump handles outgoing messages and periodic pings.
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.connMu.RLock()
		if c.wsConn != nil {
			c.wsConn.Close()
		}
		c.connMu.RUnlock()
	}()
	for {
		select {
		case msg, _ := <-c.sendCh:
			c.connMu.RLock()
			conn := c.wsConn
			c.connMu.RUnlock()
			if conn == nil {
				return
			}
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			msgBytes, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Failed to marshal message: %v", err)
				continue
			}
			if err := conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
				log.Printf("Write error: %v", err)
				go c.handleReconnect()
				return
			}
		case <-ticker.C:
			c.connMu.RLock()
			conn := c.wsConn
			c.connMu.RUnlock()
			if conn == nil {
				return
			}
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Ping error: %v", err)
				go c.handleReconnect()
				return
			}
		case <-c.doneCh:
			return
		}
	}
}

// SendMessage enqueues a message to be sent over the WebSocket.
func (c *Client) SendMessage(msg Message) error {
	// Ensure the message has the correct sender ID
	msg.From = c.UserID

	// Add timestamp if not present
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	// Sign the message with our private key
	if err := c.signMessage(&msg); err != nil {
		return fmt.Errorf("failed to sign message: %v", err)
	}

	// Enqueue the signed message
	select {
	case c.sendCh <- msg:
		return nil
	case <-time.After(10 * time.Second):
		return errors.New("send message timeout")
	}
}

// BroadcastMessage creates a broadcast message (with a proper timestamp) and enqueues it.
func (c *Client) BroadcastMessage(content string) error {
	msg := Message{
		From:      c.UserID,
		To:        "broadcast",
		Content:   content,
		Timestamp: time.Now(),
	}
	return c.SendMessage(msg)
}

// Messages returns the channel for received messages.
func (c *Client) Messages() <-chan Message {
	return c.recvCh
}

// SendCh returns the send channel (used for testing spoofing attempts)
func (c *Client) SendCh() chan<- Message {
	return c.sendCh
}

// Disconnect cleanly closes the WebSocket connection.
func (c *Client) Disconnect() error {
	select {
	case <-c.doneCh:
		// Already closed.
	default:
		close(c.doneCh)
	}
	c.connMu.Lock()
	defer c.connMu.Unlock()
	if c.wsConn != nil {
		closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Client disconnecting")
		if err := c.wsConn.WriteControl(websocket.CloseMessage, closeMsg, time.Now().Add(10*time.Second)); err != nil {
			log.Printf("Error sending close message: %v", err)
		}
		err := c.wsConn.Close()
		c.wsConn = nil
		return err
	}
	return nil
}

// handleReconnect attempts to re-establish the WebSocket connection using exponential backoff.
func (c *Client) handleReconnect() {
	c.connMu.Lock()
	if c.wsConn != nil {
		c.wsConn.Close()
		c.wsConn = nil
	}
	c.connMu.Unlock()

	interval := c.reconnectInterval
	for {
		log.Printf("Attempting to reconnect...")
		if err := c.Connect(); err == nil {
			log.Printf("Reconnected successfully")
			return
		}
		log.Printf("Reconnect failed; retrying in %v", interval)
		time.Sleep(interval)
		if interval < 60*time.Second {
			interval *= 2
		}
	}
}
