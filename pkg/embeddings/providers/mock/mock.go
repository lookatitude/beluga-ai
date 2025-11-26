package mock

import (
	"context"
	"math/rand"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Config holds configuration for mock embedder.
type Config struct {
	Dimension    int
	Seed         int64
	RandomizeNil bool
	Enabled      bool
}

// HealthChecker interface for health checks.
type HealthChecker interface {
	Check(ctx context.Context) error
}

// MockEmbedder is a mock implementation of the Embedder interface for testing.
type MockEmbedder struct {
	tracer trace.Tracer
	config *Config
	rng    *rand.Rand
	mu     sync.Mutex
}

// NewMockEmbedder creates a new MockEmbedder with the given configuration.
func NewMockEmbedder(config *Config, tracer trace.Tracer) (*MockEmbedder, error) {
	if config == nil {
		return nil, iface.NewEmbeddingError(iface.ErrCodeInvalidConfig, "config cannot be nil")
	}

	if config.Dimension <= 0 {
		return nil, iface.NewEmbeddingError(iface.ErrCodeInvalidConfig, "dimension must be positive")
	}

	src := rand.NewSource(config.Seed)
	rng := rand.New(src)

	return &MockEmbedder{
		config: config,
		tracer: tracer,
		rng:    rng,
	}, nil
}

// EmbedDocuments mocks embedding multiple documents.
func (m *MockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	ctx, span := m.tracer.Start(ctx, "mock.embed_documents",
		trace.WithAttributes(
			attribute.String("provider", "mock"),
			attribute.String("model", "mock"),
			attribute.Int("document_count", len(texts)),
			attribute.Int("dimension", m.config.Dimension),
			attribute.Bool("randomize_nil", m.config.RandomizeNil),
		))
	defer span.End()

	defer func() {
	}()

	m.mu.Lock()
	defer m.mu.Unlock()

	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		if text == "" && !m.config.RandomizeNil {
			embeddings[i] = make([]float32, m.config.Dimension) // Zero vector for empty string
		} else {
			embedding := make([]float32, m.config.Dimension)
			for j := 0; j < m.config.Dimension; j++ {
				embedding[j] = m.rng.Float32()
			}
			embeddings[i] = embedding
		}
	}

	span.SetAttributes(
		attribute.Int("output_dimension", m.config.Dimension),
	)

	return embeddings, nil
}

// EmbedQuery mocks embedding a single query.
func (m *MockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	ctx, span := m.tracer.Start(ctx, "mock.embed_query",
		trace.WithAttributes(
			attribute.String("provider", "mock"),
			attribute.String("model", "mock"),
			attribute.Int("query_length", len(text)),
			attribute.Int("dimension", m.config.Dimension),
			attribute.Bool("randomize_nil", m.config.RandomizeNil),
		))
	defer span.End()

	defer func() {
	}()

	m.mu.Lock()
	defer m.mu.Unlock()

	if text == "" && !m.config.RandomizeNil {
		span.SetAttributes(attribute.String("result_type", "zero_vector"))
		result := make([]float32, m.config.Dimension) // Zero vector

		return result, nil
	}

	embedding := make([]float32, m.config.Dimension)
	for i := 0; i < m.config.Dimension; i++ {
		embedding[i] = m.rng.Float32()
	}

	span.SetAttributes(
		attribute.String("result_type", "random_vector"),
		attribute.Int("output_dimension", m.config.Dimension),
	)

	return embedding, nil
}

// GetDimension returns the mock dimension.
func (m *MockEmbedder) GetDimension(ctx context.Context) (int, error) {
	_, span := m.tracer.Start(ctx, "mock.get_dimension",
		trace.WithAttributes(
			attribute.String("provider", "mock"),
			attribute.String("model", "mock"),
			attribute.Int("dimension", m.config.Dimension),
		))
	defer span.End()

	return m.config.Dimension, nil
}

// Check performs a health check on the mock embedder.
func (m *MockEmbedder) Check(ctx context.Context) error {
	_, span := m.tracer.Start(ctx, "mock.health_check")
	defer span.End()

	// Mock embedder is always healthy
	return nil
}

// Ensure MockEmbedder implements the interface.
var (
	_ iface.Embedder = (*MockEmbedder)(nil)
	_ HealthChecker  = (*MockEmbedder)(nil)
)
