package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lookatitude/beluga-ai/pkg/embeddings"
)

func main() {
	fmt.Println("🔄 Beluga AI Embeddings Package Usage Example")
	fmt.Println("=============================================")

	ctx := context.Background()

	// Example 1: Create Embedder Factory
	fmt.Println("\n📋 Example 1: Creating Embedder Factory")
	config := &embeddings.Config{
		Mock: &embeddings.MockConfig{
			Enabled:   true,
			Dimension: 128,
		},
	}
	config.SetDefaults()

	factory, err := embeddings.NewEmbedderFactory(config)
	if err != nil {
		log.Fatalf("Failed to create factory: %v", err)
	}
	fmt.Println("✅ Factory created successfully")

	// Example 2: Create Embedder
	fmt.Println("\n📋 Example 2: Creating Embedder")
	embedder, err := factory.NewEmbedder("mock")
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}
	fmt.Println("✅ Embedder created successfully")

	// Example 3: Get Embedding Dimension
	fmt.Println("\n📋 Example 3: Getting Embedding Dimension")
	dimension, err := embedder.GetDimension(ctx)
	if err != nil {
		log.Fatalf("Failed to get dimension: %v", err)
	}
	fmt.Printf("✅ Embedding dimension: %d\n", dimension)

	// Example 4: Embed Single Text
	fmt.Println("\n📋 Example 4: Embedding Single Text")
	text := "Machine learning is a subset of artificial intelligence"
	embedding, err := embedder.EmbedQuery(ctx, text)
	if err != nil {
		log.Fatalf("Failed to embed text: %v", err)
	}
	fmt.Printf("✅ Text embedded successfully (dimension: %d)\n", len(embedding))
	fmt.Printf("   First 5 values: %v\n", embedding[:min(5, len(embedding))])

	// Example 5: Embed Multiple Documents
	fmt.Println("\n📋 Example 5: Embedding Multiple Documents")
	documents := []string{
		"Artificial intelligence is transforming industries",
		"Deep learning uses neural networks",
		"Natural language processing enables text understanding",
	}
	embeddings, err := embedder.EmbedDocuments(ctx, documents)
	if err != nil {
		log.Fatalf("Failed to embed documents: %v", err)
	}
	fmt.Printf("✅ Embedded %d documents successfully\n", len(embeddings))
	for i, emb := range embeddings {
		fmt.Printf("   Document %d: dimension %d\n", i+1, len(emb))
	}

	// Example 6: Health Check
	fmt.Println("\n📋 Example 6: Health Check")
	err = factory.CheckHealth(ctx, "mock")
	if err != nil {
		log.Printf("⚠️  Health check failed: %v", err)
	} else {
		fmt.Println("✅ Health check passed")
	}

	// Example 7: List Available Providers
	fmt.Println("\n📋 Example 7: Listing Available Providers")
	providers := factory.GetAvailableProviders()
	fmt.Printf("✅ Available providers: %v\n", providers)

	fmt.Println("\n✨ All examples completed successfully!")
	fmt.Println("\nFor more examples, see:")
	fmt.Println("  - examples/rag/simple/ - RAG pipeline with embeddings")
	fmt.Println("  - examples/rag/advanced/ - Advanced RAG patterns")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
