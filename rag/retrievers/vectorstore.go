// Package retrievers provides implementations of the rag.Retriever interface.
package retrievers

import (
	"context"
	"errors" // Added missing import
	"fmt"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/rag"
	"github.com/lookatitude/beluga-ai/schema"
)

// VectorStoreRetriever implements the rag.Retriever interface using an underlying rag.VectorStore.
type VectorStoreRetriever struct {
	vectorStore rag.VectorStore // The vector store to retrieve from
	options     []core.Option   // Default options for search (e.g., k, score_threshold, filter)
}

// NewVectorStoreRetriever creates a new VectorStoreRetriever.
// Options provided here become the default search options, but can be overridden
// during Invoke/GetRelevantDocuments calls.
func NewVectorStoreRetriever(vectorStore rag.VectorStore, options ...core.Option) *VectorStoreRetriever {
	return &VectorStoreRetriever{
		vectorStore: vectorStore,
		options:     options,
	}
}

// GetRelevantDocuments retrieves documents from the vector store based on the query.
func (r *VectorStoreRetriever) GetRelevantDocuments(ctx context.Context, query string, options ...core.Option) ([]schema.Document, error) {
	// Combine default options with call-specific options
	// Call-specific options take precedence
	combinedOptions := make(map[string]any)
	for _, opt := range r.options {
		opt.Apply(&combinedOptions)
	}
	for _, opt := range options {
		opt.Apply(&combinedOptions)
	}

	// Extract common search parameters
	k := 4 // Default k value
	if kOpt, ok := combinedOptions["k"].(int); ok && kOpt > 0 {
		k = kOpt
	}

	// Convert map back to []core.Option for passing to the vector store
	finalOptions := make([]core.Option, 0, len(combinedOptions))
	for key, val := range combinedOptions {
		// This is a bit hacky, assumes options can be reconstructed this way.
		// A better approach might be to have the VectorStore accept the map directly,
		// or define specific option types for k, threshold, filter etc.
		switch key {
		case "score_threshold":
			if threshold, ok := val.(float32); ok {
				finalOptions = append(finalOptions, rag.WithScoreThreshold(threshold))
			}
		case "metadata_filter":
			if filter, ok := val.(map[string]any); ok {
				finalOptions = append(finalOptions, rag.WithMetadataFilter(filter))
			}
			// Add other known options here
		}
	}

	return r.vectorStore.SimilaritySearch(ctx, query, k, finalOptions...)
}

// --- core.Runnable Implementation ---

// Invoke implements the core.Runnable interface.
func (r *VectorStoreRetriever) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	query, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("invalid input type for VectorStoreRetriever: expected string, got %T", input)
	}
	return r.GetRelevantDocuments(ctx, query, options...)
}

// Batch implements the core.Runnable interface.
func (r *VectorStoreRetriever) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	// Basic batch implementation by calling Invoke sequentially.
	results := make([]any, len(inputs))
	var firstErr error
	for i, input := range inputs {
		output, err := r.Invoke(ctx, input, options...)
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("error processing batch item %d: %w", i, err)
		}
		results[i] = output
	}
	return results, firstErr
}

// Stream implements the core.Runnable interface.
// Streaming is not typically applicable to retrievers, so it returns an error.
func (r *VectorStoreRetriever) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	return nil, errors.New("streaming is not supported by VectorStoreRetriever")
}

// Compile-time check to ensure VectorStoreRetriever implements interfaces.
var _ rag.Retriever = (*VectorStoreRetriever)(nil)
var _ core.Runnable = (*VectorStoreRetriever)(nil)
