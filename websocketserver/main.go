package main

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"websocketserver/auth"
	"websocketserver/config"
	"websocketserver/db"
	"websocketserver/ws"
)

func downloadLinuxHandler(w http.ResponseWriter, r *http.Request) {
	// Specify the path to the binary file you wish to serve.
	filePath := "./install/binaries/linux_dk"
	fileName := "dk"

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		log.Printf("Error opening file %s: %v", filePath, err)
		return
	}
	defer file.Close()

	// Set the headers to indicate a file download.
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	w.Header().Set("Content-Type", "application/octet-stream")

	// Stream the file to the client.
	if _, err := io.Copy(w, file); err != nil {
		http.Error(w, "Error downloading file", http.StatusInternalServerError)
		log.Printf("Error copying file data: %v", err)
	}
}

func downloadMacHandler(w http.ResponseWriter, r *http.Request) {
	// Specify the path to the binary file you wish to serve.
	filePath := "./install/binaries/mac_dk"
	fileName := "dk"

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		log.Printf("Error opening file %s: %v", filePath, err)
		return
	}
	defer file.Close()

	// Set the headers to indicate a file download.
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	w.Header().Set("Content-Type", "application/octet-stream")

	// Stream the file to the client.
	if _, err := io.Copy(w, file); err != nil {
		http.Error(w, "Error downloading file", http.StatusInternalServerError)
		log.Printf("Error copying file data: %v", err)
	}
}

func provideInstallationScriptHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/x-shellscript")
	http.ServeFile(w, r, "./install/install.sh")
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println("Parsing Files")
	tmpl := template.Must(template.ParseFiles("templates/index.html"))

	log.Println("Parsed")
	tmpl.Execute(w, nil)

	log.Println("Executed")
}

func main() {
	// Load configuration. It is assumed that your configuration provides at least one secure address.
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

	// Initialize WebSocket server with rate limiting.
	wsServer := ws.NewServer(
		database,
		authService,
		cfg.MessageRateLimit,
		cfg.MessageBurstLimit,
	)

	// Setup HTTPS routes using the multiplexer.
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", wsServer.HandleWebSocket)
	mux.HandleFunc("/auth/register", authService.HandleRegistration)
	mux.HandleFunc("/auth/login", authService.HandleLogin)
	mux.HandleFunc("/", serveHome)
	mux.HandleFunc("/auth/users/", authService.HandleGetUserInfo)
	mux.HandleFunc("/download/linux", downloadLinuxHandler)
	mux.HandleFunc("/download/mac", downloadMacHandler)
	mux.HandleFunc("/install.sh", provideInstallationScriptHandler)

	// static files (e.g., JS libs like htmx)
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Create the HTTPS server instance.
	httpsSrv := &http.Server{
		Addr:    cfg.ServerAddr, // For example: ":443" (ensure this matches your configuration for HTTPS)
		Handler: mux,
	}

	// Create the HTTP server instance with a redirect handler.
	// This handler redirects all HTTP traffic to HTTPS.
	httpSrv := &http.Server{
		Addr: ":80", // Standard HTTP port
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Remove the default port if it's present in the host
			host := r.Host
			if strings.HasSuffix(host, ":80") {
				host = strings.TrimSuffix(host, ":80")
			}
			target := "https://" + host + r.URL.RequestURI()
			// Use StatusPermanentRedirect (308) to indicate that the resource has permanently moved.
			http.Redirect(w, r, target, http.StatusPermanentRedirect)
		}),
	}

	// Start the HTTPS server in a separate goroutine.
	go func() {
		log.Printf("Starting HTTPS server on %s", cfg.ServerAddr)
		if err := httpsSrv.ListenAndServeTLS("server.crt", "server.key"); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTPS server error: %v", err)
		}
	}()

	// Start the HTTP server in another goroutine.
	go func() {
		log.Printf("Starting HTTP server on %s (redirecting to HTTPS)", httpSrv.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for termination signal to gracefully shutdown both servers.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpsSrv.Shutdown(ctx); err != nil {
		log.Fatalf("HTTPS server forced to shutdown: %v", err)
	}
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Fatalf("HTTP server forced to shutdown: %v", err)
	}
	log.Println("Servers shut down successfully")
}
