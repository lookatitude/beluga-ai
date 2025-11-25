package vectorstores

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/schema"
	vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

// MockEmbedder for testing with configurable behavior
type MockEmbedder struct {
	embedDocumentsFunc      func(ctx context.Context, texts []string) ([][]float32, error)
	embedQueryFunc          func(ctx context.Context, text string) ([]float32, error)
	embedDocumentsCallCount int
	embedQueryCallCount     int
	mu                      sync.RWMutex
}

func (m *MockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	m.mu.Lock()
	m.embedDocumentsCallCount++
	m.mu.Unlock()

	if m.embedDocumentsFunc != nil {
	ctx := context.Background()
		return m.embedDocumentsFunc(ctx, texts)
	}
	// Return mock embeddings
	result := make([][]float32, len(texts))
	for i := range result {
		result[i] = []float32{0.1, 0.2, 0.3}
	}
	return result, nil
}

func (m *MockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	m.mu.Lock()
	m.embedQueryCallCount++
	m.mu.Unlock()

	if m.embedQueryFunc != nil {
		return m.embedQueryFunc(ctx, text)
	}
	return []float32{0.1, 0.2, 0.3}, nil
}

func (m *MockEmbedder) GetEmbedDocumentsCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.embedDocumentsCallCount
}

func (m *MockEmbedder) GetEmbedQueryCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.embedQueryCallCount
}

// MockVectorStore for testing
type MockVectorStore struct {
	addDocumentsFunc            func(ctx context.Context, documents []schema.Document, opts ...Option) ([]string, error)
	deleteDocumentsFunc         func(ctx context.Context, ids []string, opts ...Option) error
	similaritySearchFunc        func(ctx context.Context, queryVector []float32, k int, opts ...Option) ([]schema.Document, []float32, error)
	similaritySearchByQueryFunc func(ctx context.Context, query string, k int, embedder Embedder, opts ...Option) ([]schema.Document, []float32, error)
	asRetrieverFunc             func(opts ...Option) Retriever
	getNameFunc                 func() string

	// Call counters for verification
	addDocumentsCount            int
	deleteDocumentsCount         int
	similaritySearchCount        int
	similaritySearchByQueryCount int
	asRetrieverCount             int
	getNameCount                 int
	mu                           sync.RWMutex
}

func (m *MockVectorStore) AddDocuments(ctx context.Context, documents []schema.Document, opts ...Option) ([]string, error) {
	m.mu.Lock()
	m.addDocumentsCount++
	m.mu.Unlock()

	if m.addDocumentsFunc != nil {
		return m.addDocumentsFunc(ctx, documents, opts...)
	}
	ids := make([]string, len(documents))
	for i := range ids {
		ids[i] = "mock-id-" + string(rune(i+'a'))
	}
	return ids, nil
}

func (m *MockVectorStore) DeleteDocuments(ctx context.Context, ids []string, opts ...Option) error {
	m.mu.Lock()
	m.deleteDocumentsCount++
	m.mu.Unlock()

	if m.deleteDocumentsFunc != nil {
		return m.deleteDocumentsFunc(ctx, ids, opts...)
	}
	return nil
}

func (m *MockVectorStore) SimilaritySearch(ctx context.Context, queryVector []float32, k int, opts ...Option) ([]schema.Document, []float32, error) {
	m.mu.Lock()
	m.similaritySearchCount++
	m.mu.Unlock()

	if m.similaritySearchFunc != nil {
		return m.similaritySearchFunc(ctx, queryVector, k, opts...)
	}
	return []schema.Document{}, []float32{}, nil
}

func (m *MockVectorStore) SimilaritySearchByQuery(ctx context.Context, query string, k int, embedder Embedder, opts ...Option) ([]schema.Document, []float32, error) {
	m.mu.Lock()
	m.similaritySearchByQueryCount++
	m.mu.Unlock()

	if m.similaritySearchByQueryFunc != nil {
		return m.similaritySearchByQueryFunc(ctx, query, k, embedder, opts...)
	}
	return []schema.Document{}, []float32{}, nil
}

func (m *MockVectorStore) AsRetriever(opts ...Option) Retriever {
	m.mu.Lock()
	m.asRetrieverCount++
	m.mu.Unlock()

	if m.asRetrieverFunc != nil {
		return m.asRetrieverFunc(opts...)
	}
	return &MockRetriever{}
}

func (m *MockVectorStore) GetName() string {
	m.mu.Lock()
	m.getNameCount++
	m.mu.Unlock()

	if m.getNameFunc != nil {
		return m.getNameFunc()
	}
	return "mock"
}

// MockRetriever for testing
type MockRetriever struct {
	getRelevantDocumentsFunc  func(ctx context.Context, query string) ([]schema.Document, error)
	getRelevantDocumentsCount int
	mu                        sync.RWMutex
}

func (m *MockRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
	m.mu.Lock()
	m.getRelevantDocumentsCount++
	m.mu.Unlock()

	if m.getRelevantDocumentsFunc != nil {
		return m.getRelevantDocumentsFunc(ctx, query)
	}
	return []schema.Document{}, nil
}

// MockMetricsCollector for testing observability
type MockMetricsCollector struct {
	recordDocumentsAddedCount   int
	recordDocumentsDeletedCount int
	recordSearchCount           int
	recordEmbeddingCount        int
	recordErrorCount            int
	recordMemoryUsageCount      int
	recordDiskUsageCount        int
	mu                          sync.RWMutex
	lastStoreName               string
	lastCount                   int
	lastErrorCode               string
}

func (m *MockMetricsCollector) RecordDocumentsAdded(ctx context.Context, count int, storeName string) {
	m.mu.Lock()
	m.recordDocumentsAddedCount++
	m.lastStoreName = storeName
	m.lastCount = count
	m.mu.Unlock()
}

func (m *MockMetricsCollector) RecordDocumentsDeleted(ctx context.Context, count int, storeName string) {
	m.mu.Lock()
	m.recordDocumentsDeletedCount++
	m.lastStoreName = storeName
	m.lastCount = count
	m.mu.Unlock()
}

func (m *MockMetricsCollector) RecordSearch(ctx context.Context, duration time.Duration, resultCount int, storeName string) {
	m.mu.Lock()
	m.recordSearchCount++
	m.lastStoreName = storeName
	m.mu.Unlock()
}

func (m *MockMetricsCollector) RecordEmbedding(ctx context.Context, duration time.Duration, textCount int, storeName string) {
	m.mu.Lock()
	m.recordEmbeddingCount++
	m.lastStoreName = storeName
	m.mu.Unlock()
}

func (m *MockMetricsCollector) RecordError(ctx context.Context, errorCode string, storeName string) {
	m.mu.Lock()
	m.recordErrorCount++
	m.lastErrorCode = errorCode
	m.lastStoreName = storeName
	m.mu.Unlock()
}

func (m *MockMetricsCollector) RecordMemoryUsage(ctx context.Context, bytes int64, storeName string) {
	m.mu.Lock()
	m.recordMemoryUsageCount++
	m.lastStoreName = storeName
	m.mu.Unlock()
}

func (m *MockMetricsCollector) RecordDiskUsage(ctx context.Context, bytes int64, storeName string) {
	m.mu.Lock()
	m.recordDiskUsageCount++
	m.lastStoreName = storeName
	m.mu.Unlock()
}

func (m *MockMetricsCollector) GetRecordDocumentsAddedCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.recordDocumentsAddedCount
}

func (m *MockMetricsCollector) GetRecordDocumentsDeletedCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.recordDocumentsDeletedCount
}

func (m *MockMetricsCollector) GetRecordErrorCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.recordErrorCount
}

func (m *MockMetricsCollector) GetLastStoreName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastStoreName
}

func (m *MockMetricsCollector) GetLastErrorCode() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastErrorCode
}

// MockTracerProvider for testing observability
type MockTracerProvider struct {
	startSpanCount            int
	startAddDocumentsCount    int
	startDeleteDocumentsCount int
	startSearchCount          int
	startEmbeddingCount       int
	mu                        sync.RWMutex
	lastOperation             string
}

func (m *MockTracerProvider) StartSpan(ctx context.Context, operation string, opts ...attribute.KeyValue) (context.Context, func()) {
	m.mu.Lock()
	m.startSpanCount++
	m.lastOperation = operation
	m.mu.Unlock()

	return ctx, func() {}
}

func (m *MockTracerProvider) StartAddDocumentsSpan(ctx context.Context, storeName string, docCount int) (context.Context, func()) {
	m.mu.Lock()
	m.startSpanCount++
	m.startAddDocumentsCount++
	m.lastOperation = "vectorstore.add_documents"
	m.mu.Unlock()

	return ctx, func() {}
}

func (m *MockTracerProvider) StartDeleteDocumentsSpan(ctx context.Context, storeName string, docCount int) (context.Context, func()) {
	m.mu.Lock()
	m.startSpanCount++
	m.startDeleteDocumentsCount++
	m.lastOperation = "vectorstore.delete_documents"
	m.mu.Unlock()

	return ctx, func() {}
}

func (m *MockTracerProvider) StartSearchSpan(ctx context.Context, storeName string, queryLength int, k int) (context.Context, func()) {
	m.mu.Lock()
	m.startSpanCount++
	m.startSearchCount++
	m.lastOperation = "vectorstore.search"
	m.mu.Unlock()

	return ctx, func() {}
}

func (m *MockTracerProvider) StartEmbeddingSpan(ctx context.Context, storeName string, textCount int) (context.Context, func()) {
	m.mu.Lock()
	m.startSpanCount++
	m.startEmbeddingCount++
	m.lastOperation = "vectorstore.embedding"
	m.mu.Unlock()

	return ctx, func() {}
}

func (m *MockTracerProvider) GetStartSpanCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.startSpanCount
}

func (m *MockTracerProvider) GetLastOperation() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastOperation
}

// MockLogger for testing logging functionality
type MockLogger struct {
	logAttrsCount int
	lastLevel     slog.Level
	lastMessage   string
	lastAttrs     []slog.Attr
	mu            sync.RWMutex
}

func (m *MockLogger) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (m *MockLogger) LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	m.mu.Lock()
	m.logAttrsCount++
	m.lastLevel = level
	m.lastMessage = msg
	m.lastAttrs = attrs
	m.mu.Unlock()
}

func (m *MockLogger) GetLogAttrsCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.logAttrsCount
}

func (m *MockLogger) GetLastMessage() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastMessage
}

func (m *MockLogger) GetLastLevel() slog.Level {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastLevel
}

// Test setup function to register providers
func setupTestProviders() {
	// For testing, we'll directly use the inmemory provider constructor
	// instead of relying on global registration to avoid import cycles
}

// Table-driven tests for VectorStore operations using mocks
func TestVectorStoreOperations(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name       string
		setupStore func() VectorStore
		testFunc   func(t *testing.T, store VectorStore)
	}{
		{
			name: "AddDocuments_DefaultBehavior",
			setupStore: func() VectorStore {
				return &MockVectorStore{}
			},
			testFunc: func(t *testing.T, store VectorStore) {
				docs := []schema.Document{
					schema.NewDocument("test content 1", map[string]string{"source": "test1"}),
					schema.NewDocument("test content 2", map[string]string{"source": "test2"}),
				}

				ids, err := store.AddDocuments(ctx, docs)
				require.NoError(t, err)
				assert.Len(t, ids, 2) // Mock returns one ID per document
				assert.Equal(t, "mock-id-a", ids[0])
				assert.Equal(t, "mock-id-b", ids[1])
			},
		},
		{
			name: "SimilaritySearch_DefaultBehavior",
			setupStore: func() VectorStore {
				return &MockVectorStore{}
			},
			testFunc: func(t *testing.T, store VectorStore) {
				queryVector := []float32{0.1, 0.2, 0.3}
				results, scores, err := store.SimilaritySearch(ctx, queryVector, 5)
				require.NoError(t, err)
				assert.Len(t, results, 0)
				assert.Len(t, scores, 0)
			},
		},
		{
			name: "SimilaritySearchByQuery_DefaultBehavior",
			setupStore: func() VectorStore {
				return &MockVectorStore{}
			},
			testFunc: func(t *testing.T, store VectorStore) {
				embedder := &MockEmbedder{}
				results, scores, err := store.SimilaritySearchByQuery(ctx, "machine learning", 5, embedder)
				require.NoError(t, err)
				assert.Len(t, results, 0)
				assert.Len(t, scores, 0)
			},
		},
		{
			name: "DeleteDocuments_DefaultBehavior",
			setupStore: func() VectorStore {
				return &MockVectorStore{}
			},
			testFunc: func(t *testing.T, store VectorStore) {
				err := store.DeleteDocuments(ctx, []string{"test-id"})
				assert.NoError(t, err)
			},
		},
		{
			name: "AsRetriever_DefaultBehavior",
			setupStore: func() VectorStore {
				return &MockVectorStore{}
			},
			testFunc: func(t *testing.T, store VectorStore) {
				retriever := store.AsRetriever()
				assert.NotNil(t, retriever)
				assert.Equal(t, "mock", store.GetName())
			},
		},
		{
			name: "CustomBehavior_AddDocuments",
			setupStore: func() VectorStore {
				return &MockVectorStore{
					addDocumentsFunc: func(ctx context.Context, documents []schema.Document, opts ...Option) ([]string, error) {
						return []string{"custom-id-1", "custom-id-2"}, nil
					},
				}
			},
			testFunc: func(t *testing.T, store VectorStore) {
				docs := []schema.Document{schema.NewDocument("test", nil)}
				ids, err := store.AddDocuments(ctx, docs)
				require.NoError(t, err)
				assert.Equal(t, []string{"custom-id-1", "custom-id-2"}, ids)
			},
		},
		{
			name: "ErrorSimulation_AddDocuments",
			setupStore: func() VectorStore {
				return &MockVectorStore{
					addDocumentsFunc: func(ctx context.Context, documents []schema.Document, opts ...Option) ([]string, error) {
						return nil, errors.New("add documents failed")
					},
				}
			},
			testFunc: func(t *testing.T, store VectorStore) {
				docs := []schema.Document{schema.NewDocument("test", nil)}
				_, err := store.AddDocuments(ctx, docs)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "add documents failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.setupStore()
			tt.testFunc(t, store)
		})
	}
}

// TestMockEmbedder tests the MockEmbedder functionality
func TestMockEmbedder(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name      string
		setupFunc func() *MockEmbedder
		testFunc  func(t *testing.T, embedder *MockEmbedder)
	}{
		{
			name: "DefaultEmbedDocuments",
			setupFunc: func() *MockEmbedder {
				return &MockEmbedder{}
			},
			testFunc: func(t *testing.T, embedder *MockEmbedder) {
				texts := []string{"test document 1", "test document 2"}
				embeddings, err := embedder.EmbedDocuments(ctx, texts)
				require.NoError(t, err)
				assert.Len(t, embeddings, 2)
				assert.Len(t, embeddings[0], 3)
				assert.Equal(t, []float32{0.1, 0.2, 0.3}, embeddings[0])
			},
		},
		{
			name: "CustomEmbedDocuments",
			setupFunc: func() *MockEmbedder {
				return &MockEmbedder{
					embedDocumentsFunc: func(ctx context.Context, texts []string) ([][]float32, error) {
						result := make([][]float32, len(texts))
						for i := range result {
							result[i] = []float32{0.5, 0.6, 0.7}
						}
						return result, nil
					},
				}
			},
			testFunc: func(t *testing.T, embedder *MockEmbedder) {
				texts := []string{"custom test"}
				embeddings, err := embedder.EmbedDocuments(ctx, texts)
				require.NoError(t, err)
				assert.Equal(t, []float32{0.5, 0.6, 0.7}, embeddings[0])
			},
		},
		{
			name: "EmbedDocumentsError",
			setupFunc: func() *MockEmbedder {
				return &MockEmbedder{
					embedDocumentsFunc: func(ctx context.Context, texts []string) ([][]float32, error) {
						return nil, errors.New("embedding failed")
					},
				}
			},
			testFunc: func(t *testing.T, embedder *MockEmbedder) {
				texts := []string{"test"}
				_, err := embedder.EmbedDocuments(ctx, texts)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "embedding failed")
			},
		},
		{
			name: "CallCountTracking",
			setupFunc: func() *MockEmbedder {
				return &MockEmbedder{}
			},
			testFunc: func(t *testing.T, embedder *MockEmbedder) {
				ctx := context.Background()

				// Test EmbedDocuments call count
				texts := []string{"test1", "test2"}
				embedder.EmbedDocuments(ctx, texts)
				assert.Equal(t, 1, embedder.GetEmbedDocumentsCallCount())

				embedder.EmbedDocuments(ctx, texts)
				assert.Equal(t, 2, embedder.GetEmbedDocumentsCallCount())

				// Test EmbedQuery call count
				embedder.EmbedQuery(ctx, "test query")
				assert.Equal(t, 1, embedder.GetEmbedQueryCallCount())

				embedder.EmbedQuery(ctx, "another query")
				assert.Equal(t, 2, embedder.GetEmbedQueryCallCount())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			embedder := tt.setupFunc()
			tt.testFunc(t, embedder)
		})
	}
}

// TestMockVectorStore tests the MockVectorStore functionality
func TestMockVectorStore(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func() *MockVectorStore
		testFunc  func(t *testing.T, store *MockVectorStore)
	}{
		{
			name: "DefaultBehavior",
			setupFunc: func() *MockVectorStore {
				return &MockVectorStore{}
			},
			testFunc: func(t *testing.T, store *MockVectorStore) {
				ctx := context.Background()

				// Test AddDocuments
				docs := []schema.Document{schema.NewDocument("test", nil)}
				ids, err := store.AddDocuments(ctx, docs)
				require.NoError(t, err)
				assert.Len(t, ids, 1)
				assert.Equal(t, 1, store.addDocumentsCount)

				// Test SimilaritySearch
				_, _, err = store.SimilaritySearch(ctx, []float32{0.1, 0.2}, 5)
				require.NoError(t, err)
				assert.Equal(t, 1, store.similaritySearchCount)

				// Test GetName
				name := store.GetName()
				assert.Equal(t, "mock", name)
				assert.Equal(t, 1, store.getNameCount)
			},
		},
		{
			name: "CustomBehavior",
			setupFunc: func() *MockVectorStore {
				return &MockVectorStore{
					addDocumentsFunc: func(ctx context.Context, documents []schema.Document, opts ...Option) ([]string, error) {
						return []string{"custom-id"}, nil
					},
					getNameFunc: func() string {
						return "custom-store"
					},
				}
			},
			testFunc: func(t *testing.T, store *MockVectorStore) {
				ctx := context.Background()

				// Test custom AddDocuments
				docs := []schema.Document{schema.NewDocument("test", nil)}
				ids, err := store.AddDocuments(ctx, docs)
				require.NoError(t, err)
				assert.Equal(t, []string{"custom-id"}, ids)

				// Test custom GetName
				name := store.GetName()
				assert.Equal(t, "custom-store", name)
			},
		},
		{
			name: "ErrorSimulation",
			setupFunc: func() *MockVectorStore {
				return &MockVectorStore{
					addDocumentsFunc: func(ctx context.Context, documents []schema.Document, opts ...Option) ([]string, error) {
						return nil, errors.New("add documents failed")
					},
				}
			},
			testFunc: func(t *testing.T, store *MockVectorStore) {
				ctx := context.Background()
				docs := []schema.Document{schema.NewDocument("test", nil)}
				_, err := store.AddDocuments(ctx, docs)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "add documents failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.setupFunc()
			tt.testFunc(t, store)
		})
	}
}

// Table-driven tests for configuration options
func TestConfigOptions(t *testing.T) {
	tests := []struct {
		name         string
		options      []Option
		validateFunc func(t *testing.T, config *vectorstoresiface.Config)
	}{
		{
			name:    "DefaultConfig",
			options: []Option{},
			validateFunc: func(t *testing.T, config *vectorstoresiface.Config) {
				assert.Equal(t, 5, config.SearchK)
				assert.Equal(t, float32(0.0), config.ScoreThreshold)
				assert.Nil(t, config.Embedder)
				assert.Nil(t, config.MetadataFilters)
				assert.Nil(t, config.ProviderConfig)
			},
		},
		{
			name:    "WithSearchK",
			options: []Option{WithSearchK(10)},
			validateFunc: func(t *testing.T, config *vectorstoresiface.Config) {
				assert.Equal(t, 10, config.SearchK)
			},
		},
		{
			name:    "WithScoreThreshold",
			options: []Option{WithScoreThreshold(0.8)},
			validateFunc: func(t *testing.T, config *vectorstoresiface.Config) {
				assert.Equal(t, float32(0.8), config.ScoreThreshold)
			},
		},
		{
			name:    "WithEmbedder",
			options: []Option{WithEmbedder(&MockEmbedder{})},
			validateFunc: func(t *testing.T, config *vectorstoresiface.Config) {
				assert.NotNil(t, config.Embedder)
			},
		},
		{
			name:    "WithMetadataFilter",
			options: []Option{WithMetadataFilter("category", "test")},
			validateFunc: func(t *testing.T, config *vectorstoresiface.Config) {
				assert.Contains(t, config.MetadataFilters, "category")
				assert.Equal(t, "test", config.MetadataFilters["category"])
			},
		},
		{
			name: "WithMetadataFilters",
			options: []Option{WithMetadataFilters(map[string]interface{}{
				"category": "tech",
				"status":   "published",
			})},
			validateFunc: func(t *testing.T, config *vectorstoresiface.Config) {
				assert.Contains(t, config.MetadataFilters, "category")
				assert.Contains(t, config.MetadataFilters, "status")
				assert.Equal(t, "tech", config.MetadataFilters["category"])
				assert.Equal(t, "published", config.MetadataFilters["status"])
			},
		},
		{
			name:    "WithProviderConfig",
			options: []Option{WithProviderConfig("connection_string", "postgres://test")},
			validateFunc: func(t *testing.T, config *vectorstoresiface.Config) {
				assert.Contains(t, config.ProviderConfig, "connection_string")
				assert.Equal(t, "postgres://test", config.ProviderConfig["connection_string"])
			},
		},
		{
			name: "WithProviderConfigs",
			options: []Option{WithProviderConfigs(map[string]interface{}{
				"host": "localhost",
				"port": 5432,
			})},
			validateFunc: func(t *testing.T, config *vectorstoresiface.Config) {
				assert.Contains(t, config.ProviderConfig, "host")
				assert.Contains(t, config.ProviderConfig, "port")
				assert.Equal(t, "localhost", config.ProviderConfig["host"])
				assert.Equal(t, 5432, config.ProviderConfig["port"])
			},
		},
		{
			name: "MultipleOptions",
			options: []Option{
				WithSearchK(20),
				WithScoreThreshold(0.9),
				WithEmbedder(&MockEmbedder{}),
				WithMetadataFilter("category", "ai"),
				WithProviderConfig("table_name", "documents"),
			},
			validateFunc: func(t *testing.T, config *vectorstoresiface.Config) {
				assert.Equal(t, 20, config.SearchK)
				assert.Equal(t, float32(0.9), config.ScoreThreshold)
				assert.NotNil(t, config.Embedder)
				assert.Contains(t, config.MetadataFilters, "category")
				assert.Contains(t, config.ProviderConfig, "table_name")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewDefaultConfig()
			ApplyOptions(config, tt.options...)
			tt.validateFunc(t, config)
		})
	}
}

// Table-driven tests for error handling
func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name         string
		errorFunc    func() error
		validateFunc func(t *testing.T, err error)
	}{
		{
			name: "NewVectorStoreError",
			errorFunc: func() error {
				return NewVectorStoreError(ErrCodeUnknownProvider, "provider %s not found", "test")
			},
			validateFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "provider test not found")

				var vsErr *VectorStoreError
				assert.True(t, AsVectorStoreError(err, &vsErr))
				assert.Equal(t, ErrCodeUnknownProvider, vsErr.Code)
			},
		},
		{
			name: "WrapError",
			errorFunc: func() error {
				cause := errors.New("connection failed")
				return WrapError(cause, ErrCodeConnectionFailed, "failed to connect to database")
			},
			validateFunc: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to connect to database")

				var vsErr *VectorStoreError
				assert.True(t, AsVectorStoreError(err, &vsErr))
				assert.Equal(t, ErrCodeConnectionFailed, vsErr.Code)
				assert.NotNil(t, vsErr.Cause)
			},
		},
		{
			name: "IsVectorStoreError",
			errorFunc: func() error {
				return NewVectorStoreError(ErrCodeInvalidConfig, "invalid configuration")
			},
			validateFunc: func(t *testing.T, err error) {
				assert.True(t, IsVectorStoreError(err, ErrCodeInvalidConfig))
				assert.False(t, IsVectorStoreError(err, ErrCodeUnknownProvider))
			},
		},
		{
			name: "ErrorWrappingChain",
			errorFunc: func() error {
				cause := errors.New("underlying error")
				wrapped1 := WrapError(cause, ErrCodeStorageFailed, "storage operation failed")
				return WrapError(wrapped1, ErrCodeRetrievalFailed, "retrieval operation failed")
			},
			validateFunc: func(t *testing.T, err error) {
				assert.Error(t, err)

				// Should find the most recent error in the chain
				var vsErr *VectorStoreError
				assert.True(t, AsVectorStoreError(err, &vsErr))
				assert.Equal(t, ErrCodeRetrievalFailed, vsErr.Code)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.errorFunc()
			tt.validateFunc(t, err)
		})
	}
}

// Test for observability (metrics, logging, tracing)
func TestObservability(t *testing.T) {
	ctx := context.Background()

	t.Run("MetricsCollector", func(t *testing.T) {
		mockMetrics := &MockMetricsCollector{}

		// Test document operations
		mockMetrics.RecordDocumentsAdded(ctx, 5, "test-store")
		assert.Equal(t, 1, mockMetrics.GetRecordDocumentsAddedCount())
		assert.Equal(t, "test-store", mockMetrics.GetLastStoreName())

		mockMetrics.RecordDocumentsDeleted(ctx, 2, "test-store")
		assert.Equal(t, 1, mockMetrics.GetRecordDocumentsDeletedCount())

		// Test error recording
		mockMetrics.RecordError(ctx, "test_error", "test-store")
		assert.Equal(t, 1, mockMetrics.GetRecordErrorCount())
		assert.Equal(t, "test_error", mockMetrics.GetLastErrorCode())
	})

	t.Run("TracerProvider", func(t *testing.T) {
		mockTracer := &MockTracerProvider{}

		ctx, spanEnd := mockTracer.StartAddDocumentsSpan(ctx, "test-store", 5)
		assert.Equal(t, 1, mockTracer.GetStartSpanCount())
		assert.Equal(t, "vectorstore.add_documents", mockTracer.GetLastOperation())
		spanEnd()

		ctx, spanEnd = mockTracer.StartSearchSpan(ctx, "test-store", 10, 5)
		assert.Equal(t, 2, mockTracer.GetStartSpanCount())
		assert.Equal(t, "vectorstore.search", mockTracer.GetLastOperation())
		spanEnd()

		// Test that generic StartSpan also works
		mockTracer.StartSpan(ctx, "custom.operation")
		assert.Equal(t, 3, mockTracer.GetStartSpanCount())
		assert.Equal(t, "custom.operation", mockTracer.GetLastOperation())
	})

	t.Run("Logger", func(t *testing.T) {
		mockLogger := &MockLogger{}

		mockLogger.LogAttrs(ctx, slog.LevelInfo, "test message",
			slog.String("key", "value"))

		assert.Equal(t, 1, mockLogger.GetLogAttrsCount())
		assert.Equal(t, "test message", mockLogger.GetLastMessage())
		assert.Equal(t, slog.LevelInfo, mockLogger.GetLastLevel())
	})
}

// Test for factory and provider management
func TestFactoryAndProviders(t *testing.T) {
	t.Run("ListProviders", func(t *testing.T) {
		providers := ListProviders()
		assert.Contains(t, providers, "inmemory")
		assert.Contains(t, providers, "pgvector")
		assert.Greater(t, len(providers), 0)
	})

	t.Run("ValidateProvider", func(t *testing.T) {
		tests := []struct {
			provider string
			expected bool
		}{
			{"inmemory", true},
			{"pgvector", true},
			{"nonexistent", false},
			{"", false},
		}

		for _, tt := range tests {
			result := ValidateProvider(tt.provider)
			assert.Equal(t, tt.expected, result, "Provider: %s", tt.provider)
		}
	})

	t.Run("StoreFactory", func(t *testing.T) {
		factory := NewStoreFactory()
		assert.NotNil(t, factory)

		// Test registering a provider
		mockCreator := func(ctx context.Context, config vectorstoresiface.Config) (VectorStore, error) {
			return &MockVectorStore{}, nil
		}

		factory.Register("test-provider", mockCreator)

		// Test creating with registered provider
		ctx := context.Background()
		config := NewDefaultConfig()
		store, err := factory.Create(ctx, "test-provider", *config)
		assert.NoError(t, err)
		assert.NotNil(t, store)
		assert.Equal(t, "mock", store.GetName())

		// Test creating with unregistered provider
		_, err = factory.Create(ctx, "nonexistent", *config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("GlobalFactory", func(t *testing.T) {
		// Test registering with global factory
		mockCreator := func(ctx context.Context, config vectorstoresiface.Config) (VectorStore, error) {
			return &MockVectorStore{}, nil
		}

		RegisterProvider("test-global", mockCreator)

		// Test creating with global factory
		ctx := context.Background()
		config := NewDefaultConfig()
		store, err := NewVectorStore(ctx, "test-global", *config)
		assert.NoError(t, err)
		assert.NotNil(t, store)
	})
}

// Test for global instances (logger, tracer, metrics)
func TestGlobalInstances(t *testing.T) {
	t.Run("GlobalLogger", func(t *testing.T) {
		logger := NewLogger(nil)
		SetGlobalLogger(logger)

		retrievedLogger := GetGlobalLogger()
		assert.Equal(t, logger, retrievedLogger)
		assert.NotNil(t, retrievedLogger)
	})

	t.Run("GlobalTracer", func(t *testing.T) {
		tracer := NewTracerProvider("test")
		SetGlobalTracer(tracer)

		retrievedTracer := GetGlobalTracer()
		assert.Equal(t, tracer, retrievedTracer)
		assert.NotNil(t, retrievedTracer)
	})

	t.Run("GlobalMetrics", func(t *testing.T) {
		// Note: Global metrics setter is in metrics.go, so we can't test it here
		// without importing that package. This is a limitation of the current design.
		// In a real integration test, this would be tested with the full metrics collector.
	})
}

// Test for convenience functions
func TestConvenienceFunctions(t *testing.T) {
	ctx := context.Background()
	embedder := &MockEmbedder{}

	t.Run("AddDocumentsConvenience", func(t *testing.T) {
		mockStore := &MockVectorStore{}
		docs := []schema.Document{
			schema.NewDocument("convenience test", nil),
		}

		ids, err := AddDocuments(ctx, mockStore, docs, embedder)
		assert.NoError(t, err)
		assert.Len(t, ids, 1)
	})

	t.Run("SearchByQueryConvenience", func(t *testing.T) {
		mockStore := &MockVectorStore{}
		results, scores, err := SearchByQuery(ctx, mockStore, "test query", 5, embedder)
		assert.NoError(t, err)
		assert.NotNil(t, results)
		assert.NotNil(t, scores)
	})

	t.Run("SearchByVectorConvenience", func(t *testing.T) {
		mockStore := &MockVectorStore{}
		queryVector := []float32{0.1, 0.2, 0.3}
		results, scores, err := SearchByVector(ctx, mockStore, queryVector, 5)
		assert.NoError(t, err)
		assert.NotNil(t, results)
		assert.NotNil(t, scores)
	})

	t.Run("DeleteDocumentsConvenience", func(t *testing.T) {
		mockStore := &MockVectorStore{}
		ids := []string{"test-id"}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Delete using convenience function
		err := DeleteDocuments(ctx, mockStore, ids)
		assert.NoError(t, err)
	})

	t.Run("AsRetrieverConvenience", func(t *testing.T) {
		mockStore := &MockVectorStore{}
		retriever := AsRetriever(mockStore)
		assert.NotNil(t, retriever)

		results, err := retriever.GetRelevantDocuments(ctx, "test query")
		assert.NoError(t, err)
		assert.NotNil(t, results)
	})
}

// Test for edge cases and error conditions
func TestEdgeCases(t *testing.T) {
	ctx := context.Background()
	embedder := &MockEmbedder{}

	t.Run("EmptyDocuments", func(t *testing.T) {
		mockStore := &MockVectorStore{}
		ids, err := mockStore.AddDocuments(ctx, []schema.Document{})
		assert.NoError(t, err)
		assert.Len(t, ids, 0)
	})

	t.Run("NilEmbedder", func(t *testing.T) {
		// Test with mock store that simulates nil embedder behavior
		mockStore := &MockVectorStore{
			addDocumentsFunc: func(ctx context.Context, documents []schema.Document, opts ...Option) ([]string, error) {
				if len(documents) == 0 {
					return []string{}, nil
				}
				// Simulate successful operation with pre-computed embeddings
				ids := make([]string, len(documents))
				for i := range ids {
					ids[i] = "mock-id-" + string(rune(i+'a'))
				}
				return ids, nil
			},
		}

		docs := []schema.Document{
			schema.NewDocument("test content", nil),
		}

		ids, err := mockStore.AddDocuments(ctx, docs)
		assert.NoError(t, err)
		assert.Len(t, ids, 1)
	})

	t.Run("LargeBatchSize", func(t *testing.T) {
		mockStore := &MockVectorStore{}
		docs := []schema.Document{
			schema.NewDocument("large batch test", nil),
		}

		ids, err := BatchAddDocuments(ctx, mockStore, docs, 1000, embedder)
		assert.NoError(t, err)
		assert.Len(t, ids, 1)
	})

	t.Run("ZeroBatchSize", func(t *testing.T) {
		mockStore := &MockVectorStore{}
		docs := []schema.Document{
			schema.NewDocument("zero batch test", nil),
		}

		ids, err := BatchAddDocuments(ctx, mockStore, docs, 0, embedder)
		assert.NoError(t, err)
		assert.Len(t, ids, 1) // Should default to some reasonable batch size
	})
}

// Integration test template (for future expansion)
func TestIntegrationTemplate(t *testing.T) {
	t.Skip("Integration test template - enable when integration environment is available")

	// This is a template for integration tests that would run against real services
	// Uncomment and modify when you have integration test infrastructure

	/*

		t.Run("FullWorkflow", func(t *testing.T) {
			// 1. Create vector store
			store, err := NewInMemoryStore(ctx, WithEmbedder(&MockEmbedder{}))
			require.NoError(t, err)

			// 2. Add documents
			docs := []schema.Document{
				schema.NewDocument("Integration test document 1", map[string]string{"category": "test"}),
				schema.NewDocument("Integration test document 2", map[string]string{"category": "test"}),
			}
			ids, err := store.AddDocuments(ctx, docs)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
			require.NoError(t, err)
			assert.Len(t, ids, 2)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
			// 3. Search documents
			results, scores, err := store.SimilaritySearchByQuery(ctx, "integration test", 10, &MockEmbedder{})
			require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
			assert.Greater(t, len(results), 0)
			assert.Len(t, scores, len(results))

			// 4. Delete documents
			err = store.DeleteDocuments(ctx, ids[:1])
			require.NoError(t, err)

			// 5. Verify deletion
			finalResults, _, err := store.SimilaritySearch(ctx, []float32{0.1, 0.2, 0.3}, 10)
			require.NoError(t, err)
			assert.Len(t, finalResults, 1)
		})
	*/
}

// Benchmark tests for performance-critical operations
func BenchmarkVectorStoreOperations(b *testing.B) {
	b.Skip("Skipping vector store operations benchmark due to type compatibility issues with inmemory store")
	// TODO: Re-enable when inmemory store implements main VectorStore interface
}

func BenchmarkBatchOperations(b *testing.B) {
	b.Skip("Skipping batch operations benchmark due to type compatibility issues with inmemory store")
	// TODO: Re-enable when inmemory store implements main VectorStore interface
}

func BenchmarkMockEmbedder(b *testing.B) {
	ctx := context.Background()
	embedder := &MockEmbedder{}

	b.Run("EmbedDocuments", func(b *testing.B) {
		texts := []string{"test document 1", "test document 2", "test document 3"}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			embedder.EmbedDocuments(ctx, texts)
		}
	})

	b.Run("EmbedQuery", func(b *testing.B) {
		query := "test query"

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			embedder.EmbedQuery(ctx, query)
		}
	})
}

func BenchmarkConfigOperations(b *testing.B) {
	b.Run("ApplyOptions", func(b *testing.B) {
		options := []Option{
			WithSearchK(10),
			WithScoreThreshold(0.8),
			WithEmbedder(&MockEmbedder{}),
			WithMetadataFilter("category", "test"),
			WithProviderConfig("key", "value"),
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			newConfig := NewDefaultConfig()
			ApplyOptions(newConfig, options...)
		}
	})

	b.Run("CloneConfig", func(b *testing.B) {
		// CloneConfig is not exported from main package, skipping this benchmark
		b.Skip("CloneConfig benchmark skipped - function not exported from main package")
	})
}

// Performance comparison benchmarks
func BenchmarkVectorStoreComparison(b *testing.B) {
	b.Skip("Skipping vector store comparison benchmark due to type compatibility issues with inmemory store")
	// TODO: Re-enable when inmemory store implements main VectorStore interface
}
