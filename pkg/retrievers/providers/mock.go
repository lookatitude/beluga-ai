// Package providers provides concrete implementations of retrievers.
package providers

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/retrievers"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// MockRetriever is a mock implementation of the core.Retriever interface for testing purposes.
// It generates random documents based on the query and can be configured to simulate different scenarios.
type MockRetriever struct {
	name           string
	documents      []schema.Document
	defaultK       int
	scoreThreshold float32
	logger         *slog.Logger
}

// NewMockRetriever creates a new MockRetriever with the specified configuration.
func NewMockRetriever(name string, documents []schema.Document, options ...MockOption) *MockRetriever {
	mr := &MockRetriever{
		name:           name,
		documents:      documents,
		defaultK:       4,
		scoreThreshold: 0.0,
		logger:         slog.Default(),
	}

	for _, option := range options {
		option(mr)
	}

	return mr
}

// MockOption is a functional option for configuring MockRetriever.
type MockOption func(*MockRetriever)

// WithDefaultK sets the default number of documents to return.
func WithDefaultK(k int) MockOption {
	return func(mr *MockRetriever) {
		mr.defaultK = k
	}
}

// WithScoreThreshold sets the minimum similarity score threshold.
func WithScoreThreshold(threshold float32) MockOption {
	return func(mr *MockRetriever) {
		mr.scoreThreshold = threshold
	}
}

// WithLogger sets the logger for the MockRetriever.
func WithLogger(logger *slog.Logger) MockOption {
	return func(mr *MockRetriever) {
		mr.logger = logger
	}
}

// GetRelevantDocuments retrieves documents from the mock collection.
// It randomly selects documents and assigns them similarity scores.
func (mr *MockRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
	mr.logger.Info("mock retriever retrieving documents",
		"query", query,
		"available_documents", len(mr.documents),
		"default_k", mr.defaultK,
	)

	if len(mr.documents) == 0 {
		return []schema.Document{}, nil
	}

	// Simulate some processing time
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(time.Duration(rand.Intn(50)) * time.Millisecond):
	}

	// Randomly select documents up to defaultK
	k := mr.defaultK
	if k > len(mr.documents) {
		k = len(mr.documents)
	}

	selectedDocs := make([]schema.Document, 0, k)
	usedIndices := make(map[int]bool)

	for len(selectedDocs) < k {
		idx := rand.Intn(len(mr.documents))
		if !usedIndices[idx] {
			usedIndices[idx] = true
			doc := mr.documents[idx]

			// Add a mock score to the document metadata
			score := rand.Float32()
			if doc.Metadata == nil {
				doc.Metadata = make(map[string]string)
			}
			doc.Metadata["similarity_score"] = fmt.Sprintf("%.6f", score)

			// Only include documents above the threshold
			if score >= mr.scoreThreshold {
				selectedDocs = append(selectedDocs, doc)
			}
		}
	}

	mr.logger.Info("mock retriever completed",
		"documents_returned", len(selectedDocs),
	)

	return selectedDocs, nil
}

// Invoke implements the core.Runnable interface.
func (mr *MockRetriever) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	query, ok := input.(string)
	if !ok {
		return nil, retrievers.NewRetrieverErrorWithMessage("MockRetriever.Invoke", nil, retrievers.ErrCodeInvalidInput,
			fmt.Sprintf("invalid input type for MockRetriever: expected string, got %T", input))
	}
	return mr.GetRelevantDocuments(ctx, query)
}

// Batch implements the core.Runnable interface.
func (mr *MockRetriever) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	var firstErr error
	for i, input := range inputs {
		output, err := mr.Invoke(ctx, input, options...)
		if err != nil && firstErr == nil {
			firstErr = retrievers.NewRetrieverErrorWithMessage("MockRetriever.Batch", err, retrievers.ErrCodeRetrievalFailed,
				fmt.Sprintf("error processing batch item %d", i))
		}
		results[i] = output
	}
	return results, firstErr
}

// Stream implements the core.Runnable interface.
// MockRetriever does not support streaming.
func (mr *MockRetriever) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	return nil, retrievers.NewRetrieverErrorWithMessage("MockRetriever.Stream", nil, retrievers.ErrCodeInvalidInput,
		"streaming is not supported by MockRetriever")
}

// AddDocument adds a document to the mock retriever's collection.
func (mr *MockRetriever) AddDocument(doc schema.Document) {
	mr.documents = append(mr.documents, doc)
	mr.logger.Info("added document to mock retriever",
		"total_documents", len(mr.documents),
	)
}

// Clear removes all documents from the mock retriever.
func (mr *MockRetriever) Clear() {
	mr.documents = nil
	mr.logger.Info("cleared all documents from mock retriever")
}

// Count returns the number of documents in the mock retriever.
func (mr *MockRetriever) Count() int {
	return len(mr.documents)
}

// Compile-time check to ensure MockRetriever implements interfaces.
var _ core.Retriever = (*MockRetriever)(nil)
var _ core.Runnable = (*MockRetriever)(nil)
