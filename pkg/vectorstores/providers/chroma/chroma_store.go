// Package chroma provides a Chroma vector store implementation.
// Chroma is an open-source embedding database that provides a simple
// and developer-friendly interface for vector storage and search.
//
// Features:
// - Simple REST API
// - Developer-friendly interface
// - Metadata filtering
// - Collection management
//
// Requirements:
// - Chroma server running (local or remote)
// - Collection name
// - Embedding dimension
//
// Example:
//
//	import "github.com/lookatitude/beluga-ai/pkg/vectorstores"
//
//	store, err := vectorstores.NewVectorStore(ctx, "chroma",
//		vectorstores.WithEmbedder(embedder),
//		vectorstores.WithProviderConfig("url", "http://localhost:8000"),
//		vectorstores.WithProviderConfig("collection_name", "documents"),
//		vectorstores.WithProviderConfig("embedding_dimension", 768),
//	)
package chroma

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
	vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
)

// ChromaStore implements the VectorStore interface using Chroma API.
type ChromaStore struct {
	url            string
	collectionName string
	name           string
	embeddingDim   int
	mu             sync.RWMutex
}

// NewChromaStoreFromConfig creates a new ChromaStore from configuration.
// This is used by the factory pattern for creating stores via the registry.
// Chroma is an open-source embedding database with a simple REST API.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - config: Configuration containing embedder, URL, collection name, and embedding dimension
//
// Returns:
//   - VectorStore: A new Chroma vector store instance
//   - error: Configuration errors or connection failures
//
// Example:
//
//	config := vectorstoresiface.Config{
//	    Embedder: embedder,
//	    ProviderConfig: map[string]any{
//	        "url":                "http://localhost:8000",
//	        "collection_name":    "documents",
//	        "embedding_dimension": 768,
//	    },
//	}
//	store, err := chroma.NewChromaStoreFromConfig(ctx, config)
//
// Example usage can be found in examples/rag/simple/main.go.
func NewChromaStoreFromConfig(ctx context.Context, config vectorstoresiface.Config) (vectorstores.VectorStore, error) {
	// Extract chroma-specific configuration from ProviderConfig
	providerConfig, ok := config.ProviderConfig["chroma"]
	if !ok {
		providerConfig = make(map[string]any)
	}

	// Extract connection parameters
	url, _ := providerConfig.(map[string]any)["url"].(string)
	if url == "" {
		if urlVal, ok := config.ProviderConfig["url"].(string); ok {
			url = urlVal
		}
	}
	if url == "" {
		url = "http://localhost:8000" // Default Chroma URL
	}

	collectionName, _ := providerConfig.(map[string]any)["collection_name"].(string)
	if collectionName == "" {
		if collVal, ok := config.ProviderConfig["collection_name"].(string); ok {
			collectionName = collVal
		}
	}
	if collectionName == "" {
		collectionName = "beluga_documents" // Default collection name
	}

	embeddingDim, _ := providerConfig.(map[string]any)["embedding_dimension"].(int)
	if embeddingDim == 0 {
		if dimVal, ok := config.ProviderConfig["embedding_dimension"].(int); ok {
			embeddingDim = dimVal
		}
	}
	if embeddingDim == 0 {
		embeddingDim = 1536 // Default OpenAI embedding dimension
	}

	store := &ChromaStore{
		url:            url,
		collectionName: collectionName,
		embeddingDim:   embeddingDim,
		name:           "chroma",
	}

	// TODO: Initialize Chroma client connection
	// This will involve:
	// 1. Creating a Chroma client with URL
	// 2. Checking if collection exists, create if not
	// 3. Verifying collection configuration matches embedding dimension

	return store, nil
}

// AddDocuments stores documents with their embeddings in Chroma.
func (s *ChromaStore) AddDocuments(ctx context.Context, documents []schema.Document, opts ...vectorstores.Option) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Apply options
	config := vectorstores.NewDefaultConfig()
	for _, opt := range opts {
		opt(config)
	}

	// Generate embeddings if embedder is provided
	var embeddings [][]float32
	var err error
	if config.Embedder != nil {
		texts := make([]string, len(documents))
		for i, doc := range documents {
			texts[i] = doc.GetContent()
		}
		embeddings, err = config.Embedder.EmbedDocuments(ctx, texts)
		if err != nil {
			return nil, vectorstores.NewVectorStoreErrorWithMessage("AddDocuments", vectorstores.ErrCodeEmbeddingFailed,
				"failed to generate embeddings for documents", err)
		}
	} else {
		// Check if documents already have embeddings
		embeddings = make([][]float32, len(documents))
		for i, doc := range documents {
			// Check if document has embedding field set
			if len(doc.Embedding) > 0 {
				embeddings[i] = doc.Embedding
			} else {
				return nil, vectorstores.NewVectorStoreErrorWithMessage("AddDocuments", vectorstores.ErrCodeEmbeddingFailed,
					fmt.Sprintf("no embedder provided and document %d has no embedding", i), nil)
			}
		}
	}

	// Prepare data for Chroma add
	ids := make([]string, len(documents))
	metadatas := make([]map[string]any, len(documents))
	contents := make([]string, len(documents))

	for i, doc := range documents {
		// Generate ID if not present
		id := doc.ID
		if id == "" {
			id = fmt.Sprintf("doc-%d-%d", time.Now().UnixNano(), i)
		}
		ids[i] = id

		// Prepare metadata (exclude embedding)
		metadata := make(map[string]any)
		for k, v := range doc.Metadata {
			if k != "embedding" {
				metadata[k] = v
			}
		}
		metadatas[i] = metadata
		contents[i] = doc.GetContent()
	}

	// TODO: Call Chroma add API
	// This will involve:
	// 1. Preparing the add request with ids, embeddings, metadatas, documents
	// 2. Calling Chroma REST API to add documents
	// 3. Handling errors and retries
	// 4. Returning document IDs

	// Placeholder: Return IDs for now
	return ids, nil
}

// DeleteDocuments removes documents from Chroma.
func (s *ChromaStore) DeleteDocuments(ctx context.Context, ids []string, opts ...vectorstores.Option) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(ids) == 0 {
		return nil
	}

	// TODO: Call Chroma delete API
	// This will involve:
	// 1. Preparing the delete request with IDs
	// 2. Calling Chroma API to delete documents
	// 3. Handling errors and retries

	return nil
}

// SimilaritySearch finds the k most similar documents to a query vector.
func (s *ChromaStore) SimilaritySearch(ctx context.Context, queryVector []float32, k int, opts ...vectorstores.Option) ([]schema.Document, []float32, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Apply options
	config := vectorstores.NewDefaultConfig()
	for _, opt := range opts {
		opt(config)
	}

	// Use k from options if provided, otherwise use parameter
	_ = k // TODO: Use k when implementing query
	if config.SearchK > 0 {
		_ = config.SearchK // TODO: Use config.SearchK when implementing query
	}

	// TODO: Call Chroma query API
	// This will involve:
	// 1. Preparing the query request with query_embeddings and n_results
	// 2. Applying where filter if metadata filters are provided
	// 3. Calling Chroma API to query documents
	// 4. Processing results and converting to schema.Document
	// 5. Applying score threshold if configured
	// 6. Returning documents and scores

	// Placeholder: Return empty results for now
	return []schema.Document{}, []float32{}, nil
}

// SimilaritySearchByQuery performs similarity search using a text query.
func (s *ChromaStore) SimilaritySearchByQuery(ctx context.Context, query string, k int, embedder vectorstores.Embedder, opts ...vectorstores.Option) ([]schema.Document, []float32, error) {
	// Generate embedding for query
	if embedder == nil {
		// Try to get embedder from options
		config := vectorstores.NewDefaultConfig()
		for _, opt := range opts {
			opt(config)
		}
		embedder = config.Embedder
	}

	if embedder == nil {
		return nil, nil, vectorstores.NewVectorStoreErrorWithMessage("SimilaritySearchByQuery", vectorstores.ErrCodeEmbeddingFailed,
			"embedder is required for SimilaritySearchByQuery", nil)
	}

	queryVector, err := embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, nil, vectorstores.NewVectorStoreErrorWithMessage("SimilaritySearchByQuery", vectorstores.ErrCodeEmbeddingFailed,
			"failed to generate embedding for query", err)
	}

	return s.SimilaritySearch(ctx, queryVector, k, opts...)
}

// AsRetriever returns a Retriever implementation based on this VectorStore.
func (s *ChromaStore) AsRetriever(opts ...vectorstores.Option) vectorstores.Retriever {
	return &ChromaRetriever{
		store: s,
		opts:  opts,
	}
}

// GetName returns the name of the vector store.
func (s *ChromaStore) GetName() string {
	return s.name
}

// ChromaRetriever implements the Retriever interface for ChromaStore.
type ChromaRetriever struct {
	store *ChromaStore
	opts  []vectorstores.Option
}

// GetRelevantDocuments retrieves relevant documents for a query.
func (r *ChromaRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
	// Apply default search options
	opts := append(r.opts, vectorstores.WithSearchK(5))

	// Get embedder from options
	config := vectorstores.NewDefaultConfig()
	for _, opt := range opts {
		opt(config)
	}

	docs, _, err := r.store.SimilaritySearchByQuery(ctx, query, 5, config.Embedder, opts...)
	return docs, err
}

// Ensure ChromaStore implements the VectorStore interface.
var _ vectorstores.VectorStore = (*ChromaStore)(nil)

// Ensure ChromaRetriever implements the Retriever interface.
var _ vectorstores.Retriever = (*ChromaRetriever)(nil)
