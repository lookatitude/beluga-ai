package openai

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/internal/mock"
	"github.com/sashabaranov/go-openai"
	"go.opentelemetry.io/otel"
)

func TestNewOpenAIEmbedder(t *testing.T) {
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
			name: "empty api key",
			config: &Config{
				APIKey: "",
				Model:  "text-embedding-ada-002",
			},
			wantErr: true,
			errCode: iface.ErrCodeInvalidConfig,
		},
		{
			name: "valid config",
			config: &Config{
				APIKey: "sk-test123",
				Model:  "text-embedding-ada-002",
			},
			wantErr: false,
		},
		{
			name: "config with base URL",
			config: &Config{
				APIKey:  "sk-test123",
				Model:   "text-embedding-ada-002",
				BaseURL: "https://custom.openai.com",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			embedder, err := NewOpenAIEmbedderWithClient(tt.config, tracer, &mock.OpenAIClientMock{})
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOpenAIEmbedderWithClient() error = %v, wantErr %v", err, tt.wantErr)
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
					t.Error("NewOpenAIEmbedderWithClient() returned nil embedder")
				}
			}
		})
	}
}

func TestNewOpenAIEmbedderWithClient_NilClient(t *testing.T) {
	tracer := otel.Tracer("test")
	config := &Config{APIKey: "sk-test", Model: "test-model"}

	_, err := NewOpenAIEmbedderWithClient(config, tracer, nil)
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

func TestOpenAIEmbedder_EmbedDocuments(t *testing.T) {
	tracer := otel.Tracer("test")
	ctx := context.Background()

	tests := []struct {
		name            string
		documents       []string
		mockSetup       func(*mock.OpenAIClientMock)
		wantErr         bool
		expectedCount   int
		expectedDim     int
	}{
		{
			name:      "empty documents",
			documents: []string{},
			mockSetup: func(m *mock.OpenAIClientMock) {
				response := openai.EmbeddingResponse{
					Object: "list",
					Data:   []openai.Embedding{},
					Model:  "text-embedding-ada-002",
				}
				m.SetCreateEmbeddingsResponse(response)
			},
			wantErr:       false,
			expectedCount: 0,
		},
		{
			name:      "single document",
			documents: []string{"Hello world"},
			mockSetup: func(m *mock.OpenAIClientMock) {
				embedding := make([]float32, 1536)
				for i := range embedding {
					embedding[i] = float32(i) / 1536.0
				}
				response := openai.EmbeddingResponse{
					Object: "list",
					Data: []openai.Embedding{
						{
							Object:    "embedding",
							Embedding: embedding,
							Index:     0,
						},
					},
					Model: "text-embedding-ada-002",
				}
				m.SetCreateEmbeddingsResponse(response)
			},
			wantErr:       false,
			expectedCount: 1,
			expectedDim:   1536,
		},
		{
			name:      "multiple documents",
			documents: []string{"First doc", "Second doc", "Third doc"},
			mockSetup: func(m *mock.OpenAIClientMock) {
				data := make([]openai.Embedding, 3)
				for i := 0; i < 3; i++ {
					embedding := make([]float32, 1536)
					for j := range embedding {
						embedding[j] = float32(i+j) / 1536.0
					}
					data[i] = openai.Embedding{
						Object:    "embedding",
						Embedding: embedding,
						Index:     i,
					}
				}
				response := openai.EmbeddingResponse{
					Object: "list",
					Data:   data,
					Model:  "text-embedding-ada-002",
				}
				m.SetCreateEmbeddingsResponse(response)
			},
			wantErr:       false,
			expectedCount: 3,
			expectedDim:   1536,
		},
		{
			name:      "api error",
			documents: []string{"Test document"},
			mockSetup: func(m *mock.OpenAIClientMock) {
				m.SetCreateEmbeddingsError(errors.New("API rate limit exceeded"))
			},
			wantErr: true,
		},
		{
			name:      "response count mismatch",
			documents: []string{"Doc 1", "Doc 2"},
			mockSetup: func(m *mock.OpenAIClientMock) {
				// Return only 1 embedding for 2 documents
				embedding := make([]float32, 1536)
				response := openai.EmbeddingResponse{
					Object: "list",
					Data: []openai.Embedding{
						{
							Object:    "embedding",
							Embedding: embedding,
							Index:     0,
						},
					},
					Model: "text-embedding-ada-002",
				}
				m.SetCreateEmbeddingsResponse(response)
			},
			wantErr: true,
		},
		{
			name:      "index mismatch",
			documents: []string{"Doc 1"},
			mockSetup: func(m *mock.OpenAIClientMock) {
				embedding := make([]float32, 1536)
				response := openai.EmbeddingResponse{
					Object: "list",
					Data: []openai.Embedding{
						{
							Object:    "embedding",
							Embedding: embedding,
							Index:     5, // Wrong index
						},
					},
					Model: "text-embedding-ada-002",
				}
				m.SetCreateEmbeddingsResponse(response)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mock.NewOpenAIClientMock()
			tt.mockSetup(mockClient)

			config := &Config{APIKey: "sk-test", Model: "text-embedding-ada-002"}
			embedder, err := NewOpenAIEmbedderWithClient(config, tracer, mockClient)
			if err != nil {
				t.Fatalf("Failed to create embedder: %v", err)
			}

			embeddings, err := embedder.EmbedDocuments(ctx, tt.documents)
			if (err != nil) != tt.wantErr {
				t.Errorf("EmbedDocuments() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(embeddings) != tt.expectedCount {
					t.Errorf("Expected %d embeddings, got %d", tt.expectedCount, len(embeddings))
				}

				if tt.expectedCount > 0 {
					for i, embedding := range embeddings {
						if len(embedding) != tt.expectedDim {
							t.Errorf("Embedding %d: expected dimension %d, got %d", i, tt.expectedDim, len(embedding))
						}
					}
				}

				// Verify API call was made
				if len(tt.documents) > 0 {
					if len(mockClient.CreateEmbeddingsCalls) != 1 {
						t.Errorf("Expected 1 API call, got %d", len(mockClient.CreateEmbeddingsCalls))
					}
				}
			}
		})
	}
}

func TestOpenAIEmbedder_EmbedQuery(t *testing.T) {
	tracer := otel.Tracer("test")
	ctx := context.Background()

	tests := []struct {
		name        string
		query       string
		mockSetup   func(*mock.OpenAIClientMock)
		wantErr     bool
		expectedDim int
		errCode     string
	}{
		{
			name:  "empty query",
			query: "",
			mockSetup: func(m *mock.OpenAIClientMock) {
				// Should not be called
			},
			wantErr: true,
			errCode: iface.ErrCodeInvalidParameters,
		},
		{
			name:  "valid query",
			query: "What is machine learning?",
			mockSetup: func(m *mock.OpenAIClientMock) {
				embedding := make([]float32, 3072) // 3-large dimensions
				for i := range embedding {
					embedding[i] = float32(i) / 3072.0
				}
				response := openai.EmbeddingResponse{
					Object: "list",
					Data: []openai.Embedding{
						{
							Object:    "embedding",
							Embedding: embedding,
							Index:     0,
						},
					},
					Model: "text-embedding-3-large",
				}
				m.SetCreateEmbeddingsResponse(response)
			},
			wantErr:     false,
			expectedDim: 3072,
		},
		{
			name:  "api error",
			query: "Test query",
			mockSetup: func(m *mock.OpenAIClientMock) {
				m.SetCreateEmbeddingsError(errors.New("API authentication failed"))
			},
			wantErr: true,
			errCode: iface.ErrCodeEmbeddingFailed,
		},
		{
			name:  "wrong response count",
			query: "Test query",
			mockSetup: func(m *mock.OpenAIClientMock) {
				// Return 2 embeddings for 1 query
				embedding := make([]float32, 1536)
				response := openai.EmbeddingResponse{
					Object: "list",
					Data: []openai.Embedding{
						{Object: "embedding", Embedding: embedding, Index: 0},
						{Object: "embedding", Embedding: embedding, Index: 1},
					},
					Model: "text-embedding-ada-002",
				}
				m.SetCreateEmbeddingsResponse(response)
			},
			wantErr: true,
			errCode: iface.ErrCodeEmbeddingFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mock.NewOpenAIClientMock()
			tt.mockSetup(mockClient)

			config := &Config{APIKey: "sk-test", Model: "text-embedding-ada-002"}
			embedder, err := NewOpenAIEmbedderWithClient(config, tracer, mockClient)
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
				// Verify API call was made
				if len(mockClient.CreateEmbeddingsCalls) != 1 {
					t.Errorf("Expected 1 API call, got %d", len(mockClient.CreateEmbeddingsCalls))
				}
			}
		})
	}
}

func TestOpenAIEmbedder_GetDimension(t *testing.T) {
	tracer := otel.Tracer("test")
	ctx := context.Background()

	tests := []struct {
		name      string
		model     string
		expected  int
	}{
		{
			name:     "ada-002 model",
			model:    "text-embedding-ada-002",
			expected: 1536,
		},
		{
			name:     "3-small model",
			model:    "text-embedding-3-small",
			expected: 1536,
		},
		{
			name:     "3-large model",
			model:    "text-embedding-3-large",
			expected: 3072,
		},
		{
			name:     "unknown model",
			model:    "unknown-model",
			expected: 1536, // Defaults to ada-002
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mock.NewOpenAIClientMock()
			config := &Config{APIKey: "sk-test", Model: tt.model}
			embedder, err := NewOpenAIEmbedderWithClient(config, tracer, mockClient)
			if err != nil {
				t.Fatalf("Failed to create embedder: %v", err)
			}

			dimension, err := embedder.GetDimension(ctx)
			if err != nil {
				t.Fatalf("GetDimension() failed: %v", err)
			}

			if dimension != tt.expected {
				t.Errorf("Expected dimension %d, got %d", tt.expected, dimension)
			}
		})
	}
}

func TestOpenAIEmbedder_Check(t *testing.T) {
	tracer := otel.Tracer("test")
	ctx := context.Background()

	tests := []struct {
		name      string
		mockSetup func(*mock.OpenAIClientMock)
		wantErr   bool
	}{
		{
			name: "successful health check",
			mockSetup: func(m *mock.OpenAIClientMock) {
				embedding := make([]float32, 1536)
				response := openai.EmbeddingResponse{
					Object: "list",
					Data: []openai.Embedding{
						{Object: "embedding", Embedding: embedding, Index: 0},
					},
					Model: "text-embedding-ada-002",
				}
				m.SetCreateEmbeddingsResponse(response)
			},
			wantErr: false,
		},
		{
			name: "failed health check",
			mockSetup: func(m *mock.OpenAIClientMock) {
				m.SetCreateEmbeddingsError(errors.New("health check failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mock.NewOpenAIClientMock()
			tt.mockSetup(mockClient)

			config := &Config{APIKey: "sk-test", Model: "text-embedding-ada-002"}
			embedder, err := NewOpenAIEmbedderWithClient(config, tracer, mockClient)
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

func TestOpenAIEmbedder_InterfaceCompliance(t *testing.T) {
	tracer := otel.Tracer("test")
	config := &Config{APIKey: "sk-test", Model: "text-embedding-ada-002"}
	mockClient := mock.NewOpenAIClientMock()

	embedder, err := NewOpenAIEmbedderWithClient(config, tracer, mockClient)
	if err != nil {
		t.Fatalf("Failed to create embedder: %v", err)
	}

	// Test interface compliance
	var _ iface.Embedder = embedder
	var _ HealthChecker = embedder
}

func TestOpenAIEmbedder_ContextCancellation(t *testing.T) {
	tracer := otel.Tracer("test")

	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	mockClient := mock.NewOpenAIClientMock()
	mockClient.SetCreateEmbeddingsError(context.Canceled) // Simulate cancellation

	config := &Config{APIKey: "sk-test", Model: "text-embedding-ada-002"}
	embedder, err := NewOpenAIEmbedderWithClient(config, tracer, mockClient)
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

func TestOpenAIEmbedder_ConcurrentAccess(t *testing.T) {
	tracer := otel.Tracer("test")
	ctx := context.Background()

	// Test concurrent creation of multiple embedders
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- true }()

			mockClient := mock.NewOpenAIClientMock()
			// Set up mock with appropriate response for this embedder
			embedding := make([]float32, 1536)
			response := openai.EmbeddingResponse{
				Object: "list",
				Data: []openai.Embedding{
					{Object: "embedding", Embedding: embedding, Index: 0},
				},
				Model: "text-embedding-ada-002",
			}
			mockClient.SetCreateEmbeddingsResponse(response)

			config := &Config{APIKey: "sk-test", Model: "text-embedding-ada-002"}
			embedder, err := NewOpenAIEmbedderWithClient(config, tracer, mockClient)
			if err != nil {
				t.Errorf("Goroutine %d: failed to create embedder: %v", id, err)
				return
			}

			// Test operations on this embedder
			_, err = embedder.EmbedQuery(ctx, "query from goroutine "+string(rune(id+'0')))
			if err != nil {
				t.Errorf("Goroutine %d: failed to embed query: %v", id, err)
			}

			dimension, err := embedder.GetDimension(ctx)
			if err != nil {
				t.Errorf("Goroutine %d: failed to get dimension: %v", id, err)
			}
			if dimension != 1536 {
				t.Errorf("Goroutine %d: expected dimension 1536, got %d", id, dimension)
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}
}

// Benchmark tests
func BenchmarkOpenAIEmbedder_EmbedQuery(b *testing.B) {
	tracer := otel.Tracer("benchmark")
	ctx := context.Background()

	mockClient := mock.NewOpenAIClientMock()
	embedding := make([]float32, 1536)
	response := openai.EmbeddingResponse{
		Object: "list",
		Data: []openai.Embedding{
			{Object: "embedding", Embedding: embedding, Index: 0},
		},
		Model: "text-embedding-ada-002",
	}
	mockClient.SetCreateEmbeddingsResponse(response)

	config := &Config{APIKey: "sk-test", Model: "text-embedding-ada-002"}
	embedder, err := NewOpenAIEmbedderWithClient(config, tracer, mockClient)
	if err != nil {
		b.Fatalf("Failed to create embedder: %v", err)
	}

	query := "This is a benchmark query for embedding performance"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = embedder.EmbedQuery(ctx, query)
	}
}

func BenchmarkOpenAIEmbedder_EmbedDocuments(b *testing.B) {
	tracer := otel.Tracer("benchmark")
	ctx := context.Background()

	mockClient := mock.NewOpenAIClientMock()
	embeddings := make([]openai.Embedding, 5)
	for i := 0; i < 5; i++ {
		embedding := make([]float32, 1536)
		embeddings[i] = openai.Embedding{
			Object:    "embedding",
			Embedding: embedding,
			Index:     i,
		}
	}
	response := openai.EmbeddingResponse{
		Object: "list",
		Data:   embeddings,
		Model:  "text-embedding-ada-002",
	}
	mockClient.SetCreateEmbeddingsResponse(response)

	config := &Config{APIKey: "sk-test", Model: "text-embedding-ada-002"}
	embedder, err := NewOpenAIEmbedderWithClient(config, tracer, mockClient)
	if err != nil {
		b.Fatalf("Failed to create embedder: %v", err)
	}

	documents := []string{
		"This is document one for benchmarking",
		"This is document two for benchmarking",
		"This is document three for benchmarking",
		"This is document four for benchmarking",
		"This is document five for benchmarking",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = embedder.EmbedDocuments(ctx, documents)
	}
}
