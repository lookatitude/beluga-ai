package cohere

import (
	"context"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/stretchr/testify/mock"
)

// AdvancedMockCohereEmbedder provides a comprehensive mock implementation for testing Cohere embedder.
type AdvancedMockCohereEmbedder struct {
	mock.Mock
	mu                sync.RWMutex
	callCount         int
	shouldError       bool
	errorToReturn     error
	embeddings        [][]float32
	embeddingIndex    int
	dimension         int
	simulateDelay     time.Duration
	simulateRateLimit bool
	rateLimitCount    int
}

// NewAdvancedMockCohereEmbedder creates a new advanced mock with configurable behavior.
func NewAdvancedMockCohereEmbedder(dimension int) *AdvancedMockCohereEmbedder {
	mock := &AdvancedMockCohereEmbedder{
		dimension: dimension,
		embeddings: make([][]float32, 0),
	}
	mock.generateDefaultEmbeddings(10)
	return mock
}

// generateDefaultEmbeddings creates random embeddings for testing.
func (m *AdvancedMockCohereEmbedder) generateDefaultEmbeddings(count int) {
	m.embeddings = make([][]float32, count)
	for i := 0; i < count; i++ {
		embedding := make([]float32, m.dimension)
		for j := 0; j < m.dimension; j++ {
			embedding[j] = float32(i+j) / float32(m.dimension) // Deterministic pattern for testing
		}
		m.embeddings[i] = embedding
	}
}

// EmbedDocuments creates embeddings for a batch of document texts.
func (m *AdvancedMockCohereEmbedder) EmbedDocuments(ctx context.Context, documents []string) ([][]float32, error) {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	simulateRateLimit := m.simulateRateLimit
	rateLimitCount := m.rateLimitCount
	embeddingIndex := m.embeddingIndex
	embeddingsCopy := make([][]float32, len(m.embeddings))
	for i := range m.embeddings {
		embeddingsCopy[i] = make([]float32, len(m.embeddings[i]))
		copy(embeddingsCopy[i], m.embeddings[i])
	}
	m.mu.Unlock()

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	m.mu.Lock()
	if simulateRateLimit && rateLimitCount > 5 {
		m.mu.Unlock()
		return nil, iface.NewEmbeddingError(iface.ErrCodeEmbeddingFailed, "rate limit exceeded")
	}
	m.rateLimitCount++
	m.mu.Unlock()

	if shouldError {
		return nil, errorToReturn
	}

	results := make([][]float32, len(documents))
	for i := range documents {
		m.mu.Lock()
		currentIndex := embeddingIndex
		if currentIndex < len(embeddingsCopy) {
			embedding := make([]float32, len(embeddingsCopy[currentIndex]))
			copy(embedding, embeddingsCopy[currentIndex])
			results[i] = embedding
			embeddingIndex = (embeddingIndex + 1) % len(embeddingsCopy)
			m.embeddingIndex = embeddingIndex
		} else {
			// Generate deterministic embedding
			embedding := make([]float32, m.dimension)
			for j := 0; j < m.dimension; j++ {
				embedding[j] = float32(i+j) / float32(m.dimension)
			}
			results[i] = embedding
		}
		m.mu.Unlock()
	}

	return results, nil
}

// EmbedQuery creates an embedding for a single query text.
func (m *AdvancedMockCohereEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	m.mu.Lock()
	m.callCount++
	shouldError := m.shouldError
	errorToReturn := m.errorToReturn
	simulateRateLimit := m.simulateRateLimit
	rateLimitCount := m.rateLimitCount
	embeddingIndex := m.embeddingIndex
	embeddingsCopy := make([][]float32, len(m.embeddings))
	for i := range m.embeddings {
		embeddingsCopy[i] = make([]float32, len(m.embeddings[i]))
		copy(embeddingsCopy[i], m.embeddings[i])
	}
	m.mu.Unlock()

	if m.simulateDelay > 0 {
		time.Sleep(m.simulateDelay)
	}

	m.mu.Lock()
	if simulateRateLimit && rateLimitCount > 5 {
		m.mu.Unlock()
		return nil, iface.NewEmbeddingError(iface.ErrCodeEmbeddingFailed, "rate limit exceeded")
	}
	m.rateLimitCount++
	m.mu.Unlock()

	if shouldError {
		return nil, errorToReturn
	}

	m.mu.Lock()
	var embedding []float32
	if embeddingIndex < len(embeddingsCopy) {
		embedding = make([]float32, len(embeddingsCopy[embeddingIndex]))
		copy(embedding, embeddingsCopy[embeddingIndex])
		m.embeddingIndex = (embeddingIndex + 1) % len(embeddingsCopy)
		m.mu.Unlock()
		return embedding, nil
	}
	m.mu.Unlock()

	// Generate deterministic embedding
	embedding = make([]float32, m.dimension)
	for j := 0; j < m.dimension; j++ {
		embedding[j] = float32(j) / float32(m.dimension)
	}
	return embedding, nil
}

// GetDimension returns the dimension of embeddings produced by this embedder.
func (m *AdvancedMockCohereEmbedder) GetDimension(ctx context.Context) (int, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()

	if m.shouldError {
		return 0, m.errorToReturn
	}

	return m.dimension, nil
}

// Check implements the HealthChecker interface.
func (m *AdvancedMockCohereEmbedder) Check(ctx context.Context) error {
	_, err := m.GetDimension(ctx)
	return err
}

// SetError configures the mock to return an error.
func (m *AdvancedMockCohereEmbedder) SetError(shouldError bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = shouldError
	m.errorToReturn = err
}

// SetDelay configures the mock to simulate delay.
func (m *AdvancedMockCohereEmbedder) SetDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateDelay = delay
}

// SetRateLimit configures the mock to simulate rate limiting.
func (m *AdvancedMockCohereEmbedder) SetRateLimit(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateRateLimit = enabled
}

// GetCallCount returns the number of times methods have been called.
func (m *AdvancedMockCohereEmbedder) GetCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.callCount
}

// Reset resets the mock state.
func (m *AdvancedMockCohereEmbedder) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callCount = 0
	m.shouldError = false
	m.errorToReturn = nil
	m.embeddingIndex = 0
	m.rateLimitCount = 0
	m.simulateRateLimit = false
	m.simulateDelay = 0
}

// Ensure AdvancedMockCohereEmbedder implements the interfaces.
var (
	_ iface.Embedder   = (*AdvancedMockCohereEmbedder)(nil)
	_ HealthChecker    = (*AdvancedMockCohereEmbedder)(nil)
)
