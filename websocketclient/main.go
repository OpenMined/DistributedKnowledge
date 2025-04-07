package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"flag"
	// "fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	wsclient "websocketclient/lib"
)

func loadOrCreateKeys(privateKeyPath, publicKeyPath string) (ed25519.PublicKey, ed25519.PrivateKey, error) {
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		publicKey, privateKey, err := ed25519.GenerateKey(nil)
		if err != nil {
			return nil, nil, err
		}
		if err := os.WriteFile(privateKeyPath, []byte(hex.EncodeToString(privateKey)), 0600); err != nil {
			return nil, nil, err
		}
		if err := os.WriteFile(publicKeyPath, []byte(hex.EncodeToString(publicKey)), 0600); err != nil {
			return nil, nil, err
		}
		return publicKey, privateKey, nil
	}

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

func main() {
	privateKeyPath := flag.String("private", "private_key", "Path to the private key file")
	publicKeyPath := flag.String("public", "public_key", "Path to the public key file")
	userID := flag.String("userId", "JohnDoe", "User ID")
	// targetId := flag.String("targetId", "Jana Doe", "Target User ID for direct messages")
	flag.Parse()

	publicKey, privateKey, err := loadOrCreateKeys(*privateKeyPath, *publicKeyPath)
	if err != nil {
		log.Fatalf("Failed to load or create keys: %v", err)
	}

	client := wsclient.NewClient("https://201.54.9.216:8080", *userID, privateKey, publicKey)
	client.SetInsecure(true)

	if err := client.Register(*userID); err != nil {
		log.Printf("Registration failed: %v", err)
	}

	if err := client.Login(); err != nil {
		log.Fatalf("Login failed: %v", err)
	}

	if err := client.Connect(); err != nil {
		log.Fatalf("WebSocket connection failed: %v", err)
	}

	// Launch a goroutine to log received messages.
	go func() {
		for msg := range client.Messages() {
			// Add signature verification status to the output
			var verificationStatus string
			switch msg.Status {
			case "verified":
				verificationStatus = "✓ VERIFIED"
			case "invalid_signature":
				verificationStatus = "❌ INVALID SIGNATURE"
			case "unverified":
				verificationStatus = "⚠️ UNVERIFIED (couldn't get sender's public key)"
			case "unsigned":
				verificationStatus = "⚠️ UNSIGNED (no signature provided)"
			default:
				verificationStatus = msg.Status
			}

			log.Printf("Received message from %s [%s]: %s", msg.From, verificationStatus, msg.Content)
		}
	}()

	// Send 10 direct messages and 10 broadcast messages.
	// for i := range [10]int{} {
	// 	log.Println("Sending direct message...")
	// 	err = client.SendMessage(wsclient.Message{
	// 		From:      *userID,
	// 		To:        *targetId,
	// 		Content:   fmt.Sprintf("%d - Hello, %s!", i, *targetId),
	// 		Timestamp: time.Now(),
	// 	})
	// 	if err != nil {
	// 		log.Printf("Send message error: %v", err)
	// 	}
	//
	// 	log.Println("Sending broadcast message...")
	// 	err = client.BroadcastMessage(fmt.Sprintf("%d - Hello, everyone!", i))
	// 	if err != nil {
	// 		log.Printf("Broadcast error: %v", err)
	// 	}
	//
	// 	time.Sleep(100 * time.Millisecond)
	// }

	// Wait for an interrupt signal to gracefully shut down.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	log.Println("Interrupt received, shutting down gracefully...")
	if err := client.Disconnect(); err != nil {
		log.Printf("Error during disconnect: %v", err)
	}
	time.Sleep(1 * time.Second)
	log.Println("Shutdown complete.")
}
