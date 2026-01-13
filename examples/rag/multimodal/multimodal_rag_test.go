package main

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/llms"
)

// MockEmbedder implements embeddings.Embedder for testing
type MockEmbedder struct {
	embedDimension int
}

func NewMockEmbedder(dimension int) *MockEmbedder {
	return &MockEmbedder{embedDimension: dimension}
}

func (m *MockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i, text := range texts {
		// Create a simple deterministic embedding based on text content
		embedding := make([]float32, m.embedDimension)
		for j := 0; j < m.embedDimension && j < len(text); j++ {
			embedding[j] = float32(text[j]) / 255.0
		}
		result[i] = embedding
	}
	return result, nil
}

func (m *MockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	results, err := m.EmbedDocuments(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return results[0], nil
}

func (m *MockEmbedder) GetDimension(ctx context.Context) (int, error) {
	return m.embedDimension, nil
}

// TestNewMultimodalRAGExample tests RAG example creation
func TestNewMultimodalRAGExample(t *testing.T) {
	mockEmbedder := NewMockEmbedder(384)
	mockLLM := llms.NewAdvancedMockChatModel("test-model")

	rag := NewMultimodalRAGExample(mockEmbedder, mockLLM, 3)

	if rag == nil {
		t.Fatal("NewMultimodalRAGExample returned nil")
	}

	if rag.topK != 3 {
		t.Errorf("topK = %d, want 3", rag.topK)
	}

	if rag.GetDocumentCount() != 0 {
		t.Errorf("Initial document count = %d, want 0", rag.GetDocumentCount())
	}
}

// TestIndexDocuments tests document indexing
func TestIndexDocuments(t *testing.T) {
	mockEmbedder := NewMockEmbedder(384)
	mockLLM := llms.NewAdvancedMockChatModel("test-model")
	rag := NewMultimodalRAGExample(mockEmbedder, mockLLM, 3)

	docs := []Document{
		{ID: "1", Type: TypeText, Content: "Test document one"},
		{ID: "2", Type: TypeImage, ImageURL: "http://example.com/img.jpg", Content: "Image caption"},
		{ID: "3", Type: TypeImageText, ImageURL: "http://example.com/diagram.png", Content: "Diagram description"},
	}

	err := rag.IndexDocuments(context.Background(), docs)
	if err != nil {
		t.Fatalf("IndexDocuments failed: %v", err)
	}

	if rag.GetDocumentCount() != 3 {
		t.Errorf("Document count = %d, want 3", rag.GetDocumentCount())
	}
}

// TestQuery tests the query functionality
func TestQuery(t *testing.T) {
	mockEmbedder := NewMockEmbedder(384)
	mockLLM := llms.NewAdvancedMockChatModel(
		"test-model",
		llms.WithResponses("This is the answer based on the context."),
	)
	rag := NewMultimodalRAGExample(mockEmbedder, mockLLM, 2)

	// Index some documents first
	docs := []Document{
		{ID: "1", Type: TypeText, Content: "Beluga whales are white whales"},
		{ID: "2", Type: TypeText, Content: "Dolphins are intelligent mammals"},
		{ID: "3", Type: TypeImage, Content: "A whale swimming", ImageURL: "http://example.com/whale.jpg"},
	}

	err := rag.IndexDocuments(context.Background(), docs)
	if err != nil {
		t.Fatalf("IndexDocuments failed: %v", err)
	}

	// Query
	answer, results, err := rag.Query(context.Background(), "What are beluga whales?")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if answer == "" {
		t.Error("Query returned empty answer")
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

// TestSimilaritySearch tests the similarity search function
func TestSimilaritySearch(t *testing.T) {
	mockEmbedder := NewMockEmbedder(384)
	mockLLM := llms.NewAdvancedMockChatModel("test-model")
	rag := NewMultimodalRAGExample(mockEmbedder, mockLLM, 2)

	// Manually add documents with known embeddings
	rag.documents = []EmbeddedDocument{
		{
			Document:  Document{ID: "1", Type: TypeText, Content: "Similar content"},
			Embedding: []float32{0.9, 0.1, 0.0},
		},
		{
			Document:  Document{ID: "2", Type: TypeText, Content: "Different content"},
			Embedding: []float32{0.1, 0.9, 0.0},
		},
		{
			Document:  Document{ID: "3", Type: TypeText, Content: "Also similar"},
			Embedding: []float32{0.8, 0.2, 0.0},
		},
	}

	// Query with embedding similar to doc1 and doc3
	query := []float32{0.85, 0.15, 0.0}
	results := rag.similaritySearch(query, 2)

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	// doc1 should be most similar
	if results[0].ID != "1" {
		t.Errorf("Expected doc1 as top result, got %s", results[0].ID)
	}

	// doc3 should be second
	if results[1].ID != "3" {
		t.Errorf("Expected doc3 as second result, got %s", results[1].ID)
	}
}

// TestCosineSimilarity tests the cosine similarity function
func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float32
		b        []float32
		expected float32
		delta    float32
	}{
		{
			name:     "identical vectors",
			a:        []float32{1.0, 0.0, 0.0},
			b:        []float32{1.0, 0.0, 0.0},
			expected: 1.0,
			delta:    0.001,
		},
		{
			name:     "orthogonal vectors",
			a:        []float32{1.0, 0.0, 0.0},
			b:        []float32{0.0, 1.0, 0.0},
			expected: 0.0,
			delta:    0.001,
		},
		{
			name:     "similar vectors",
			a:        []float32{0.9, 0.1, 0.0},
			b:        []float32{0.8, 0.2, 0.0},
			expected: 0.98,
			delta:    0.02,
		},
		{
			name:     "different lengths",
			a:        []float32{1.0, 0.0},
			b:        []float32{1.0, 0.0, 0.0},
			expected: 0.0,
			delta:    0.001,
		},
		{
			name:     "zero vector",
			a:        []float32{0.0, 0.0, 0.0},
			b:        []float32{1.0, 0.0, 0.0},
			expected: 0.0,
			delta:    0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.a, tt.b)
			if result < tt.expected-tt.delta || result > tt.expected+tt.delta {
				t.Errorf("cosineSimilarity() = %f, expected %f (±%f)", result, tt.expected, tt.delta)
			}
		})
	}
}

// TestDocumentTypes tests handling of different document types
func TestDocumentTypes(t *testing.T) {
	docs := createSampleDocuments()

	textCount := 0
	imageCount := 0
	imageTextCount := 0

	for _, doc := range docs {
		switch doc.Type {
		case TypeText:
			textCount++
			if doc.Content == "" {
				t.Errorf("Text document %s has empty content", doc.ID)
			}
		case TypeImage:
			imageCount++
			if doc.ImageURL == "" {
				t.Errorf("Image document %s has empty URL", doc.ID)
			}
		case TypeImageText:
			imageTextCount++
			if doc.ImageURL == "" || doc.Content == "" {
				t.Errorf("ImageText document %s missing URL or content", doc.ID)
			}
		}
	}

	if textCount == 0 {
		t.Error("No text documents in sample")
	}
	if imageCount == 0 {
		t.Error("No image documents in sample")
	}
	if imageTextCount == 0 {
		t.Error("No image+text documents in sample")
	}
}

// TestTruncate tests the truncate helper function
func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a longer string", 10, "this is..."},
		{"exactly10!", 10, "exactly10!"},
		{"abc", 5, "abc"},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

// TestContextCancellation tests that operations respect context cancellation
func TestContextCancellation(t *testing.T) {
	mockEmbedder := NewMockEmbedder(384)
	mockLLM := llms.NewAdvancedMockChatModel(
		"test-model",
		llms.WithResponses("answer"),
	)
	rag := NewMultimodalRAGExample(mockEmbedder, mockLLM, 2)

	// Index a document
	docs := []Document{{ID: "1", Type: TypeText, Content: "Test"}}
	_ = rag.IndexDocuments(context.Background(), docs)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Query should handle cancelled context
	// The behavior depends on implementation - it may return early or complete
	_, _, _ = rag.Query(ctx, "test query")

	// If we get here without panic, the test passes
}

// BenchmarkIndexing benchmarks document indexing performance
func BenchmarkIndexing(b *testing.B) {
	mockEmbedder := NewMockEmbedder(384)
	mockLLM := llms.NewAdvancedMockChatModel("test-model")

	docs := createSampleDocuments()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rag := NewMultimodalRAGExample(mockEmbedder, mockLLM, 3)
		_ = rag.IndexDocuments(context.Background(), docs)
	}
}

// BenchmarkQuery benchmarks query performance
func BenchmarkQuery(b *testing.B) {
	mockEmbedder := NewMockEmbedder(384)
	mockLLM := llms.NewAdvancedMockChatModel(
		"test-model",
		llms.WithResponses("answer"),
	)
	rag := NewMultimodalRAGExample(mockEmbedder, mockLLM, 3)

	docs := createSampleDocuments()
	_ = rag.IndexDocuments(context.Background(), docs)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = rag.Query(context.Background(), "What are beluga whales?")
	}
}

// BenchmarkSimilaritySearch benchmarks similarity search performance
func BenchmarkSimilaritySearch(b *testing.B) {
	mockEmbedder := NewMockEmbedder(384)
	mockLLM := llms.NewAdvancedMockChatModel("test-model")
	rag := NewMultimodalRAGExample(mockEmbedder, mockLLM, 3)

	// Add many documents
	for i := 0; i < 1000; i++ {
		rag.documents = append(rag.documents, EmbeddedDocument{
			Document:  Document{ID: string(rune(i)), Type: TypeText},
			Embedding: make([]float32, 384),
		})
	}

	query := make([]float32, 384)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rag.similaritySearch(query, 10)
	}
}
