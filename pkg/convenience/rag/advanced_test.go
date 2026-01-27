package rag

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// TestPipeline_AddDocuments tests adding documents to the pipeline.
func TestPipeline_AddDocuments(t *testing.T) {
	tests := []struct {
		name        string
		docs        []schema.Document
		expectCount int
		expectErr   bool
	}{
		{
			name: "add single document",
			docs: []schema.Document{
				schema.NewDocument("Test content", nil),
			},
			expectCount: 1,
			expectErr:   false,
		},
		{
			name: "add multiple documents",
			docs: []schema.Document{
				schema.NewDocument("First document", nil),
				schema.NewDocument("Second document", nil),
				schema.NewDocument("Third document", nil),
			},
			expectCount: 3,
			expectErr:   false,
		},
		{
			name:        "add empty documents",
			docs:        []schema.Document{},
			expectCount: 0,
			expectErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEmbedder := NewMockEmbedder()
			pipeline, err := NewBuilder().
				WithEmbedder(mockEmbedder).
				Build(context.Background())
			if err != nil {
				t.Fatalf("failed to build pipeline: %v", err)
			}

			err = pipeline.AddDocumentsRaw(context.Background(), tt.docs)

			if tt.expectErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			count := pipeline.GetDocumentCount()
			if count != tt.expectCount {
				t.Errorf("expected document count %d, got %d", tt.expectCount, count)
			}
		})
	}
}

// TestPipeline_Search tests the search functionality.
func TestPipeline_Search(t *testing.T) {
	mockEmbedder := NewMockEmbedder()
	pipeline, err := NewBuilder().
		WithEmbedder(mockEmbedder).
		WithTopK(3).
		Build(context.Background())
	if err != nil {
		t.Fatalf("failed to build pipeline: %v", err)
	}

	// Add test documents
	docs := []schema.Document{
		schema.NewDocument("Machine learning is great", nil),
		schema.NewDocument("Deep learning is powerful", nil),
		schema.NewDocument("Natural language processing", nil),
	}
	err = pipeline.AddDocumentsRaw(context.Background(), docs)
	if err != nil {
		t.Fatalf("failed to add documents: %v", err)
	}

	// Search
	results, scores, err := pipeline.Search(context.Background(), "machine learning", 2)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) > 2 {
		t.Errorf("expected at most 2 results, got %d", len(results))
	}
	if len(results) != len(scores) {
		t.Errorf("results and scores length mismatch: %d vs %d", len(results), len(scores))
	}
}

// TestPipeline_Query tests query functionality.
func TestPipeline_Query(t *testing.T) {
	t.Run("query without LLM fails", func(t *testing.T) {
		mockEmbedder := NewMockEmbedder()
		pipeline, err := NewBuilder().
			WithEmbedder(mockEmbedder).
			Build(context.Background())
		if err != nil {
			t.Fatalf("failed to build pipeline: %v", err)
		}

		_, err = pipeline.Query(context.Background(), "test query")
		if err == nil {
			t.Error("expected error for query without LLM")
		}

		code := GetErrorCode(err)
		if code != ErrCodeNoLLM {
			t.Errorf("expected error code %s, got %s", ErrCodeNoLLM, code)
		}
	})

	t.Run("successful query", func(t *testing.T) {
		mockEmbedder := NewMockEmbedder()
		mockLLM := NewMockChatModel()
		mockLLM.SetGenerateResponse("This is the answer based on the context.")

		pipeline, err := NewBuilder().
			WithEmbedder(mockEmbedder).
			WithLLM(mockLLM).
			Build(context.Background())
		if err != nil {
			t.Fatalf("failed to build pipeline: %v", err)
		}

		// Add test documents
		docs := []schema.Document{
			schema.NewDocument("Relevant information for the query", nil),
		}
		err = pipeline.AddDocumentsRaw(context.Background(), docs)
		if err != nil {
			t.Fatalf("failed to add documents: %v", err)
		}

		answer, err := pipeline.Query(context.Background(), "What is the information?")
		if err != nil {
			t.Fatalf("query failed: %v", err)
		}

		if answer == "" {
			t.Error("expected non-empty answer")
		}
	})
}

// TestPipeline_QueryWithSources tests query with source retrieval.
func TestPipeline_QueryWithSources(t *testing.T) {
	mockEmbedder := NewMockEmbedder()
	mockLLM := NewMockChatModel()
	mockLLM.SetGenerateResponse("Answer with sources")

	pipeline, err := NewBuilder().
		WithEmbedder(mockEmbedder).
		WithLLM(mockLLM).
		WithTopK(2).
		Build(context.Background())
	if err != nil {
		t.Fatalf("failed to build pipeline: %v", err)
	}

	// Add test documents
	docs := []schema.Document{
		schema.NewDocument("Source document 1", nil),
		schema.NewDocument("Source document 2", nil),
	}
	err = pipeline.AddDocumentsRaw(context.Background(), docs)
	if err != nil {
		t.Fatalf("failed to add documents: %v", err)
	}

	answer, sources, err := pipeline.QueryWithSources(context.Background(), "test query")
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	if answer == "" {
		t.Error("expected non-empty answer")
	}
	if len(sources) == 0 {
		t.Error("expected sources to be returned")
	}
}

// TestPipeline_Clear tests clearing the pipeline.
func TestPipeline_Clear(t *testing.T) {
	mockEmbedder := NewMockEmbedder()
	pipeline, err := NewBuilder().
		WithEmbedder(mockEmbedder).
		Build(context.Background())
	if err != nil {
		t.Fatalf("failed to build pipeline: %v", err)
	}

	// Add documents
	docs := []schema.Document{
		schema.NewDocument("Test document", nil),
	}
	err = pipeline.AddDocumentsRaw(context.Background(), docs)
	if err != nil {
		t.Fatalf("failed to add documents: %v", err)
	}

	if pipeline.GetDocumentCount() != 1 {
		t.Errorf("expected document count 1, got %d", pipeline.GetDocumentCount())
	}

	// Clear
	err = pipeline.Clear(context.Background())
	if err != nil {
		t.Fatalf("clear failed: %v", err)
	}

	if pipeline.GetDocumentCount() != 0 {
		t.Errorf("expected document count 0 after clear, got %d", pipeline.GetDocumentCount())
	}
}

// TestPipeline_ScoreThreshold tests score threshold filtering.
func TestPipeline_ScoreThreshold(t *testing.T) {
	mockEmbedder := NewMockEmbedder()
	pipeline, err := NewBuilder().
		WithEmbedder(mockEmbedder).
		WithScoreThreshold(0.99). // Very high threshold
		Build(context.Background())
	if err != nil {
		t.Fatalf("failed to build pipeline: %v", err)
	}

	// Add test documents
	docs := []schema.Document{
		schema.NewDocument("Test document", nil),
	}
	err = pipeline.AddDocumentsRaw(context.Background(), docs)
	if err != nil {
		t.Fatalf("failed to add documents: %v", err)
	}

	// Search with high threshold should return fewer results
	results, _, err := pipeline.Search(context.Background(), "test", 10)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	// With a very high threshold, we may get no results
	t.Logf("Results with high threshold: %d", len(results))
}

// TestPipeline_Concurrent tests concurrent access to the pipeline.
func TestPipeline_Concurrent(t *testing.T) {
	mockEmbedder := NewMockEmbedder()
	pipeline, err := NewBuilder().
		WithEmbedder(mockEmbedder).
		Build(context.Background())
	if err != nil {
		t.Fatalf("failed to build pipeline: %v", err)
	}

	// Add initial documents
	docs := []schema.Document{
		schema.NewDocument("Initial document", nil),
	}
	err = pipeline.AddDocumentsRaw(context.Background(), docs)
	if err != nil {
		t.Fatalf("failed to add documents: %v", err)
	}

	const numGoroutines = 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*2)

	// Concurrent searches
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, _, err := pipeline.Search(context.Background(), fmt.Sprintf("query %d", id), 5)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	// Concurrent document additions
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			doc := schema.NewDocument(fmt.Sprintf("Concurrent doc %d", id), nil)
			err := pipeline.AddDocumentsRaw(context.Background(), []schema.Document{doc})
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent operation failed: %v", err)
	}
}

// TestPipeline_IngestFromPaths tests the not-yet-implemented path ingestion.
func TestPipeline_IngestFromPaths(t *testing.T) {
	mockEmbedder := NewMockEmbedder()
	pipeline, err := NewBuilder().
		WithEmbedder(mockEmbedder).
		Build(context.Background())
	if err != nil {
		t.Fatalf("failed to build pipeline: %v", err)
	}

	err = pipeline.IngestFromPaths(context.Background(), []string{"./docs"})
	if err == nil {
		t.Error("expected error for unimplemented path ingestion")
	}

	code := GetErrorCode(err)
	if code != ErrCodeDocumentLoad {
		t.Errorf("expected error code %s, got %s", ErrCodeDocumentLoad, code)
	}
}

// TestPipeline_IngestDocuments tests the document ingestion with no paths.
func TestPipeline_IngestDocuments(t *testing.T) {
	mockEmbedder := NewMockEmbedder()
	pipeline, err := NewBuilder().
		WithEmbedder(mockEmbedder).
		Build(context.Background())
	if err != nil {
		t.Fatalf("failed to build pipeline: %v", err)
	}

	// Should not error when no paths configured
	err = pipeline.IngestDocuments(context.Background())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
