package rag

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// Pipeline defines the simplified interface for RAG pipelines.
// It provides a streamlined API for document ingestion, retrieval, and query answering.
type Pipeline interface {
	// Query executes a RAG query and returns the generated answer.
	// This requires an LLM to be configured for generation.
	Query(ctx context.Context, query string) (string, error)

	// QueryWithSources executes a RAG query and returns the answer along with source documents.
	// This requires an LLM to be configured for generation.
	QueryWithSources(ctx context.Context, query string) (string, []schema.Document, error)

	// IngestDocuments loads and processes documents from the configured paths.
	// Documents are split into chunks and stored in the vector store.
	IngestDocuments(ctx context.Context) error

	// IngestFromPaths loads and processes documents from the specified paths.
	// This allows dynamic document ingestion beyond the initially configured paths.
	IngestFromPaths(ctx context.Context, paths []string) error

	// AddDocuments directly adds pre-loaded documents to the vector store.
	// Documents will be split according to the configured splitter settings.
	AddDocuments(ctx context.Context, docs []schema.Document) error

	// AddDocumentsRaw adds documents without splitting.
	// Use this for pre-chunked documents or when no splitting is needed.
	AddDocumentsRaw(ctx context.Context, docs []schema.Document) error

	// Search performs similarity search and returns matching documents with scores.
	Search(ctx context.Context, query string, k int) ([]schema.Document, []float32, error)

	// GetDocumentCount returns the number of documents in the vector store.
	GetDocumentCount() int

	// Clear removes all documents from the vector store.
	Clear(ctx context.Context) error
}

// QueryResult contains the result of a RAG query.
type QueryResult struct {
	// Answer is the generated response to the query.
	Answer string

	// Sources are the documents used to generate the answer.
	Sources []schema.Document

	// Scores are the similarity scores of the source documents.
	Scores []float32
}
