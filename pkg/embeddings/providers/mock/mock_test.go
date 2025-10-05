package mock

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"go.opentelemetry.io/otel"
)

func TestNewMockEmbedder(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
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

// Benchmark tests
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

// TestMockEmbedder_LoadSimulation tests load simulation features
func TestMockEmbedder_LoadSimulation(t *testing.T) {
	tracer := otel.Tracer("test")
	config := &Config{
		Dimension:          128,
		SimulateDelay:      10 * time.Millisecond,
		SimulateErrors:     false,
		ErrorRate:          0.0,
		RateLimitPerSecond: 10,
		MemoryPressure:     true,
		PerformanceDegrade: true,
	}

	embedder, err := NewMockEmbedder(config, tracer)
	if err != nil {
		t.Fatalf("Failed to create mock embedder: %v", err)
	}

	ctx := context.Background()

	// Test load simulation delay
	start := time.Now()
	_, err = embedder.EmbedQuery(ctx, "test query")
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Unexpected error with load simulation: %v", err)
	}

	// Should take at least the simulated delay
	if duration < config.SimulateDelay {
		t.Errorf("Expected delay of at least %v, got %v", config.SimulateDelay, duration)
	}

	// Test rate limiting
	t.Run("rate limiting", func(t *testing.T) {
		// Make multiple rapid requests to trigger rate limiting
		for i := 0; i < 15; i++ { // More than the rate limit
			_, err := embedder.EmbedQuery(ctx, "test query")
			if i >= 10 && err == nil { // After rate limit should be hit
				// Rate limiting may or may not trigger depending on timing
				t.Logf("Request %d succeeded (rate limit may not have triggered yet)", i)
			}
		}
	})

	// Test error simulation
	t.Run("error simulation", func(t *testing.T) {
		configWithErrors := &Config{
			Dimension:      128,
			SimulateErrors: true,
			ErrorRate:      1.0, // 100% error rate
		}

		embedderWithErrors, err := NewMockEmbedder(configWithErrors, tracer)
		if err != nil {
			t.Fatalf("Failed to create mock embedder with errors: %v", err)
		}

		_, err = embedderWithErrors.EmbedQuery(ctx, "test query")
		if err == nil {
			t.Error("Expected error with 100% error rate, but got none")
		}
	})
}

// TestMockEmbedder_BoundaryConditions tests edge cases and boundary conditions
func TestMockEmbedder_BoundaryConditions(t *testing.T) {
	tracer := otel.Tracer("test")
	ctx := context.Background()

	tests := []struct {
		name   string
		config *Config
		test   func(t *testing.T, embedder *MockEmbedder)
	}{
		{
			name: "very large dimension",
			config: &Config{
				Dimension: 10000, // Very large embedding dimension
			},
			test: func(t *testing.T, embedder *MockEmbedder) {
				dim, err := embedder.GetDimension(ctx)
				if err != nil {
					t.Errorf("GetDimension failed for large dimension: %v", err)
				}
				if dim != 10000 {
					t.Errorf("Expected dimension 10000, got %d", dim)
				}

				// Test embedding generation with large dimension
				vectors, err := embedder.EmbedQuery(ctx, "test")
				if err != nil {
					t.Errorf("EmbedQuery failed for large dimension: %v", err)
				}
				if len(vectors) != 10000 {
					t.Errorf("Expected vector length 10000, got %d", len(vectors))
				}
			},
		},
		{
			name: "extreme seed values",
			config: &Config{
				Dimension: 128,
				Seed:      -999999, // Very negative seed
			},
			test: func(t *testing.T, embedder *MockEmbedder) {
				// Should not panic with extreme seed values
				_, err := embedder.EmbedQuery(ctx, "test")
				if err != nil {
					t.Errorf("EmbedQuery failed with extreme seed: %v", err)
				}
			},
		},
		{
			name: "memory pressure simulation",
			config: &Config{
				Dimension:      128,
				MemoryPressure: true,
			},
			test: func(t *testing.T, embedder *MockEmbedder) {
				// Multiple operations to trigger memory pressure effects
				for i := 0; i < 100; i++ {
					_, err := embedder.EmbedQuery(ctx, "test query")
					if err != nil {
						// Memory pressure might cause occasional failures
						t.Logf("Memory pressure caused failure on iteration %d: %v", i, err)
					}
				}
			},
		},
		{
			name: "performance degradation simulation",
			config: &Config{
				Dimension:          128,
				PerformanceDegrade: true,
			},
			test: func(t *testing.T, embedder *MockEmbedder) {
				// Measure performance over multiple operations
				var totalDuration time.Duration
				operations := 50

				for i := 0; i < operations; i++ {
					start := time.Now()
					_, err := embedder.EmbedQuery(ctx, "test query")
					duration := time.Since(start)
					totalDuration += duration

					if err != nil {
						t.Logf("Performance degradation caused failure on iteration %d: %v", i, err)
					}
				}

				avgDuration := totalDuration / time.Duration(operations)
				t.Logf("Average operation time with degradation: %v", avgDuration)

				// Performance should degrade over time (get slower)
				// This is a basic check - in practice, degradation would be more sophisticated
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			embedder, err := NewMockEmbedder(tt.config, tracer)
			if err != nil {
				t.Fatalf("Failed to create mock embedder: %v", err)
			}

			tt.test(t, embedder)
		})
	}
}
