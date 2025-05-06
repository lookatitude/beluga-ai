// Package rag defines interfaces for components involved in Retrieval Augmented Generation (RAG).
// RAG pipelines typically involve loading data, splitting it, embedding it, storing it,
// and retrieving relevant parts to augment prompts sent to language models.
package rag

import (
	"context"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/schema"
)

// Loader defines the interface for loading data from various sources (files, URLs, databases, etc.)
// and converting it into a sequence of schema.Document objects.
type Loader interface {
	// Load reads all data from the source and returns it as a slice of Documents.
	Load(ctx context.Context) ([]schema.Document, error)
	// LazyLoad provides an alternative way to load data, returning a channel that yields
	// documents one by one as they become available.
	// This is useful for large datasets or sources where loading everything at once is inefficient.
	// Errors encountered during loading should be sent on the channel.
	// The channel yields items of type schema.Document or error.
	LazyLoad(ctx context.Context) (<-chan any, error)
}

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

	// SimilaritySearch performs a similarity search using a query string.
	// It embeds the query using the configured embedder and finds the `k` most similar documents.
	// Options can include metadata filters, search parameters specific to the store, etc.
	SimilaritySearch(ctx context.Context, query string, k int, options ...core.Option) ([]schema.Document, error)

	// SimilaritySearchByVector performs a similarity search using a pre-computed query vector.
	// Options can include metadata filters, search parameters specific to the store, etc.
	SimilaritySearchByVector(ctx context.Context, embedding []float32, k int, options ...core.Option) ([]schema.Document, error)

	// TODO: Consider adding UpdateDocuments method.
	// TODO: Consider adding MaxMarginalRelevanceSearch (MMR) method.

	// AsRetriever returns a Retriever instance based on this VectorStore.
	// This allows the VectorStore to be easily used in chains/graphs that expect a Retriever.
	// Options can configure the retriever behavior (e.g., search type, k value, filters).
	AsRetriever(options ...core.Option) Retriever
}

// Retriever defines a generic interface for fetching relevant documents based on a query string.
// While often backed by a VectorStore, retrievers can implement other strategies
// (e.g., keyword search, database lookups, hybrid approaches).
// Retrievers implement the core.Runnable interface, making them easily pluggable into chains.
type Retriever interface {
	core.Runnable // Input: string (query), Output: []schema.Document

	// GetRelevantDocuments retrieves documents considered relevant to the given query string.
	GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error)
}

// --- Options for VectorStore/Retriever ---

// optionFunc is a helper type for creating options from functions.
type optionFunc func(*map[string]any)

func (f optionFunc) Apply(config *map[string]any) {
	f(config)
}

// WithEmbedder specifies an Embedder to use for an operation (e.g., AddDocuments).
type WithEmbedderOption struct {
	Embedder Embedder
}

func (o WithEmbedderOption) Apply(config *map[string]any) {
	(*config)["embedder"] = o.Embedder
}
func WithEmbedder(embedder Embedder) core.Option {
	return WithEmbedderOption{Embedder: embedder}
}

// WithScoreThreshold sets a minimum similarity score threshold for retrieved documents.
type WithScoreThresholdOption struct {
	Threshold float32
}

func (o WithScoreThresholdOption) Apply(config *map[string]any) {
	(*config)["score_threshold"] = o.Threshold
}
func WithScoreThreshold(threshold float32) core.Option {
	return WithScoreThresholdOption{Threshold: threshold}
}

// WithMetadataFilter applies a filter based on document metadata.
// The exact filter format depends on the VectorStore implementation.
type WithMetadataFilterOption struct {
	Filter map[string]any
}

func (o WithMetadataFilterOption) Apply(config *map[string]any) {
	(*config)["metadata_filter"] = o.Filter
}
func WithMetadataFilter(filter map[string]any) core.Option {
	return WithMetadataFilterOption{Filter: filter}
}

// TODO:
// - Implement specific Loaders (e.g., FileLoader, WebLoader, DirectoryLoader) in rag/loaders/
// - Implement specific Splitters (e.g., CharacterSplitter, RecursiveCharacterSplitter, TokenSplitter) in rag/splitters/
// - Implement specific Embedders (e.g., OpenAIEmbedder, OllamaEmbedder, BedrockEmbedder adapters) in rag/embedders/
// - Implement specific VectorStores (e.g., InMemoryVectorStore, adapters for PgVector, Chroma, Pinecone, Weaviate, etc.) in rag/vectorstores/
// - Implement specific Retrievers (e.g., VectorStoreRetriever, MultiQueryRetriever, ContextualCompressionRetriever) in rag/retrievers/
