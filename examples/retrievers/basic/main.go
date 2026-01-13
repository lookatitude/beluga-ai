package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	_ "github.com/lookatitude/beluga-ai/pkg/embeddings/providers/mock"
	"github.com/lookatitude/beluga-ai/pkg/retrievers"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
	_ "github.com/lookatitude/beluga-ai/pkg/vectorstores/providers/inmemory"
)

func main() {
	fmt.Println("🔍 Beluga AI - Retrievers Package Example")
	fmt.Println("=========================================")

	ctx := context.Background()

	// Step 1: Create embedder
	fmt.Println("\n📋 Step 1: Creating embedder...")
	embedderConfig := &embeddings.Config{
		Mock: &embeddings.MockConfig{
			Enabled:   true,
			Dimension: 128,
		},
	}
	embedderConfig.SetDefaults()
	embedderFactory, err := embeddings.NewEmbedderFactory(embedderConfig)
	if err != nil {
		log.Fatalf("Failed to create embedder factory: %v", err)
	}
	embedder, err := embedderFactory.NewEmbedder("mock")
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}
	fmt.Println("✅ Embedder created")

	// Step 2: Create vector store
	fmt.Println("\n📋 Step 2: Creating vector store...")
	store, err := vectorstores.NewInMemoryStore(ctx,
		vectorstores.WithEmbedder(embedder),
	)
	if err != nil {
		log.Fatalf("Failed to create vector store: %v", err)
	}
	fmt.Println("✅ Vector store created")

	// Step 3: Add documents to store
	fmt.Println("\n📋 Step 3: Adding documents...")
	documents := []schema.Document{
		schema.NewDocument("Machine learning is a subset of artificial intelligence", map[string]string{}),
		schema.NewDocument("Deep learning uses neural networks with multiple layers", map[string]string{}),
		schema.NewDocument("Natural language processing enables computers to understand text", map[string]string{}),
		schema.NewDocument("Computer vision allows machines to interpret visual information", map[string]string{}),
	}
	ids, err := store.AddDocuments(ctx, documents)
	if err != nil {
		log.Fatalf("Failed to add documents: %v", err)
	}
	fmt.Printf("✅ Added %d documents with IDs: %v\n", len(ids), ids)

	// Step 4: Create retriever
	fmt.Println("\n📋 Step 4: Creating retriever...")
	retriever, err := retrievers.NewVectorStoreRetriever(store,
		retrievers.WithDefaultK(3),
	)
	if err != nil {
		log.Fatalf("Failed to create retriever: %v", err)
	}
	fmt.Println("✅ Retriever created")

	// Step 5: Retrieve relevant documents
	fmt.Println("\n📋 Step 5: Retrieving relevant documents...")
	query := "What is artificial intelligence?"
	relevantDocs, err := retriever.GetRelevantDocuments(ctx, query)
	if err != nil {
		log.Fatalf("Failed to retrieve documents: %v", err)
	}
	fmt.Printf("✅ Retrieved %d relevant documents:\n", len(relevantDocs))
	for i, doc := range relevantDocs {
		fmt.Printf("  %d. %s (score: %.4f)\n", i+1, doc.GetContent(), doc.Score)
	}

	fmt.Println("\n✨ Example completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Use real vector stores (Qdrant, Pinecone, etc.)")
	fmt.Println("- Configure score thresholds and top-k values")
	fmt.Println("- Integrate with RAG pipelines")
}
