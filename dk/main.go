package main

import (
	"context"
	dk_client "dk/client"
	"dk/core"
	mcp_server "dk/mcp"
	"dk/utils"
	"flag"
	"github.com/mark3labs/mcp-go/server"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func loadParameters() utils.Parameters {
	params := utils.Parameters{}

	// These flags remain unchanged.
	params.PrivateKeyPath = flag.String("private", "path/to/private_key.pem", "Path to the private key file in PEM format")
	params.PublicKeyPath = flag.String("public", "path/to/public_key.pem", "Path to the public key file in PEM format")
	params.UserID = flag.String("userId", "defaultUser", "User ID for authentication")
	// Keep the rag_sources flag so that it isn’t nil.
	params.RagSourcesFile = flag.String("rag_sources", "/path/to/rag_sources.jsonl", "Path to the JSONL file containing source data")
	params.ServerURL = flag.String("server", "https://localhost:8080", "Address to the websocket server")

	// New flag for projectPath (base directory).
	projectPath := flag.String("project_path", "~/.config", "Base directory for project configuration")

	flag.Parse()

	// Generate the dependent file paths from the projectPath.
	basePath := *projectPath
	queriesFile := basePath + "/queries.json"
	answersFile := basePath + "/answers.json"
	automaticApprovalFile := basePath + "/automatic_approval.json"
	vectorDBPath := basePath + "/vector_db"
	descriptionSourceFile := basePath + "/description.json"
	modelConfigFile := basePath + "/model_config.json"

	// Set the values in the Parameters struct using the generated strings.
	params.QueriesFile = &queriesFile
	params.AnswersFile = &answersFile
	params.AutomaticApprovalFile = &automaticApprovalFile
	params.VectorDBPath = &vectorDBPath
	params.ModelConfigFile = &modelConfigFile
	params.DescriptionSourceFile = &descriptionSourceFile

	return params
}

func main() {
	params := loadParameters()
	rootCtx := context.Background()

	// Ensure the model configuration file exists.
	// ensureModelConfigFile(*params.ModelConfigFile)

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
			// Add LLM provider to MCP context if available.
			if llmProvider != nil {
				ctx = core.WithLLMProvider(ctx, llmProvider)
			}
			return ctx
		}),
	)

	rootCtx = utils.WithParams(rootCtx, params)
	go core.HandleRequests(rootCtx)

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
