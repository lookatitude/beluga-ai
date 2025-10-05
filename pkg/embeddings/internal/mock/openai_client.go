package mock

import (
	"context"
	"sync"

	"github.com/sashabaranov/go-openai"
)

// OpenAIClientMock implements the OpenAI API client interface for testing
type OpenAIClientMock struct {
	mu sync.Mutex

	// CreateEmbeddings behavior
	CreateEmbeddingsFunc func(ctx context.Context, req openai.EmbeddingRequestConverter) (openai.EmbeddingResponse, error)
	CreateEmbeddingsCalls []CreateEmbeddingsCall

	// Error injection
	ShouldFailEmbeddings bool
	EmbeddingsError      error
}

// CreateEmbeddingsCall records a call to CreateEmbeddings
type CreateEmbeddingsCall struct {
	Ctx context.Context
	Req openai.EmbeddingRequestConverter
}

// NewOpenAIClientMock creates a new mock OpenAI client
func NewOpenAIClientMock() *OpenAIClientMock {
	return &OpenAIClientMock{
		CreateEmbeddingsCalls: make([]CreateEmbeddingsCall, 0),
	}
}

// CreateEmbeddings mocks the OpenAI embeddings API call
func (m *OpenAIClientMock) CreateEmbeddings(ctx context.Context, req openai.EmbeddingRequestConverter) (openai.EmbeddingResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CreateEmbeddingsCalls = append(m.CreateEmbeddingsCalls, CreateEmbeddingsCall{
		Ctx: ctx,
		Req: req,
	})

	if m.ShouldFailEmbeddings {
		return openai.EmbeddingResponse{}, m.EmbeddingsError
	}

	if m.CreateEmbeddingsFunc != nil {
		return m.CreateEmbeddingsFunc(ctx, req)
	}

	// Default behavior: return mock embeddings
	// Since req is an interface, we can't directly access fields.
	// In the test setup, the custom response function should be used.
	// For basic cases, return a single embedding.
	dimension := 1536 // Default ada-002 dimension

	embeddings := []openai.Embedding{
		{
			Object:    "embedding",
			Embedding: make([]float32, dimension),
			Index:     0,
		},
	}

	// Fill with mock data
	for j := 0; j < dimension; j++ {
		embeddings[0].Embedding[j] = float32(j) / float32(dimension)
	}

	return openai.EmbeddingResponse{
		Object: "list",
		Data:   embeddings,
		Model:  "text-embedding-ada-002",
		Usage: openai.Usage{
			PromptTokens:     5,
			TotalTokens:      5,
		},
	}, nil
}

// Reset resets the mock state
func (m *OpenAIClientMock) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CreateEmbeddingsCalls = make([]CreateEmbeddingsCall, 0)
	m.ShouldFailEmbeddings = false
	m.EmbeddingsError = nil
	m.CreateEmbeddingsFunc = nil
}

// SetCreateEmbeddingsResponse sets a custom response function for embeddings
func (m *OpenAIClientMock) SetCreateEmbeddingsResponse(response openai.EmbeddingResponse) {
	m.CreateEmbeddingsFunc = func(ctx context.Context, req openai.EmbeddingRequestConverter) (openai.EmbeddingResponse, error) {
		return response, nil
	}
}

// SetCreateEmbeddingsError sets the mock to return an error
func (m *OpenAIClientMock) SetCreateEmbeddingsError(err error) {
	m.ShouldFailEmbeddings = true
	m.EmbeddingsError = err
}
