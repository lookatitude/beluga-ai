// Package vectorstore provides comprehensive tests for vector store memory implementations.
package vectorstore

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/memory/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
	"github.com/stretchr/testify/assert"
)

// MockRetriever is a mock implementation of core.Retriever for testing
type MockRetriever struct {
	invokeFunc           func(ctx context.Context, input any, options ...core.Option) (any, error)
	batchFunc            func(ctx context.Context, inputs []any, options ...core.Option) ([]any, error)
	streamFunc           func(ctx context.Context, input any, options ...core.Option) (<-chan any, error)
	invokeError          error
	batchError           error
	streamError          error
	addDocumentsFunc     func(ctx context.Context, documents []schema.Document) ([]string, error)
	addDocumentsError    error
	getRelevantDocsFunc  func(ctx context.Context, query string) ([]schema.Document, error)
	getRelevantDocsError error
}

func NewMockRetriever() *MockRetriever {
	return &MockRetriever{}
}

func (m *MockRetriever) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	if m.invokeError != nil {
		return nil, m.invokeError
	}
	if m.invokeFunc != nil {
		return m.invokeFunc(ctx, input, options...)
	}
	// Default behavior - return empty document list
	return []schema.Document{}, nil
}

func (m *MockRetriever) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	if m.batchError != nil {
		return nil, m.batchError
	}
	if m.batchFunc != nil {
		return m.batchFunc(ctx, inputs, options...)
	}
	// Default behavior - return mock results
	results := make([]any, len(inputs))
	for i := range inputs {
		results[i] = []schema.Document{}
	}
	return results, nil
}

func (m *MockRetriever) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	if m.streamError != nil {
		return nil, m.streamError
	}
	if m.streamFunc != nil {
		return m.streamFunc(ctx, input, options...)
	}
	// Default behavior - return a channel with a single response
	ch := make(chan any, 1)
	ch <- []schema.Document{}
	close(ch)
	return ch, nil
}

func (m *MockRetriever) WithConfig(config map[string]any) core.Runnable {
	return m
}

func (m *MockRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
	if m.getRelevantDocsError != nil {
		return nil, m.getRelevantDocsError
	}
	if m.getRelevantDocsFunc != nil {
		return m.getRelevantDocsFunc(ctx, query)
	}
	// Default behavior - return mock documents
	return []schema.Document{
		{PageContent: "Mock relevant document", Metadata: map[string]string{"score": "0.9"}},
	}, nil
}

func (m *MockRetriever) AddDocuments(ctx context.Context, documents []schema.Document) ([]string, error) {
	if m.addDocumentsError != nil {
		return nil, m.addDocumentsError
	}
	if m.addDocumentsFunc != nil {
		return m.addDocumentsFunc(ctx, documents)
	}
	// Default behavior - return document IDs
	ids := make([]string, len(documents))
	for i := range documents {
		ids[i] = "doc_" + string(rune(i+'0'))
	}
	return ids, nil
}

// MockEmbedder is a mock implementation of embeddingsiface.Embedder for testing
type MockEmbedder struct {
	embedFunc           func(ctx context.Context, texts []string) ([][]float32, error)
	embedDocumentsFunc  func(ctx context.Context, texts []string) ([][]float32, error)
	embedError          error
	embedDocumentsError error
}

func NewMockEmbedder() *MockEmbedder {
	return &MockEmbedder{}
}

func (m *MockEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if m.embedError != nil {
		return nil, m.embedError
	}
	if m.embedFunc != nil {
		return m.embedFunc(ctx, texts)
	}
	// Default behavior - return mock embeddings
	embeddings := make([][]float32, len(texts))
	for i := range texts {
		embeddings[i] = []float32{0.1, 0.2, 0.3}
	}
	return embeddings, nil
}

func (m *MockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if m.embedDocumentsError != nil {
		return nil, m.embedDocumentsError
	}
	if m.embedDocumentsFunc != nil {
		return m.embedDocumentsFunc(ctx, texts)
	}
	// Default behavior - same as Embed
	return m.Embed(ctx, texts)
}

func (m *MockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	// Default behavior - return a single embedding
	return []float32{0.1, 0.2, 0.3}, nil
}

func (m *MockEmbedder) GetDimension(ctx context.Context) (int, error) {
	// Default behavior - return fixed dimension
	return 3, nil
}

// MockVectorStore is a mock implementation of vectorstores.VectorStore for testing
type MockVectorStore struct {
	similaritySearchFunc         func(ctx context.Context, queryVector []float32, k int, options ...vectorstores.Option) ([]schema.Document, []float32, error)
	similaritySearchByQueryFunc  func(ctx context.Context, query string, k int, embedder vectorstores.Embedder, options ...vectorstores.Option) ([]schema.Document, []float32, error)
	addDocumentsFunc             func(ctx context.Context, documents []schema.Document, options ...vectorstores.Option) ([]string, error)
	deleteDocumentsFunc          func(ctx context.Context, ids []string, options ...vectorstores.Option) error
	asRetrieverFunc              func(options ...vectorstores.Option) vectorstores.Retriever
	similaritySearchError        error
	similaritySearchByQueryError error
	addDocumentsError            error
	deleteDocumentsError         error
	name                         string
}

func NewMockVectorStore() *MockVectorStore {
	return &MockVectorStore{
		name: "mock_vectorstore",
	}
}

func (m *MockVectorStore) SimilaritySearch(ctx context.Context, queryVector []float32, k int, options ...vectorstores.Option) ([]schema.Document, []float32, error) {
	if m.similaritySearchError != nil {
		return nil, nil, m.similaritySearchError
	}
	if m.similaritySearchFunc != nil {
		return m.similaritySearchFunc(ctx, queryVector, k, options...)
	}
	// Default behavior - return mock documents
	docs := []schema.Document{
		{PageContent: "Mock document 1", Metadata: map[string]string{"score": "0.9"}},
		{PageContent: "Mock document 2", Metadata: map[string]string{"score": "0.8"}},
	}
	scores := []float32{0.9, 0.8}
	return docs, scores, nil
}

func (m *MockVectorStore) SimilaritySearchByQuery(ctx context.Context, query string, k int, embedder vectorstores.Embedder, options ...vectorstores.Option) ([]schema.Document, []float32, error) {
	if m.similaritySearchByQueryError != nil {
		return nil, nil, m.similaritySearchByQueryError
	}
	if m.similaritySearchByQueryFunc != nil {
		return m.similaritySearchByQueryFunc(ctx, query, k, embedder, options...)
	}
	// Default behavior - return mock documents
	docs := []schema.Document{
		{PageContent: "Mock document 1", Metadata: map[string]string{"score": "0.9"}},
		{PageContent: "Mock document 2", Metadata: map[string]string{"score": "0.8"}},
	}
	scores := []float32{0.9, 0.8}
	return docs, scores, nil
}

func (m *MockVectorStore) AddDocuments(ctx context.Context, documents []schema.Document, options ...vectorstores.Option) ([]string, error) {
	if m.addDocumentsError != nil {
		return nil, m.addDocumentsError
	}
	if m.addDocumentsFunc != nil {
		return m.addDocumentsFunc(ctx, documents, options...)
	}
	// Default behavior - return document IDs
	ids := make([]string, len(documents))
	for i := range documents {
		ids[i] = "doc_" + string(rune(i+'0'))
	}
	return ids, nil
}

func (m *MockVectorStore) DeleteDocuments(ctx context.Context, ids []string, options ...vectorstores.Option) error {
	if m.deleteDocumentsError != nil {
		return m.deleteDocumentsError
	}
	if m.deleteDocumentsFunc != nil {
		return m.deleteDocumentsFunc(ctx, ids, options...)
	}
	// Default behavior - no-op
	return nil
}

func (m *MockVectorStore) AsRetriever(options ...vectorstores.Option) vectorstores.Retriever {
	if m.asRetrieverFunc != nil {
		return m.asRetrieverFunc(options...)
	}
	// Default behavior - return nil (would normally return a retriever)
	return nil
}

func (m *MockVectorStore) GetName() string {
	return m.name
}

// TestNewVectorStoreMemory tests the constructor
func TestNewVectorStoreMemory(t *testing.T) {
	retriever := NewMockRetriever()
	memory := NewVectorStoreMemory(retriever, "test_memory", false, 5)

	assert.NotNil(t, memory)
	assert.Equal(t, retriever, memory.Retriever)
	assert.Equal(t, "test_memory", memory.MemoryKey)
	assert.False(t, memory.ReturnDocs)
	assert.Equal(t, 5, memory.NumDocsToKeep)
}

// TestNewVectorStoreMemory_Defaults tests default values
func TestNewVectorStoreMemory_Defaults(t *testing.T) {
	retriever := NewMockRetriever()
	memory := NewVectorStoreMemory(retriever, "", true, 0)

	assert.Equal(t, "history", memory.MemoryKey) // Default memory key
	assert.True(t, memory.ReturnDocs)
	assert.Equal(t, 4, memory.NumDocsToKeep) // Default number of documents
}

// TestVectorStoreMemory_MemoryVariables tests the MemoryVariables method
func TestVectorStoreMemory_MemoryVariables(t *testing.T) {
	retriever := NewMockRetriever()
	memory := NewVectorStoreMemory(retriever, "custom_memory", false, 5)

	variables := memory.MemoryVariables()
	assert.Equal(t, []string{"custom_memory"}, variables)
}

// TestVectorStoreMemory_LoadMemoryVariables_ReturnDocs tests loading with ReturnDocs=true
func TestVectorStoreMemory_LoadMemoryVariables_ReturnDocs(t *testing.T) {
	ctx := context.Background()
	retriever := NewMockRetriever()

	// Mock retriever to return documents
	retriever.invokeFunc = func(ctx context.Context, input any, options ...core.Option) (any, error) {
		return []schema.Document{
			{PageContent: "Document 1", Metadata: map[string]string{"score": "0.9"}},
			{PageContent: "Document 2", Metadata: map[string]string{"score": "0.8"}},
			{PageContent: "Document 3", Metadata: map[string]string{"score": "0.7"}},
		}, nil
	}

	memory := NewVectorStoreMemory(retriever, "memory", true, 2)
	memory.InputKey = "query"

	inputs := map[string]any{"query": "test query"}

	vars, err := memory.LoadMemoryVariables(ctx, inputs)
	assert.NoError(t, err)
	assert.Contains(t, vars, "memory")

	docs, ok := vars["memory"].([]schema.Document)
	assert.True(t, ok)
	assert.Len(t, docs, 2) // Should be limited to NumDocsToKeep
}

// TestVectorStoreMemory_LoadMemoryVariables_ReturnFormattedString tests loading with ReturnDocs=false
func TestVectorStoreMemory_LoadMemoryVariables_ReturnFormattedString(t *testing.T) {
	ctx := context.Background()
	retriever := NewMockRetriever()

	// Mock retriever to return documents
	retriever.invokeFunc = func(ctx context.Context, input any, options ...core.Option) (any, error) {
		return []schema.Document{
			{PageContent: "Document 1", Metadata: map[string]string{"score": "0.9"}},
			{PageContent: "Document 2", Metadata: map[string]string{"score": "0.8"}},
		}, nil
	}

	memory := NewVectorStoreMemory(retriever, "memory", false, 5)
	memory.InputKey = "query"

	inputs := map[string]any{"query": "test query"}

	vars, err := memory.LoadMemoryVariables(ctx, inputs)
	assert.NoError(t, err)
	assert.Contains(t, vars, "memory")

	formatted, ok := vars["memory"].(string)
	assert.True(t, ok)
	assert.Contains(t, formatted, "Relevant context:")
	assert.Contains(t, formatted, "Document 1")
	assert.Contains(t, formatted, "Document 2")
}

// TestVectorStoreMemory_LoadMemoryVariables_AutoDetectInputKey tests automatic input key detection
func TestVectorStoreMemory_LoadMemoryVariables_AutoDetectInputKey(t *testing.T) {
	ctx := context.Background()
	retriever := NewMockRetriever()

	// Mock retriever to return documents
	retriever.invokeFunc = func(ctx context.Context, input any, options ...core.Option) (any, error) {
		return []schema.Document{{PageContent: "Test doc"}}, nil
	}

	memory := NewVectorStoreMemory(retriever, "memory", false, 5)
	// Clear InputKey to trigger auto-detection
	memory.InputKey = ""

	testCases := []struct {
		name     string
		inputs   map[string]any
		expected string
	}{
		{
			name:     "Single key",
			inputs:   map[string]any{"question": "test"},
			expected: "question",
		},
		{
			name:     "Standard input key",
			inputs:   map[string]any{"input": "test", "other": "value"},
			expected: "input",
		},
		{
			name:     "Question key",
			inputs:   map[string]any{"question": "test", "other": "value"},
			expected: "question",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vars, err := memory.LoadMemoryVariables(ctx, tc.inputs)
			if tc.expected != "" {
				assert.NoError(t, err)
				assert.Contains(t, vars, "memory")
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// TestVectorStoreMemory_LoadMemoryVariables_ErrorHandling tests various error conditions
func TestVectorStoreMemory_LoadMemoryVariables_ErrorHandling(t *testing.T) {
	testCases := []struct {
		name           string
		setupMemory    func(*VectorStoreMemory)
		setupRetriever func(*MockRetriever)
		inputs         map[string]any
		expectedError  string
	}{
		{
			name: "Missing input key",
			setupMemory: func(m *VectorStoreMemory) {
				m.InputKey = "missing_key"
			},
			inputs:        map[string]any{"other_key": "value"},
			expectedError: "error",
		},
		{
			name: "Non-string input",
			setupMemory: func(m *VectorStoreMemory) {
				m.InputKey = "input"
			},
			inputs:        map[string]any{"input": 123},
			expectedError: "error",
		},
		{
			name: "Retriever error",
			setupRetriever: func(r *MockRetriever) {
				r.invokeError = errors.New("retriever failed")
			},
			inputs:        map[string]any{"input": "query"},
			expectedError: "error",
		},
		{
			name: "Unexpected retriever response type",
			setupRetriever: func(r *MockRetriever) {
				r.invokeFunc = func(ctx context.Context, input any, options ...core.Option) (any, error) {
					return "unexpected string response", nil
				}
			},
			inputs:        map[string]any{"input": "query"},
			expectedError: "error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			retriever := NewMockRetriever()
			if tc.setupRetriever != nil {
				tc.setupRetriever(retriever)
			}

			memory := NewVectorStoreMemory(retriever, "memory", false, 5)
			if tc.setupMemory != nil {
				tc.setupMemory(memory)
			}

			_, err := memory.LoadMemoryVariables(ctx, tc.inputs)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedError)
		})
	}
}

// TestVectorStoreMemory_SaveContext tests saving context
func TestVectorStoreMemory_SaveContext(t *testing.T) {
	ctx := context.Background()
	retriever := NewMockRetriever()

	// Mock retriever AddDocuments
	var savedDocs []schema.Document
	retriever.addDocumentsFunc = func(ctx context.Context, documents []schema.Document) ([]string, error) {
		savedDocs = documents
		return []string{"doc_1"}, nil
	}

	memory := NewVectorStoreMemory(retriever, "memory", false, 5)
	memory.InputKey = "input"
	memory.OutputKey = "output"

	inputs := map[string]any{"input": "Hello"}
	outputs := map[string]any{"output": "Hi there!"}

	err := memory.SaveContext(ctx, inputs, outputs)
	assert.NoError(t, err)

	// Verify document was saved
	assert.Len(t, savedDocs, 1)
	assert.Contains(t, savedDocs[0].PageContent, "Input: Hello")
	assert.Contains(t, savedDocs[0].PageContent, "Output: Hi there!")
}

// TestVectorStoreMemory_SaveContext_AutoDetectKeys tests automatic key detection
func TestVectorStoreMemory_SaveContext_AutoDetectKeys(t *testing.T) {
	ctx := context.Background()
	retriever := NewMockRetriever()

	retriever.addDocumentsFunc = func(ctx context.Context, documents []schema.Document) ([]string, error) {
		return []string{"doc_1"}, nil
	}

	memory := NewVectorStoreMemory(retriever, "memory", false, 5)
	// Clear keys to trigger auto-detection
	memory.InputKey = ""
	memory.OutputKey = ""

	inputs := map[string]any{"query": "Hello"}
	outputs := map[string]any{"response": "Hi there!"}

	err := memory.SaveContext(ctx, inputs, outputs)
	assert.NoError(t, err)
}

// TestVectorStoreMemory_SaveContext_ErrorHandling tests error handling in SaveContext
func TestVectorStoreMemory_SaveContext_ErrorHandling(t *testing.T) {
	testCases := []struct {
		name           string
		setupMemory    func(*VectorStoreMemory)
		setupRetriever func(*MockRetriever)
		inputs         map[string]any
		outputs        map[string]any
		expectedError  string
	}{
		{
			name: "Missing input key",
			setupMemory: func(m *VectorStoreMemory) {
				m.InputKey = "missing_key"
			},
			inputs:        map[string]any{"other_key": "value"},
			outputs:       map[string]any{"output": "response"},
			expectedError: "error",
		},
		{
			name: "Missing output key",
			setupMemory: func(m *VectorStoreMemory) {
				m.OutputKey = "missing_key"
			},
			inputs:        map[string]any{"input": "query"},
			outputs:       map[string]any{"other_key": "response"},
			expectedError: "error",
		},
		{
			name: "Non-string input",
			setupMemory: func(m *VectorStoreMemory) {
				m.InputKey = "input"
				m.OutputKey = "output"
			},
			inputs:        map[string]any{"input": 123},
			outputs:       map[string]any{"output": "response"},
			expectedError: "error",
		},
		{
			name: "Non-string output",
			setupMemory: func(m *VectorStoreMemory) {
				m.InputKey = "input"
				m.OutputKey = "output"
			},
			inputs:        map[string]any{"input": "query"},
			outputs:       map[string]any{"output": 456},
			expectedError: "error",
		},
		{
			name: "Retriever does not support AddDocuments",
			setupRetriever: func(r *MockRetriever) {
				// Remove AddDocuments method by using a different retriever
				r.addDocumentsFunc = nil
			},
			inputs:        map[string]any{"input": "query"},
			outputs:       map[string]any{"output": "response"},
			expectedError: "", // This should not error, just log a warning
		},
		{
			name: "AddDocuments error",
			setupRetriever: func(r *MockRetriever) {
				r.addDocumentsError = errors.New("add documents failed")
			},
			inputs:        map[string]any{"input": "query"},
			outputs:       map[string]any{"output": "response"},
			expectedError: "error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			retriever := NewMockRetriever()
			if tc.setupRetriever != nil {
				tc.setupRetriever(retriever)
			}

			memory := NewVectorStoreMemory(retriever, "memory", false, 5)
			if tc.setupMemory != nil {
				tc.setupMemory(memory)
			}

			err := memory.SaveContext(ctx, tc.inputs, tc.outputs)
			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestVectorStoreMemory_Clear tests the Clear method
func TestVectorStoreMemory_Clear(t *testing.T) {
	ctx := context.Background()
	retriever := NewMockRetriever()
	memory := NewVectorStoreMemory(retriever, "memory", false, 5)

	// Clear should not error (it's currently a no-op)
	err := memory.Clear(ctx)
	assert.NoError(t, err)
}

// TestFormatDocuments tests the formatDocuments function
func TestFormatDocuments(t *testing.T) {
	docs := []schema.Message{
		schema.NewHumanMessage("Document 1"),
		schema.NewAIMessage("Document 2"),
	}

	result := formatDocuments(docs)
	assert.Contains(t, result, "Relevant context:")
	assert.Contains(t, result, "- Document 1")
	assert.Contains(t, result, "- Document 2")
}

// TestFormatDocuments_Empty tests with empty document list
func TestFormatDocuments_Empty(t *testing.T) {
	result := formatDocuments([]schema.Message{})
	assert.Equal(t, "Relevant context:\n", result)
}

// TestNewVectorStoreRetrieverMemory tests the constructor
func TestNewVectorStoreRetrieverMemory(t *testing.T) {
	embedder := NewMockEmbedder()
	vectorStore := NewMockVectorStore()

	memory := NewVectorStoreRetrieverMemory(embedder, vectorStore)

	assert.NotNil(t, memory)
	assert.Equal(t, embedder, memory.Embedder)
	assert.Equal(t, vectorStore, memory.VectorStore)
	assert.Equal(t, "history", memory.MemoryKey)
	assert.Equal(t, "input", memory.InputKey)
	assert.Equal(t, "output", memory.OutputKey)
	assert.False(t, memory.ReturnDocs)
	assert.False(t, memory.ExcludeInputKey)
	assert.Equal(t, 5, memory.TopK)
}

// TestNewVectorStoreRetrieverMemory_WithOptions tests constructor with options
func TestNewVectorStoreRetrieverMemory_WithOptions(t *testing.T) {
	embedder := NewMockEmbedder()
	vectorStore := NewMockVectorStore()

	memory := NewVectorStoreRetrieverMemory(embedder, vectorStore,
		WithMemoryKey("custom_memory"),
		WithInputKey("query"),
		WithOutputKey("response"),
		WithReturnDocs(true),
		WithExcludeInputKey(true),
		WithK(10),
	)

	assert.Equal(t, "custom_memory", memory.MemoryKey)
	assert.Equal(t, "query", memory.InputKey)
	assert.Equal(t, "response", memory.OutputKey)
	assert.True(t, memory.ReturnDocs)
	assert.True(t, memory.ExcludeInputKey)
	assert.Equal(t, 10, memory.TopK)
}

// TestVectorStoreRetrieverMemory_MemoryVariables tests the MemoryVariables method
func TestVectorStoreRetrieverMemory_MemoryVariables(t *testing.T) {
	embedder := NewMockEmbedder()
	vectorStore := NewMockVectorStore()

	memory := NewVectorStoreRetrieverMemory(embedder, vectorStore, WithMemoryKey("test_memory"))

	variables := memory.MemoryVariables()
	assert.Equal(t, []string{"test_memory"}, variables)
}

// TestVectorStoreRetrieverMemory_LoadMemoryVariables tests loading memory variables
func TestVectorStoreRetrieverMemory_LoadMemoryVariables(t *testing.T) {
	ctx := context.Background()
	embedder := NewMockEmbedder()
	vectorStore := NewMockVectorStore()

	// Mock vector store to return specific documents
	vectorStore.similaritySearchByQueryFunc = func(ctx context.Context, query string, k int, embedder vectorstores.Embedder, options ...vectorstores.Option) ([]schema.Document, []float32, error) {
		return []schema.Document{
			{PageContent: "Relevant document 1"},
			{PageContent: "Relevant document 2"},
		}, []float32{0.9, 0.8}, nil
	}

	memory := NewVectorStoreRetrieverMemory(embedder, vectorStore, WithReturnDocs(true))

	inputs := map[string]any{"input": "test query"}

	vars, err := memory.LoadMemoryVariables(ctx, inputs)
	assert.NoError(t, err)
	assert.Contains(t, vars, "history")
	assert.Contains(t, vars, "retrieved_docs")

	// Check formatted string
	history, ok := vars["history"].(string)
	assert.True(t, ok)
	assert.Contains(t, history, "Relevant document 1")
	assert.Contains(t, history, "Relevant document 2")

	// Check documents
	docs, ok := vars["retrieved_docs"].([]schema.Document)
	assert.True(t, ok)
	assert.Len(t, docs, 2)
}

// TestVectorStoreRetrieverMemory_LoadMemoryVariables_NoReturnDocs tests loading without returning docs
func TestVectorStoreRetrieverMemory_LoadMemoryVariables_NoReturnDocs(t *testing.T) {
	ctx := context.Background()
	embedder := NewMockEmbedder()
	vectorStore := NewMockVectorStore()

	vectorStore.similaritySearchByQueryFunc = func(ctx context.Context, query string, k int, embedder vectorstores.Embedder, options ...vectorstores.Option) ([]schema.Document, []float32, error) {
		return []schema.Document{
			{PageContent: "Document 1"},
		}, []float32{0.9}, nil
	}

	memory := NewVectorStoreRetrieverMemory(embedder, vectorStore, WithReturnDocs(false))

	inputs := map[string]any{"input": "test query"}

	vars, err := memory.LoadMemoryVariables(ctx, inputs)
	assert.NoError(t, err)
	assert.Contains(t, vars, "history")
	assert.NotContains(t, vars, "retrieved_docs")

	history, ok := vars["history"].(string)
	assert.True(t, ok)
	assert.Contains(t, history, "Document 1")
}

// TestVectorStoreRetrieverMemory_LoadMemoryVariables_ErrorHandling tests error handling
func TestVectorStoreRetrieverMemory_LoadMemoryVariables_ErrorHandling(t *testing.T) {
	testCases := []struct {
		name             string
		setupMemory      func(*VectorStoreRetrieverMemory)
		setupVectorStore func(*MockVectorStore)
		inputs           map[string]any
		expectedError    string
	}{
		{
			name:          "Non-string input",
			inputs:        map[string]any{"input": 123},
			expectedError: "error",
		},
		{
			name: "Vector store error",
			setupVectorStore: func(vs *MockVectorStore) {
				vs.similaritySearchError = errors.New("similarity search failed")
			},
			inputs:        map[string]any{"input": "query"},
			expectedError: "error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			embedder := NewMockEmbedder()
			vectorStore := NewMockVectorStore()
			if tc.setupVectorStore != nil {
				tc.setupVectorStore(vectorStore)
			}

			memory := NewVectorStoreRetrieverMemory(embedder, vectorStore)
			if tc.setupMemory != nil {
				tc.setupMemory(memory)
			}

			_, err := memory.LoadMemoryVariables(ctx, tc.inputs)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedError)
		})
	}
}

// TestVectorStoreRetrieverMemory_SaveContext tests saving context
func TestVectorStoreRetrieverMemory_SaveContext(t *testing.T) {
	ctx := context.Background()
	embedder := NewMockEmbedder()
	vectorStore := NewMockVectorStore()

	// Mock vector store AddDocuments
	var savedDocs []schema.Document
	vectorStore.addDocumentsFunc = func(ctx context.Context, documents []schema.Document, options ...vectorstores.Option) ([]string, error) {
		savedDocs = documents
		return []string{"doc_1"}, nil
	}

	memory := NewVectorStoreRetrieverMemory(embedder, vectorStore)

	inputs := map[string]any{"input": "Hello"}
	outputs := map[string]any{"output": "Hi there!"}

	err := memory.SaveContext(ctx, inputs, outputs)
	assert.NoError(t, err)

	// Verify document was saved
	assert.Len(t, savedDocs, 1)
	assert.Contains(t, savedDocs[0].PageContent, "Input: Hello")
	assert.Contains(t, savedDocs[0].PageContent, "Output: Hi there!")
	assert.Equal(t, "Hello", savedDocs[0].Metadata["input"])
	assert.Equal(t, "Hi there!", savedDocs[0].Metadata["output"])
	assert.Equal(t, "conversation", savedDocs[0].Metadata["source"])
}

// TestVectorStoreRetrieverMemory_SaveContext_ErrorHandling tests error handling in SaveContext
func TestVectorStoreRetrieverMemory_SaveContext_ErrorHandling(t *testing.T) {
	testCases := []struct {
		name             string
		setupVectorStore func(*MockVectorStore)
		inputs           map[string]any
		outputs          map[string]any
		expectedError    string
	}{
		{
			name:          "Non-string input",
			inputs:        map[string]any{"input": 123},
			outputs:       map[string]any{"output": "response"},
			expectedError: "error",
		},
		{
			name:          "Non-string output",
			inputs:        map[string]any{"input": "query"},
			outputs:       map[string]any{"output": 456},
			expectedError: "error",
		},
		{
			name: "Vector store error",
			setupVectorStore: func(vs *MockVectorStore) {
				vs.addDocumentsError = errors.New("add documents failed")
			},
			inputs:        map[string]any{"input": "query"},
			outputs:       map[string]any{"output": "response"},
			expectedError: "error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			embedder := NewMockEmbedder()
			vectorStore := NewMockVectorStore()
			if tc.setupVectorStore != nil {
				tc.setupVectorStore(vectorStore)
			}

			memory := NewVectorStoreRetrieverMemory(embedder, vectorStore)

			err := memory.SaveContext(ctx, tc.inputs, tc.outputs)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedError)
		})
	}
}

// TestVectorStoreRetrieverMemory_Clear tests the Clear method
func TestVectorStoreRetrieverMemory_Clear(t *testing.T) {
	ctx := context.Background()
	embedder := NewMockEmbedder()
	vectorStore := NewMockVectorStore()

	memory := NewVectorStoreRetrieverMemory(embedder, vectorStore)

	// Clear should not error (it's currently a no-op with logging)
	err := memory.Clear(ctx)
	assert.NoError(t, err)
}

// TestInterfaceCompliance tests that both implementations comply with the Memory interface
func TestVectorStoreInterfaceCompliance(t *testing.T) {
	embedder := NewMockEmbedder()
	vectorStore := NewMockVectorStore()

	// Test VectorStoreMemory
	retriever := NewMockRetriever()
	vectorStoreMemory := NewVectorStoreMemory(retriever, "memory", false, 5)
	var _ iface.Memory = vectorStoreMemory

	// Test VectorStoreRetrieverMemory
	vectorStoreRetrieverMemory := NewVectorStoreRetrieverMemory(embedder, vectorStore)
	var _ iface.Memory = vectorStoreRetrieverMemory
}

// BenchmarkVectorStoreMemory_LoadMemoryVariables benchmarks LoadMemoryVariables performance
func BenchmarkVectorStoreMemory_LoadMemoryVariables(b *testing.B) {
	ctx := context.Background()
	retriever := NewMockRetriever()

	retriever.invokeFunc = func(ctx context.Context, input any, options ...core.Option) (any, error) {
		return []schema.Document{
			{PageContent: "Benchmark document 1"},
			{PageContent: "Benchmark document 2"},
		}, nil
	}

	memory := NewVectorStoreMemory(retriever, "memory", false, 5)
	memory.InputKey = "query"

	inputs := map[string]any{"query": "test query"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		memory.LoadMemoryVariables(ctx, inputs)
	}
}

// BenchmarkVectorStoreRetrieverMemory_LoadMemoryVariables benchmarks LoadMemoryVariables performance
func BenchmarkVectorStoreRetrieverMemory_LoadMemoryVariables(b *testing.B) {
	ctx := context.Background()
	embedder := NewMockEmbedder()
	vectorStore := NewMockVectorStore()

	vectorStore.similaritySearchByQueryFunc = func(ctx context.Context, query string, k int, embedder vectorstores.Embedder, options ...vectorstores.Option) ([]schema.Document, []float32, error) {
		return []schema.Document{
			{PageContent: "Benchmark document"},
		}, []float32{0.9}, nil
	}

	memory := NewVectorStoreRetrieverMemory(embedder, vectorStore)

	inputs := map[string]any{"input": "test query"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		memory.LoadMemoryVariables(ctx, inputs)
	}
}
