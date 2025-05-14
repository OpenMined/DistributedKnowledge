package core

import (
	"context"
	"dk/utils"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/philippgille/chromem-go"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

func SetupChromemCollection(vectorPath string) *chromem.Collection {
	// Setup chromem-go
	db, err := chromem.NewPersistentDB(vectorPath, false)
	if err != nil {
		panic(err)
	}

	embeddingModel := "nomic-embed-text"

	// Create collection if it wasn't loaded from persistent storage yet.
	// You can pass nil as embedding function to use the default (OpenAI text-embedding-3-small),
	// which is very good and cheap. It would require the OPENAI_API_KEY environment
	// variable to be set.
	// For this example we choose to use a locally running embedding model though.
	// It requires Ollama to serve its API at "http://localhost:11434/api".
	collection, err := db.GetOrCreateCollection("PersonalKnowledge", nil, chromem.NewEmbeddingFuncOllama(embeddingModel, ""))
	if err != nil {
		panic(err)
	}
	return collection
}

func RetrieveDocuments(ctx context.Context, question string, numResults int, metadataFilter map[string]string) ([]Document, error) {
	chromemCollection, err := utils.ChromemCollectionFromContext(ctx)
	if err != nil {
		log.Printf("[RAG] Failed to get Chromem collection from context: %v", err)
		return nil, err
	}

	// For the Ollama embedding model, a prefix is required to differentiate between a query and a document.
	// The documents were stored with "search_document: " as a prefix, so we use "search_query: " here.
	query := "search_query: " + question

	// Query the collection for the top 'numResults' similar documents.
	var docRes []chromem.Result

	// Create combined filter with always-active filter + any custom metadata filters
	filter := map[string]string{"active": "true"}

	// Add any custom metadata filters
	for key, value := range metadataFilter {
		filter[key] = value
	}

	// Get the total document count to avoid requesting more than available
	totalCount := chromemCollection.Count()
	log.Printf("[RAG] Query request: %s, numResults: %d, filters: %v", query, numResults, filter)
	log.Printf("[RAG] Total document count: %d", totalCount)

	// Use the smaller of numResults or totalCount to avoid "nResults must be <= number of documents" error
	queryLimit := numResults
	if totalCount < numResults {
		queryLimit = totalCount
	}
	log.Printf("[RAG] Adjusted query limit: %d", queryLimit)

	// Only query if we have documents
	if queryLimit > 0 {
		var err error
		log.Printf("[RAG] Executing query with limit: %d, filter: %v", queryLimit, filter)
		docRes, err = chromemCollection.Query(ctx, query, queryLimit, filter, nil)
		if err != nil {
			log.Printf("[RAG] Query error: %v", err)
			// If there's an error and it might be due to no documents matching the filter,
			// return an empty result instead of an error
			if strings.Contains(err.Error(), "nResults must be <= number of documents") {
				log.Printf("[RAG] No documents match the filter criteria, returning empty results")
				return []Document{}, nil
			}
			return nil, fmt.Errorf("error querying collection: %w", err)
		}
		log.Printf("[RAG] Query returned %d results", len(docRes))
	} else {
		log.Printf("[RAG] Query limit is 0, skipping query and returning empty results")
		return []Document{}, nil
	}

	var results []Document = []Document{}
	for _, res := range docRes {
		// Cut off the prefix we added before adding the document (see comment above).
		// This is specific to the "nomic-embed-text" model.
		contentString := strings.TrimPrefix(res.Content, "search_document: ")

		// Extract metadata from the document's metadata map
		metadata := make(map[string]string)
		for key, value := range res.Metadata {
			// Skip the "file" field as it's handled separately
			if key != "file" {
				metadata[key] = value
			}
		}

		content := Document{
			FileName: res.Metadata["file"],
			Content:  contentString,
			Metadata: metadata,
			Score:    res.Similarity,
		}
		results = append(results, content)
	}

	log.Printf("[RAG] Processed %d results", len(results))
	if len(results) > numResults {
		results = results[:numResults]
		log.Printf("[RAG] Trimmed results to %d items", len(results))
	}

	return results, nil
}

func RemoveDocument(ctx context.Context, filename string) error {
	chromemCollection, err := utils.ChromemCollectionFromContext(ctx)
	if err != nil {
		log.Printf("[RAG] %v", err)
		return nil
	}

	if strings.TrimSpace(filename) == "" {
		return errors.New("filename must be non‑empty")
	}

	where := map[string]string{"file": filename}

	if err := chromemCollection.Delete(ctx, where, nil); err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}
	return nil
}

func AddDocument(ctx context.Context, fileName string, fileContent string, UpdateDescriptions bool, metadata map[string]string) error {
	chromemCollection, err := utils.ChromemCollectionFromContext(ctx)
	if err != nil {
		log.Printf("[RAG] %v", err)
		return nil
	}
	content := "search_document: " + fileContent

	// Format current time in the required format
	currentTime := time.Now().Format("Jan 2, 2006, 03:04 PM")

	// Create metadata map with the required "file", "active", and "date" fields
	docMetadata := map[string]string{
		"file":   fileName,
		"active": "true",
		"date":   currentTime,
	}

	// Add additional metadata if provided
	for key, value := range metadata {
		docMetadata[key] = value
	}

	newDoc := chromem.Document{
		ID:       uuid.NewString(),
		Metadata: docMetadata,
		Content:  content,
	}

	err = chromemCollection.AddDocument(ctx, newDoc)
	if err != nil {
		return err
	}

	dkClient, err := utils.DkFromContext(ctx)
	if err != nil {
		panic(err)
	}

	if UpdateDescriptions {
		descriptions, err := utils.GetDescriptions(ctx)
		if err != nil {
			return err
		}

		llmProvider, err := LLMProviderFromContext(ctx)
		if err != nil {
			panic(err)
		}

		description, err := llmProvider.GenerateDescription(ctx, fileContent)
		if err != nil {
			panic(err)
		}
		descriptions = append(descriptions, description)

		dkClient.SetUserDescriptions(descriptions)
		utils.UpdateDescriptions(ctx, descriptions)
	}
	return nil
}

func FeedChromem(ctx context.Context, sourcePath string, update bool) {
	chromemCollection, err := utils.ChromemCollectionFromContext(ctx)
	if err != nil {
		log.Printf("[RAG] %v", err)
		return
	}

	// If the collection already has docs and update == false, bail early.
	if chromemCollection.Count() > 0 && !update {
		log.Println("[RAG] collection already populated – nothing to do")
		return
	}

	// Nothing to read? Fine – just return.
	fi, err := os.Stat(sourcePath)
	if err != nil || fi.Size() == 0 {
		log.Printf("[RAG] '%s' empty or missing – waiting for first upload", sourcePath)
		return
	}

	// Feed chromem with documents
	var docs []chromem.Document
	var descriptions []string
	if chromemCollection.Count() == 0 || update {
		// Here we use a DBpedia sample, where each line contains the lead section/introduction
		// to some Wikipedia article and its category.
		f, err := os.Open(sourcePath)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		d := json.NewDecoder(f)
		for i := 1; ; i++ {
			var article struct {
				Text     string `json:"text"`
				FileName string `json:"file"`
			}
			err := d.Decode(&article)
			if err == io.EOF {
				break
			} else if err != nil {

				panic(err)
			}

			llmProvider, err := LLMProviderFromContext(ctx)
			if err != nil {

				panic(err)
			}

			description, err := llmProvider.GenerateDescription(ctx, article.Text)
			if err != nil {
				panic(err)
			}
			descriptions = append(descriptions, description)

			// The embeddings model we use in this example ("nomic-embed-text")
			// fare better with a prefix to differentiate between document and query.
			// We'll have to cut it off later when we retrieve the documents.
			// An alternative is to create the embedding with `chromem.NewDocument()`,
			// and then change back the content before adding it do the collection
			// with `collection.AddDocument()`.
			content := "search_document: " + article.Text

			docs = append(docs, chromem.Document{
				ID: uuid.NewString(),
				Metadata: map[string]string{
					"file":        article.FileName,
					"description": description,
				},
				Content: content, //"search_document: " + article.Text,
			})
		}

		dkClient, err := utils.DkFromContext(ctx)
		if err != nil {
			panic(err)
		}

		dkClient.SetUserDescriptions(descriptions)
		utils.UpdateDescriptions(ctx, descriptions)

		log.Println("Adding documents to chromem-go, including creating their embeddings via Ollama API...")
		if len(docs) == 0 {
			log.Println("There's no content to generate the RAG. Skipping it for now")
			return
		}
		err = chromemCollection.AddDocuments(ctx, docs, runtime.NumCPU())
		if err != nil {
			// panic(err)
		}
	} else {
		log.Println("Not reading JSON lines because collection was loaded from persistent storage.")
	}
}

func GetDocument(ctx context.Context, filterName string, filterValue string, nElements int) (*Document, error) {
	if strings.TrimSpace(filterValue) == "" {
		return nil, errors.New("filterValue shouldn't be empty")
	}

	col, err := utils.ChromemCollectionFromContext(ctx)
	if err != nil {
		return nil, err
	}

	where := map[string]string{filterName: filterValue}

	// chromem-go requires a non‑empty queryText; a throw‑away literal is fine.
	const dummyQuery = "search_query: _"
	results, err := col.Query(ctx, dummyQuery, 1, where, nil)
	if len(results) == 0 {
		return nil, nil // caller turns this into 404
	}
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	content := strings.TrimPrefix(results[0].Content, "search_document: ")

	// Extract metadata from the document's metadata map
	metadata := make(map[string]string)
	for key, value := range results[0].Metadata {
		metadata[key] = value
		// if strings.HasPrefix(key, "metadata_") {
		// Strip the "metadata_" prefix and use the rest as the key
		// }
	}

	return &Document{
		FileName: results[0].Metadata["file"],
		Content:  content,
		Metadata: metadata,
		Score:    results[0].Similarity,
	}, nil
}

// GetDocuments returns all documents that match the given filter criteria
func GetDocuments(ctx context.Context, filterName string, filterValue string, nElements int) ([]Document, error) {
	if strings.TrimSpace(filterValue) == "" {
		return nil, errors.New("filterValue shouldn't be empty")
	}

	col, err := utils.ChromemCollectionFromContext(ctx)
	if err != nil {
		return nil, err
	}

	where := map[string]string{filterName: filterValue}

	// chromem-go requires a non‑empty queryText; a throw‑away literal is fine.
	const dummyQuery = "search_query: _"
	results, err := col.Query(ctx, dummyQuery, nElements, where, nil)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	if len(results) == 0 {
		return []Document{}, nil
	}

	documents := make([]Document, 0, len(results))
	for _, res := range results {
		content := strings.TrimPrefix(res.Content, "search_document: ")

		// Extract metadata from the document's metadata map
		metadata := make(map[string]string)
		for key, value := range res.Metadata {
			// Skip the "file" field as it's handled separately
			if key != "file" {
				metadata[key] = value
			}
		}

		documents = append(documents, Document{
			FileName: res.Metadata["file"],
			Content:  content,
			Metadata: metadata,
			Score:    res.Similarity,
		})
	}

	return documents, nil
}

// UpdateDocument overwrites (or creates) the document identified by fileName.
// It re‑uses the existing helpers to keep the behaviour (embeddings, description
// list, etc.) consistent in one place.
func UpdateDocument(ctx context.Context, fileName, newContent string, metadata map[string]string) error {
	// Remove first – we don't care if the old doc did not exist.
	if err := RemoveDocument(ctx, fileName); err != nil {
		return err
	}
	return AddDocument(ctx, fileName, newContent, false, metadata)
}

// AppendDocument appends new content to an existing document identified by fileName.
// If the document doesn't exist, it creates a new one with the provided content.
func AppendDocument(ctx context.Context, fileName, newContent string, metadata map[string]string) error {
	// Try to get the existing document
	existingDoc, err := GetDocument(ctx, "file", fileName, 1)
	if err != nil {
		return err
	}

	// If document exists, append the new content
	if existingDoc != nil {
		newContent = existingDoc.Content + "\n\n" + newContent
	}

	// Remove existing document and add with combined content
	if err := RemoveDocument(ctx, fileName); err != nil {
		return err
	}
	return AddDocument(ctx, fileName, newContent, false, metadata)
}

// ToggleActiveMetadata retrieves documents based on a filter and toggles the 'active' key
// in their metadata. If 'active' is present, it removes it; if not present, it adds it.
func ToggleActiveMetadata(ctx context.Context, filterField string, filterValue string) error {
	// Get collection to check document count
	chromemCollection, err := utils.ChromemCollectionFromContext(ctx)
	if err != nil {
		return fmt.Errorf("Failed to get the vector db collection: %w", err)
	}

	// Use collection count instead of a fixed value to avoid "nResults must be <= number of documents" error
	count := chromemCollection.Count()

	// Get all documents matching the filter criteria
	// Using a large number to get all potential matches
	documents, err := GetDocuments(ctx, filterField, filterValue, count)
	if err != nil {
		return fmt.Errorf("failed to get documents: %w", err)
	}

	if len(documents) == 0 {
		// No documents found matching the criteria
		return nil
	}

	for _, doc := range documents {
		// Create a new metadata map
		// newMetadata := make(map[string]string)
		// for key, value := range doc.Metadata {
		// 	newMetadata[key] = value
		// }

		val, exists := doc.Metadata["active"]

		// Check if the 'active' key exists in the metadata
		if val == "true" || !exists { // _, exists := doc.Metadata["active"]; exists {
			// Remove the 'active' key from metadata
			// newMetadata["active"] = "false"
			doc.Metadata["active"] = "false"
		} else {
			// Add 'active' key to metadata if it doesn't exist
			// newMetadata["active"] = "true"
			doc.Metadata["active"] = "true"
		}

		// Remove and re-add the document with updated metadata
		if err := RemoveDocument(ctx, doc.FileName); err != nil {
			return fmt.Errorf("failed to remove document %s: %w", doc.FileName, err)
		}

		if err := AddDocument(ctx, doc.FileName, doc.Content, false, doc.Metadata); err != nil {
			return fmt.Errorf("failed to re-add document %s: %w", doc.FileName, err)
		}
	}

	return nil
}

// DeleteAllDocuments removes all documents from the collection in stages:
// 1. First deletes documents with metadata "active" = "true"
// 2. Then deletes documents with metadata "active" = "false"
func DeleteAllDocuments(ctx context.Context) error {
	chromemCollection, err := utils.ChromemCollectionFromContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to get the vector db collection: %w", err)
	}

	// First delete documents with "active" = "true"
	filter := map[string]string{"active": "true"}
	if err := chromemCollection.Delete(ctx, filter, nil); err != nil {
		return fmt.Errorf("failed to delete documents with active=true: %w", err)
	}

	// Then delete documents with "active" = "false"
	filter = map[string]string{"active": "false"}
	if err := chromemCollection.Delete(ctx, filter, nil); err != nil {
		return fmt.Errorf("failed to delete documents with active=false: %w", err)
	}

	return nil
}

// CheckChromemHealth verifies that the Chromem database is working properly
// It attempts a basic query to validate the database connection and functionality
func CheckChromemHealth(ctx context.Context) error {
	log.Printf("[RAG] Running health check on Chromem database")

	chromemCollection, err := utils.ChromemCollectionFromContext(ctx)
	if err != nil {
		log.Printf("[RAG] Health check failed: Could not get Chromem collection from context: %v", err)
		return fmt.Errorf("failed to get Chromem collection: %w", err)
	}

	// Check collection count
	count := chromemCollection.Count()
	log.Printf("[RAG] Health check: database contains %d documents", count)

	if count > 0 {
		// Try retrieving at least one document to validate the database
		const dummyQuery = "search_query: _"

		// First try without any filter to ensure basic functionality
		log.Printf("[RAG] Health check: attempting basic query without filters")
		_, err = chromemCollection.Query(ctx, dummyQuery, 1, nil, nil)
		if err != nil {
			log.Printf("[RAG] Health check failed: Basic query test failed: %v", err)
			return fmt.Errorf("database basic query test failed: %w", err)
		}

		// Then try with active:true filter to test filter functionality
		log.Printf("[RAG] Health check: attempting query with active:true filter")
		filter := map[string]string{"active": "true"}
		results, err := chromemCollection.Query(ctx, dummyQuery, count, filter, nil)

		if err != nil {
			log.Printf("[RAG] Health check warning: Filter query test failed: %v", err)
			// Don't return error here, as some collections might not have the active field
		} else {
			log.Printf("[RAG] Health check: found %d documents with active:true", len(results))
		}
	}

	log.Printf("[RAG] Health check completed successfully")
	return nil
}

// EnsureDocumentMetadata scans all documents and ensures they have the required metadata fields
// This is useful for fixing documents that might be missing the 'active' flag or other required fields
func EnsureDocumentMetadata(ctx context.Context) (map[string]int, error) {
	log.Printf("[RAG] Starting document metadata validation and repair")

	chromemCollection, err := utils.ChromemCollectionFromContext(ctx)
	if err != nil {
		log.Printf("[RAG] Metadata repair failed: Could not get Chromem collection: %v", err)
		return nil, fmt.Errorf("failed to get Chromem collection: %w", err)
	}

	// Get the total document count
	count := chromemCollection.Count()
	log.Printf("[RAG] Processing %d documents for metadata validation", count)

	if count == 0 {
		log.Printf("[RAG] No documents to process")
		return map[string]int{"total": 0, "fixed": 0}, nil
	}

	// Get all documents using a dummy query with large limit
	const dummyQuery = "search_query: _"
	results, err := chromemCollection.Query(ctx, dummyQuery, count, nil, nil)
	if err != nil {
		log.Printf("[RAG] Failed to retrieve documents for metadata validation: %v", err)
		return nil, fmt.Errorf("failed to retrieve documents: %w", err)
	}

	log.Printf("[RAG] Retrieved %d documents for metadata validation", len(results))

	// Track statistics
	stats := map[string]int{
		"total":             len(results),
		"fixed":             0,
		"missing_active":    0,
		"missing_date":      0,
		"already_compliant": 0,
	}

	// Process each document
	for _, doc := range results {
		needsUpdate := false
		updatedMetadata := make(map[string]string)

		// Copy existing metadata
		for k, v := range doc.Metadata {
			updatedMetadata[k] = v
		}

		// Ensure 'active' field exists
		if _, exists := updatedMetadata["active"]; !exists {
			updatedMetadata["active"] = "true"
			needsUpdate = true
			stats["missing_active"]++
			log.Printf("[RAG] Document '%s' is missing 'active' field, will be set to 'true'",
				doc.Metadata["file"])
		}

		// Ensure 'date' field exists
		if _, exists := updatedMetadata["date"]; !exists {
			// Format current time in the required format
			currentTime := time.Now().Format("Jan 2, 2006, 03:04 PM")
			updatedMetadata["date"] = currentTime
			needsUpdate = true
			stats["missing_date"]++
			log.Printf("[RAG] Document '%s' is missing 'date' field, will be set to '%s'",
				doc.Metadata["file"], currentTime)
		}

		// Update document if needed
		if needsUpdate {
			// Get the document filename
			filename := doc.Metadata["file"]
			if filename == "" {
				log.Printf("[RAG] Warning: Document has no filename, skipping repair")
				continue
			}

			// Extract content (remove the search_document prefix)
			content := strings.TrimPrefix(doc.Content, "search_document: ")

			// Remove and re-add the document
			log.Printf("[RAG] Updating document '%s' with fixed metadata", filename)

			if err := RemoveDocument(ctx, filename); err != nil {
				log.Printf("[RAG] Error removing document '%s': %v", filename, err)
				continue
			}

			if err := AddDocument(ctx, filename, content, false, updatedMetadata); err != nil {
				log.Printf("[RAG] Error re-adding document '%s': %v", filename, err)
				continue
			}

			stats["fixed"]++
		} else {
			stats["already_compliant"]++
		}
	}

	log.Printf("[RAG] Metadata validation complete: %d documents processed, %d fixed",
		stats["total"], stats["fixed"])

	return stats, nil
}
