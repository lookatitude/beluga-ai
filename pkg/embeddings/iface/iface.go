package iface

import "context"

// Embedder defines the interface for generating text embeddings.
// Implementations of this interface will provide access to different
// embedding models (e.g., OpenAI, local models, etc.).
type Embedder interface {
	// EmbedDocuments generates embeddings for a batch of documents.
	// It takes a context for cancellation and deadline propagation, and a slice
	// of strings (documents) and returns a slice of float32 slices (embeddings)
	// or an error if the process fails.
	EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)

	// EmbedQuery generates an embedding for a single query text.
	// It takes a context and a string (query) and returns a float32 slice
	// (embedding) or an error if the process fails.
	EmbedQuery(ctx context.Context, text string) ([]float32, error)

	// GetDimension returns the dimensionality of the embeddings produced by this embedder.
	// This can be useful for configuring vector stores or other components that
	// need to know the embedding size.
	// Returns 0 if the dimension is not fixed or known in advance (though this is rare for most providers).
	GetDimension(ctx context.Context) (int, error)
}

