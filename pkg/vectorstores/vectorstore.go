package vectorstores

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// VectorStore is an interface for storing and querying vector embeddings of documents.
// It is a key component for retrieval-augmented generation (RAG).	ype VectorStore interface {
	// AddDocuments adds a list of documents to the vector store.
	// It typically involves generating embeddings for the documents using an Embedder
	// and then storing both the documents and their embeddings.
	AddDocuments(ctx context.Context, docs []schema.Document, embedder embeddings.Embedder) error

	// SimilaritySearch performs a similarity search for a given query vector.
	// It returns a list of documents that are most similar to the query vector,
	// along with their similarity scores.
	// `k` specifies the number of top similar documents to return.
	SimilaritySearch(ctx context.Context, queryVector []float32, k int) ([]schema.Document, []float32, error)

	// SimilaritySearchByQuery performs a similarity search using a query string.
	// It first generates an embedding for the query string using the provided Embedder
	// and then performs a similarity search with the resulting vector.
	SimilaritySearchByQuery(ctx context.Context, query string, k int, embedder embeddings.Embedder) ([]schema.Document, []float32, error)

	// GetName returns the name of the vector store provider (e.g., "inmemory", "pinecone", "pgvector").
	GetName() string

	// TODO: Add methods for deleting documents, updating documents, filtering, etc.
}

// Config is a generic configuration structure for VectorStore providers.
// Specific providers can embed this and add their own fields.
type Config struct {
	Type         string                 // e.g., "inmemory", "pinecone", "pgvector"
	Name         string                 // A unique name for this vector store configuration instance (optional, for lookup)
	ProviderArgs map[string]interface{} // Provider-specific arguments, e.g., API keys, connection strings, embedding dimensions
}

// Factory defines the interface for creating VectorStore instances.	ype Factory interface {
	CreateVectorStore(ctx context.Context, config Config) (VectorStore, error)
}

