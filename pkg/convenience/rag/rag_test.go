package rag

import (
	"context"
	"errors"
	"testing"
)

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder()

	if builder.topK != 5 {
		t.Errorf("expected default topK 5, got %d", builder.topK)
	}
	if builder.chunkSize != 1000 {
		t.Errorf("expected default chunkSize 1000, got %d", builder.chunkSize)
	}
	if builder.overlap != 200 {
		t.Errorf("expected default overlap 200, got %d", builder.overlap)
	}
}

func TestBuilder_WithMethods(t *testing.T) {
	mockEmbedder := NewMockEmbedder()
	mockLLM := NewMockChatModel()

	builder := NewBuilder().
		WithDocumentSource("./docs", "md", "txt").
		WithTopK(10).
		WithChunkSize(500).
		WithOverlap(100).
		WithEmbedder(mockEmbedder).
		WithLLM(mockLLM).
		WithSystemPrompt("Custom prompt").
		WithReturnSources(false).
		WithScoreThreshold(0.5)

	if len(builder.docPaths) != 1 {
		t.Errorf("expected 1 doc path, got %d", len(builder.docPaths))
	}
	if builder.topK != 10 {
		t.Errorf("expected topK 10, got %d", builder.topK)
	}
	if builder.chunkSize != 500 {
		t.Errorf("expected chunkSize 500, got %d", builder.chunkSize)
	}
	if builder.overlap != 100 {
		t.Errorf("expected overlap 100, got %d", builder.overlap)
	}
	if builder.embedder != mockEmbedder {
		t.Error("expected embedder to be set")
	}
	if builder.llm != mockLLM {
		t.Error("expected LLM to be set")
	}
	if builder.systemPrompt != "Custom prompt" {
		t.Errorf("expected systemPrompt 'Custom prompt', got %s", builder.systemPrompt)
	}
	if builder.returnSources != false {
		t.Error("expected returnSources to be false")
	}
	if builder.scoreThreshold != 0.5 {
		t.Errorf("expected scoreThreshold 0.5, got %f", builder.scoreThreshold)
	}
}

func TestBuilder_WithDocumentSource_Multiple(t *testing.T) {
	builder := NewBuilder().
		WithDocumentSource("./docs1", "md").
		WithDocumentSource("./docs2", "txt", "html")

	if len(builder.docPaths) != 2 {
		t.Errorf("expected 2 doc paths, got %d", len(builder.docPaths))
	}
	if len(builder.extensions) != 3 {
		t.Errorf("expected 3 extensions, got %d", len(builder.extensions))
	}
}

func TestBuilder_Build_MissingEmbedder(t *testing.T) {
	builder := NewBuilder()

	_, err := builder.Build(context.Background())
	if err == nil {
		t.Error("expected error for missing embedder")
	}

	var ragErr *Error
	if !errors.As(err, &ragErr) {
		t.Fatal("expected *Error type")
	}
	if ragErr.Code != ErrCodeMissingEmbedder {
		t.Errorf("expected error code %s, got %s", ErrCodeMissingEmbedder, ragErr.Code)
	}
}

func TestBuilder_Build_WithEmbedder(t *testing.T) {
	mockEmbedder := NewMockEmbedder()

	pipeline, err := NewBuilder().
		WithEmbedder(mockEmbedder).
		Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pipeline == nil {
		t.Fatal("expected pipeline to be non-nil")
	}
}

func TestBuilder_Build_WithLLM(t *testing.T) {
	mockEmbedder := NewMockEmbedder()
	mockLLM := NewMockChatModel()

	pipeline, err := NewBuilder().
		WithEmbedder(mockEmbedder).
		WithLLM(mockLLM).
		Build(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pipeline == nil {
		t.Fatal("expected pipeline to be non-nil")
	}
}

func TestBuilder_Getters(t *testing.T) {
	builder := NewBuilder().
		WithDocumentSource("./docs", "md").
		WithTopK(15).
		WithChunkSize(800).
		WithOverlap(150)

	if len(builder.GetDocPaths()) != 1 {
		t.Errorf("expected 1 doc path, got %d", len(builder.GetDocPaths()))
	}
	if len(builder.GetExtensions()) != 1 {
		t.Errorf("expected 1 extension, got %d", len(builder.GetExtensions()))
	}
	if builder.GetTopK() != 15 {
		t.Errorf("expected topK 15, got %d", builder.GetTopK())
	}
	if builder.GetChunkSize() != 800 {
		t.Errorf("expected chunkSize 800, got %d", builder.GetChunkSize())
	}
	if builder.GetOverlap() != 150 {
		t.Errorf("expected overlap 150, got %d", builder.GetOverlap())
	}
}
