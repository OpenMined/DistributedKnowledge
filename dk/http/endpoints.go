package http

import (
	"context"
	"dk/core"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

// RagRequest represents the JSON structure for POST /rag requests
type RagRequest struct {
	Filename    string   `json:"filename"`
	FileContent string   `json:"filecontent"`
	Metadata    []string `json:"metadata,omitempty"`
}

// RagResponse represents the structure for GET /rag responses
type RagResponse struct {
	Documents []core.Document `json:"documents"`
}

// ErrorResponse represents the structure for error responses
type ErrorResponse struct {
	Error string `json:"error"`
}

// SingleDocumentResponse is returned by GET /rag/{file_name}
type SingleDocumentResponse struct {
	Document core.Document `json:"document"`
}

// PatchRagRequest is used by PATCH /rag
type PatchRagRequest struct {
	Filename    string   `json:"filename"`
	FileContent string   `json:"filecontent"`
	Metadata    []string `json:"metadata,omitempty"`
}

// setupHTTPServer initializes and starts the HTTP server
func SetupHTTPServer(ctx context.Context, port string) {
	mux := http.NewServeMux()

	// GET /rag/{file_name} – fetch one document by exact file name
	mux.HandleFunc("GET /rag/{filename}", func(w http.ResponseWriter, r *http.Request) {
		fileName := r.PathValue("filename") // Go 1.22+ path parameter helper
		if fileName == "" {
			sendErrorResponse(w, "File name is required", http.StatusBadRequest)
			return
		}

		doc, err := core.GetDocument(ctx, fileName)
		if err != nil {
			sendErrorResponse(w, "Failed to retrieve document: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if doc == nil {

			sendErrorResponse(w, "Document not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SingleDocumentResponse{Document: *doc})
	})

	// PATCH /rag – replace a document’s content
	mux.HandleFunc("PATCH /rag", func(w http.ResponseWriter, r *http.Request) {
		var req PatchRagRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if req.Filename == "" || req.FileContent == "" {
			sendErrorResponse(w, "Filename and file content are required", http.StatusBadRequest)
			return
		}

		// Update (remove ‑ then add) the document.
		if err := core.UpdateDocument(ctx, req.Filename, req.FileContent, req.Metadata...); err != nil {
			sendErrorResponse(w, "Failed to update document: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "Document updated successfully"})
	})

	// POST /rag - Add document to vector database
	mux.HandleFunc("POST /rag", func(w http.ResponseWriter, r *http.Request) {
		var req RagRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Filename == "" || req.FileContent == "" {
			sendErrorResponse(w, "Filename and file content are required", http.StatusBadRequest)
			return
		}

		if err := core.AddDocument(ctx, req.Filename, req.FileContent, true, req.Metadata...); err != nil {
			sendErrorResponse(w, "Failed to add document: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "Document added successfully"})
	})

	// GET /rag - Retrieve documents based on query
	mux.HandleFunc("GET /rag", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		if query == "" {
			sendErrorResponse(w, "Query parameter is required", http.StatusBadRequest)
			return
		}

		numResultsStr := r.URL.Query().Get("num_results")
		numResults := 5 // default value
		if numResultsStr != "" {
			var err error
			numResults, err = strconv.Atoi(numResultsStr)
			if err != nil || numResults <= 0 {
				sendErrorResponse(w, "Invalid num_results parameter", http.StatusBadRequest)
				return
			}
		}

		docs, err := core.RetrieveDocuments(ctx, query, numResults)
		if err != nil {
			sendErrorResponse(w, "Failed to retrieve documents: "+err.Error(), http.StatusInternalServerError)
			return
		}

		response := RagResponse{Documents: docs}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	// DELETE /rag - Remove document from vector database
	mux.HandleFunc("DELETE /rag", func(w http.ResponseWriter, r *http.Request) {
		filename := r.URL.Query().Get("filename")
		if filename == "" {
			sendErrorResponse(w, "Filename parameter is required", http.StatusBadRequest)
			return
		}

		if err := core.RemoveDocument(ctx, filename); err != nil {
			sendErrorResponse(w, "Failed to remove document: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "Document removed successfully"})
	})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Start the server in a goroutine
	go func() {
		log.Printf("Starting HTTP server on port %s", port)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()
}

// sendErrorResponse is a helper function to send error responses
func sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}
