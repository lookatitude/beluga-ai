// Package main demonstrates multimodal RAG (Retrieval-Augmented Generation) workflows.
// This example shows how to use multimodal models with vector stores for multimodal document retrieval.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
	vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
)

func main() {
	ctx := context.Background()

	// Example: Multimodal RAG with image documents
	fmt.Println("=== Multimodal RAG Example ===")
	multimodalRAGExample(ctx)
}

func multimodalRAGExample(ctx context.Context) {
	// Create multimodal embedder
	embedder, err := embeddings.NewEmbedder(ctx, "openai", embeddings.Config{
		OpenAI: &embeddings.OpenAIConfig{
			APIKey: os.Getenv("OPENAI_API_KEY"),
			Model:  "text-embedding-3-large",
		},
	})
	if err != nil {
		log.Printf("Failed to create embedder: %v", err)
		return
	}

	// Check if embedder supports multimodal
	multiEmbedder, ok := embedder.(embeddingsiface.MultimodalEmbedder)
	if !ok || !multiEmbedder.SupportsMultimodal() {
		fmt.Println("Note: Embedder does not support multimodal, using text-only mode")
		// Fall back to text-only RAG
		textOnlyRAGExample(ctx, embedder)
		return
	}

	// Create vector store
	store, err := vectorstores.NewVectorStore(ctx, "qdrant", vectorstoresiface.Config{
		Embedder: embedder,
		// Add other Qdrant-specific config here
	})
	if err != nil {
		log.Printf("Failed to create vector store: %v", err)
		return
	}

	// Create multimodal documents with images
	documents := []schema.Document{
		{
			PageContent: "A beautiful sunset over the ocean with orange and pink hues",
			Metadata: map[string]string{
				"image_url": "https://example.com/sunset.jpg",
				"image_type": "image/jpeg",
				"category": "nature",
			},
		},
		{
			PageContent: "A modern city skyline at night with bright lights",
			Metadata: map[string]string{
				"image_url": "https://example.com/city.jpg",
				"image_type": "image/jpeg",
				"category": "urban",
			},
		},
	}

	// Generate multimodal embeddings and store documents
	fmt.Println("Storing multimodal documents...")
	for i, doc := range documents {
		embedding, err := multiEmbedder.EmbedQueryMultimodal(ctx, doc)
		if err != nil {
			log.Printf("Failed to generate embedding for document %d: %v", i, err)
			continue
		}

		doc.Embedding = embedding
		_, err = store.AddDocuments(ctx, []schema.Document{doc})
		if err != nil {
			log.Printf("Failed to add document %d: %v", i, err)
			continue
		}
		fmt.Printf("Stored document %d: %s\n", i+1, doc.PageContent)
	}

	// Create query document with image
	queryDoc := schema.Document{
		PageContent: "Find images similar to this ocean scene",
		Metadata: map[string]string{
			"image_url": "https://example.com/query-ocean.jpg",
			"image_type": "image/jpeg",
		},
	}

	// Generate query embedding
	queryEmbedding, err := multiEmbedder.EmbedQueryMultimodal(ctx, queryDoc)
	if err != nil {
		log.Printf("Failed to generate query embedding: %v", err)
		return
	}

	// Search for similar documents
	fmt.Println("\nSearching for similar documents...")
	docs, scores, err := store.SimilaritySearch(ctx, queryEmbedding, 3)
	if err != nil {
		log.Printf("Failed to search: %v", err)
		return
	}

	// Display results
	fmt.Println("\nSearch Results:")
	for i, doc := range docs {
		fmt.Printf("\nResult %d (score: %.4f):\n", i+1, scores[i])
		fmt.Printf("  Content: %s\n", doc.PageContent)
		if imageURL, ok := doc.Metadata["image_url"]; ok {
			fmt.Printf("  Image URL: %s\n", imageURL)
		}
		if category, ok := doc.Metadata["category"]; ok {
			fmt.Printf("  Category: %s\n", category)
		}
	}
}

func textOnlyRAGExample(ctx context.Context, embedder embeddingsiface.Embedder) {
	// Fallback to text-only RAG if multimodal not supported
	fmt.Println("Using text-only RAG mode")

	store, err := vectorstores.NewVectorStore(ctx, "qdrant", vectorstoresiface.Config{
		Embedder: embedder,
	})
	if err != nil {
		log.Printf("Failed to create vector store: %v", err)
		return
	}

	// Create text-only documents
	documents := []schema.Document{
		{PageContent: "Artificial intelligence is transforming industries"},
		{PageContent: "Machine learning enables computers to learn from data"},
		{PageContent: "Deep learning uses neural networks with multiple layers"},
	}

	// Store documents
	for _, doc := range documents {
		embedding, err := embedder.EmbedQuery(ctx, doc.PageContent)
		if err != nil {
			log.Printf("Failed to generate embedding: %v", err)
			continue
		}
		doc.Embedding = embedding
		_, err = store.AddDocuments(ctx, []schema.Document{doc})
		if err != nil {
			log.Printf("Failed to add document: %v", err)
		}
	}

	// Search
	queryEmbedding, _ := embedder.EmbedQuery(ctx, "What is AI?")
	docs, scores, err := store.SimilaritySearch(ctx, queryEmbedding, 2)
	if err != nil {
		log.Printf("Search failed: %v", err)
		return
	}

	fmt.Println("\nText-only RAG Results:")
	for i, doc := range docs {
		fmt.Printf("Result %d (score: %.4f): %s\n", i+1, scores[i], doc.PageContent)
	}
}
