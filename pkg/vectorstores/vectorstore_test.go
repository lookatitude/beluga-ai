package vectorstores

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
)

// MockEmbedder is a mock implementation of the iface.Embedder for testing.
type MockEmbedder struct {
	MockEmbedDocuments func(ctx context.Context, texts []string) ([][]float32, error)
	MockEmbedQuery    func(ctx context.Context, text string) ([]float32, error)
	MockGetDimension  func(ctx context.Context) (int, error)
}

func (m *MockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if m.MockEmbedDocuments != nil {
		return m.MockEmbedDocuments(ctx, texts)
	}
	return nil, nil
}

func (m *MockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	if m.MockEmbedQuery != nil {
		return m.MockEmbedQuery(ctx, text)
	}
	return nil, nil
}

func (m *MockEmbedder) GetDimension(ctx context.Context) (int, error) {
	if m.MockGetDimension != nil {
		return m.MockGetDimension(ctx)
	}
	return 0, nil
}

// MockVectorStore is a mock implementation of the VectorStore interface for testing.
type MockVectorStore struct {
	MockAddDocuments            func(ctx context.Context, docs []schema.Document, embedder iface.Embedder) error
	MockSimilaritySearch        func(ctx context.Context, queryVector []float32, k int) ([]schema.Document, []float32, error)
	MockSimilaritySearchByQuery func(ctx context.Context, query string, k int, embedder iface.Embedder) ([]schema.Document, []float32, error)
	MockGetName                 func() string
}

func (m *MockVectorStore) AddDocuments(ctx context.Context, docs []schema.Document, embedder iface.Embedder) error {
	if m.MockAddDocuments != nil {
		return m.MockAddDocuments(ctx, docs, embedder)
	}
	return nil
}

func (m *MockVectorStore) SimilaritySearch(ctx context.Context, queryVector []float32, k int) ([]schema.Document, []float32, error) {
	if m.MockSimilaritySearch != nil {
		return m.MockSimilaritySearch(ctx, queryVector, k)
	}
	return nil, nil, nil
}

func (m *MockVectorStore) SimilaritySearchByQuery(ctx context.Context, query string, k int, embedder iface.Embedder) ([]schema.Document, []float32, error) {
	if m.MockSimilaritySearchByQuery != nil {
		return m.MockSimilaritySearchByQuery(ctx, query, k, embedder)
	}
	return nil, nil, nil
}

func (m *MockVectorStore) GetName() string {
	if m.MockGetName != nil {
		return m.MockGetName()
	}
	return "mock"
}

func TestConfig(t *testing.T) {
	config := Config{
		Type: "test_type",
		Name: "test_name",
		ProviderArgs: map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		},
	}

	assert.Equal(t, "test_type", config.Type)
	assert.Equal(t, "test_name", config.Name)
	assert.Equal(t, "value1", config.ProviderArgs["key1"])
	assert.Equal(t, 123, config.ProviderArgs["key2"])
}

// TestVectorStoreInterface can be used to ensure mock or actual implementations adhere to the interface.
// This is more of a compile-time check and a pattern for testing interface satisfaction.
func TestVectorStoreInterface(t *testing.T) {
	var _ VectorStore = (*MockVectorStore)(nil)
	// To test a real implementation (e.g., InMemoryVectorStore if it were in this package):
	// var _ VectorStore = (*InMemoryVectorStore)(nil) // Assuming InMemoryVectorStore exists and implements VectorStore
}

// TestFactoryInterface can be used to ensure mock or actual factory implementations adhere to the interface.
func TestFactoryInterface(t *testing.T) {
	// Example of how you might test a factory implementation if one existed in this package
	// type MockFactory struct{}
	// func (mf *MockFactory) CreateVectorStore(ctx context.Context, config Config) (VectorStore, error) {
	// 	 return &MockVectorStore{}, nil
	// }
	// var _ Factory = (*MockFactory)(nil)
	assert.True(t, true, "Placeholder test for Factory interface. Actual tests would involve a concrete factory implementation.")
}

