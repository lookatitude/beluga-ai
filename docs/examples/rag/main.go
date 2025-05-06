// docs/examples/rag/main.go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/embedders/ollama"
	"github.com/lookatitude/beluga-ai/rag/loaders"
	"github.com/lookatitude/beluga-ai/rag/retrievers"
	"github.com/lookatitude/beluga-ai/rag/splitters"
	"github.com/lookatitude/beluga-ai/rag/vectorstores"
)

func main() {
	 ctx := context.Background()

	 // Load configuration (needed for Ollama embedder)
	 err := config.LoadConfig()
	 if err != nil {
	 	 log.Fatalf("Failed to load configuration: %v", err)
	 }

	 // 1. Load Documents
	 fmt.Println("--- Loading Documents ---")
	 loader := loaders.NewFileLoader("./sample.txt") // Assumes sample.txt is in the same dir
	 docs, err := loader.Load(ctx)
	 if err != nil {
	 	 log.Fatalf("Failed to load documents: %v", err)
	 }
	 fmt.Printf("Loaded %d document(s)\n", len(docs))
	 // fmt.Printf("Document content: %s\n", docs[0].PageContent)

	 // 2. Split Documents
	 fmt.Println("\n--- Splitting Documents ---")
	 splitter := splitters.NewCharacterSplitter(
	 	 splitters.WithChunkSize(50),
	 	 splitters.WithChunkOverlap(10),
	 )
	 splitDocs, err := splitter.SplitDocuments(ctx, docs)
	 if err != nil {
	 	 log.Fatalf("Failed to split documents: %v", err)
	 }
	 fmt.Printf("Split into %d chunks\n", len(splitDocs))
	 // for i, chunk := range splitDocs {
	 // 	 fmt.Printf(" Chunk %d: %s\n", i, chunk.PageContent)
	 // }

	 // 3. Create Embedder (using Ollama)
	 fmt.Println("\n--- Initializing Embedder ---")
	 embedder, err := ollama.NewOllamaEmbedder(
	 	 ollama.WithOllamaEmbedderBaseURL(config.Cfg.LLMs.Ollama.BaseURL),
	 	 // Use a model known for embeddings if available, otherwise default
	 	 // ollama.WithOllamaEmbedderModel("nomic-embed-text"), 
	 )
	 if err != nil {
	 	 log.Fatalf("Failed to create Ollama embedder: %v", err)
	 }
	 fmt.Println("Embedder initialized.")

	 // 4. Create Vector Store (In-Memory)
	 fmt.Println("\n--- Initializing Vector Store ---")
	 // Note: For persistent storage, use pgvector.NewPgVectorStore
	 store, err := vectorstores.NewInMemoryVectorStore(embedder)
	 if err != nil {
	 	 log.Fatalf("Failed to create in-memory vector store: %v", err)
	 }
	 fmt.Println("In-memory vector store initialized.")

	 // 5. Add Documents to Vector Store
	 fmt.Println("\n--- Adding Documents to Store ---")
	 _, err = store.AddDocuments(ctx, splitDocs)
	 if err != nil {
	 	 log.Fatalf("Failed to add documents to vector store: %v", err)
	 }
	 fmt.Println("Documents added to store.")

	 // 6. Create Retriever
	 fmt.Println("\n--- Initializing Retriever ---")
	 retriever := retrievers.NewVectorStoreRetriever(store, retrievers.WithTopK(2)) // Retrieve top 2 relevant chunks
	 fmt.Println("Retriever initialized.")

	 // 7. Retrieve Relevant Documents
	 fmt.Println("\n--- Retrieving Documents ---")
	 query := "What is RAG?"
	 fmt.Printf("Query: %s\n", query)
	 retrievedDocs, err := retriever.GetRelevantDocuments(ctx, query)
	 if err != nil {
	 	 log.Fatalf("Failed to retrieve documents: %v", err)
	 }

	 fmt.Println("Retrieved Documents:")
	 if len(retrievedDocs) == 0 {
	 	 fmt.Println(" No relevant documents found.")
	 } else {
	 	 for i, doc := range retrievedDocs {
	 	 	 fmt.Printf(" [%d] %s\n", i+1, doc.PageContent)
	 	 	 // fmt.Printf("   Metadata: %v\n", doc.Metadata)
	 	 }
	 }
}

