package ollama

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/internal/mock"
	"go.opentelemetry.io/otel"
)

func TestNewOllamaEmbedder(t *testing.T) {
	tracer := otel.Tracer("test")

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errCode string
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errCode: iface.ErrCodeInvalidConfig,
		},
		{
			name: "empty model",
			config: &Config{
				Model: "",
			},
			wantErr: true,
			errCode: iface.ErrCodeInvalidConfig,
		},
		{
			name: "valid config",
			config: &Config{
				Model: "nomic-embed-text",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			embedder, err := NewOllamaEmbedderWithClient(tt.config, tracer, &mock.OllamaClientMock{})
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOllamaEmbedderWithClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got nil")
					return
				}
				var embErr *iface.EmbeddingError
				if !iface.AsEmbeddingError(err, &embErr) {
					t.Errorf("Expected EmbeddingError, got %T", err)
					return
				}
				if embErr.Code != tt.errCode {
					t.Errorf("Expected error code %s, got %s", tt.errCode, embErr.Code)
				}
			} else {
				if embedder == nil {
					t.Error("NewOllamaEmbedderWithClient() returned nil embedder")
				}
			}
		})
	}
}

func TestNewOllamaEmbedderWithClient_NilClient(t *testing.T) {
	tracer := otel.Tracer("test")
	config := &Config{Model: "test-model"}

	_, err := NewOllamaEmbedderWithClient(config, tracer, nil)
	if err == nil {
		t.Error("Expected error for nil client")
	}

	var embErr *iface.EmbeddingError
	if !iface.AsEmbeddingError(err, &embErr) {
		t.Errorf("Expected EmbeddingError, got %T", err)
	}
	if embErr.Code != iface.ErrCodeConnectionFailed {
		t.Errorf("Expected error code %s, got %s", iface.ErrCodeConnectionFailed, embErr.Code)
	}
}

func TestOllamaEmbedder_EmbedDocuments(t *testing.T) {
	tracer := otel.Tracer("test")
	ctx := context.Background()

	tests := []struct {
		name            string
		documents       []string
		mockSetup       func(*mock.OllamaClientMock)
		wantErr         bool
		expectedCount   int
		expectedDim     int
	}{
		{
			name:      "empty documents",
			documents: []string{},
			mockSetup: func(m *mock.OllamaClientMock) {
				// No calls expected
			},
			wantErr:       false,
			expectedCount: 0,
		},
		{
			name:      "single document",
			documents: []string{"Hello world"},
			mockSetup: func(m *mock.OllamaClientMock) {
				embedding := make([]float64, 768)
				for i := range embedding {
					embedding[i] = float64(i) / 768.0
				}
				m.SetEmbeddingsResponse(embedding)
			},
			wantErr:       false,
			expectedCount: 1,
			expectedDim:   768,
		},
		{
			name:      "multiple documents",
			documents: []string{"First doc", "Second doc", "Third doc"},
			mockSetup: func(m *mock.OllamaClientMock) {
				embedding := make([]float64, 512)
				for i := range embedding {
					embedding[i] = float64(i) / 512.0
				}
				m.SetEmbeddingsResponse(embedding)
			},
			wantErr:       false,
			expectedCount: 3,
			expectedDim:   512,
		},
		{
			name:      "api error on first document",
			documents: []string{"First", "Second"},
			mockSetup: func(m *mock.OllamaClientMock) {
				m.SetEmbeddingsError(errors.New("API error"))
			},
			wantErr:       true,
			expectedCount: 2, // Should return partial results
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mock.NewOllamaClientMock()
			tt.mockSetup(mockClient)

			config := &Config{Model: "test-model"}
			embedder, err := NewOllamaEmbedderWithClient(config, tracer, mockClient)
			if err != nil {
				t.Fatalf("Failed to create embedder: %v", err)
			}

			embeddings, err := embedder.EmbedDocuments(ctx, tt.documents)
			if (err != nil) != tt.wantErr {
				t.Errorf("EmbedDocuments() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(embeddings) != tt.expectedCount {
				t.Errorf("Expected %d embeddings, got %d", tt.expectedCount, len(embeddings))
			}

			if !tt.wantErr && tt.expectedCount > 0 {
				for i, embedding := range embeddings {
					if embedding == nil {
						continue // Allow nil embeddings for error cases
					}
					if len(embedding) != tt.expectedDim {
						t.Errorf("Embedding %d: expected dimension %d, got %d", i, tt.expectedDim, len(embedding))
					}
				}
			}

			// Verify API calls
			if len(tt.documents) > 0 {
				if len(mockClient.EmbeddingsCalls) != len(tt.documents) {
					t.Errorf("Expected %d API calls, got %d", len(tt.documents), len(mockClient.EmbeddingsCalls))
				}

				for i, call := range mockClient.EmbeddingsCalls {
					if call.Req.Model != config.Model {
						t.Errorf("Call %d: expected model %s, got %s", i, config.Model, call.Req.Model)
					}
					if call.Req.Prompt != tt.documents[i] {
						t.Errorf("Call %d: expected prompt %q, got %q", i, tt.documents[i], call.Req.Prompt)
					}
				}
			}
		})
	}
}

func TestOllamaEmbedder_EmbedQuery(t *testing.T) {
	tracer := otel.Tracer("test")
	ctx := context.Background()

	tests := []struct {
		name        string
		query       string
		mockSetup   func(*mock.OllamaClientMock)
		wantErr     bool
		expectedDim int
		errCode     string
	}{
		{
			name:  "empty query",
			query: "",
			mockSetup: func(m *mock.OllamaClientMock) {
				// Should not be called
			},
			wantErr: true,
			errCode: iface.ErrCodeInvalidParameters,
		},
		{
			name:  "valid query",
			query: "What is machine learning?",
			mockSetup: func(m *mock.OllamaClientMock) {
				embedding := make([]float64, 384)
				for i := range embedding {
					embedding[i] = float64(i) / 384.0
				}
				m.SetEmbeddingsResponse(embedding)
			},
			wantErr:     false,
			expectedDim: 384,
		},
		{
			name:  "api error",
			query: "Test query",
			mockSetup: func(m *mock.OllamaClientMock) {
				m.SetEmbeddingsError(errors.New("API connection failed"))
			},
			wantErr: true,
			errCode: iface.ErrCodeEmbeddingFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mock.NewOllamaClientMock()
			tt.mockSetup(mockClient)

			config := &Config{Model: "test-model"}
			embedder, err := NewOllamaEmbedderWithClient(config, tracer, mockClient)
			if err != nil {
				t.Fatalf("Failed to create embedder: %v", err)
			}

			embedding, err := embedder.EmbedQuery(ctx, tt.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("EmbedQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got nil")
					return
				}
				var embErr *iface.EmbeddingError
				if iface.AsEmbeddingError(err, &embErr) && embErr.Code != tt.errCode {
					t.Errorf("Expected error code %s, got %s", tt.errCode, embErr.Code)
				}
			} else {
				if len(embedding) != tt.expectedDim {
					t.Errorf("Expected dimension %d, got %d", tt.expectedDim, len(embedding))
				}
				// Verify API call
				if len(mockClient.EmbeddingsCalls) != 1 {
					t.Errorf("Expected 1 API call, got %d", len(mockClient.EmbeddingsCalls))
				} else {
					call := mockClient.EmbeddingsCalls[0]
					if call.Req.Model != config.Model {
						t.Errorf("Expected model %s, got %s", config.Model, call.Req.Model)
					}
					if call.Req.Prompt != tt.query {
						t.Errorf("Expected prompt %q, got %q", tt.query, call.Req.Prompt)
					}
				}
			}
		})
	}
}

func TestOllamaEmbedder_GetDimension(t *testing.T) {
	tracer := otel.Tracer("test")
	ctx := context.Background()

	config := &Config{Model: "test-model"}
	mockClient := mock.NewOllamaClientMock()
	embedder, err := NewOllamaEmbedderWithClient(config, tracer, mockClient)
	if err != nil {
		t.Fatalf("Failed to create embedder: %v", err)
	}

	dimension, err := embedder.GetDimension(ctx)
	if err != nil {
		t.Fatalf("GetDimension() failed: %v", err)
	}

	// Ollama returns 0 for unknown dimension
	if dimension != 0 {
		t.Errorf("Expected dimension 0 (unknown), got %d", dimension)
	}
}

func TestOllamaEmbedder_Check(t *testing.T) {
	tracer := otel.Tracer("test")
	ctx := context.Background()

	tests := []struct {
		name      string
		mockSetup func(*mock.OllamaClientMock)
		wantErr   bool
	}{
		{
			name: "successful health check",
			mockSetup: func(m *mock.OllamaClientMock) {
				embedding := []float64{0.1, 0.2, 0.3}
				m.SetEmbeddingsResponse(embedding)
			},
			wantErr: false,
		},
		{
			name: "failed health check",
			mockSetup: func(m *mock.OllamaClientMock) {
				m.SetEmbeddingsError(errors.New("health check failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mock.NewOllamaClientMock()
			tt.mockSetup(mockClient)

			config := &Config{Model: "test-model"}
			embedder, err := NewOllamaEmbedderWithClient(config, tracer, mockClient)
			if err != nil {
				t.Fatalf("Failed to create embedder: %v", err)
			}

			err = embedder.Check(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Check() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOllamaEmbedder_InterfaceCompliance(t *testing.T) {
	tracer := otel.Tracer("test")
	config := &Config{Model: "test-model"}
	mockClient := mock.NewOllamaClientMock()

	embedder, err := NewOllamaEmbedderWithClient(config, tracer, mockClient)
	if err != nil {
		t.Fatalf("Failed to create embedder: %v", err)
	}

	// Test interface compliance
	var _ iface.Embedder = embedder
	var _ HealthChecker = embedder
}

func TestOllamaEmbedder_ContextCancellation(t *testing.T) {
	tracer := otel.Tracer("test")

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	mockClient := mock.NewOllamaClientMock()
	mockClient.SetEmbeddingsError(context.Canceled) // Simulate cancellation

	config := &Config{Model: "test-model"}
	embedder, err := NewOllamaEmbedderWithClient(config, tracer, mockClient)
	if err != nil {
		t.Fatalf("Failed to create embedder: %v", err)
	}

	_, err = embedder.EmbedQuery(ctx, "test query")
	if err == nil {
		t.Error("Expected error due to context cancellation")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

func TestOllamaEmbedder_ConcurrentAccess(t *testing.T) {
	tracer := otel.Tracer("test")
	ctx := context.Background()

	mockClient := mock.NewOllamaClientMock()
	embedding := make([]float64, 256)
	for i := range embedding {
		embedding[i] = float64(i) / 256.0
	}
	mockClient.SetEmbeddingsResponse(embedding)

	config := &Config{Model: "test-model"}
	embedder, err := NewOllamaEmbedderWithClient(config, tracer, mockClient)
	if err != nil {
		t.Fatalf("Failed to create embedder: %v", err)
	}

	// Test concurrent access
	done := make(chan bool, 2)

	go func() {
		_, err := embedder.EmbedQuery(ctx, "query 1")
		if err != nil {
			t.Errorf("Concurrent query 1 failed: %v", err)
		}
		done <- true
	}()

	go func() {
		_, err := embedder.EmbedQuery(ctx, "query 2")
		if err != nil {
			t.Errorf("Concurrent query 2 failed: %v", err)
		}
		done <- true
	}()

	// Wait for both goroutines
	for i := 0; i < 2; i++ {
		<-done
	}
}

// Benchmark tests
func BenchmarkOllamaEmbedder_EmbedQuery(b *testing.B) {
	tracer := otel.Tracer("benchmark")
	ctx := context.Background()

	mockClient := mock.NewOllamaClientMock()
	embedding := make([]float64, 768)
	for i := range embedding {
		embedding[i] = float64(i) / 768.0
	}
	mockClient.SetEmbeddingsResponse(embedding)

	config := &Config{Model: "test-model"}
	embedder, err := NewOllamaEmbedderWithClient(config, tracer, mockClient)
	if err != nil {
		b.Fatalf("Failed to create embedder: %v", err)
	}

	query := "This is a benchmark query for embedding performance"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = embedder.EmbedQuery(ctx, query)
	}
}

func BenchmarkOllamaEmbedder_EmbedDocuments(b *testing.B) {
	tracer := otel.Tracer("benchmark")
	ctx := context.Background()

	mockClient := mock.NewOllamaClientMock()
	embedding := make([]float64, 512)
	for i := range embedding {
		embedding[i] = float64(i) / 512.0
	}
	mockClient.SetEmbeddingsResponse(embedding)

	config := &Config{Model: "test-model"}
	embedder, err := NewOllamaEmbedderWithClient(config, tracer, mockClient)
	if err != nil {
		b.Fatalf("Failed to create embedder: %v", err)
	}

	documents := []string{
		"This is document one",
		"This is document two",
		"This is document three",
		"This is document four",
		"This is document five",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = embedder.EmbedDocuments(ctx, documents)
	}
}
