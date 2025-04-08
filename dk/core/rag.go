package core

import (
	"context"
	"dk/utils"
	"encoding/json"
	"github.com/philippgille/chromem-go"
	"io"
	"log"
	"os"
	"runtime"
	"strconv"
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

func FeedChromem(ctx context.Context, sourcePath string, update bool) {
	chromemCollection, err := utils.ChromemCollectionFromContext(ctx)
	if err != nil {
		panic(err)
	}

	// Feed chromem with documents
	var docs []chromem.Document
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

			// The embeddings model we use in this example ("nomic-embed-text")
			// fare better with a prefix to differentiate between document and query.
			// We'll have to cut it off later when we retrieve the documents.
			// An alternative is to create the embedding with `chromem.NewDocument()`,
			// and then change back the content before adding it do the collection
			// with `collection.AddDocument()`.
			content := "search_document: " + article.Text

			docs = append(docs, chromem.Document{
				ID:       strconv.Itoa(i),
				Metadata: map[string]string{"file": article.FileName},
				Content:  content,
			})
		}
		log.Println("Adding documents to chromem-go, including creating their embeddings via Ollama API...")
		if len(docs) == 0 {
			log.Println("There's no content to generate the RAG. Skipping it for now")
			return
		}
		err = chromemCollection.AddDocuments(ctx, docs, runtime.NumCPU())
		if err != nil {
			panic(err)
		}
	} else {
		log.Println("Not reading JSON lines because collection was loaded from persistent storage.")
	}
}
