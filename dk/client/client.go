package lib

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha512"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"filippo.io/edwards25519"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/nacl/box"
)

// Message represents the structure of messages exchanged with the server.
type Message struct {
	ID               int       `json:"id,omitempty"`
	From             string    `json:"from"`
	To               string    `json:"to"`
	Timestamp        time.Time `json:"timestamp,omitempty"`
	Content          string    `json:"content"`
	Status           string    `json:"status,omitempty"`
	Signature        string    `json:"signature,omitempty"`          // Base64-encoded signature of message content
	IsForwardMessage bool      `json:"is_forward_message,omitempty"` // Indicates if this is a forward message
}

// EncryptedMessage is the structure that will be marshaled into the Message.Content field
// for direct messages. It contains the envelope (asymmetrically encrypted symmetric key)
// and the symmetrically encrypted message content.
type EncryptedMessage struct {
	// Data to allow the receiver to recover the AES key.
	EphemeralPublicKey string `json:"ephemeral_public_key"`
	KeyNonce           string `json:"key_nonce"`
	EncryptedKey       string `json:"encrypted_key"`
	// Data for AES-GCM encryption of the message content.
	DataNonce        string `json:"data_nonce"`
	EncryptedContent string `json:"encrypted_content"`
}

// UserStatusResponse holds the list of online and offline usernames.
type UserStatusResponse struct {
	Online  []string `json:"online"`
	Offline []string `json:"offline"`
}

// Client represents the WebSocket client as before.
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

func (c *Client) Token() string {
	return c.jwtToken
}

func (c *Client) SetReconnectInterval(interval time.Duration) {
	c.reconnectInterval = interval
}

// GetUserDescriptions retrieves the list of descriptions for the specified userID.
// It makes an HTTP GET request to the /user/descriptions/<user_id> endpoint.
// Since no authentication is required for this endpoint, the request is sent without an Authorization header.
func (c *Client) GetUserDescriptions(userID string) ([]string, error) {
	// Construct the endpoint URL using the base server URL and the user ID.
	endpoint := fmt.Sprintf("%s/user/descriptions/%s", c.serverURL, userID)

	// Create a new HTTP GET request.
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w", err)
	}

	// Execute the GET request using the client's HTTP client.
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP GET request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check if the response status code is OK.
	if resp.StatusCode != http.StatusOK {
		// Read the response body to include any error message.
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user descriptions: %s (status code %d)", string(bodyBytes), resp.StatusCode)
	}

	// Decode the JSON array response into a slice of strings.
	var descriptions []string
	if err := json.NewDecoder(resp.Body).Decode(&descriptions); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Return the descriptions slice.
	return descriptions, nil
}

// SetUserDescriptions sends the provided list of descriptions to the server's
// "/user/descriptions" endpoint using a POST request. If the descriptions slice
// is empty, it returns an error. It expects that the client already has a valid
// JWT token stored in c.jwtToken.
func (c *Client) SetUserDescriptions(descriptions []string) error {
	if len(descriptions) == 0 {
		return fmt.Errorf("descriptions list cannot be empty")
	}

	// Marshal the slice of strings into JSON.
	payload, err := json.Marshal(descriptions)
	if err != nil {
		return fmt.Errorf("failed to marshal descriptions: %w", err)
	}

	// Construct the endpoint URL.
	endpoint := fmt.Sprintf("%s/user/descriptions", c.serverURL)

	// Create a new HTTP POST request with the JSON payload.
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set the required headers.
	req.Header.Set("Content-Type", "application/json")
	if c.jwtToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.jwtToken)
	} else {
		return fmt.Errorf("JWT token is not set; please login first")
	}

	// Execute the request using the client's HTTP client.
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check if the response status is OK.
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to set descriptions: %s (status code %d)", string(bodyBytes), resp.StatusCode)
	}

	return nil
}

// GetActiveUsers performs an HTTP GET request to the serverURL + "/active-users" endpoint,
// retrieves the active and inactive user lists, and returns a UserStatusResponse.
// It follows best practices for error handling and resource management.
func (c *Client) GetActiveUsers() (*UserStatusResponse, error) {
	// Build the endpoint URL.
	endpoint := fmt.Sprintf("%s/active-users", c.serverURL)

	// Create a new HTTP GET request.
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request for active users: %w", err)
	}

	// Include the Authorization header if JWT token is set.
	if c.jwtToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.jwtToken)
	}

	// Optionally, you could add a context with timeout here:
	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()
	// req = req.WithContext(ctx)

	// Execute the request using the client's HTTP client.
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request to %s failed: %w", endpoint, err)
	}
	defer resp.Body.Close()

	// Check for a successful response.
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Decode the JSON response into the UserStatusResponse struct.
	var userStatus UserStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&userStatus); err != nil {
		return nil, fmt.Errorf("failed to decode active users response: %w", err)
	}

	return &userStatus, nil
}

// signMessage generates a cryptographic signature of the message content.
// It now signs the (possibly encrypted) message.Content, so that recipients first verify
// the integrity/authenticity of the envelope before decryption.
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

// verifyMessageSignature verifies that a message was signed by the claimed sender.
// Returns true if signature is valid, false otherwise.
func (c *Client) verifyMessageSignature(msg Message, senderPubKey ed25519.PublicKey) bool {
	// Skip verification for messages without signatures
	if msg.Signature == "" {
		return false
	}

	// Use the provided timestamp for verification.
	timestampValue := msg.Timestamp.UnixNano()
	log.Printf("Using timestamp for signature verification: %d", timestampValue)

	// Create the same canonical representation as used for signing.
	canonicalMsg := fmt.Sprintf("%s|%s|%d|%s",
		msg.From,
		msg.To,
		timestampValue,
		msg.Content)

	// Decode signature.
	signature, err := base64.StdEncoding.DecodeString(msg.Signature)
	if err != nil {
		log.Printf("Failed to decode signature: %v", err)
		return false
	}

	// Verify signature using sender's public key.
	return ed25519.Verify(senderPubKey, []byte(canonicalMsg), signature)
}

// GetUserPublicKey fetches a user's public key for verification.
func (c *Client) GetUserPublicKey(userID string) (ed25519.PublicKey, error) {
	// Check cache first (read lock)
	c.pubKeyCacheMu.RLock()
	pubKey, found := c.pubKeyCache[userID]
	c.pubKeyCacheMu.RUnlock()

	if found {
		return pubKey, nil
	}

	// Not in cache, need to fetch from server.
	endpoint := fmt.Sprintf("%s/auth/users/%s", c.serverURL, userID)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	// Add authorization header.
	if c.jwtToken != "" {
		req.Header.Add("Authorization", "Bearer "+c.jwtToken)
	}

	// Send request.
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user public key: %s", string(body))
	}

	// Parse response.
	var userInfo struct {
		UserID    string `json:"user_id"`
		PublicKey string `json:"public_key"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	// Decode base64 public key.
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
		b, _ := io.ReadAll(resp.Body)
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
		b, _ := io.ReadAll(resp.Body)
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
		b, _ := io.ReadAll(resp2.Body)
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

			// Skip decryption/signature verification for system messages and forward messages.
			if msg.From == "system" || msg.IsForwardMessage {
				if msg.IsForwardMessage {
					log.Printf("Received forward message, skipping decryption/verification")
				}
				c.recvCh <- msg
				continue
			}

			// Verify the message signature if present.
			if msg.Signature != "" {
				// Get sender's public key.
				senderPubKey, err := c.GetUserPublicKey(msg.From)
				if err != nil {
					log.Printf("Failed to get public key for user %s: %v", msg.From, err)
					// We still deliver the message but add a warning about unverified signature.
					msg.Status = "unverified"
					c.recvCh <- msg
					continue
				}

				// Verify signature.
				if !c.verifyMessageSignature(msg, senderPubKey) {
					log.Printf("WARNING: Invalid signature for message from %s", msg.From)
					// We still deliver the message but mark it as having an invalid signature.
					msg.Status = "invalid_signature"
					c.recvCh <- msg
					continue
				}

				// Signature valid, add verified status.
				if msg.Status == "" || msg.Status == "pending" {
					msg.Status = "verified"
				}
			} else {
				// No signature present.
				if msg.Status == "" {
					msg.Status = "unsigned"
				}
			}

			// If the message is a direct message to this client, attempt decryption.
			if msg.To == c.UserID {
				plaintext, err := decryptDirectMessage(msg.Content, c.privateKey)
				if err != nil {
					log.Printf("Failed to decrypt message from %s: %v", msg.From, err)
					msg.Status = "decryption_failed"
				} else {
					msg.Content = plaintext
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

			// Skip encryption and signing for forward messages
			if !msg.IsForwardMessage {
				// For direct messages (non-broadcast), encrypt the message content.
				if msg.To != "broadcast" {
					recipientPub, err := c.GetUserPublicKey(msg.To)
					if err != nil {
						log.Printf("Failed to get recipient public key: %v", err)
						continue
					}
					encryptedContent, err := encryptDirectMessage(msg.Content, recipientPub, c.privateKey)
					if err != nil {
						log.Printf("Failed to encrypt message: %v", err)
						continue
					}
					msg.Content = encryptedContent
				}

				// Sign the message with our private key.
				if err := c.signMessage(&msg); err != nil {
					log.Printf("Failed to sign message: %v", err)
					continue
				}
			} else {
				// For forward messages, just log that we're skipping encryption and signing
				log.Printf("Skipping encryption and signing for forward message")
			}

			// Add timestamp if not present.
			if msg.Timestamp.IsZero() {
				msg.Timestamp = time.Now()
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
	// Ensure the message has the correct sender ID.
	msg.From = c.UserID

	// Add timestamp if not present.
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}

	// Enqueue the message (encryption will be done in writePump for direct messages).
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

// SendCh returns the send channel (used for testing spoofing attempts).
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

// ---------------------- Helper Functions for Hybrid Encryption ----------------------

// encryptDirectMessage applies a hybrid encryption to the plaintext direct message.
// It first encrypts the plaintext with a random AES-GCM key, then encrypts this symmetric key
// using NaCl's box with an ephemeral key pair and the recipient’s X25519 public key.
func encryptDirectMessage(plaintext string, recipientEdPub ed25519.PublicKey, senderEdPriv ed25519.PrivateKey) (string, error) {
	// Generate a random 256-bit symmetric key.
	symKey := make([]byte, 32)
	if _, err := rand.Read(symKey); err != nil {
		return "", fmt.Errorf("failed to generate symmetric key: %v", err)
	}

	// Encrypt the plaintext using AES-GCM.
	block, err := aes.NewCipher(symKey)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %v", err)
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create AES-GCM: %v", err)
	}
	dataNonce := make([]byte, aesgcm.NonceSize())
	if _, err := rand.Read(dataNonce); err != nil {
		return "", fmt.Errorf("failed to generate AES nonce: %v", err)
	}
	ciphertext := aesgcm.Seal(nil, dataNonce, []byte(plaintext), nil)

	// Convert recipient's Ed25519 public key to X25519 public key.
	recipientX25519, err := convertEd25519PublicKeyToX25519(recipientEdPub)
	if err != nil {
		return "", fmt.Errorf("failed to convert recipient public key: %v", err)
	}

	// Generate an ephemeral key pair for asymmetric encryption.
	ephemeralPub, ephemeralPriv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to generate ephemeral key pair: %v", err)
	}
	// Generate nonce for NaCl box.
	boxNonce := make([]byte, 24)
	if _, err := rand.Read(boxNonce); err != nil {
		return "", fmt.Errorf("failed to generate box nonce: %v", err)
	}
	// Encrypt the symmetric key using NaCl box.
	encryptedSymKey := box.Seal(nil, symKey, (*[24]byte)(boxNonce), &recipientX25519, ephemeralPriv)

	// Create the envelope.
	env := EncryptedMessage{
		EphemeralPublicKey: base64.StdEncoding.EncodeToString(ephemeralPub[:]),
		KeyNonce:           base64.StdEncoding.EncodeToString(boxNonce),
		EncryptedKey:       base64.StdEncoding.EncodeToString(encryptedSymKey),
		DataNonce:          base64.StdEncoding.EncodeToString(dataNonce),
		EncryptedContent:   base64.StdEncoding.EncodeToString(ciphertext),
	}
	jsonBytes, err := json.Marshal(env)
	if err != nil {
		return "", fmt.Errorf("failed to marshal encrypted envelope: %v", err)
	}
	return string(jsonBytes), nil
}

// decryptDirectMessage reverses the hybrid encryption.
// It extracts the envelope fields from the JSON in ciphertext, decrypts the symmetric AES key
// using our converted X25519 private key, and then uses AES-GCM to decrypt the bulk message.
func decryptDirectMessage(encryptedEnvelope string, receiverEdPriv ed25519.PrivateKey) (string, error) {
	var env EncryptedMessage
	if err := json.Unmarshal([]byte(encryptedEnvelope), &env); err != nil {
		return "", fmt.Errorf("failed to unmarshal encrypted envelope: %v", err)
	}

	// Decode the ephemeral public key.
	ephemeralPubBytes, err := base64.StdEncoding.DecodeString(env.EphemeralPublicKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode ephemeral public key: %v", err)
	}
	if len(ephemeralPubBytes) != 32 {
		return "", errors.New("ephemeral public key has invalid length")
	}
	var ephemeralPub [32]byte
	copy(ephemeralPub[:], ephemeralPubBytes)

	// Convert our Ed25519 private key to X25519.
	receiverXPriv, err := convertEd25519PrivateKeyToX25519(receiverEdPriv)
	if err != nil {
		return "", fmt.Errorf("failed to convert receiver private key: %v", err)
	}

	// Decode the nonce and the asymmetrically encrypted symmetric key.
	boxNonceBytes, err := base64.StdEncoding.DecodeString(env.KeyNonce)
	if err != nil {
		return "", fmt.Errorf("failed to decode box nonce: %v", err)
	}
	if len(boxNonceBytes) != 24 {
		return "", errors.New("box nonce has invalid length")
	}
	var boxNonce [24]byte
	copy(boxNonce[:], boxNonceBytes)

	encryptedSymKey, err := base64.StdEncoding.DecodeString(env.EncryptedKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode encrypted symmetric key: %v", err)
	}
	// Decrypt the symmetric key.
	symKey, ok := box.Open(nil, encryptedSymKey, &boxNonce, &ephemeralPub, &receiverXPriv)
	if !ok {
		return "", errors.New("failed to decrypt symmetric key")
	}

	// Now decode the AES nonce and the encrypted content.
	dataNonce, err := base64.StdEncoding.DecodeString(env.DataNonce)
	if err != nil {
		return "", fmt.Errorf("failed to decode data nonce: %v", err)
	}
	encryptedContent, err := base64.StdEncoding.DecodeString(env.EncryptedContent)
	if err != nil {
		return "", fmt.Errorf("failed to decode encrypted content: %v", err)
	}
	// Decrypt the bulk message using AES-GCM.
	block, err := aes.NewCipher(symKey)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %v", err)
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create AES-GCM: %v", err)
	}
	plaintext, err := aesgcm.Open(nil, dataNonce, encryptedContent, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt content: %v", err)
	}
	return string(plaintext), nil
}

// convertEd25519PublicKeyToX25519 converts an Ed25519 public key to the corresponding X25519 key.
// It uses the edwards25519 package to decode and then compute the Montgomery u-coordinate.
func convertEd25519PublicKeyToX25519(edPub ed25519.PublicKey) ([32]byte, error) {
	var x25519Pub [32]byte
	if len(edPub) != ed25519.PublicKeySize {
		return x25519Pub, errors.New("invalid ed25519 public key length")
	}
	var A edwards25519.Point
	_, err := A.SetBytes(edPub)
	if err != nil {
		return x25519Pub, fmt.Errorf("failed to decode ed25519 public key: %v", err)
	}
	// Get the Montgomery form as a slice.
	mont := A.BytesMontgomery()
	// if err != nil {
	// 	return x25519Pub, fmt.Errorf("failed to convert to Montgomery form: %v", err)
	// }
	if len(mont) < 32 {
		return x25519Pub, fmt.Errorf("montgomery representation has invalid size: %d", len(mont))
	}
	// Copy the first 32 bytes of the slice into the array.
	copy(x25519Pub[:], mont[:32])
	return x25519Pub, nil
}

func convertEd25519PrivateKeyToX25519(edPriv ed25519.PrivateKey) ([32]byte, error) {
	var x25519Priv [32]byte
	if len(edPriv) != ed25519.PrivateKeySize {
		return x25519Priv, errors.New("invalid ed25519 private key length")
	}
	// Get the seed (first 32 bytes).
	seed := edPriv[:32]
	// Hash the seed using SHA-512.
	h := sha512.Sum512(seed)
	// Use the first 32 bytes of the hash.
	x := h[:32]
	copy(x25519Priv[:], x)
	// Perform the clamping as required by X25519.
	x25519Priv[0] &= 248
	x25519Priv[31] &= 127
	x25519Priv[31] |= 64
	return x25519Priv, nil
}

// convertEd25519PrivateKeyToX25519 converts an Ed25519 private key to an X25519 private key.
// The conversion uses the first 32 bytes (the seed) and performs clamping as required.
// func convertEd25519PrivateKeyToX25519(edPriv ed25519.PrivateKey) ([32]byte, error) {
// 	var x25519Priv [32]byte
// 	if len(edPriv) != ed25519.PrivateKeySize {
// 		return x25519Priv, errors.New("invalid ed25519 private key length")
// 	}
// 	// The seed is stored in the first 32 bytes.
// 	seed := edPriv[:32]
// 	copy(x25519Priv[:], seed)
// 	// Clamp as required by X25519.
// 	x25519Priv[0] &= 248
// 	x25519Priv[31] &= 127
// 	x25519Priv[31] |= 64
// 	return x25519Priv, nil
// }
