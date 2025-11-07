// Package providers provides tests for retriever provider implementations.
package providers

import (
	"context"
	"fmt"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/retrievers"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestMockRetriever(t *testing.T) {
	// Create test documents
	docs := []schema.Document{
		{
			PageContent: "This is a test document about machine learning.",
			Metadata: map[string]string{
				"source": "test",
				"id":     "doc1",
			},
		},
		{
			PageContent: "Another document about artificial intelligence.",
			Metadata: map[string]string{
				"source": "test",
				"id":     "doc2",
			},
		},
		{
			PageContent: "This document discusses natural language processing.",
			Metadata: map[string]string{
				"source": "test",
				"id":     "doc3",
			},
		},
	}

	mockRetriever := NewMockRetriever("test-retriever", docs,
		WithDefaultK(2),
		WithScoreThreshold(0.0),
	)

	t.Run("GetRelevantDocuments", func(t *testing.T) {
		ctx := context.Background()
		result, err := mockRetriever.GetRelevantDocuments(ctx, "machine learning")

		if err != nil {
			t.Fatalf("GetRelevantDocuments() error = %v", err)
		}

		if len(result) > 2 { // Should not exceed defaultK
			t.Errorf("GetRelevantDocuments() returned %d documents, want at most 2", len(result))
		}

		// Check that all returned documents have similarity scores
		for _, doc := range result {
			if doc.Metadata == nil {
				t.Error("Document metadata is nil")
				continue
			}
			if _, exists := doc.Metadata["similarity_score"]; !exists {
				t.Error("Document missing similarity_score in metadata")
			}
		}
	})

	t.Run("Invoke", func(t *testing.T) {
		ctx := context.Background()
		result, err := mockRetriever.Invoke(ctx, "test query")

		if err != nil {
			t.Fatalf("Invoke() error = %v", err)
		}

		docs, ok := result.([]schema.Document)
		if !ok {
			t.Errorf("Invoke() returned %T, want []schema.Document", result)
		}

		if len(docs) > 2 {
			t.Errorf("Invoke() returned %d documents, want at most 2", len(docs))
		}
	})

	t.Run("Invoke with invalid input", func(t *testing.T) {
		ctx := context.Background()
		_, err := mockRetriever.Invoke(ctx, 123) // Invalid input type

		if err == nil {
			t.Error("Invoke() with invalid input should return error")
		}

		// Check if it's a RetrieverError
		if _, ok := err.(*retrievers.RetrieverError); !ok {
			t.Errorf("Invoke() error = %T, want *retrievers.RetrieverError", err)
		}
	})

	t.Run("Batch", func(t *testing.T) {
		ctx := context.Background()
		inputs := []any{"query1", "query2", "query3"}

		results, err := mockRetriever.Batch(ctx, inputs)
		if err != nil {
			t.Fatalf("Batch() error = %v", err)
		}

		if len(results) != len(inputs) {
			t.Errorf("Batch() returned %d results, want %d", len(results), len(inputs))
		}
	})

	t.Run("Stream", func(t *testing.T) {
		ctx := context.Background()
		_, err := mockRetriever.Stream(ctx, "test query")

		if err == nil {
			t.Error("Stream() should return error (not supported)")
		}
	})

	t.Run("AddDocument", func(t *testing.T) {
		newDoc := schema.Document{
			PageContent: "New test document",
			Metadata: map[string]string{
				"source": "test",
				"id":     "new_doc",
			},
		}

		initialCount := mockRetriever.Count()
		mockRetriever.AddDocument(newDoc)

		if mockRetriever.Count() != initialCount+1 {
			t.Errorf("AddDocument() count = %d, want %d", mockRetriever.Count(), initialCount+1)
		}
	})

	t.Run("Clear", func(t *testing.T) {
		mockRetriever.Clear()

		if mockRetriever.Count() != 0 {
			t.Errorf("Clear() count = %d, want 0", mockRetriever.Count())
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
}

func BenchmarkMockRetriever_GetRelevantDocuments(b *testing.B) {
	// Create a larger set of documents for benchmarking
	docs := make([]schema.Document, 100)
	for i := range docs {
		docs[i] = schema.Document{
			PageContent: fmt.Sprintf("This is test document number %d with some content.", i),
			Metadata: map[string]string{
				"id":    fmt.Sprintf("%d", i),
				"index": fmt.Sprintf("%d", i),
			},
		}
	}

	mockRetriever := NewMockRetriever("benchmark-retriever", docs,
		WithDefaultK(10),
	)

	ctx := context.Background()
	query := "test query"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := mockRetriever.GetRelevantDocuments(ctx, query)
		if err != nil {
			b.Fatalf("GetRelevantDocuments() error = %v", err)
		}
	}
}
