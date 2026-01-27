package rag

import (
	"context"
	"errors"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/core"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// MockEmbedder is a mock implementation of embeddingsiface.Embedder for testing.
type MockEmbedder struct {
	EmbedDocumentsFunc func(ctx context.Context, texts []string) ([][]float32, error)
	EmbedQueryFunc     func(ctx context.Context, text string) ([]float32, error)
	GetDimensionFunc   func(ctx context.Context) (int, error)

	mu              sync.Mutex
	embedDocCalls   [][]string
	embedQueryCalls []string
	dimension       int
}

// NewMockEmbedder creates a new MockEmbedder with default behavior.
func NewMockEmbedder() *MockEmbedder {
	m := &MockEmbedder{
		dimension: 384, // Default dimension
	}
	m.EmbedDocumentsFunc = func(ctx context.Context, texts []string) ([][]float32, error) {
		embeddings := make([][]float32, len(texts))
		for i := range texts {
			embeddings[i] = make([]float32, m.dimension)
			// Fill with simple values based on text length for predictability
			for j := 0; j < m.dimension; j++ {
				embeddings[i][j] = float32(len(texts[i])+j) / 1000.0
			}
		}
		return embeddings, nil
	}
	m.EmbedQueryFunc = func(ctx context.Context, text string) ([]float32, error) {
		embedding := make([]float32, m.dimension)
		for i := 0; i < m.dimension; i++ {
			embedding[i] = float32(len(text)+i) / 1000.0
		}
		return embedding, nil
	}
	m.GetDimensionFunc = func(ctx context.Context) (int, error) {
		return m.dimension, nil
	}
	return m
}

// EmbedDocuments calls the mock function.
func (m *MockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	m.mu.Lock()
	m.embedDocCalls = append(m.embedDocCalls, texts)
	m.mu.Unlock()

	if m.EmbedDocumentsFunc != nil {
		return m.EmbedDocumentsFunc(ctx, texts)
	}
	return nil, nil
}

// EmbedQuery calls the mock function.
func (m *MockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	m.mu.Lock()
	m.embedQueryCalls = append(m.embedQueryCalls, text)
	m.mu.Unlock()

	if m.EmbedQueryFunc != nil {
		return m.EmbedQueryFunc(ctx, text)
	}
	return nil, nil
}

// GetDimension returns the embedding dimension.
func (m *MockEmbedder) GetDimension(ctx context.Context) (int, error) {
	if m.GetDimensionFunc != nil {
		return m.GetDimensionFunc(ctx)
	}
	return m.dimension, nil
}

// SetDimension sets the embedding dimension.
func (m *MockEmbedder) SetDimension(dim int) {
	m.dimension = dim
}

// SetEmbedError sets an error for embed operations.
func (m *MockEmbedder) SetEmbedError(err error) {
	m.EmbedDocumentsFunc = func(ctx context.Context, texts []string) ([][]float32, error) {
		return nil, err
	}
	m.EmbedQueryFunc = func(ctx context.Context, text string) ([]float32, error) {
		return nil, err
	}
}

// GetEmbedDocCalls returns all texts passed to EmbedDocuments.
func (m *MockEmbedder) GetEmbedDocCalls() [][]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.embedDocCalls
}

// GetEmbedQueryCalls returns all queries passed to EmbedQuery.
func (m *MockEmbedder) GetEmbedQueryCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.embedQueryCalls
}

// MockChatModel is a mock implementation of llmsiface.ChatModel for testing.
type MockChatModel struct {
	InvokeFunc       func(ctx context.Context, input any, options ...core.Option) (any, error)
	BatchFunc        func(ctx context.Context, inputs []any, options ...core.Option) ([]any, error)
	StreamFunc       func(ctx context.Context, input any, options ...core.Option) (<-chan any, error)
	GenerateFunc     func(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error)
	StreamChatFunc   func(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan llmsiface.AIMessageChunk, error)
	BindToolsFunc    func(tools []core.Tool) llmsiface.ChatModel
	GetModelNameFunc func() string
	GetProviderFunc  func() string
	CheckHealthFunc  func() map[string]any

	mu            sync.Mutex
	generateCalls [][]schema.Message
}

// NewMockChatModel creates a new MockChatModel with default behavior.
func NewMockChatModel() *MockChatModel {
	m := &MockChatModel{}
	m.InvokeFunc = func(ctx context.Context, input any, options ...core.Option) (any, error) {
		return "Mock response", nil
	}
	m.BatchFunc = func(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
		results := make([]any, len(inputs))
		for i := range inputs {
			results[i] = "Mock response"
		}
		return results, nil
	}
	m.StreamFunc = func(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
		ch := make(chan any, 1)
		go func() {
			defer close(ch)
			ch <- "Mock response"
		}()
		return ch, nil
	}
	m.GenerateFunc = func(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
		return schema.NewAIMessage("Mock AI response"), nil
	}
	m.StreamChatFunc = func(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan llmsiface.AIMessageChunk, error) {
		ch := make(chan llmsiface.AIMessageChunk, 1)
		go func() {
			defer close(ch)
			ch <- llmsiface.AIMessageChunk{Content: "Mock streamed response"}
		}()
		return ch, nil
	}
	m.GetModelNameFunc = func() string {
		return "mock-model"
	}
	m.GetProviderFunc = func() string {
		return "mock-provider"
	}
	m.CheckHealthFunc = func() map[string]any {
		return map[string]any{"status": "healthy"}
	}
	return m
}

// Invoke calls the mock function.
func (m *MockChatModel) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	if m.InvokeFunc != nil {
		return m.InvokeFunc(ctx, input, options...)
	}
	return "Mock response", nil
}

// Batch calls the mock function.
func (m *MockChatModel) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	if m.BatchFunc != nil {
		return m.BatchFunc(ctx, inputs, options...)
	}
	results := make([]any, len(inputs))
	for i := range inputs {
		results[i] = "Mock response"
	}
	return results, nil
}

// Stream calls the mock function.
func (m *MockChatModel) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	if m.StreamFunc != nil {
		return m.StreamFunc(ctx, input, options...)
	}
	ch := make(chan any, 1)
	go func() {
		defer close(ch)
		ch <- "Mock response"
	}()
	return ch, nil
}

// Generate calls the mock function.
func (m *MockChatModel) Generate(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
	m.mu.Lock()
	m.generateCalls = append(m.generateCalls, messages)
	m.mu.Unlock()

	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, messages, options...)
	}
	return schema.NewAIMessage("Mock AI response"), nil
}

// StreamChat calls the mock function.
func (m *MockChatModel) StreamChat(ctx context.Context, messages []schema.Message, options ...core.Option) (<-chan llmsiface.AIMessageChunk, error) {
	if m.StreamChatFunc != nil {
		return m.StreamChatFunc(ctx, messages, options...)
	}
	ch := make(chan llmsiface.AIMessageChunk, 1)
	go func() {
		defer close(ch)
		ch <- llmsiface.AIMessageChunk{Content: "Mock streamed response"}
	}()
	return ch, nil
}

// BindTools creates a new ChatModel with tools bound.
func (m *MockChatModel) BindTools(tools []core.Tool) llmsiface.ChatModel {
	if m.BindToolsFunc != nil {
		return m.BindToolsFunc(tools)
	}
	return m
}

// GetModelName returns the mock model name.
func (m *MockChatModel) GetModelName() string {
	if m.GetModelNameFunc != nil {
		return m.GetModelNameFunc()
	}
	return "mock-model"
}

// GetProviderName returns the mock provider name.
func (m *MockChatModel) GetProviderName() string {
	if m.GetProviderFunc != nil {
		return m.GetProviderFunc()
	}
	return "mock-provider"
}

// CheckHealth returns the mock health status.
func (m *MockChatModel) CheckHealth() map[string]any {
	if m.CheckHealthFunc != nil {
		return m.CheckHealthFunc()
	}
	return map[string]any{"status": "healthy"}
}

// SetGenerateResponse sets a specific response for Generate calls.
func (m *MockChatModel) SetGenerateResponse(content string) {
	m.GenerateFunc = func(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
		return schema.NewAIMessage(content), nil
	}
}

// SetGenerateError sets an error for Generate calls.
func (m *MockChatModel) SetGenerateError(err error) {
	m.GenerateFunc = func(ctx context.Context, messages []schema.Message, options ...core.Option) (schema.Message, error) {
		return nil, err
	}
}

// GetGenerateCalls returns all message lists passed to Generate.
func (m *MockChatModel) GetGenerateCalls() [][]schema.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.generateCalls
}

// Ensure MockChatModel implements llmsiface.ChatModel.
var _ llmsiface.ChatModel = (*MockChatModel)(nil)

// Common test errors.
var (
	ErrMockEmbedder = errors.New("mock embedder error")
	ErrMockLLM      = errors.New("mock LLM error")
)

// CreateTestDocuments creates a set of test documents.
func CreateTestDocuments(count int) []schema.Document {
	docs := make([]schema.Document, count)
	for i := 0; i < count; i++ {
		docs[i] = schema.NewDocument(
			"This is test document content for document number "+string(rune('A'+i)),
			map[string]string{"index": string(rune('0' + i))},
		)
	}
	return docs
}
