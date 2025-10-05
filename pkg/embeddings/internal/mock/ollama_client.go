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
	EmbeddingsFunc  func(ctx context.Context, req *api.EmbeddingRequest) (*api.EmbeddingResponse, error)
	EmbeddingsCalls []EmbeddingsCall

	// Show behavior
	ShowFunc  func(ctx context.Context, req *api.ShowRequest) (*api.ShowResponse, error)
	ShowCalls []ShowCall

	// Error injection
	ShouldFailEmbeddings bool
	EmbeddingsError      error
	ShouldFailShow       bool
	ShowError            error
}

// EmbeddingsCall records a call to Embeddings
type EmbeddingsCall struct {
	Ctx context.Context
	Req *api.EmbeddingRequest
}

// ShowCall records a call to Show
type ShowCall struct {
	Ctx context.Context
	Req *api.ShowRequest
}

// NewOllamaClientMock creates a new mock Ollama client
func NewOllamaClientMock() *OllamaClientMock {
	return &OllamaClientMock{
		EmbeddingsCalls: make([]EmbeddingsCall, 0),
		ShowCalls:       make([]ShowCall, 0),
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
	m.ShowCalls = make([]ShowCall, 0)
	m.ShouldFailEmbeddings = false
	m.EmbeddingsError = nil
	m.EmbeddingsFunc = nil
	m.ShouldFailShow = false
	m.ShowError = nil
	m.ShowFunc = nil
}

// SetEmbeddingsResponse sets a custom response function for embeddings
func (m *OllamaClientMock) SetEmbeddingsResponse(embedding []float64) {
	m.EmbeddingsFunc = func(ctx context.Context, req *api.EmbeddingRequest) (*api.EmbeddingResponse, error) {
		return &api.EmbeddingResponse{
			Embedding: embedding,
		}, nil
	}
}

// Show mocks the Ollama show API call
func (m *OllamaClientMock) Show(ctx context.Context, req *api.ShowRequest) (*api.ShowResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ShowCalls = append(m.ShowCalls, ShowCall{
		Ctx: ctx,
		Req: req,
	})

	if m.ShouldFailShow {
		return nil, m.ShowError
	}

	if m.ShowFunc != nil {
		return m.ShowFunc(ctx, req)
	}

	// Default behavior: return mock model info
	return &api.ShowResponse{
		Modelfile: "# Mock model configuration\nFROM llama2:7b\nPARAMETER temperature 0.8",
	}, nil
}

// SetEmbeddingsError sets the mock to return an error
func (m *OllamaClientMock) SetEmbeddingsError(err error) {
	m.ShouldFailEmbeddings = true
	m.EmbeddingsError = err
}

// SetShowResponse sets a custom response function for show
func (m *OllamaClientMock) SetShowResponse(modelfile string) {
	m.ShowFunc = func(ctx context.Context, req *api.ShowRequest) (*api.ShowResponse, error) {
		return &api.ShowResponse{
			Modelfile: modelfile,
		}, nil
	}
}

// SetShowError sets the mock to return an error for show
func (m *OllamaClientMock) SetShowError(err error) {
	m.ShouldFailShow = true
	m.ShowError = err
}
