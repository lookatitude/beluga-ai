package mockembedder

import (
	"context"
	"sync"

	"github.com/lookatitude/beluga-ai/rag/embedding"
)

// Compile-time interface check
var _ embedding.Embedder = (*MockEmbedder)(nil)

// MockEmbedder is a configurable mock for the Embedder interface.
// It records all Embed calls and can return preset embeddings or errors.
type MockEmbedder struct {
	mu sync.Mutex

	embeddings [][]float32
	dimensions int
	err        error
	embedFn    func(ctx context.Context, texts []string) ([][]float32, error)

	embedCalls int
	lastTexts  []string
}

// Option configures a MockEmbedder.
type Option func(*MockEmbedder)

// New creates a MockEmbedder with the given options.
func New(opts ...Option) *MockEmbedder {
	m := &MockEmbedder{
		dimensions: 384, // default embedding dimension
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// WithEmbeddings configures the mock to return the given embeddings from Embed.
// The embeddings should be a slice where each element is a float32 vector.
func WithEmbeddings(embeddings [][]float32) Option {
	return func(m *MockEmbedder) {
		m.embeddings = embeddings
	}
}

// WithDimensions sets the dimensionality returned by Dimensions().
func WithDimensions(dim int) Option {
	return func(m *MockEmbedder) {
		m.dimensions = dim
	}
}

// WithError configures the mock to return the given error from Embed and EmbedSingle.
func WithError(err error) Option {
	return func(m *MockEmbedder) {
		m.err = err
	}
}

// WithEmbedFunc sets a custom function to call on Embed, overriding
// the canned embeddings/error.
func WithEmbedFunc(fn func(ctx context.Context, texts []string) ([][]float32, error)) Option {
	return func(m *MockEmbedder) {
		m.embedFn = fn
	}
}

// Embed produces embeddings for a batch of texts. It returns the configured
// embeddings or error, and records the call for later inspection.
func (m *MockEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.embedCalls++
	m.lastTexts = make([]string, len(texts))
	copy(m.lastTexts, texts)

	if m.embedFn != nil {
		return m.embedFn(ctx, texts)
	}

	if m.err != nil {
		return nil, m.err
	}

	if m.embeddings != nil {
		// If we have preset embeddings, repeat them as needed
		result := make([][]float32, len(texts))
		for i := range texts {
			idx := i % len(m.embeddings)
			result[i] = make([]float32, len(m.embeddings[idx]))
			copy(result[i], m.embeddings[idx])
		}
		return result, nil
	}

	// Default: return zero vectors of the configured dimension
	result := make([][]float32, len(texts))
	for i := range result {
		result[i] = make([]float32, m.dimensions)
	}
	return result, nil
}

// EmbedSingle embeds a single text and returns its vector.
func (m *MockEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := m.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return embeddings[0], nil
}

// Dimensions returns the configured dimensionality of embedding vectors.
func (m *MockEmbedder) Dimensions() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.dimensions
}

// EmbedCalls returns the number of times Embed or EmbedSingle has been called.
func (m *MockEmbedder) EmbedCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.embedCalls
}

// LastTexts returns the texts passed to the most recent Embed or EmbedSingle call.
func (m *MockEmbedder) LastTexts() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.lastTexts))
	copy(result, m.lastTexts)
	return result
}

// SetEmbeddings updates the canned embeddings for subsequent calls.
func (m *MockEmbedder) SetEmbeddings(embeddings [][]float32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.embeddings = embeddings
	m.err = nil
}

// SetError updates the error for subsequent calls.
func (m *MockEmbedder) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
	m.embeddings = nil
}

// Reset clears all recorded calls and configuration.
func (m *MockEmbedder) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.embedCalls = 0
	m.lastTexts = nil
	m.embeddings = nil
	m.err = nil
	m.embedFn = nil
	m.dimensions = 384
}
