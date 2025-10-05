package mock

import (
	"context"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Config holds configuration for mock embedder
type Config struct {
	Dimension    int
	Seed         int64
	RandomizeNil bool
	Enabled      bool
	// Load simulation settings
	SimulateDelay      time.Duration
	SimulateErrors     bool
	ErrorRate          float64 // 0.0 to 1.0
	RateLimitPerSecond int
	MemoryPressure     bool // Simulate memory pressure
	PerformanceDegrade bool // Gradually degrade performance
}

// HealthChecker interface for health checks
type HealthChecker interface {
	Check(ctx context.Context) error
}

// MockEmbedder is a mock implementation of the Embedder interface for testing.
type MockEmbedder struct {
	config *Config
	tracer trace.Tracer
	mu     sync.Mutex
	rng    *rand.Rand
	// Load simulation state
	requestCount    int64
	startTime       time.Time
	rateLimitTokens int64
	lastRefillTime  time.Time
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

	now := time.Now()
	mock := &MockEmbedder{
		config:          config,
		tracer:          tracer,
		rng:             rng,
		startTime:       now,
		lastRefillTime:  now,
		rateLimitTokens: int64(config.RateLimitPerSecond),
	}

	return mock, nil
}

// simulateLoadEffects applies configured load simulation effects
func (m *MockEmbedder) simulateLoadEffects(ctx context.Context, span trace.Span) error {
	// Increment request count
	atomic.AddInt64(&m.requestCount, 1)

	// Simulate delay if configured
	if m.config.SimulateDelay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(m.config.SimulateDelay):
			// Delay completed
		}
	}

	// Simulate rate limiting
	if m.config.RateLimitPerSecond > 0 {
		if !m.checkRateLimit() {
			span.SetAttributes(attribute.String("rate_limit", "exceeded"))
			return iface.NewEmbeddingError(iface.ErrCodeEmbeddingFailed, "rate limit exceeded")
		}
	}

	// Simulate random errors
	if m.config.SimulateErrors && m.rng.Float64() < m.config.ErrorRate {
		span.SetAttributes(attribute.String("simulated_error", "true"))
		return iface.NewEmbeddingError(iface.ErrCodeEmbeddingFailed, "simulated random error")
	}

	// Simulate memory pressure
	if m.config.MemoryPressure {
		// Allocate some memory to simulate pressure
		_ = make([]byte, 1024*1024) // 1MB allocation (will be GC'd)
		runtime.GC()                // Force garbage collection
		span.SetAttributes(attribute.String("memory_pressure", "simulated"))
	}

	// Simulate performance degradation
	if m.config.PerformanceDegrade {
		requestCount := atomic.LoadInt64(&m.requestCount)
		// Add increasing delay based on request count (up to 100ms)
		degradeDelay := time.Duration(requestCount%100) * time.Millisecond
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(degradeDelay):
			// Performance degradation delay
		}
		span.SetAttributes(attribute.Int64("performance_degradation_ms", int64(degradeDelay.Milliseconds())))
	}

	return nil
}

// checkRateLimit implements token bucket rate limiting
func (m *MockEmbedder) checkRateLimit() bool {
	now := time.Now()

	// Refill tokens based on time elapsed
	elapsed := now.Sub(m.lastRefillTime)
	tokensToAdd := int64(elapsed.Seconds() * float64(m.config.RateLimitPerSecond))

	if tokensToAdd > 0 {
		atomic.AddInt64(&m.rateLimitTokens, tokensToAdd)
		// Cap at maximum tokens per second
		for {
			current := atomic.LoadInt64(&m.rateLimitTokens)
			if current <= int64(m.config.RateLimitPerSecond) {
				break
			}
			if atomic.CompareAndSwapInt64(&m.rateLimitTokens, current, int64(m.config.RateLimitPerSecond)) {
				break
			}
		}
		m.lastRefillTime = now
	}

	// Try to consume a token
	for {
		current := atomic.LoadInt64(&m.rateLimitTokens)
		if current <= 0 {
			return false
		}
		if atomic.CompareAndSwapInt64(&m.rateLimitTokens, current, current-1) {
			return true
		}
	}
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

	// Apply load simulation effects
	if err := m.simulateLoadEffects(ctx, span); err != nil {
		return nil, err
	}

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

	// Apply load simulation effects
	if err := m.simulateLoadEffects(ctx, span); err != nil {
		return nil, err
	}

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

// Check performs a health check on the mock embedder
func (m *MockEmbedder) Check(ctx context.Context) error {
	_, span := m.tracer.Start(ctx, "mock.health_check")
	defer span.End()

	// Mock embedder is always healthy
	return nil
}

// Ensure MockEmbedder implements the interface.
var _ iface.Embedder = (*MockEmbedder)(nil)
var _ HealthChecker = (*MockEmbedder)(nil)
