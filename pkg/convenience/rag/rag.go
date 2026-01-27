// Package rag provides a simplified API for building RAG (Retrieval-Augmented Generation) pipelines.
// It reduces the boilerplate typically required to set up document loading, embedding,
// storage, and retrieval.
//
// Example usage:
//
//	pipeline, err := rag.NewBuilder().
//	    WithDocumentSource("./docs", "md", "txt").
//	    WithEmbedder(embedder).
//	    WithInMemoryVectorStore().
//	    WithLLM(llm).
//	    Build(ctx)
//
//	answer, sources, err := pipeline.QueryWithSources(ctx, "What is X?")
package rag

import (
	"context"

	embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/textsplitters"
	"github.com/lookatitude/beluga-ai/pkg/textsplitters/iface"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores/providers/inmemory"
)

// Document is an alias for schema.Document for convenience.
type Document = schema.Document

// NewDocument creates a new Document for convenience.
// This is a wrapper around schema.NewDocument.
func NewDocument(content string, metadata map[string]string) Document {
	return schema.NewDocument(content, metadata)
}

// Builder provides a fluent interface for constructing RAG pipelines.
type Builder struct {
	// Document source configuration
	docPaths   []string
	extensions []string

	// Chunking configuration
	topK      int
	chunkSize int
	overlap   int

	// Embedder configuration
	embedder         embeddingsiface.Embedder
	embedderProvider string
	embedderAPIKey   string

	// Vector store configuration
	vectorStore     *inmemory.InMemoryVectorStore
	vectorStoreType string

	// LLM configuration
	llm         llmsiface.ChatModel
	llmProvider string
	llmAPIKey   string
	llmModel    string

	// Query configuration
	systemPrompt   string
	returnSources  bool
	scoreThreshold float32

	// Metrics
	metrics *Metrics
}

// NewBuilder creates a new RAG pipeline builder with sensible defaults.
func NewBuilder() *Builder {
	return &Builder{
		topK:           5,
		chunkSize:      1000,
		overlap:        200,
		returnSources:  true,
		scoreThreshold: 0.0,
	}
}

// WithDocumentSource adds a document source directory with file extensions.
func (b *Builder) WithDocumentSource(path string, extensions ...string) *Builder {
	b.docPaths = append(b.docPaths, path)
	b.extensions = append(b.extensions, extensions...)
	return b
}

// WithTopK sets the number of documents to retrieve.
func (b *Builder) WithTopK(k int) *Builder {
	b.topK = k
	return b
}

// WithChunkSize sets the chunk size for text splitting.
func (b *Builder) WithChunkSize(size int) *Builder {
	b.chunkSize = size
	return b
}

// WithOverlap sets the overlap between chunks.
func (b *Builder) WithOverlap(overlap int) *Builder {
	b.overlap = overlap
	return b
}

// WithEmbedder sets a pre-configured embedder instance.
func (b *Builder) WithEmbedder(embedder embeddingsiface.Embedder) *Builder {
	b.embedder = embedder
	return b
}

// WithEmbedderProvider sets the embedder provider for automatic resolution.
func (b *Builder) WithEmbedderProvider(provider, apiKey string) *Builder {
	b.embedderProvider = provider
	b.embedderAPIKey = apiKey
	return b
}

// WithVectorStore sets a pre-configured vector store instance.
func (b *Builder) WithVectorStore(store *inmemory.InMemoryVectorStore) *Builder {
	b.vectorStore = store
	return b
}

// WithInMemoryVectorStore configures an in-memory vector store.
func (b *Builder) WithInMemoryVectorStore() *Builder {
	b.vectorStoreType = "inmemory"
	return b
}

// WithLLM sets a pre-configured LLM (ChatModel) instance.
func (b *Builder) WithLLM(llm llmsiface.ChatModel) *Builder {
	b.llm = llm
	return b
}

// WithLLMProvider sets the LLM provider for automatic resolution.
func (b *Builder) WithLLMProvider(provider, apiKey, model string) *Builder {
	b.llmProvider = provider
	b.llmAPIKey = apiKey
	b.llmModel = model
	return b
}

// WithSystemPrompt sets the system prompt for query generation.
func (b *Builder) WithSystemPrompt(prompt string) *Builder {
	b.systemPrompt = prompt
	return b
}

// WithReturnSources configures whether to return source documents with queries.
func (b *Builder) WithReturnSources(enabled bool) *Builder {
	b.returnSources = enabled
	return b
}

// WithScoreThreshold sets the minimum similarity score threshold for retrieval.
func (b *Builder) WithScoreThreshold(threshold float32) *Builder {
	b.scoreThreshold = threshold
	return b
}

// WithMetrics sets the metrics instance for the pipeline.
func (b *Builder) WithMetrics(metrics *Metrics) *Builder {
	b.metrics = metrics
	return b
}

// Build creates and returns a configured Pipeline instance.
// It validates the configuration and creates all necessary components.
//
// Returns an error if:
//   - No embedder is configured (either via WithEmbedder or WithEmbedderProvider)
//   - Component creation fails
func (b *Builder) Build(ctx context.Context) (Pipeline, error) {
	const op = "Build"

	// Get or create metrics
	metrics := b.metrics
	if metrics == nil {
		metrics = GetMetrics()
		if metrics == nil {
			metrics = NoOpMetrics()
		}
	}

	// Start tracing span
	ctx, span := metrics.StartBuildSpan(ctx)
	if span != nil {
		defer span.End()
	}

	// Validate embedder configuration
	if b.embedder == nil && b.embedderProvider == "" {
		metrics.RecordBuild(ctx, false)
		return nil, NewError(op, ErrCodeMissingEmbedder, ErrMissingEmbedder)
	}

	// Resolve embedder
	var embedder embeddingsiface.Embedder
	if b.embedder != nil {
		embedder = b.embedder
	} else if b.embedderProvider != "" {
		// Provider-based creation would go here
		metrics.RecordBuild(ctx, false)
		return nil, NewErrorWithMessage(op, ErrCodeEmbedderCreation,
			"provider-based embedder creation not yet implemented, use WithEmbedder", nil)
	}

	// Create or use vector store
	var vectorStore *inmemory.InMemoryVectorStore
	if b.vectorStore != nil {
		vectorStore = b.vectorStore
	} else {
		// Default to in-memory vector store
		vectorStore = inmemory.NewInMemoryVectorStore(embedder)
	}

	// Create text splitter
	splitter, err := textsplitters.NewRecursiveCharacterTextSplitter(
		textsplitters.WithRecursiveChunkSize(b.chunkSize),
		textsplitters.WithRecursiveChunkOverlap(b.overlap),
	)
	if err != nil {
		metrics.RecordBuild(ctx, false)
		return nil, NewError(op, ErrCodeSplitterCreation, err)
	}

	// Create the RAG pipeline
	pipeline := &ragPipeline{
		embedder:       embedder,
		vectorStore:    vectorStore,
		llm:            b.llm,
		splitter:       splitter,
		docPaths:       b.docPaths,
		extensions:     b.extensions,
		topK:           b.topK,
		systemPrompt:   b.systemPrompt,
		scoreThreshold: b.scoreThreshold,
		metrics:        metrics,
	}

	metrics.RecordBuild(ctx, true)
	return pipeline, nil
}

// Getter methods for builder configuration

// GetDocPaths returns the configured document paths.
func (b *Builder) GetDocPaths() []string {
	return b.docPaths
}

// GetExtensions returns the configured file extensions.
func (b *Builder) GetExtensions() []string {
	return b.extensions
}

// GetTopK returns the configured top-k value.
func (b *Builder) GetTopK() int {
	return b.topK
}

// GetChunkSize returns the configured chunk size.
func (b *Builder) GetChunkSize() int {
	return b.chunkSize
}

// GetOverlap returns the configured overlap.
func (b *Builder) GetOverlap() int {
	return b.overlap
}

// ragPipeline implements the Pipeline interface.
type ragPipeline struct {
	embedder       embeddingsiface.Embedder
	vectorStore    *inmemory.InMemoryVectorStore
	llm            llmsiface.ChatModel
	splitter       iface.TextSplitter
	docPaths       []string
	extensions     []string
	systemPrompt   string
	topK           int
	scoreThreshold float32
	documentCount  int
	metrics        *Metrics
}
