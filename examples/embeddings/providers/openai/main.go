package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	_ "github.com/lookatitude/beluga-ai/pkg/embeddings/providers/openai"
)

func main() {
	fmt.Println("🔢 Beluga AI - OpenAI Embedding Provider Example")
	fmt.Println("=================================================")

	ctx := context.Background()

	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Create embedding configuration
	config := &embeddings.Config{
		OpenAI: &embeddings.OpenAIConfig{
			APIKey:  apiKey,
			Model:   "text-embedding-3-small",
			Enabled: true,
		},
	}
	config.SetDefaults()

	// Create embedder factory
	factory, err := embeddings.NewEmbedderFactory(config)
	if err != nil {
		log.Fatalf("Failed to create factory: %v", err)
	}
	fmt.Println("✅ Created embedder factory")

	// Create OpenAI embedder
	embedder, err := factory.NewEmbedder("openai")
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}
	fmt.Println("✅ Created OpenAI embedder")

	// Get embedding dimension
	dimension, err := embedder.GetDimension(ctx)
	if err != nil {
		log.Fatalf("Failed to get dimension: %v", err)
	}
	fmt.Printf("✅ Embedding dimension: %d\n", dimension)

	// Embed a single query
	fmt.Println("\n📝 Embedding query text...")
	query := "What is machine learning?"
	queryEmbedding, err := embedder.EmbedQuery(ctx, query)
	if err != nil {
		log.Fatalf("Failed to embed query: %v", err)
	}
	fmt.Printf("✅ Query embedding length: %d\n", len(queryEmbedding))

	// Embed multiple documents
	fmt.Println("\n📚 Embedding documents...")
	documents := []string{
		"Machine learning is a subset of artificial intelligence.",
		"Deep learning uses neural networks with multiple layers.",
		"Natural language processing enables computers to understand text.",
	}
	documentEmbeddings, err := embedder.EmbedDocuments(ctx, documents)
	if err != nil {
		log.Fatalf("Failed to embed documents: %v", err)
	}
	fmt.Printf("✅ Embedded %d documents\n", len(documentEmbeddings))

	fmt.Println("\n✨ Example completed successfully!")
}
