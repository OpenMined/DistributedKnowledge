package main

import (
	"context"
	dk_client "dk/client/lib"
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
	params.PrivateKeyPath = flag.String("private", "path/to/private_key.pem", "Path to the private key file in PEM format")
	params.PublicKeyPath = flag.String("public", "path/to/public_key.pem", "Path to the public key file in PEM format")
	params.UserID = flag.String("userId", "defaultUser", "User ID for authentication")
	params.QueriesFile = flag.String("queriesFile", "default_queries.json", "Path to the JSON file containing queries")
	params.AnswersFile = flag.String("answersFile", "default_answers.json", "Path to the JSON file containing answers")
	params.AutomaticApprovalFile = flag.String("automaticApproval", "default_automatic_approval.json", "Path to the JSON file for automatic approval settings")
	params.VectorDBPath = flag.String("vector_db", "/path/to/vector_db", "Path to the vector database file")
	params.RagSourcesFile = flag.String("rag_sources", "/path/to/rag_sources.jsonl", "Path to the JSONL file containing source data")
	params.ModelConfigFile = flag.String("modelConfig", "./config/model_config.json", "Path to the LLM provider configuration file")

	params.ServerURL = flag.String("server", "https://localhost:8080", "Address to the websocket server")
	flag.Parse()
	return params
}

func main() {
	params := loadParameters()
	rootCtx := context.Background()
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

	client.SetReadLimit(1024 * 1024)
	chromemCollection := core.SetupChromemCollection(*params.VectorDBPath)
	rootCtx = utils.WithChromemCollection(rootCtx, chromemCollection)
	core.FeedChromem(rootCtx, *params.RagSourcesFile, false)

	// Load LLM model configuration and create provider
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

	mcpServer := mcp_server.NewMCPServer()

	// Store LLM provider for reuse in the MCP context
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
			// Add LLM provider to MCP context if available
			if llmProvider != nil {
				ctx = core.WithLLMProvider(ctx, llmProvider)
			}
			return ctx
		}),
	)

	rootCtx = utils.WithDK(rootCtx, client)
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
