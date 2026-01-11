// Package pinecone provides a Pinecone vector store implementation.
// Pinecone is a managed vector database service that provides serverless,
// scalable vector storage and similarity search.
//
// Features:
// - Serverless vector storage
// - Automatic scaling
// - High-performance similarity search
// - Metadata filtering
// - Multi-region support
//
// Requirements:
// - Pinecone API key
// - Pinecone project ID and index name
//
// Example:
//
//	import "github.com/lookatitude/beluga-ai/pkg/vectorstores"
//
//	store, err := vectorstores.NewVectorStore(ctx, "pinecone",
//		vectorstores.WithEmbedder(embedder),
//		vectorstores.WithProviderConfig("api_key", "your-api-key"),
//		vectorstores.WithProviderConfig("environment", "us-west1-gcp"),
//		vectorstores.WithProviderConfig("project_id", "your-project"),
//		vectorstores.WithProviderConfig("index_name", "my-index"),
//	)
package pinecone

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
	vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
)

// PineconeStore implements the VectorStore interface using Pinecone API.
type PineconeStore struct {
	apiKey      string
	environment string
	projectID   string
	indexName   string
	indexHost   string
	embeddingDim int
	name        string
	mu          sync.RWMutex
}

// PineconeConfig holds configuration specific to PineconeStore.
type PineconeConfig struct {
	APIKey        string
	Environment   string
	ProjectID     string
	IndexName     string
	IndexHost     string
	EmbeddingDim  int
}

// NewPineconeStoreFromConfig creates a new PineconeStore from configuration.
// This is used by the factory pattern for creating stores via the registry.
// Pinecone is a managed vector database service with serverless, scalable storage.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - config: Configuration containing embedder, API key, environment, project ID, index name, and embedding dimension
//
// Returns:
//   - VectorStore: A new Pinecone vector store instance
//   - error: Configuration errors or connection failures
//
// Example:
//
//	config := vectorstoresiface.Config{
//	    Embedder: embedder,
//	    ProviderConfig: map[string]any{
//	        "api_key":            "your-api-key",
//	        "environment":        "us-west1-gcp",
//	        "project_id":         "your-project",
//	        "index_name":         "my-index",
//	        "embedding_dimension": 768,
//	    },
//	}
//	store, err := pinecone.NewPineconeStoreFromConfig(ctx, config)
//
// Example usage can be found in examples/rag/simple/main.go
func NewPineconeStoreFromConfig(ctx context.Context, config vectorstoresiface.Config) (vectorstores.VectorStore, error) {
	// Extract pinecone-specific configuration from ProviderConfig
	providerConfig, ok := config.ProviderConfig["pinecone"]
	if !ok {
		providerConfig = make(map[string]any)
	}

	// Extract connection parameters
	apiKey, _ := providerConfig.(map[string]any)["api_key"].(string)
	if apiKey == "" {
		// Try direct config
		if apiKeyVal, ok := config.ProviderConfig["api_key"].(string); ok {
			apiKey = apiKeyVal
		}
	}
	if apiKey == "" {
		return nil, vectorstores.NewVectorStoreErrorWithMessage("NewPineconeStoreFromConfig", vectorstores.ErrCodeInvalidConfig,
			"api_key is required in pinecone provider config", nil)
	}

	environment, _ := providerConfig.(map[string]any)["environment"].(string)
	if environment == "" {
		if envVal, ok := config.ProviderConfig["environment"].(string); ok {
			environment = envVal
		}
	}
	if environment == "" {
		return nil, vectorstores.NewVectorStoreErrorWithMessage("NewPineconeStoreFromConfig", vectorstores.ErrCodeInvalidConfig,
			"environment is required in pinecone provider config", nil)
	}

	projectID, _ := providerConfig.(map[string]any)["project_id"].(string)
	if projectID == "" {
		if projVal, ok := config.ProviderConfig["project_id"].(string); ok {
			projectID = projVal
		}
	}
	if projectID == "" {
		return nil, vectorstores.NewVectorStoreErrorWithMessage("NewPineconeStoreFromConfig", vectorstores.ErrCodeInvalidConfig,
			"project_id is required in pinecone provider config", nil)
	}

	indexName, _ := providerConfig.(map[string]any)["index_name"].(string)
	if indexName == "" {
		if idxVal, ok := config.ProviderConfig["index_name"].(string); ok {
			indexName = idxVal
		}
	}
	if indexName == "" {
		return nil, vectorstores.NewVectorStoreErrorWithMessage("NewPineconeStoreFromConfig", vectorstores.ErrCodeInvalidConfig,
			"index_name is required in pinecone provider config", nil)
	}

	indexHost, _ := providerConfig.(map[string]any)["index_host"].(string)
	if indexHost == "" {
		if hostVal, ok := config.ProviderConfig["index_host"].(string); ok {
			indexHost = hostVal
		}
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

	store := &PineconeStore{
		apiKey:       apiKey,
		environment:  environment,
		projectID:    projectID,
		indexName:    indexName,
		indexHost:    indexHost,
		embeddingDim: embeddingDim,
		name:         "pinecone",
	}

	// TODO: Initialize Pinecone client connection
	// This will involve:
	// 1. Creating a Pinecone client with API key
	// 2. Connecting to the specified index
	// 3. Verifying index configuration matches embedding dimension

	return store, nil
}

// AddDocuments stores documents with their embeddings in Pinecone.
func (s *PineconeStore) AddDocuments(ctx context.Context, documents []schema.Document, opts ...vectorstores.Option) ([]string, error) {
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

	// Prepare vectors for Pinecone upsert
	vectors := make([]PineconeVector, len(documents))
	ids := make([]string, len(documents))

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
		metadata["content"] = doc.GetContent()

		vectors[i] = PineconeVector{
			ID:       id,
			Values:   embeddings[i],
			Metadata: metadata,
		}
	}

	// TODO: Call Pinecone upsert API
	// This will involve:
	// 1. Preparing the upsert request with vectors
	// 2. Calling Pinecone API to upsert vectors
	// 3. Handling errors and retries
	// 4. Returning document IDs

	// Placeholder: Return IDs for now
	return ids, nil
}

// DeleteDocuments removes documents from Pinecone.
func (s *PineconeStore) DeleteDocuments(ctx context.Context, ids []string, opts ...vectorstores.Option) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(ids) == 0 {
		return nil
	}

	// TODO: Call Pinecone delete API
	// This will involve:
	// 1. Preparing the delete request with IDs
	// 2. Calling Pinecone API to delete vectors
	// 3. Handling errors and retries

	return nil
}

// SimilaritySearch finds the k most similar documents to a query vector.
func (s *PineconeStore) SimilaritySearch(ctx context.Context, queryVector []float32, k int, opts ...vectorstores.Option) ([]schema.Document, []float32, error) {
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

	// TODO: Call Pinecone query API
	// This will involve:
	// 1. Preparing the query request with vector and topK
	// 2. Applying metadata filters if provided
	// 3. Calling Pinecone API to query vectors
	// 4. Processing results and converting to schema.Document
	// 5. Applying score threshold if configured
	// 6. Returning documents and scores

	// Placeholder: Return empty results for now
	return []schema.Document{}, []float32{}, nil
}

// SimilaritySearchByQuery performs similarity search using a text query.
func (s *PineconeStore) SimilaritySearchByQuery(ctx context.Context, query string, k int, embedder vectorstores.Embedder, opts ...vectorstores.Option) ([]schema.Document, []float32, error) {
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
func (s *PineconeStore) AsRetriever(opts ...vectorstores.Option) vectorstores.Retriever {
	return &PineconeRetriever{
		store: s,
		opts:  opts,
	}
}

// GetName returns the name of the vector store.
func (s *PineconeStore) GetName() string {
	return s.name
}

// PineconeVector represents a vector in Pinecone format.
type PineconeVector struct {
	ID       string         `json:"id"`
	Values  []float32      `json:"values"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// PineconeQueryRequest represents a query request to Pinecone.
type PineconeQueryRequest struct {
	Vector          []float32           `json:"vector"`
	TopK            int                  `json:"topK"`
	IncludeMetadata bool                 `json:"includeMetadata"`
	IncludeValues   bool                 `json:"includeValues"`
	Filter          map[string]any       `json:"filter,omitempty"`
	Namespace       string               `json:"namespace,omitempty"`
}

// PineconeQueryResponse represents a query response from Pinecone.
type PineconeQueryResponse struct {
	Matches []PineconeMatch `json:"matches"`
}

// PineconeMatch represents a match from Pinecone query.
type PineconeMatch struct {
	ID       string         `json:"id"`
	Score    float32        `json:"score"`
	Values   []float32      `json:"values,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// PineconeRetriever implements the Retriever interface for PineconeStore.
type PineconeRetriever struct {
	store *PineconeStore
	opts  []vectorstores.Option
}

// GetRelevantDocuments retrieves relevant documents for a query.
func (r *PineconeRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
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

// Ensure PineconeStore implements the VectorStore interface.
var _ vectorstores.VectorStore = (*PineconeStore)(nil)

// Ensure PineconeRetriever implements the Retriever interface.
var _ vectorstores.Retriever = (*PineconeRetriever)(nil)
