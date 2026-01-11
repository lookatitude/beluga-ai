package main

import (
	"context"
	"fmt"
	"log"

	_ "github.com/lookatitude/beluga-ai/pkg/embeddings/providers/mock"
	_ "github.com/lookatitude/beluga-ai/pkg/vectorstores/providers/inmemory"
	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
)

func main() {
	fmt.Println("💾 Beluga AI - In-Memory Vector Store Provider Example")
	fmt.Println("======================================================")

	ctx := context.Background()

	// Step 1: Create an embedder (required for text-based operations)
	fmt.Println("\n📋 Step 1: Creating embedder...")
	embedderConfig := embeddings.NewConfig()
	embedderConfig.Mock.Enabled = true
	embedderFactory, err := embeddings.NewEmbedderFactory(embedderConfig)
	if err != nil {
		log.Fatalf("Failed to create embedder factory: %v", err)
	}
	embedder, err := embedderFactory.NewEmbedder("mock")
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}
	fmt.Println("✅ Created mock embedder")

	// Step 2: Create in-memory vector store
	fmt.Println("\n📋 Step 2: Creating in-memory vector store...")
	store, err := vectorstores.NewInMemoryStore(ctx,
		vectorstores.WithEmbedder(embedder),
	)
	if err != nil {
		log.Fatalf("Failed to create vector store: %v", err)
	}
	fmt.Println("✅ Created in-memory vector store")

	// Step 3: Create and add documents
	fmt.Println("\n📋 Step 3: Adding documents...")
	documents := []schema.Document{
		schema.NewDocument("Machine learning is a subset of AI"),
		schema.NewDocument("Deep learning uses neural networks"),
		schema.NewDocument("Natural language processing enables text understanding"),
	}
	ids, err := store.AddDocuments(ctx, documents)
	if err != nil {
		log.Fatalf("Failed to add documents: %v", err)
	}
	fmt.Printf("✅ Added %d documents with IDs: %v\n", len(ids), ids)

	// Step 4: Search by query
	fmt.Println("\n📋 Step 4: Searching by query...")
	query := "What is artificial intelligence?"
	results, err := store.SimilaritySearchByQuery(ctx, query, 2)
	if err != nil {
		log.Fatalf("Failed to search: %v", err)
	}
	fmt.Printf("✅ Found %d results:\n", len(results))
	for i, result := range results {
		fmt.Printf("  %d. %s (score: %.4f)\n", i+1, result.GetContent(), result.GetScore())
	}

	// Step 5: Search by vector
	fmt.Println("\n📋 Step 5: Searching by vector...")
	queryEmbedding, err := embedder.EmbedQuery(ctx, query)
	if err != nil {
		log.Fatalf("Failed to embed query: %v", err)
	}
	vectorResults, err := store.SimilaritySearch(ctx, queryEmbedding, 2)
	if err != nil {
		log.Fatalf("Failed to search by vector: %v", err)
	}
	fmt.Printf("✅ Found %d results by vector search\n", len(vectorResults))

	fmt.Println("\n✨ Example completed successfully!")
}
