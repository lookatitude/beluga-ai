// Package vectorstores provides interfaces and implementations for vector storage and retrieval.
// It supports multiple vector store providers and enables efficient similarity search
// for retrieval-augmented generation (RAG) applications.
//
// Example usage:
//
//	import "github.com/lookatitude/beluga-ai/pkg/vectorstores"
//
//	// Create an in-memory vector store
//	store, err := vectorstores.NewInMemoryStore(vectorstores.WithEmbedder(embedder))
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Add documents
//	docs := []schema.Document{
//		schema.NewDocument("content 1", map[string]string{"source": "doc1"}),
//		schema.NewDocument("content 2", map[string]string{"source": "doc2"}),
//	}
//	ids, err := store.AddDocuments(ctx, docs)
//
//	// Search by query
//	results, scores, err := store.SimilaritySearchByQuery(ctx, "search query", 5, embedder)
package vectorstores

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// Embedder defines the interface for generating vector embeddings from text.
// This follows the Interface Segregation Principle by focusing on embedding functionality.
type Embedder interface {
	EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)
	EmbedQuery(ctx context.Context, text string) ([]float32, error)
}

// Retriever defines the interface for retrieving documents based on queries.
// This enables VectorStores to be used in retrieval chains and graphs.
type Retriever interface {
	GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error)
}

// VectorStore defines the core interface for storing and querying vector embeddings.
// It provides methods for document storage, deletion, and similarity search.
// Implementations should be thread-safe and handle context cancellation properly.
type VectorStore interface {
	// AddDocuments stores documents with their embeddings in the vector store.
	// It generates embeddings if an embedder is available, or uses pre-computed embeddings.
	// Returns the IDs of stored documents and any error encountered.
	//
	// Example:
	//   docs := []schema.Document{
	//       schema.NewDocument("machine learning is awesome", map[string]string{"topic": "ml"}),
	//   }
	//   ids, err := store.AddDocuments(ctx, docs, vectorstores.WithEmbedder(embedder))
	AddDocuments(ctx context.Context, documents []schema.Document, opts ...Option) ([]string, error)

	// DeleteDocuments removes documents from the store based on their IDs.
	// Returns an error if any document cannot be deleted.
	//
	// Example:
	//   err := store.DeleteDocuments(ctx, []string{"doc1", "doc2"})
	DeleteDocuments(ctx context.Context, ids []string, opts ...Option) error

	// SimilaritySearch finds the k most similar documents to a query vector.
	// Returns documents and their similarity scores (higher scores indicate better matches).
	//
	// Example:
	//   docs, scores, err := store.SimilaritySearch(ctx, queryVector, 10)
	SimilaritySearch(ctx context.Context, queryVector []float32, k int, opts ...Option) ([]schema.Document, []float32, error)

	// SimilaritySearchByQuery performs similarity search using a text query.
	// It generates an embedding for the query and then performs vector similarity search.
	//
	// Example:
	//   docs, scores, err := store.SimilaritySearchByQuery(ctx, "machine learning basics", 5, embedder)
	SimilaritySearchByQuery(ctx context.Context, query string, k int, embedder Embedder, opts ...Option) ([]schema.Document, []float32, error)

	// AsRetriever returns a Retriever implementation based on this VectorStore.
	// This enables the VectorStore to be used in retrieval chains and pipelines.
	//
	// Example:
	//   retriever := store.AsRetriever(vectorstores.WithSearchK(10))
	//   docs, err := retriever.GetRelevantDocuments(ctx, "query")
	AsRetriever(opts ...Option) Retriever

	// GetName returns the name/identifier of this vector store implementation.
	// This is used for logging, metrics, and debugging purposes.
	GetName() string
}

// Option represents a functional option for configuring VectorStore operations.
// This follows the functional options pattern for flexible configuration.
type Option func(*Config)

// Factory defines the interface for creating VectorStore instances.
// This enables dependency injection and makes testing easier.
type Factory interface {
	// CreateVectorStore creates a new VectorStore instance with the given configuration.
	// The config parameter contains provider-specific settings.
	CreateVectorStore(ctx context.Context, config Config) (VectorStore, error)
}

// StoreFactory is the global factory for creating vector store instances.
// It maintains a registry of available providers and their creation functions.
type StoreFactory struct {
	creators map[string]func(ctx context.Context, config Config) (VectorStore, error)
}

// NewStoreFactory creates a new StoreFactory instance.
func NewStoreFactory() *StoreFactory {
	return &StoreFactory{
		creators: make(map[string]func(ctx context.Context, config Config) (VectorStore, error)),
	}
}

// Register registers a new vector store provider with the factory.
func (f *StoreFactory) Register(name string, creator func(ctx context.Context, config Config) (VectorStore, error)) {
	f.creators[name] = creator
}

// Create creates a new vector store instance using the registered provider.
func (f *StoreFactory) Create(ctx context.Context, name string, config Config) (VectorStore, error) {
	creator, exists := f.creators[name]
	if !exists {
		return nil, NewVectorStoreError("unknown_provider", "vector store provider '%s' not found", name)
	}
	return creator(ctx, config)
}

// Global factory instance for easy access
var globalFactory = NewStoreFactory()

// RegisterGlobal registers a provider with the global factory.
func RegisterGlobal(name string, creator func(ctx context.Context, config Config) (VectorStore, error)) {
	globalFactory.Register(name, creator)
}

// NewVectorStore creates a vector store using the global factory.
func NewVectorStore(ctx context.Context, name string, config Config) (VectorStore, error) {
	return globalFactory.Create(ctx, name, config)
}
