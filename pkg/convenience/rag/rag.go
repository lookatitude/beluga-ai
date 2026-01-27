// Package rag provides a simplified API for building RAG (Retrieval-Augmented Generation) pipelines.
// It reduces the boilerplate typically required to set up document loading, embedding,
// storage, and retrieval.
//
// Note: This package is a work in progress. For production use, please use the
// individual packages directly (embeddings, vectorstores, llms, documentloaders).
//
// Example intended usage (future):
//
//	pipeline, err := rag.NewBuilder().
//	    WithDocumentSource("./docs", "md", "txt").
//	    WithEmbedder("openai").
//	    WithVectorStore("memory").
//	    WithLLM("openai").
//	    Build(ctx)
//
//	answer, sources, err := pipeline.Query(ctx, "What is X?")
package rag

// Builder provides a fluent interface for constructing RAG pipelines.
// This is a placeholder for future implementation.
type Builder struct {
	docPaths   []string
	extensions []string
	topK       int
	chunkSize  int
	overlap    int
}

// NewBuilder creates a new RAG pipeline builder.
func NewBuilder() *Builder {
	return &Builder{
		topK:      5,
		chunkSize: 1000,
		overlap:   200,
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
