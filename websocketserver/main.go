package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"websocketserver/auth"
	"websocketserver/config"
	"websocketserver/db"
	"websocketserver/ws"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize SQLite database and set WAL mode.
	database, err := db.Initialize("app.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Run migrations to create required tables.
	if err := db.RunMigrations(database); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize authentication service.
	authService := auth.NewService(database)

	// Initialize WebSocket server with rate limiting
	wsServer := ws.NewServer(
		database,
		authService,
		cfg.MessageRateLimit,
		cfg.MessageBurstLimit,
	)

	// Setup HTTP routes.
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", wsServer.HandleWebSocket)
	mux.HandleFunc("/auth/register", authService.HandleRegistration)
	mux.HandleFunc("/auth/login", authService.HandleLogin)
	mux.HandleFunc("/auth/users/", authService.HandleGetUserInfo)

	// Setup HTTPS server (ensure you have valid TLS certificate files).
	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: mux,
	}

	// Start the server in a separate goroutine.
	go func() {
		log.Printf("Server starting on https://localhost%s", cfg.ServerAddr)
		if err := srv.ListenAndServeTLS("server.crt", "server.key"); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to listen and serve: %v", err)
		}
	}()

	// Wait for termination signal for graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server shutdown successfully")
}
