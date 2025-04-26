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

func RetrieveDocuments(ctx context.Context, question string, numResults int) ([]Document, error) {
	chromemCollection, err := utils.ChromemCollectionFromContext(ctx)
	if err != nil {
		return nil, err
	}

	log.Printf("Number of Documents: %d", chromemCollection.Count())
	// For the Ollama embedding model, a prefix is required to differentiate between a query and a document.
	// The documents were stored with "search_document: " as a prefix, so we use "search_query: " here.
	query := "search_query: " + question

	// Query the collection for the top 'numResults' similar documents.
	tmpNumResults := numResults
	var docRes []chromem.Result
	for tmpNumResults > 0 {
		// Query the collection for the top 'numResults' similar documents.
		docRes, _ = chromemCollection.Query(ctx, query, tmpNumResults, nil, nil)
		tmpNumResults = tmpNumResults - 1
	}

	var results []Document = []Document{}
	for _, res := range docRes {
		// Cut off the prefix we added before adding the document (see comment above).
		// This is specific to the "nomic-embed-text" model.
		contentString := strings.TrimPrefix(res.Content, "search_document: ")
		content := Document{FileName: res.Metadata["file"], Content: contentString}
		results = append(results, content)
	}

	if len(results) > numResults {
		results = results[:numResults]
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

func AddDocument(ctx context.Context, fileName string, fileContent string, UpdateDescriptions bool) error {
	chromemCollection, err := utils.ChromemCollectionFromContext(ctx)
	if err != nil {
		log.Printf("[RAG] %v", err)
		return nil
	}
	content := "search_document: " + fileContent
	newDoc := chromem.Document{
		ID: uuid.NewString(),
		Metadata: map[string]string{
			"file": fileName,
		},
		Content: content,
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

func GetDocument(ctx context.Context, fileName string) (*Document, error) {
	if strings.TrimSpace(fileName) == "" {
		return nil, errors.New("filename must be non‑empty")
	}

	col, err := utils.ChromemCollectionFromContext(ctx)
	if err != nil {
		return nil, err
	}

	where := map[string]string{"file": fileName}

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
	return &Document{FileName: results[0].Metadata["file"], Content: content}, nil
}

// UpdateDocument overwrites (or creates) the document identified by fileName.
// It re‑uses the existing helpers to keep the behaviour (embeddings, description
// list, etc.) consistent in one place.
func UpdateDocument(ctx context.Context, fileName, newContent string) error {
	// Remove first – we don't care if the old doc did not exist.
	if err := RemoveDocument(ctx, fileName); err != nil {
		return err
	}
	return AddDocument(ctx, fileName, newContent, false)
}

// AppendDocument appends new content to an existing document identified by fileName.
// If the document doesn't exist, it creates a new one with the provided content.
func AppendDocument(ctx context.Context, fileName, newContent string) error {
	// Try to get the existing document
	existingDoc, err := GetDocument(ctx, fileName)
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
	return AddDocument(ctx, fileName, newContent, false)
}
