package vectorstores

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInterfaceCompliance tests that implementations properly satisfy interfaces.
func TestInterfaceCompliance(t *testing.T) {
	ctx := context.Background()

	t.Run("VectorStoreInterface", func(t *testing.T) {
		// Test that MockVectorStore implements VectorStore
		var _ VectorStore = (*MockVectorStore)(nil)

		// Test that inmemory.InMemoryVectorStore would implement VectorStore
		// (We can't directly test this due to import cycles, but this is a compile-time check)
		store := &MockVectorStore{}

		// Test all interface methods exist and are callable
		docs := []schema.Document{schema.NewDocument("test", nil)}

		_, err := store.AddDocuments(ctx, docs)
		require.NoError(t, err)

		_, _, err = store.SimilaritySearch(ctx, []float32{0.1, 0.2, 0.3}, 5)
		require.NoError(t, err)

		_, _, err = store.SimilaritySearchByQuery(ctx, "test", 5, &MockEmbedder{})
		require.NoError(t, err)

		err = store.DeleteDocuments(ctx, []string{"test-id"})
		require.NoError(t, err)

		retriever := store.AsRetriever()
		assert.NotNil(t, retriever)

		name := store.GetName()
		assert.NotEmpty(t, name)
	})

	t.Run("RetrieverInterface", func(t *testing.T) {
		// Test that MockRetriever implements Retriever
		var _ Retriever = (*MockRetriever)(nil)

		retriever := &MockRetriever{}

		// Test interface method exists and is callable
		_, err := retriever.GetRelevantDocuments(ctx, "test query")
		require.NoError(t, err)
	})

	t.Run("EmbedderInterface", func(t *testing.T) {
		// Test that MockEmbedder implements Embedder
		var _ Embedder = (*MockEmbedder)(nil)

		embedder := &MockEmbedder{}

		// Test interface methods exist and are callable
		_, err := embedder.EmbedDocuments(ctx, []string{"test"})
		require.NoError(t, err)

		_, err = embedder.EmbedQuery(ctx, "test query")
		require.NoError(t, err)
	})
}

// TestConfigInterfaceCompliance tests that Config works with all expected options.
func TestConfigInterfaceCompliance(t *testing.T) {
	config := &Config{}

	// Test that all option functions can be applied
	options := []Option{
		WithSearchK(10),
		WithScoreThreshold(0.8),
		WithEmbedder(&MockEmbedder{}),
		WithMetadataFilter("key", "value"),
		WithMetadataFilters(map[string]any{"k1": "v1", "k2": "v2"}),
		WithProviderConfig("config_key", "config_value"),
		WithProviderConfigs(map[string]any{"c1": "v1", "c2": "v2"}),
	}

	// Apply all options - this tests that the Option type is compatible
	for _, opt := range options {
		opt(config)
	}

	// Verify the config was modified
	assert.Equal(t, 10, config.SearchK)
	assert.Equal(t, float32(0.8), config.ScoreThreshold)
	assert.NotNil(t, config.Embedder)
	assert.Contains(t, config.MetadataFilters, "key")
	assert.Contains(t, config.ProviderConfig, "config_key")
}

// TestStoreFactoryCompliance tests StoreFactory functionality.
func TestStoreFactoryCompliance(t *testing.T) {
	ctx := context.Background()

	factory := NewStoreFactory()
	assert.NotNil(t, factory)

	// Test interface methods exist
	config := NewDefaultConfig()

	mockCreator := func(ctx context.Context, config Config) (VectorStore, error) {
		return &MockVectorStore{}, nil
	}

	factory.Register("test", mockCreator)

	store, err := factory.Create(ctx, "test", *config)
	require.NoError(t, err)
	assert.NotNil(t, store)
	assert.Equal(t, "mock", store.GetName())
}

// TestTypeAssertions tests that our mock implementations can be properly type-asserted.
func TestTypeAssertions(t *testing.T) {
	t.Run("VectorStoreTypeAssertion", func(t *testing.T) {
		var storeInterface VectorStore = &MockVectorStore{}

		// Test that we can assert back to concrete type
		mockStore, ok := storeInterface.(*MockVectorStore)
		assert.True(t, ok)
		assert.NotNil(t, mockStore)
	})

	t.Run("RetrieverTypeAssertion", func(t *testing.T) {
		var retrieverInterface Retriever = &MockRetriever{}

		// Test that we can assert back to concrete type
		mockRetriever, ok := retrieverInterface.(*MockRetriever)
		assert.True(t, ok)
		assert.NotNil(t, mockRetriever)
	})

	t.Run("EmbedderTypeAssertion", func(t *testing.T) {
		var embedderInterface Embedder = &MockEmbedder{}

		// Test that we can assert back to concrete type
		mockEmbedder, ok := embedderInterface.(*MockEmbedder)
		assert.True(t, ok)
		assert.NotNil(t, mockEmbedder)
	})
}

// TestInterfaceMethodSignatures tests that interface methods have correct signatures.
func TestInterfaceMethodSignatures(t *testing.T) {
	ctx := context.Background()

	t.Run("VectorStoreMethodSignatures", func(t *testing.T) {
		store := &MockVectorStore{}

		// Test method signatures by calling them with correct parameter types
		docs := []schema.Document{schema.NewDocument("test", nil)}
		vector := []float32{0.1, 0.2, 0.3}
		embedder := &MockEmbedder{}
		ids := []string{"test-id"}
		opts := []Option{}

		// These should compile and run without signature errors
		_, _ = store.AddDocuments(ctx, docs, opts...)
		_, _, _ = store.SimilaritySearch(ctx, vector, 5, opts...)
		_, _, _ = store.SimilaritySearchByQuery(ctx, "query", 5, embedder, opts...)
		_ = store.DeleteDocuments(ctx, ids, opts...)
		_ = store.AsRetriever(opts...)
		_ = store.GetName()
	})

	t.Run("RetrieverMethodSignatures", func(t *testing.T) {
		retriever := &MockRetriever{}

		// Test method signature
		_, _ = retriever.GetRelevantDocuments(ctx, "query")
	})

	t.Run("EmbedderMethodSignatures", func(t *testing.T) {
		embedder := &MockEmbedder{}

		// Test method signatures
		_, _ = embedder.EmbedDocuments(ctx, []string{"test"})
		_, _ = embedder.EmbedQuery(ctx, "query")
	})
}

// TestNilInterfaceHandling tests how interfaces handle nil implementations.
func TestNilInterfaceHandling(t *testing.T) {
	t.Run("NilVectorStore", func(t *testing.T) {
		var store VectorStore

		// Test that nil interface doesn't panic on method calls
		// (This would panic if called, but we're just testing the interface definition)
		assert.Nil(t, store)
	})

	t.Run("NilRetriever", func(t *testing.T) {
		var retriever Retriever
		assert.Nil(t, retriever)
	})

	t.Run("NilEmbedder", func(t *testing.T) {
		var embedder Embedder
		assert.Nil(t, embedder)
	})
}

// Mock implementations for interface compliance testing.
type MockVectorStore struct {
	addDocumentsFunc            func(ctx context.Context, documents []schema.Document, opts ...Option) ([]string, error)
	deleteDocumentsFunc         func(ctx context.Context, ids []string, opts ...Option) error
	similaritySearchFunc        func(ctx context.Context, queryVector []float32, k int, opts ...Option) ([]schema.Document, []float32, error)
	similaritySearchByQueryFunc func(ctx context.Context, query string, k int, embedder Embedder, opts ...Option) ([]schema.Document, []float32, error)
	asRetrieverFunc             func(opts ...Option) Retriever
	getNameFunc                 func() string
}

func (m *MockVectorStore) AddDocuments(ctx context.Context, documents []schema.Document, opts ...Option) ([]string, error) {
	if m.addDocumentsFunc != nil {
		return m.addDocumentsFunc(ctx, documents, opts...)
	}
	return []string{"mock-id"}, nil
}

func (m *MockVectorStore) DeleteDocuments(ctx context.Context, ids []string, opts ...Option) error {
	if m.deleteDocumentsFunc != nil {
		return m.deleteDocumentsFunc(ctx, ids, opts...)
	}
	return nil
}

func (m *MockVectorStore) SimilaritySearch(ctx context.Context, queryVector []float32, k int, opts ...Option) ([]schema.Document, []float32, error) {
	if m.similaritySearchFunc != nil {
		return m.similaritySearchFunc(ctx, queryVector, k, opts...)
	}
	return []schema.Document{}, []float32{}, nil
}

func (m *MockVectorStore) SimilaritySearchByQuery(ctx context.Context, query string, k int, embedder Embedder, opts ...Option) ([]schema.Document, []float32, error) {
	if m.similaritySearchByQueryFunc != nil {
		return m.similaritySearchByQueryFunc(ctx, query, k, embedder, opts...)
	}
	return []schema.Document{}, []float32{}, nil
}

func (m *MockVectorStore) AsRetriever(opts ...Option) Retriever {
	if m.asRetrieverFunc != nil {
		return m.asRetrieverFunc(opts...)
	}
	return &MockRetriever{}
}

func (m *MockVectorStore) GetName() string {
	if m.getNameFunc != nil {
		return m.getNameFunc()
	}
	return "mock"
}

type MockRetriever struct {
	getRelevantDocumentsFunc func(ctx context.Context, query string) ([]schema.Document, error)
}

func (m *MockRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
	if m.getRelevantDocumentsFunc != nil {
		return m.getRelevantDocumentsFunc(ctx, query)
	}
	return []schema.Document{}, nil
}

type MockEmbedder struct {
	embedDocumentsFunc func(ctx context.Context, texts []string) ([][]float32, error)
	embedQueryFunc     func(ctx context.Context, text string) ([]float32, error)
}

func (m *MockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if m.embedDocumentsFunc != nil {
		return m.embedDocumentsFunc(ctx, texts)
	}
	result := make([][]float32, len(texts))
	for i := range result {
		result[i] = []float32{0.1, 0.2, 0.3}
	}
	return result, nil
}

func (m *MockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	if m.embedQueryFunc != nil {
		return m.embedQueryFunc(ctx, text)
	}
	return []float32{0.1, 0.2, 0.3}, nil
}
