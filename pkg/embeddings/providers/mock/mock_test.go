package mock

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"go.opentelemetry.io/otel"
)

func TestNewMockEmbedder(t *testing.T) {
	tests := []struct {
		config  *Config
		name    string
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "valid config",
			config: &Config{
				Dimension: 128,
				Seed:      42,
			},
			wantErr: false,
		},
		{
			name: "zero dimension",
			config: &Config{
				Dimension: 0,
			},
			wantErr: true,
		},
		{
			name: "negative dimension",
			config: &Config{
				Dimension: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracer := otel.Tracer("test")
			embedder, err := NewMockEmbedder(tt.config, tracer)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewMockEmbedder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && embedder == nil {
				t.Error("NewMockEmbedder() returned nil embedder")
			}
		})
	}
}

func TestMockEmbedder_EmbedDocuments(t *testing.T) {
	config := &Config{
		Dimension: 64,
		Seed:      123,
	}
	tracer := otel.Tracer("test")
	embedder, err := NewMockEmbedder(config, tracer)
	if err != nil {
		t.Fatalf("Failed to create embedder: %v", err)
	}

	ctx := context.Background()
	documents := []string{"Hello world", "How are you?", ""}

	embeddings, err := embedder.EmbedDocuments(ctx, documents)
	if err != nil {
		t.Fatalf("EmbedDocuments failed: %v", err)
	}

	// Check dimensions
	if len(embeddings) != len(documents) {
		t.Errorf("Expected %d embeddings, got %d", len(documents), len(embeddings))
	}

	for i, embedding := range embeddings {
		if len(embedding) != config.Dimension {
			t.Errorf("Document %d: expected dimension %d, got %d", i, config.Dimension, len(embedding))
		}
	}

	// Test deterministic output with same seed
	embedder2, err := NewMockEmbedder(config, tracer)
	if err != nil {
		t.Fatalf("Failed to create second embedder: %v", err)
	}

	embeddings2, err := embedder2.EmbedDocuments(ctx, documents)
	if err != nil {
		t.Fatalf("Second EmbedDocuments failed: %v", err)
	}

	// Should produce identical results
	for i := range embeddings {
		for j := range embeddings[i] {
			if embeddings[i][j] != embeddings2[i][j] {
				t.Errorf("Embeddings differ at position [%d][%d]: %f != %f", i, j, embeddings[i][j], embeddings2[i][j])
			}
		}
	}
}

func TestMockEmbedder_EmbedQuery(t *testing.T) {
	config := &Config{
		Dimension:    32,
		RandomizeNil: true,
	}
	tracer := otel.Tracer("test")
	embedder, err := NewMockEmbedder(config, tracer)
	if err != nil {
		t.Fatalf("Failed to create embedder: %v", err)
	}

	ctx := context.Background()

	// Test normal query
	embedding, err := embedder.EmbedQuery(ctx, "Test query")
	if err != nil {
		t.Fatalf("EmbedQuery failed: %v", err)
	}

	if len(embedding) != config.Dimension {
		t.Errorf("Expected dimension %d, got %d", config.Dimension, len(embedding))
	}

	// Test empty query with RandomizeNil=true (should generate random embedding)
	emptyEmbedding, err := embedder.EmbedQuery(ctx, "")
	if err != nil {
		t.Fatalf("EmbedQuery with empty string failed: %v", err)
	}

	if len(emptyEmbedding) != config.Dimension {
		t.Errorf("Empty query: expected dimension %d, got %d", config.Dimension, len(emptyEmbedding))
	}

	// Test with RandomizeNil=false
	config.RandomizeNil = false
	embedder2, err := NewMockEmbedder(config, tracer)
	if err != nil {
		t.Fatalf("Failed to create embedder with RandomizeNil=false: %v", err)
	}

	zeroEmbedding, err := embedder2.EmbedQuery(ctx, "")
	if err != nil {
		t.Fatalf("EmbedQuery with empty string failed: %v", err)
	}

	// Should be all zeros
	for i, val := range zeroEmbedding {
		if val != 0.0 {
			t.Errorf("Zero embedding at position %d should be 0.0, got %f", i, val)
		}
	}
}

func TestMockEmbedder_GetDimension(t *testing.T) {
	config := &Config{
		Dimension: 256,
	}
	tracer := otel.Tracer("test")
	embedder, err := NewMockEmbedder(config, tracer)
	if err != nil {
		t.Fatalf("Failed to create embedder: %v", err)
	}

	ctx := context.Background()
	dimension, err := embedder.GetDimension(ctx)
	if err != nil {
		t.Fatalf("GetDimension failed: %v", err)
	}

	if dimension != config.Dimension {
		t.Errorf("Expected dimension %d, got %d", config.Dimension, dimension)
	}
}

func TestMockEmbedder_Check(t *testing.T) {
	config := &Config{
		Dimension: 128,
	}
	tracer := otel.Tracer("test")
	embedder, err := NewMockEmbedder(config, tracer)
	if err != nil {
		t.Fatalf("Failed to create embedder: %v", err)
	}

	ctx := context.Background()
	err = embedder.Check(ctx)
	if err != nil {
		t.Errorf("Check should always succeed for mock embedder, got error: %v", err)
	}
}

func TestMockEmbedder_InterfaceCompliance(t *testing.T) {
	config := &Config{
		Dimension: 128,
	}
	tracer := otel.Tracer("test")
	embedder, err := NewMockEmbedder(config, tracer)
	if err != nil {
		t.Fatalf("Failed to create embedder: %v", err)
	}

	// Test interface compliance
	var _ iface.Embedder = embedder
	var _ HealthChecker = embedder
}

// Benchmark tests.
func BenchmarkMockEmbedder_EmbedDocuments(b *testing.B) {
	config := &Config{
		Dimension: 128,
	}
	tracer := otel.Tracer("benchmark")
	embedder, err := NewMockEmbedder(config, tracer)
	if err != nil {
		b.Fatalf("Failed to create embedder: %v", err)
	}

	ctx := context.Background()
	documents := []string{"Hello world", "This is a test", "Benchmarking embeddings"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = embedder.EmbedDocuments(ctx, documents)
	}
}

func BenchmarkMockEmbedder_EmbedQuery(b *testing.B) {
	config := &Config{
		Dimension: 128,
	}
	tracer := otel.Tracer("benchmark")
	embedder, err := NewMockEmbedder(config, tracer)
	if err != nil {
		b.Fatalf("Failed to create embedder: %v", err)
	}

	ctx := context.Background()
	query := "This is a benchmark query"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = embedder.EmbedQuery(ctx, query)
	}
}
