import * as crypto from 'crypto'
import * as nacl from 'tweetnacl'
import * as ed2curve from 'ed2curve'
import WebSocket from 'ws'
import { Message, EncryptedMessage, UserStatusResponse } from '../../shared/types'

// Client represents the WebSocket client as before.
export class Client {
  private userID: string
  private privateKey: Uint8Array
  private publicKey: Uint8Array

  private serverURL: string
  private jwtToken: string = ''

  // The WebSocket connection is protected by a mutex in Go, we'll simulate this behavior
  private wsConn: WebSocket | null = null
  private wsConnMutex: { locked: boolean } = { locked: false }

  private recvCh: {
    push: (msg: Message) => void
    queue: Message[]
    callbacks: ((msg: Message) => void)[]
  }
  private sendCh: { push: (msg: Message) => void; queue: Message[] }
  private doneCh: { closed: boolean; close: () => void; onClose: (callback: () => void) => void }

  // Cache of user public keys for signature verification
  private pubKeyCache: Map<string, Uint8Array>
  private pubKeyCacheMutex: { locked: boolean } = { locked: false }

  private reconnectInterval: number
  private insecure: boolean = false
  private isReconnecting: boolean = false

  // NewClient creates a new Client instance.
  constructor(serverURL: string, userID: string, privateKey: Uint8Array, publicKey: Uint8Array) {
    this.serverURL = serverURL
    this.userID = userID
    this.privateKey = privateKey
    this.publicKey = publicKey

    // Create channels equivalent
    this.recvCh = {
      push: (msg: Message) => {
        this.recvCh.queue.push(msg)
        this.recvCh.callbacks.forEach((cb) => cb(msg))
      },
      queue: [],
      callbacks: []
    }

    this.sendCh = {
      push: (msg: Message) => {
        this.sendCh.queue.push(msg)
      },
      queue: []
    }

    this.doneCh = {
      closed: false,
      close: () => {
        this.doneCh.closed = true
      },
      onClose: (callback: () => void) => {
        // This would add to a list of callbacks, but we're keeping it simple
      }
    }

    this.pubKeyCache = new Map<string, Uint8Array>()
    this.reconnectInterval = 5000 // 5 seconds in milliseconds

    // Add own public key to cache
    this.pubKeyCache.set(userID, publicKey)
  }

  getToken(): string {
    return this.jwtToken
  }

  setReconnectInterval(interval: number): void {
    this.reconnectInterval = interval
  }

  // GetUserDescriptions retrieves the list of descriptions for the specified userID.
  async getUserDescriptions(userID: string): Promise<string[]> {
    // Construct the endpoint URL using the base server URL and the user ID.
    const endpoint = `${this.serverURL}/user/descriptions/${userID}`

    try {
      // Create and execute the request
      const response = await fetch(endpoint, {
        method: 'GET',
        headers: this.insecure ? { Insecure: 'true' } : {}
      })

      if (!response.ok) {
        const bodyText = await response.text()
        throw new Error(
          `Failed to get user descriptions: ${bodyText} (status code ${response.status})`
        )
      }

      // Parse the JSON response
      const descriptions = await response.json()
      return descriptions as string[]
    } catch (error) {
      throw new Error(`HTTP GET request failed: ${error}`)
    }
  }

  // SetUserDescriptions sends the provided list of descriptions to the server
  async setUserDescriptions(descriptions: string[]): Promise<void> {
    if (descriptions.length === 0) {
      throw new Error('descriptions list cannot be empty')
    }

    // Construct the endpoint URL
    const endpoint = `${this.serverURL}/user/descriptions`

    try {
      // Create and execute the request
      const response = await fetch(endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: this.jwtToken ? `Bearer ${this.jwtToken}` : '',
          ...(this.insecure ? { Insecure: 'true' } : {})
        },
        body: JSON.stringify(descriptions)
      })

      if (!response.ok) {
        const bodyText = await response.text()
        throw new Error(`Failed to set descriptions: ${bodyText} (status code ${response.status})`)
      }
    } catch (error) {
      throw new Error(`HTTP request failed: ${error}`)
    }
  }

  // signMessage generates a cryptographic signature of the message content.
  private signMessage(msg: Message): void {
    // Ensure timestamp exists
    if (!msg.timestamp) {
      msg.timestamp = new Date()
    }

    // Create a canonical representation of the message for signing
    // Format: from|to|timestamp|content
    // Convert JavaScript timestamp (milliseconds) to nanoseconds like in Go
    const timestampNanos = msg.timestamp.getTime() * 1000000
    const canonicalMsg = `${msg.from}|${msg.to}|${timestampNanos}|${msg.content}`

    // Sign the canonical message
    const signature = nacl.sign.detached(new TextEncoder().encode(canonicalMsg), this.privateKey)

    // Store base64-encoded signature
    msg.signature = Buffer.from(signature).toString('base64')
  }

  // verifyMessageSignature verifies that a message was signed by the claimed sender.
  private verifyMessageSignature(msg: Message, senderPubKey: Uint8Array): boolean {
    // Skip verification for messages without signatures
    if (!msg.signature) {
      return false
    }

    // Ensure the timestamp is a Date object
    if (!(msg.timestamp instanceof Date)) {
      try {
        if (typeof msg.timestamp === 'string') {
          msg.timestamp = new Date(msg.timestamp)
        } else if (typeof msg.timestamp === 'object') {
          const timestampStr = String(msg.timestamp)
          msg.timestamp = new Date(timestampStr)
        } else {
          msg.timestamp = new Date()
        }
      } catch (err) {
        msg.timestamp = new Date()
      }
    }

    // Use the provided timestamp for verification.
    // Convert JavaScript timestamp (milliseconds) to nanoseconds like in Go
    const timestampValue = msg.timestamp.getTime() * 1000000

    // Create the same canonical representation as used for signing.
    const canonicalMsg = `${msg.from}|${msg.to}|${timestampValue}|${msg.content}`

    // Decode signature.
    let signature: Uint8Array
    try {
      signature = new Uint8Array(Buffer.from(msg.signature, 'base64'))
    } catch (err) {
      console.log(`Failed to decode signature: ${err}`)
      return false
    }

    // Verify signature using sender's public key.
    return nacl.sign.detached.verify(
      new TextEncoder().encode(canonicalMsg),
      signature,
      senderPubKey
    )
  }

  // GetUserPublicKey fetches a user's public key for verification.
  async getUserPublicKey(userID: string): Promise<Uint8Array> {
    // Check cache first
    if (this.pubKeyCache.has(userID)) {
      return this.pubKeyCache.get(userID)!
    }

    // Not in cache, need to fetch from server.
    const endpoint = `${this.serverURL}/auth/users/${userID}`

    try {
      const response = await fetch(endpoint, {
        method: 'GET',
        headers: {
          Authorization: this.jwtToken ? `Bearer ${this.jwtToken}` : '',
          ...(this.insecure ? { Insecure: 'true' } : {})
        }
      })

      if (!response.ok) {
        const body = await response.text()
        throw new Error(`Failed to get user public key: ${body}`)
      }

      // Parse response.
      const userInfo = await response.json()

      // Decode base64 public key.
      const pubKeyBytes = Buffer.from(userInfo.public_key, 'base64')

      // Cache the public key
      this.pubKeyCache.set(userID, new Uint8Array(pubKeyBytes))

      return new Uint8Array(pubKeyBytes)
    } catch (error) {
      throw new Error(`Failed to get public key: ${error}`)
    }
  }

  // SetInsecure configures the client to skip TLS verification (for testing only).
  setInsecure(insecure: boolean): void {
    this.insecure = insecure
  }

  setReadLimit(limit: number): void {
    // WebSocket in Node.js doesn't have a direct equivalent to SetReadLimit
    // We can implement a custom solution that validates message size on receipt
    if (this.wsConn) {
      // Add a check in the message handler
      const originalOnMessage = this.wsConn.onmessage
      this.wsConn.onmessage = (event) => {
        if (typeof event.data === 'string' && event.data.length > limit) {
          console.log(`Message exceeds size limit (${event.data.length} > ${limit})`)
          // Optionally close connection or handle oversized message
          return
        }
        // Process normal-sized messages
        if (originalOnMessage) {
          originalOnMessage(event)
        }
      }
    }
  }

  // Register calls the /auth/register endpoint.
  async register(username: string): Promise<void> {
    const endpoint = `${this.serverURL}/auth/register`
    const payload = {
      user_id: this.userID,
      username: username,
      public_key: Buffer.from(this.publicKey).toString('base64')
    }

    try {
      const response = await fetch(endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(this.insecure ? { Insecure: 'true' } : {})
        },
        body: JSON.stringify(payload)
      })

      if (response.status !== 201) {
        const responseText = await response.text()
        throw new Error(`Registration failed: ${responseText}`)
      }
    } catch (error) {
      throw new Error(`Registration request failed: ${error}`)
    }
  }

  // Login performs challenge–response authentication using /auth/login.
  async login(): Promise<void> {
    // 1) Request a challenge
    const challengeURL = `${this.serverURL}/auth/login`
    const challengePayload = { user_id: this.userID }
    let resp: Response
    try {
      resp = await fetch(challengeURL, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(this.insecure ? { Insecure: 'true' } : {})
        },
        body: JSON.stringify(challengePayload)
      })
    } catch (err) {
      throw new Error(`Login challenge request failed: ${err}`)
    }
    if (!resp.ok) {
      const bodyText = await resp.text()
      throw new Error(`Login challenge failed: ${bodyText} (status ${resp.status})`)
    }

    // 2) Parse out the Base64-encoded challenge string
    let json1: { challenge?: string }
    try {
      json1 = await resp.json()
    } catch (err) {
      throw new Error(`Failed to parse challenge response: ${err}`)
    }
    const challenge = json1.challenge
    if (!challenge) {
      throw new Error('Challenge not found in response')
    }

    // 3) Sign the challenge string (auth.go does ed25519.Verify over the UTF-8 bytes of that Base64 text)
    const encoder = new TextEncoder()
    const sigBytes = nacl.sign.detached(encoder.encode(challenge), this.privateKey)
    const sigB64 = Buffer.from(sigBytes).toString('base64')

    // 4) Send signature back for verification and receive JWT
    const verifyURL = `${this.serverURL}/auth/login?verify=true`
    const verifyPayload = { user_id: this.userID, signature: sigB64 }
    let resp2: Response
    try {
      resp2 = await fetch(verifyURL, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(this.insecure ? { Insecure: 'true' } : {})
        },
        body: JSON.stringify(verifyPayload)
      })
    } catch (err) {
      throw new Error(`Login verification request failed: ${err}`)
    }
    if (!resp2.ok) {
      const bodyText = await resp2.text()
      throw new Error(`Login verification failed: ${bodyText} (status ${resp2.status})`)
    }

    // 5) Extract token
    let json2: { token?: string }
    try {
      json2 = await resp2.json()
    } catch (err) {
      throw new Error(`Failed to parse verification response: ${err}`)
    }
    if (!json2.token) {
      throw new Error('Token not found in verification response')
    }

    // 6) Store JWT for subsequent calls
    this.jwtToken = json2.token
  }

  // Connect opens a WebSocket connection and launches the read and write pumps.
  async connect(): Promise<void> {
    const wsURL = `${this.serverURL}/ws?token=${this.jwtToken}`
    let parsedURL = new URL(wsURL)

    // Convert HTTP(S) to WS(S) accordingly.
    if (parsedURL.protocol === 'https:') {
      parsedURL.protocol = 'wss:'
    } else if (parsedURL.protocol === 'http:') {
      parsedURL.protocol = 'ws:'
    }

    try {
      // Create WebSocket connection
      this.wsConn = new WebSocket(parsedURL.toString(), {
        rejectUnauthorized: !this.insecure
      })

      // Set up event handlers
      this.wsConn.on('open', () => {
        console.log('WebSocket connection established')
        this.startReadPump()
        this.startWritePump()
      })

      this.wsConn.on('error', (error) => {
        console.log(`WebSocket error: ${error}`)
        this.handleReconnect()
      })

      this.wsConn.on('close', () => {
        console.log('WebSocket connection closed')
        this.handleReconnect()
      })

      this.wsConn.on('pong', () => {
        // Reset read deadline in Go, but not directly applicable in Node.js
        console.log('Received pong from server')
      })
    } catch (error) {
      throw new Error(`Failed to connect to WebSocket: ${error}`)
    }
  }

  private startReadPump(): void {
    if (!this.wsConn) return

    this.wsConn.on('message', async (data: WebSocket.Data) => {
      try {
        const msgBytes = data.toString()
        const msg = JSON.parse(msgBytes) as Message

        // Make sure timestamp is a Date object
        if (msg.timestamp && !(msg.timestamp instanceof Date)) {
          try {
            // If it's a string or number, convert to Date
            if (typeof msg.timestamp === 'string') {
              msg.timestamp = new Date(msg.timestamp)
            } else if (typeof msg.timestamp === 'object') {
              // If it's already a JSON object but not a Date instance,
              // create a new Date object from timestamp string
              // Using a safe approach to handle various formats
              const timestampStr = String(msg.timestamp)
              msg.timestamp = new Date(timestampStr)
            }
          } catch (err) {
            console.log(`Failed to parse timestamp: ${err}, using current time instead`)
            msg.timestamp = new Date() // Fallback to current time
          }
        }

        // Skip decryption/signature verification for system messages.
        if (msg.from === 'system') {
          this.recvCh.push(msg)
          return
        }

        // Verify the message signature if present.
        if (msg.signature) {
          try {
            // Get sender's public key.
            const senderPubKey = await this.getUserPublicKey(msg.from)

            // Verify signature.
            if (!this.verifyMessageSignature(msg, senderPubKey)) {
              console.log(`WARNING: Invalid signature for message from ${msg.from}`)
              // We still deliver the message but mark it as having an invalid signature.
              msg.status = 'invalid_signature'
              this.recvCh.push(msg)
              return
            }

            // Signature valid, add verified status.
            if (!msg.status || msg.status === 'pending') {
              msg.status = 'verified'
            }
          } catch (error) {
            console.log(`Failed to get public key for user ${msg.from}: ${error}`)
            // We still deliver the message but add a warning about unverified signature.
            msg.status = 'unverified'
            this.recvCh.push(msg)
            return
          }
        } else {
          // No signature present.
          if (!msg.status) {
            msg.status = 'unsigned'
          }
        }

        // If the message is a direct message to this client, attempt decryption.
        if (msg.to === this.userID) {
          try {
            // Decrypt the message content
            const plaintext = await this.decryptDirectMessage(msg.content)

            // The decryptDirectMessage method now always returns a properly formatted MessageContent JSON string
            // so we can directly assign it to msg.content
            msg.content = plaintext
          } catch (error) {
            console.error(`Critical error in message decryption from ${msg.from}: ${error}`)
            msg.status = 'decryption_failed'

            // For stability, create a placeholder message content
            msg.content = JSON.stringify({
              text: `Message could not be decrypted: ${error instanceof Error ? error.message : String(error)}`,
              messageType: 'text'
            })
          }
        }

        this.recvCh.push(msg)
      } catch (error) {
        console.log(`Failed to process message: ${error}`)
      }
    })
  }

  private startWritePump(): void {
    // Setup ping interval (equivalent to ticker in Go)
    const pingInterval = setInterval(() => {
      if (this.doneCh.closed) {
        clearInterval(pingInterval)
        return
      }

      if (this.wsConn && this.wsConn.readyState === WebSocket.OPEN) {
        this.wsConn.ping()
      } else {
        clearInterval(pingInterval)
        this.handleReconnect()
      }
    }, 54000) // 54 seconds

    // Process sendCh queue
    const processSendQueue = setInterval(() => {
      if (this.doneCh.closed) {
        clearInterval(processSendQueue)
        return
      }

      if (
        this.sendCh.queue.length > 0 &&
        this.wsConn &&
        this.wsConn.readyState === WebSocket.OPEN
      ) {
        const msg = this.sendCh.queue.shift()
        if (msg) {
          this.processAndSendMessage(msg)
        }
      }
    }, 100) // Check every 100ms
  }

  private async processAndSendMessage(msg: Message): Promise<void> {
    try {
      // For direct messages (non-broadcast), encrypt the message content.
      if (msg.to !== 'broadcast') {
        const recipientPub = await this.getUserPublicKey(msg.to)
        const encryptedContent = await this.encryptDirectMessage(msg.content, recipientPub)
        msg.content = encryptedContent
      }

      // Add timestamp if not present.
      if (!msg.timestamp) {
        msg.timestamp = new Date()
      }

      // Sign the message with our private key.
      this.signMessage(msg)

      // Send the message
      const msgBytes = JSON.stringify(msg)
      this.wsConn!.send(msgBytes)
    } catch (error) {
      console.log(`Failed to send message: ${error}`)
      this.handleReconnect()
    }
  }

  // SendMessage enqueues a message to be sent over the WebSocket.
  sendMessage(msg: Message): void {
    // Ensure the message has the correct sender ID.
    msg.from = this.userID

    // Add timestamp if not present.
    if (!msg.timestamp) {
      msg.timestamp = new Date()
    }

    // Enqueue the message
    this.sendCh.push(msg)
  }

  // BroadcastMessage creates a broadcast message and enqueues it.
  broadcastMessage(content: string): void {
    const msg: Message = {
      from: this.userID,
      to: 'broadcast',
      content: content,
      timestamp: new Date()
    }
    this.sendMessage(msg)
  }

  // Messages returns the channel for received messages.
  onMessage(callback: (msg: Message) => void): void {
    this.recvCh.callbacks.push(callback)
  }

  // Disconnect cleanly closes the WebSocket connection.
  disconnect(): void {
    this.doneCh.close()
    this.isReconnecting = false // Ensure reconnection attempts stop

    if (this.wsConn) {
      this.wsConn.close(1000, 'Client disconnecting')
      this.wsConn = null
    }
  }

  // Get active and inactive users
  async getUserActiveStatus(): Promise<UserStatusResponse> {
    // Build the endpoint URL
    const endpoint = `${this.serverURL}/active-users`

    try {
      // Create AbortController for timeout
      const controller = new AbortController()
      const timeoutId = setTimeout(() => controller.abort(), 5000) // 5 second timeout

      // Create and execute the request
      const response = await fetch(endpoint, {
        method: 'GET',
        headers: {
          Authorization: this.jwtToken ? `Bearer ${this.jwtToken}` : '',
          ...(this.insecure ? { Insecure: 'true' } : {})
        },
        signal: controller.signal
      })

      // Clear the timeout
      clearTimeout(timeoutId)

      if (!response.ok) {
        const bodyText = await response.text()
        console.error(`Failed response body: ${bodyText}`)
        // Return default empty response instead of throwing
        return {
          active_users: [],
          inactive_users: []
        }
      }

      try {
        // Parse the JSON response directly
        const userStatus = await response.json()

        // Validate and sanitize the response
        if (!userStatus || typeof userStatus !== 'object') {
          console.error('Invalid response format: not an object')
          return { active_users: [], inactive_users: [] }
        }

        // Use the proper field names from the server (online/offline)
        let activeUsers: string[] = []
        let inactiveUsers: string[] = []

        // Using the original format: { online: [], offline: [] }
        if (Array.isArray(userStatus.online)) {
          activeUsers = userStatus.online
        } else if (Array.isArray(userStatus.active_users)) {
          // Fallback
          activeUsers = userStatus.active_users
        }

        if (Array.isArray(userStatus.offline)) {
          inactiveUsers = userStatus.offline
        } else if (Array.isArray(userStatus.inactive_users)) {
          // Fallback
          inactiveUsers = userStatus.inactive_users
        }

        const sanitizedResponse: UserStatusResponse = {
          active_users: activeUsers,
          inactive_users: inactiveUsers
        }

        return sanitizedResponse
      } catch (parseError) {
        console.error('Failed to parse JSON response:', parseError)
        return {
          active_users: [],
          inactive_users: []
        }
      }
    } catch (error) {
      console.error('Error in getUserActiveStatus:', error)
      // Return empty response instead of throwing
      return {
        active_users: [],
        inactive_users: []
      }
    }
  }

  // handleReconnect attempts to re-establish the WebSocket connection using exponential backoff.
  private async handleReconnect(): Promise<void> {
    // Skip if already trying to reconnect
    if (this.isReconnecting) {
      return
    }

    this.isReconnecting = true

    if (this.wsConn) {
      this.wsConn.terminate()
      this.wsConn = null
    }

    let interval = this.reconnectInterval

    // Always wait for the initial backoff interval before first attempt
    console.log(`Connection lost; will attempt reconnect in ${interval}ms`)
    await new Promise((resolve) => setTimeout(resolve, interval))

    while (!this.doneCh.closed) {
      console.log('Attempting to reconnect...')
      try {
        await this.connect()
        console.log('Reconnected successfully')
        this.isReconnecting = false
        return
      } catch (error) {
        // Increase the interval for the next attempt
        if (interval < 60000) {
          // 60 seconds max
          interval *= 2
        }
        console.log(`Reconnect failed; retrying in ${interval}ms`)
        await new Promise((resolve) => setTimeout(resolve, interval))
      }
    }

    this.isReconnecting = false
  }

  // ---------------------- Helper Functions for Hybrid Encryption ----------------------

  // encryptDirectMessage applies a hybrid encryption to the plaintext direct message.
  private async encryptDirectMessage(
    plaintext: string,
    recipientEdPub: Uint8Array
  ): Promise<string> {
    try {
      // Generate a random 256-bit symmetric key
      const symKey = crypto.randomBytes(32)

      // Create AES-GCM cipher with 12-byte nonce (96 bits)
      const dataNonce = crypto.randomBytes(12)
      const cipher = crypto.createCipheriv('aes-256-gcm', symKey, dataNonce)

      // Encrypt the plaintext
      let ciphertext = cipher.update(plaintext, 'utf8')
      ciphertext = Buffer.concat([ciphertext, cipher.final()])
      const authTag = cipher.getAuthTag() // 16 bytes (128 bits) auth tag

      // Combine ciphertext and auth tag - ensuring same format as Go
      const encryptedData = Buffer.concat([ciphertext, authTag])

      // Convert recipient's Ed25519 public key to X25519
      const recipientX25519 = ed2curve.convertPublicKey(recipientEdPub)
      if (!recipientX25519) {
        throw new Error("Failed to convert recipient's Ed25519 public key to X25519")
      }

      // Generate ephemeral key pair for X25519
      const ephemeralKeyPair = nacl.box.keyPair()

      // Generate 24-byte nonce (192 bits) for box encryption
      const boxNonce = crypto.randomBytes(24)

      // Encrypt the symmetric key
      const encryptedSymKey = nacl.box(
        symKey,
        new Uint8Array(boxNonce),
        new Uint8Array(recipientX25519),
        ephemeralKeyPair.secretKey
      )

      if (!encryptedSymKey) {
        throw new Error('Failed to encrypt symmetric key')
      }

      // Create the encrypted envelope - matching the format in Go
      const env: EncryptedMessage = {
        ephemeral_public_key: Buffer.from(ephemeralKeyPair.publicKey).toString('base64'),
        key_nonce: Buffer.from(boxNonce).toString('base64'),
        encrypted_key: Buffer.from(encryptedSymKey).toString('base64'),
        data_nonce: Buffer.from(dataNonce).toString('base64'),
        encrypted_content: Buffer.from(encryptedData).toString('base64')
      }

      return JSON.stringify(env)
    } catch (error) {
      throw new Error(`Encryption error: ${error}`)
    }
  }

  // decryptDirectMessage reverses the hybrid encryption.
  private async decryptDirectMessage(encryptedEnvelope: string): Promise<string> {
    try {
      // Handle possible non-encrypted messages (for testing/dev purposes)
      try {
        // First check if the string is already a valid plain text message
        // This could happen if the message wasn't actually encrypted
        const possibleMessage = JSON.parse(encryptedEnvelope)
        if (possibleMessage && typeof possibleMessage === 'object' && 'text' in possibleMessage) {
          return encryptedEnvelope // Return as-is since it's already in the right format
        }
      } catch (e) {
        // Not a JSON object or not in the expected format, continue with normal decryption
      }

      // Parse the encryption envelope
      let env: EncryptedMessage
      try {
        env = JSON.parse(encryptedEnvelope) as EncryptedMessage
      } catch (parseError) {
        console.error(`Failed to parse encryption envelope: ${parseError}`)
        // If we can't parse it as JSON, return it as plain text
        return JSON.stringify({
          text: encryptedEnvelope,
          messageType: 'text'
        })
      }

      // Validate required fields for encryption envelope
      if (
        !env.ephemeral_public_key ||
        !env.key_nonce ||
        !env.encrypted_key ||
        !env.data_nonce ||
        !env.encrypted_content
      ) {
        console.log('Invalid encryption envelope, missing required fields')
        // Format as a properly structured message
        return JSON.stringify({
          text: encryptedEnvelope,
          messageType: 'text'
        })
      }

      // Decode the ephemeral public key
      let ephemeralPubBytes: Buffer
      try {
        ephemeralPubBytes = Buffer.from(env.ephemeral_public_key, 'base64')
        if (ephemeralPubBytes.length !== 32) {
          throw new Error(`ephemeral public key has invalid length: ${ephemeralPubBytes.length}`)
        }
      } catch (error) {
        console.error(`Failed to decode ephemeral public key: ${error}`)
        return JSON.stringify({
          text: 'Failed to decode message (invalid ephemeral key)',
          messageType: 'text'
        })
      }

      // Convert our Ed25519 private key to X25519
      const receiverXPriv = ed2curve.convertSecretKey(this.privateKey)
      if (!receiverXPriv) {
        console.error("Failed to convert receiver's Ed25519 private key to X25519")
        return JSON.stringify({
          text: 'Failed to decrypt message (key conversion error)',
          messageType: 'text'
        })
      }

      // Decode the nonce and encrypted symmetric key
      let boxNonceBytes: Buffer
      try {
        boxNonceBytes = Buffer.from(env.key_nonce, 'base64')
        if (boxNonceBytes.length !== 24) {
          throw new Error(`box nonce has invalid length: ${boxNonceBytes.length}`)
        }
      } catch (error) {
        console.error(`Failed to decode box nonce: ${error}`)
        return JSON.stringify({
          text: 'Failed to decode message (invalid nonce)',
          messageType: 'text'
        })
      }

      let encryptedSymKey: Buffer
      try {
        encryptedSymKey = Buffer.from(env.encrypted_key, 'base64')
      } catch (error) {
        console.error(`Failed to decode encrypted key: ${error}`)
        return JSON.stringify({
          text: 'Failed to decode message (invalid encrypted key)',
          messageType: 'text'
        })
      }

      // Decrypt the symmetric key
      const symKey = nacl.box.open(
        new Uint8Array(encryptedSymKey),
        new Uint8Array(boxNonceBytes),
        new Uint8Array(ephemeralPubBytes),
        receiverXPriv
      )

      if (!symKey) {
        console.error('Failed to decrypt symmetric key')
        return JSON.stringify({
          text: 'Failed to decrypt message (symmetric key decryption failed)',
          messageType: 'text'
        })
      }

      // Decode the AES nonce and encrypted content
      let dataNonce: Buffer
      let encryptedContent: Buffer
      try {
        dataNonce = Buffer.from(env.data_nonce, 'base64')
        encryptedContent = Buffer.from(env.encrypted_content, 'base64')
      } catch (error) {
        console.error(`Failed to decode data nonce or encrypted content: ${error}`)
        return JSON.stringify({
          text: 'Failed to decode message components',
          messageType: 'text'
        })
      }

      // In Node.js crypto, we need to separate the auth tag
      // AES-GCM auth tag is 16 bytes (128 bits)
      if (encryptedContent.length < 16) {
        console.error('Encrypted content too short to contain auth tag')
        return JSON.stringify({
          text: 'Failed to decrypt message (invalid encrypted content)',
          messageType: 'text'
        })
      }

      const authTag = encryptedContent.slice(encryptedContent.length - 16)
      const ciphertext = encryptedContent.slice(0, encryptedContent.length - 16)

      // Decrypt with AES-GCM
      let plaintext: Buffer
      try {
        const decipher = crypto.createDecipheriv('aes-256-gcm', Buffer.from(symKey), dataNonce)
        decipher.setAuthTag(authTag)
        let decryptedChunk = decipher.update(ciphertext)
        plaintext = Buffer.concat([decryptedChunk, decipher.final()])
      } catch (error) {
        console.error(`AES decryption failed: ${error}`)
        return JSON.stringify({
          text: 'Failed to decrypt message content',
          messageType: 'text'
        })
      }

      const plaintextStr = plaintext.toString('utf8')

      // Try to parse the result as JSON to see if it's already formatted correctly
      try {
        const parsed = JSON.parse(plaintextStr)
        if (typeof parsed === 'object' && 'text' in parsed) {
          // It's already in the right format
          return plaintextStr
        } else {
          // It's JSON but not in our message format
          return JSON.stringify({
            text: plaintextStr,
            messageType: 'text'
          })
        }
      } catch (e) {
        // Not JSON, so wrap it in our message format
        return JSON.stringify({
          text: plaintextStr,
          messageType: 'text'
        })
      }
    } catch (error) {
      console.error(`Decryption error: ${error}`)
      // For stability, return a properly formatted error message
      return JSON.stringify({
        text: `Decryption failed: ${error instanceof Error ? error.message : String(error)}`,
        messageType: 'text'
      })
    }
  }
}

// Helper functions to generate keys
export function generateKeyPair(): { privateKey: Uint8Array; publicKey: Uint8Array } {
  const keyPair = nacl.sign.keyPair()
  return {
    privateKey: keyPair.secretKey,
    publicKey: keyPair.publicKey
  }
}

// Generate and format keys for the onboarding process
export async function generateKeys(): Promise<{ privateKey: string; publicKey: string }> {
  const keyPair = generateKeyPair()

  // Format keys in hexadecimal format
  return {
    privateKey: Buffer.from(keyPair.privateKey).toString('hex'),
    publicKey: Buffer.from(keyPair.publicKey).toString('hex')
  }
}
