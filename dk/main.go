package main

import (
	"context"
	dk_client "dk/client"
	"dk/core"
	"dk/db"
	"dk/http"
	mcp_server "dk/mcp"
	"dk/utils"
	"flag"
	"github.com/mark3labs/mcp-go/server"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func loadParameters() utils.Parameters {
	params := utils.Parameters{}

	// These flags remain unchanged.
	params.PrivateKeyPath = flag.String("private", "path/to/private_key.pem", "Path to the private key file in PEM format")
	params.PublicKeyPath = flag.String("public", "path/to/public_key.pem", "Path to the public key file in PEM format")
	params.UserID = flag.String("userId", "defaultUser", "User ID for authentication")

	// Keep the rag_sources flag so that it isn't nil.
	params.RagSourcesFile = flag.String("rag_sources", "/path/to/rag_sources.jsonl", "Path to the JSONL file containing source data")
	params.ServerURL = flag.String("server", "https://localhost:8080", "Address to the websocket server")
	params.HTTPPort = flag.String("http_port", "8081", "Port for the HTTP server")
	syftboxConfigPath := flag.String("syftbox_config", "~/.syftbox", "Path to syftbox config file")
	params.SyftboxConfig = syftboxConfigPath

	// New flag for projectPath (base directory).
	projectPath := flag.String("project_path", "~/.config", "Base directory for project configuration")

	flag.Parse()

	// Expand the home directory path if needed and generate dependent file paths
	basePath, err := utils.ExpandHomePath(*projectPath)
	if err != nil {
		log.Printf("Warning: Failed to expand home directory in path %s: %v", *projectPath, err)
		// Fall back to the original path if expansion fails
		basePath = *projectPath
	}

	// Create the base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		log.Printf("Warning: Failed to create base directory %s: %v", basePath, err)
	}

	// Expand SyftboxConfig path
	expandedSyftboxConfig, err := utils.ExpandHomePath(*syftboxConfigPath)
	if err != nil {
		log.Printf("Warning: Failed to expand home directory in SyftboxConfig path %s: %v", *syftboxConfigPath, err)
		// Fall back to the original path if expansion fails
	} else {
		params.SyftboxConfig = &expandedSyftboxConfig
	}

	vectorDBPath := filepath.Join(basePath, "vector_db")
	modelConfigFile := filepath.Join(basePath, "model_config.json")
	DBPath := filepath.Join(basePath, "app.db")

	// Set the values in the Parameters struct using the generated strings.
	params.VectorDBPath = &vectorDBPath
	params.ModelConfigFile = &modelConfigFile
	params.DBPath = &DBPath

	return params
}

func main() {
	params := loadParameters()
	rootCtx := context.Background()

	// Initialize the database connection
	database, err := db.Initialize(*params.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Create the database connection object
	dbConn := &db.DatabaseConnection{
		DB: database,
	}

	defer database.Close()

	if err := db.RunMigrations(database); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize API Management System migrations
	if err := db.RunAPIMigrations(database); err != nil {
		log.Printf("Warning: Failed to run API Management migrations: %v", err)
	}

	publicKey, privateKey, err := utils.LoadOrCreateKeys(*params.PrivateKeyPath, *params.PublicKeyPath)
	if err != nil {
		log.Fatalf("Failed to load or create keys: %v", err)
	}

	client := dk_client.NewClient(*params.ServerURL, *params.UserID, privateKey, publicKey)
	client.SetInsecure(true)
	if err := client.Register(*params.UserID); err != nil {
		log.Printf("Registration failed: %v", err)
	}

	if err := client.Login(); err != nil {
		log.Fatalf("Login failed: %v", err)
	}

	if err := client.Connect(); err != nil {
		log.Fatalf("WebSocket connection failed: %v", err)
	}

	log.Printf("Token:  %s\n", client.Token())

	// Load LLM model configuration and create provider.
	modelConfig, err := core.LoadModelConfig(*params.ModelConfigFile)
	if err != nil {
		log.Printf("Warning: Failed to load model config: %v", err)
	} else {
		llmProvider, err := core.CreateLLMProvider(modelConfig)
		if err != nil {
			log.Printf("Warning: Failed to create LLM provider: %v", err)
		} else {
			rootCtx = core.WithLLMProvider(rootCtx, llmProvider)
			log.Printf("LLM provider '%s' initialized successfully with model '%s'", modelConfig.Provider, modelConfig.Model)
		}
	}
	rootCtx = utils.WithDatabaseConnection(rootCtx, dbConn)

	rootCtx = utils.WithDK(rootCtx, client)
	client.SetReadLimit(1024 * 1024)
	chromemCollection := core.SetupChromemCollection(*params.VectorDBPath)
	rootCtx = utils.WithChromemCollection(rootCtx, chromemCollection)
	core.FeedChromem(rootCtx, *params.RagSourcesFile, false)

	mcpServer := mcp_server.NewMCPServer()

	// Store LLM provider for reuse in the MCP context.
	var llmProvider core.LLMProvider
	if p, err := core.LLMProviderFromContext(rootCtx); err == nil {
		llmProvider = p
	}

	go server.ServeStdio(
		mcpServer,
		server.WithStdioContextFunc(func(ctx context.Context) context.Context {
			ctx = utils.WithParams(ctx, params)
			ctx = utils.WithChromemCollection(ctx, chromemCollection)
			ctx = utils.WithDK(ctx, client)
			ctx = utils.WithDatabaseConnection(ctx, dbConn)
			// Add LLM provider to MCP context if available.
			if llmProvider != nil {
				ctx = core.WithLLMProvider(ctx, llmProvider)
			}
			return ctx
		}),
	)

	rootCtx = utils.WithParams(rootCtx, params)
	go core.HandleRequests(rootCtx)

	// Set up the HTTP server with the database connection for usage tracking
	http.SetupHTTPServer(rootCtx, *params.HTTPPort, dbConn)

	// Start policy worker to apply scheduled policy changes
	// Check every 5 minutes for pending changes
	utils.StartPolicyWorker(rootCtx, database, 5*time.Minute)

	// Start background job to refresh usage summaries
	// Run every 6 hours to calculate and update summaries
	go func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()

		// Run once immediately at startup
		if err := db.UpdateAPIUsageSummaries(database); err != nil {
			log.Printf("Error updating API usage summaries: %v", err)
		}

		for range ticker.C {
			if err := db.UpdateAPIUsageSummaries(database); err != nil {
				log.Printf("Error updating API usage summaries: %v", err)
			}
		}
	}()

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
