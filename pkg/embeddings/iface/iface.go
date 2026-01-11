package iface

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// Embedder defines the interface for generating text embeddings.
// Implementations of this interface will provide access to different
// embedding models (e.g., OpenAI, local models, etc.).
//
// Embedder follows the Interface Segregation Principle (ISP) by providing
// focused methods specific to embedding operations.
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

// MultimodalEmbedder extends Embedder to support multimodal inputs (images, audio, video).
// Implementations can support text-only, image-only, or mixed multimodal embeddings.
type MultimodalEmbedder interface {
	Embedder

	// EmbedDocumentsMultimodal generates embeddings for a batch of documents that may contain
	// multimodal content (text, images, audio, video).
	// It takes a context and a slice of Document types and returns embeddings.
	// If a document contains only text, it should use the text content.
	// If a document contains images or other media, it should process them accordingly.
	EmbedDocumentsMultimodal(ctx context.Context, documents []schema.Document) ([][]float32, error)

	// EmbedQueryMultimodal generates an embedding for a single query that may contain
	// multimodal content (text, images, audio, video).
	// It takes a context and a Document and returns an embedding.
	EmbedQueryMultimodal(ctx context.Context, document schema.Document) ([]float32, error)

	// SupportsMultimodal returns true if this embedder supports multimodal inputs.
	// If false, EmbedDocumentsMultimodal and EmbedQueryMultimodal should fall back
	// to text-only processing.
	SupportsMultimodal() bool
}
