// Package retrievers provides implementations of the rag.Retriever interface.
package retrievers

import (
	"context"
	"errors"
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
// Options provided here become the default search options.
func NewVectorStoreRetriever(vectorStore rag.VectorStore, options ...core.Option) *VectorStoreRetriever {
	return &VectorStoreRetriever{
		vectorStore: vectorStore,
		options:     options,
	}
}

// getCombinedOptions merges the retriever's default options with call-specific options.
// Call-specific options take precedence.
func (r *VectorStoreRetriever) getCombinedOptions(callOptions ...core.Option) map[string]any {
	combined := make(map[string]any)
	// Apply retriever's default options first
	for _, opt := range r.options {
		opt.Apply(&combined)
	}
	// Then apply call-specific options, potentially overriding defaults
	for _, opt := range callOptions {
		opt.Apply(&combined)
	}
	return combined
}

// GetRelevantDocuments retrieves documents from the vector store based on the query.
// This method now adheres to the rag.Retriever interface and uses the retriever's default options.
// For call-specific options, use the Invoke method.
func (r *VectorStoreRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
	// Use the retriever's default options for this interface method.
	// The Invoke method will handle merging call-specific options.
	return r.getRelevantDocumentsWithOptions(ctx, query, r.options...)
}

// getRelevantDocumentsWithOptions is an internal helper that accepts options.
func (r *VectorStoreRetriever) getRelevantDocumentsWithOptions(ctx context.Context, query string, options ...core.Option) ([]schema.Document, error) {
	combinedOptionsMap := r.getCombinedOptions(options...)

	k := 4 // Default k value
	if kOpt, ok := combinedOptionsMap["k"].(int); ok && kOpt > 0 {
		k = kOpt
	}

	finalVSOptions := make([]core.Option, 0, len(combinedOptionsMap))
	for key, val := range combinedOptionsMap {
		switch key {
		case "score_threshold":
			if threshold, ok := val.(float32); ok {
				finalVSOptions = append(finalVSOptions, rag.WithScoreThreshold(threshold))
			}
		case "metadata_filter":
			if filter, ok := val.(map[string]any); ok {
				finalVSOptions = append(finalVSOptions, rag.WithMetadataFilter(filter))
			}
		// Note: 'k' is handled separately and passed directly to SimilaritySearch
		// Add other known options that translate to rag.VectorStore options here
		}
	}

	return r.vectorStore.SimilaritySearch(ctx, query, k, finalVSOptions...)
}


// --- core.Runnable Implementation ---

// Invoke implements the core.Runnable interface.
func (r *VectorStoreRetriever) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	query, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("invalid input type for VectorStoreRetriever: expected string, got %T", input)
	}
	// Invoke uses the internal helper that can take call-specific options
	return r.getRelevantDocumentsWithOptions(ctx, query, options...)
}

// Batch implements the core.Runnable interface.
func (r *VectorStoreRetriever) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
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

