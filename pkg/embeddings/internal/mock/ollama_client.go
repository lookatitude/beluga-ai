package mock

import (
	"context"
	"sync"

	"github.com/ollama/ollama/api"
)

// OllamaClientMock implements the Ollama API client interface for testing
type OllamaClientMock struct {
	mu sync.Mutex

	// Embeddings behavior
	EmbeddingsFunc func(ctx context.Context, req *api.EmbeddingRequest) (*api.EmbeddingResponse, error)
	EmbeddingsCalls []EmbeddingsCall

	// Error injection
	ShouldFailEmbeddings bool
	EmbeddingsError      error
}

// EmbeddingsCall records a call to Embeddings
type EmbeddingsCall struct {
	Ctx context.Context
	Req *api.EmbeddingRequest
}

// NewOllamaClientMock creates a new mock Ollama client
func NewOllamaClientMock() *OllamaClientMock {
	return &OllamaClientMock{
		EmbeddingsCalls: make([]EmbeddingsCall, 0),
	}
}

// Embeddings mocks the Ollama embeddings API call
func (m *OllamaClientMock) Embeddings(ctx context.Context, req *api.EmbeddingRequest) (*api.EmbeddingResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.EmbeddingsCalls = append(m.EmbeddingsCalls, EmbeddingsCall{
		Ctx: ctx,
		Req: req,
	})

	if m.ShouldFailEmbeddings {
		return nil, m.EmbeddingsError
	}

	if m.EmbeddingsFunc != nil {
		return m.EmbeddingsFunc(ctx, req)
	}

	// Default behavior: return mock embeddings
	dimension := 768 // Common embedding dimension
	embedding := make([]float64, dimension)
	for i := range embedding {
		embedding[i] = float64(i) / float64(dimension) // Simple mock values
	}

	return &api.EmbeddingResponse{
		Embedding: embedding,
	}, nil
}

// Reset resets the mock state
func (m *OllamaClientMock) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.EmbeddingsCalls = make([]EmbeddingsCall, 0)
	m.ShouldFailEmbeddings = false
	m.EmbeddingsError = nil
	m.EmbeddingsFunc = nil
}

// SetEmbeddingsResponse sets a custom response function for embeddings
func (m *OllamaClientMock) SetEmbeddingsResponse(embedding []float64) {
	m.EmbeddingsFunc = func(ctx context.Context, req *api.EmbeddingRequest) (*api.EmbeddingResponse, error) {
		return &api.EmbeddingResponse{
			Embedding: embedding,
		}, nil
	}
}

// SetEmbeddingsError sets the mock to return an error
func (m *OllamaClientMock) SetEmbeddingsError(err error) {
	m.ShouldFailEmbeddings = true
	m.EmbeddingsError = err
}
