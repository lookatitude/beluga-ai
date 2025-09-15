// Package iface defines the core interfaces for the retrievers package.
// These interfaces follow the Interface Segregation Principle (ISP) by being
// small, focused, and serving specific purposes in the RAG pipeline.
package iface

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// Loader defines the interface for loading data from various sources (files, URLs, databases, etc.)
// and converting it into a sequence of schema.Document objects.
// Note: Loader interface is now defined in pkg/core/interfaces.go

// Splitter defines the interface for splitting large schema.Document objects
// or raw text into smaller, more manageable chunks.
// This is crucial for embedding, as models have context limits, and retrieval works better
// with smaller, focused chunks.
type Splitter interface {
	// SplitDocuments takes a slice of existing documents and splits each one into smaller documents.
	// Metadata is typically preserved and potentially updated (e.g., adding chunk numbers).
	SplitDocuments(ctx context.Context, documents []schema.Document) ([]schema.Document, error)
	// CreateDocuments takes raw text strings and optional corresponding metadata,
	// splits the text, and creates new schema.Document objects for each chunk.
	CreateDocuments(ctx context.Context, texts []string, metadatas []map[string]any) ([]schema.Document, error)
	// SplitText splits a single raw text string into a slice of smaller strings.
	SplitText(ctx context.Context, text string) ([]string, error)
}

// Embedder defines the interface for creating vector representations (embeddings)
// of text documents or query strings.
// These embeddings capture the semantic meaning of the text.
type Embedder interface {
	// EmbedDocuments creates embeddings for a batch of document texts.
	// Returns a slice of embeddings, where each embedding corresponds to a document text.
	EmbedDocuments(ctx context.Context, documents []string) ([][]float32, error)
	// EmbedQuery creates an embedding for a single query string.
	// Query embeddings might be generated differently from document embeddings by some models.
	EmbedQuery(ctx context.Context, query string) ([]float32, error)
	// GetDimension returns the dimensionality of the embeddings produced by this embedder.
	GetDimension(ctx context.Context) (int, error)
}

// VectorStore defines the interface for storing vector embeddings and performing
// similarity searches over them.
// Vector stores are the core component for retrieving documents based on semantic similarity.
type VectorStore interface {
	// AddDocuments embeds the provided documents (if an embedder is configured with the store
	// or passed as an option) and stores them along with their embeddings and metadata.
	// Returns a slice of IDs assigned to the added documents, or an error.
	// Options might include specifying an embedder to use if not pre-configured.
	AddDocuments(ctx context.Context, documents []schema.Document, options ...core.Option) ([]string, error)

	// DeleteDocuments removes documents from the store based on their IDs.
	// Returns an error if deletion fails for any ID.
	DeleteDocuments(ctx context.Context, ids []string, options ...core.Option) error

	// SimilaritySearch performs a similarity search using a pre-computed query vector.
	// It finds the `k` most similar documents along with their similarity scores.
	// `k` specifies the number of top similar documents to return.
	SimilaritySearch(ctx context.Context, queryVector []float32, k int) ([]schema.Document, []float32, error)

	// SimilaritySearchByQuery performs a similarity search using a query string.
	// It first generates an embedding for the query string using the provided Embedder
	// and then performs a similarity search with the resulting vector.
	SimilaritySearchByQuery(ctx context.Context, query string, k int, embedder Embedder) ([]schema.Document, []float32, error)

	// TODO: Consider adding UpdateDocuments method.
	// TODO: Consider adding MaxMarginalRelevanceSearch (MMR) method.

	// AsRetriever returns a Retriever instance based on this VectorStore.
	// This allows the VectorStore to be easily used in chains/graphs that expect a Retriever.
	// Options can configure the retriever behavior (e.g., search type, k value, filters).
	AsRetriever(options ...core.Option) core.Retriever
}

// Note: Retriever interface is now defined in pkg/core/interfaces.go
