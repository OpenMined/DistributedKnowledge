package main

import (
	"context"
	"database/sql"
	"encoding/json"
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
	"websocketserver/metrics"
	"websocketserver/ws"
)

// handleGetUserDescriptions returns an HTTP GET endpoint that returns the list of descriptions
// for a specified user. The user id is provided as part of the URL path like /user/descriptions/<user_id>.
// No authentication is required.
func handleGetUserDescriptions(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only allow GET requests.
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Expecting the URL to be: /user/descriptions/<user_id>
		// Split the URL path into components.
		// For example, if the URL is "/user/descriptions/someUserID",
		// parts[0] is empty, parts[1] is "user", parts[2] is "descriptions",
		// and parts[3] is the user id.
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 4 || parts[3] == "" {
			http.Error(w, "User ID not specified in URL", http.StatusBadRequest)
			return
		}
		userID := parts[3]

		// Query the database for the descriptions JSON string.
		var storedDescriptions string
		query := "SELECT descriptions FROM user_descriptions WHERE user_id = ?"
		if err := db.QueryRow(query, userID).Scan(&storedDescriptions); err != nil {
			if err == sql.ErrNoRows {
				// If no record exists for this user, return an empty JSON array.
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("[]"))
				return
			}
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Return the retrieved descriptions. Since storedDescriptions is already a JSON array,
		// we simply send it back.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(storedDescriptions))
	}
}

// handleUserDescriptions returns an HTTP handler that allows authenticated users to set
// their descriptions list by sending a JSON array of strings. This request replaces any previously stored list.
func handleUserDescriptions(authService *auth.Service, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Allow only POST requests.
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract and validate the Authorization header.
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, "Invalid Authorization header", http.StatusUnauthorized)
			return
		}
		tokenStr := parts[1]

		// Validate the token and get the user ID.
		claims, err := auth.ParseToken(tokenStr, authService)
		if err != nil {
			http.Error(w, "Invalid token: "+err.Error(), http.StatusUnauthorized)
			return
		}
		userID, ok := claims["user_id"].(string)
		if !ok || userID == "" {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		// Read and parse the JSON payload into a slice of strings.
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		var newDescriptions []string
		if err := json.Unmarshal(body, &newDescriptions); err != nil {
			http.Error(w, "Invalid JSON payload, expected an array of strings", http.StatusBadRequest)
			return
		}

		// Optionally, ensure the array is valid (for example, not nil)
		if len(newDescriptions) == 0 {
			http.Error(w, "Descriptions list cannot be empty", http.StatusBadRequest)
			return
		}

		// Marshal the new list to JSON for storage.
		updatedList, err := json.Marshal(newDescriptions)
		if err != nil {
			http.Error(w, "Error processing descriptions list", http.StatusInternalServerError)
			return
		}

		// Begin a transaction for atomic update.
		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		commit := false
		defer func() {
			if !commit {
				tx.Rollback()
			}
		}()

		// Check for an existing record for the user.
		var existing string
		query := "SELECT descriptions FROM user_descriptions WHERE user_id = ?"
		err = tx.QueryRow(query, userID).Scan(&existing)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		if err == sql.ErrNoRows {
			// No record exists; insert a new one.
			insertQuery := "INSERT INTO user_descriptions (user_id, descriptions) VALUES (?, ?)"
			if _, err := tx.Exec(insertQuery, userID, string(updatedList)); err != nil {
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
		} else {
			// Record exists; replace the list.
			updateQuery := "UPDATE user_descriptions SET descriptions = ? WHERE user_id = ?"
			if _, err = tx.Exec(updateQuery, string(updatedList), userID); err != nil {
				http.Error(w, "Database error updating descriptions", http.StatusInternalServerError)
				return
			}
		}

		// Commit the transaction.
		if err = tx.Commit(); err != nil {
			http.Error(w, "Database commit error", http.StatusInternalServerError)
			return
		}
		commit = true

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Descriptions list updated"))
	}
}

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

func downloadWindowsHandler(w http.ResponseWriter, r *http.Request) {
	// Specify the path to the binary file you wish to serve.
	filePath := "./install/binaries/windows_dk.exe"
	fileName := "DK Installer.exe"

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

	tmpl.Execute(w, nil)
}

func serveDownload(w http.ResponseWriter, r *http.Request) {
	log.Println("Parsing Files")
	tmpl := template.Must(template.ParseFiles("templates/download.html"))

	tmpl.Execute(w, nil)
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

	metrics.InitPersistence(database)

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
	mux.HandleFunc("/auth/check-userid/", authService.HandleCheckUserID)
	mux.HandleFunc("/active-users", wsServer.ActiveUsersHandler)
	mux.HandleFunc("/", serveHome)
	mux.HandleFunc("/user/descriptions", handleUserDescriptions(authService, database))
	mux.HandleFunc("/download", serveDownload)
	mux.HandleFunc("/auth/users/", authService.HandleGetUserInfo)
	mux.HandleFunc("/download/linux", downloadLinuxHandler)
	mux.HandleFunc("/download/mac", downloadMacHandler)
	mux.HandleFunc("/download/windows", downloadWindowsHandler)
	mux.HandleFunc("/install.sh", provideInstallationScriptHandler)
	mux.HandleFunc("/user/descriptions/", handleGetUserDescriptions(database))

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
