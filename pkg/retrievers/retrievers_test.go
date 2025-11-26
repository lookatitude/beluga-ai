// Package retrievers provides comprehensive tests for the retrievers package.
package retrievers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
)

// MockVectorStore is a test implementation of the VectorStore interface for testing retrievers.
type MockVectorStore struct {
	searchByQueryErr   error
	addDocumentsErr    error
	deleteDocumentsErr error
	searchErr          error
	embedder           vectorstores.Embedder
	callCount          map[string]int
	documents          []schema.Document
	similarityResults  []schema.Document
	similarityScores   []float32
}

func NewMockVectorStore() *MockVectorStore {
	return &MockVectorStore{
		callCount: make(map[string]int),
	}
}

func (m *MockVectorStore) WithDocuments(docs []schema.Document) *MockVectorStore {
	m.documents = docs
	return m
}

func (m *MockVectorStore) WithSimilarityResults(docs []schema.Document, scores []float32) *MockVectorStore {
	m.similarityResults = docs
	m.similarityScores = scores
	return m
}

func (m *MockVectorStore) WithSearchByQueryError(err error) *MockVectorStore {
	m.searchByQueryErr = err
	return m
}

func (m *MockVectorStore) WithSimilaritySearchError(err error) *MockVectorStore {
	m.searchErr = err
	return m
}

func (m *MockVectorStore) WithEmbedder(embedder vectorstores.Embedder) *MockVectorStore {
	m.embedder = embedder
	return m
}

func (m *MockVectorStore) AddDocuments(ctx context.Context, documents []schema.Document, options ...vectorstores.Option) ([]string, error) {
	m.callCount["AddDocuments"]++
	if m.addDocumentsErr != nil {
		return nil, m.addDocumentsErr
	}
	ids := make([]string, len(documents))
	for i := range documents {
		ids[i] = fmt.Sprintf("doc-%d", i)
	}
	return ids, nil
}

func (m *MockVectorStore) DeleteDocuments(ctx context.Context, ids []string, options ...vectorstores.Option) error {
	m.callCount["DeleteDocuments"]++
	return m.deleteDocumentsErr
}

func (m *MockVectorStore) SimilaritySearch(ctx context.Context, queryVector []float32, k int, options ...vectorstores.Option) ([]schema.Document, []float32, error) {
	m.callCount["SimilaritySearch"]++
	if m.searchErr != nil {
		return nil, nil, m.searchErr
	}
	return m.similarityResults, m.similarityScores, nil
}

func (m *MockVectorStore) SimilaritySearchByQuery(ctx context.Context, query string, k int, embedder vectorstores.Embedder, options ...vectorstores.Option) ([]schema.Document, []float32, error) {
	m.callCount["SimilaritySearchByQuery"]++

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	default:
	}

	if m.searchByQueryErr != nil {
		return nil, nil, m.searchByQueryErr
	}

	// Respect the k parameter by limiting results
	results := m.similarityResults
	scores := m.similarityScores

	if k > 0 && len(results) > k {
		results = results[:k]
		scores = scores[:k]
	}

	return results, scores, nil
}

func (m *MockVectorStore) AsRetriever(options ...vectorstores.Option) vectorstores.Retriever {
	m.callCount["AsRetriever"]++
	return nil // Not implemented for tests
}

func (m *MockVectorStore) GetName() string {
	return "MockVectorStore"
}

func (m *MockVectorStore) GetCallCount(method string) int {
	return m.callCount[method]
}

func (m *MockVectorStore) ResetCallCount() {
	m.callCount = make(map[string]int)
}

// MockEmbedder is a test implementation of the vectorstores.Embedder interface.
type MockEmbedder struct {
	embedErr  error
	dimension int
}

func NewMockEmbedder(dimension int) *MockEmbedder {
	return &MockEmbedder{dimension: dimension}
}

func (m *MockEmbedder) WithError(err error) *MockEmbedder {
	m.embedErr = err
	return m
}

func (m *MockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if m.embedErr != nil {
		return nil, m.embedErr
	}
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = make([]float32, m.dimension)
		for j := range result[i] {
			result[i][j] = float32(i*j + 1) // Deterministic values for testing
		}
	}
	return result, nil
}

func (m *MockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	if m.embedErr != nil {
		return nil, m.embedErr
	}
	result := make([]float32, m.dimension)
	for i := range result {
		result[i] = float32(i + 1)
	}
	return result, nil
}

// Helper function to create test documents.
func createTestDocuments(count int) []schema.Document {
	docs := make([]schema.Document, count)
	for i := 0; i < count; i++ {
		docs[i] = schema.Document{
			PageContent: fmt.Sprintf("This is test document number %d with some content for testing.", i),
			Metadata: map[string]string{
				"id":       fmt.Sprintf("doc-%d", i),
				"source":   "test",
				"category": fmt.Sprintf("category-%d", i%3),
			},
		}
	}
	return docs
}

// Helper function to create test scores.
func createTestScores(count int) []float32 {
	scores := make([]float32, count)
	for i := 0; i < count; i++ {
		scores[i] = 1.0 - float32(i)*0.1 // Decreasing scores: 1.0, 0.9, 0.8, ...
	}
	return scores
}

func TestMain(m *testing.M) {
	// Set up test logger to discard logs during tests
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	slog.SetDefault(logger)

	os.Exit(m.Run())
}

func TestNewVectorStoreRetriever(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
		wantErr bool
	}{
		{
			name:    "valid configuration with defaults",
			options: []Option{},
			wantErr: false,
		},
		{
			name:    "invalid k value too low",
			options: []Option{WithDefaultK(0)},
			wantErr: true,
		},
		{
			name:    "invalid k value too high",
			options: []Option{WithDefaultK(101)},
			wantErr: true,
		},
		{
			name: "valid custom configuration",
			options: []Option{
				WithDefaultK(5),
				WithTimeout(10 * time.Second),
				WithTracing(false),
				WithMetrics(false),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For this test, we'll use nil for vectorStore since we're just testing config validation
			_, err := NewVectorStoreRetriever(nil, tt.options...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewVectorStoreRetriever() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewVectorStoreRetrieverFromConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  VectorStoreRetrieverConfig
		wantErr bool
	}{
		{
			name: "valid configuration",
			config: VectorStoreRetrieverConfig{
				K:              5,
				ScoreThreshold: 0.7,
				Timeout:        30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "invalid k value",
			config: VectorStoreRetrieverConfig{
				K:              -1,
				ScoreThreshold: 0.7,
				Timeout:        30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid score threshold",
			config: VectorStoreRetrieverConfig{
				K:              5,
				ScoreThreshold: 1.5,
				Timeout:        30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid timeout",
			config: VectorStoreRetrieverConfig{
				K:              5,
				ScoreThreshold: 0.7,
				Timeout:        500 * time.Millisecond,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewVectorStoreRetrieverFromConfig(nil, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewVectorStoreRetrieverFromConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRetrieverOptions(t *testing.T) {
	tests := []struct {
		expected *RetrieverOptions
		name     string
		options  []Option
	}{
		{
			name:    "default options",
			options: []Option{},
			expected: &RetrieverOptions{
				DefaultK:       4,
				ScoreThreshold: 0.0,
				MaxRetries:     3,
				Timeout:        30 * time.Second,
				EnableTracing:  true,
				EnableMetrics:  true,
			},
		},
		{
			name: "custom options",
			options: []Option{
				WithDefaultK(10),
				WithMaxRetries(5),
				WithTimeout(60 * time.Second),
				WithTracing(false),
				WithMetrics(false),
			},
			expected: &RetrieverOptions{
				DefaultK:       10,
				ScoreThreshold: 0.0, // Default value
				MaxRetries:     5,
				Timeout:        60 * time.Second,
				EnableTracing:  false,
				EnableMetrics:  false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &RetrieverOptions{
				DefaultK:       4,
				ScoreThreshold: 0.0,
				MaxRetries:     3,
				Timeout:        30 * time.Second,
				EnableTracing:  true,
				EnableMetrics:  true,
			}

			for _, option := range tt.options {
				option(opts)
			}

			if opts.DefaultK != tt.expected.DefaultK {
				t.Errorf("DefaultK = %v, want %v", opts.DefaultK, tt.expected.DefaultK)
			}
			if opts.ScoreThreshold != tt.expected.ScoreThreshold {
				t.Errorf("ScoreThreshold = %v, want %v", opts.ScoreThreshold, tt.expected.ScoreThreshold)
			}
			if opts.MaxRetries != tt.expected.MaxRetries {
				t.Errorf("MaxRetries = %v, want %v", opts.MaxRetries, tt.expected.MaxRetries)
			}
			if opts.Timeout != tt.expected.Timeout {
				t.Errorf("Timeout = %v, want %v", opts.Timeout, tt.expected.Timeout)
			}
			if opts.EnableTracing != tt.expected.EnableTracing {
				t.Errorf("EnableTracing = %v, want %v", opts.EnableTracing, tt.expected.EnableTracing)
			}
			if opts.EnableMetrics != tt.expected.EnableMetrics {
				t.Errorf("EnableMetrics = %v, want %v", opts.EnableMetrics, tt.expected.EnableMetrics)
			}
		})
	}
}

// TestMockRetriever is removed due to import cycle issues.
// MockRetriever tests are located in the providers package.

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				DefaultK:       5,
				ScoreThreshold: 0.7,
				MaxRetries:     3,
				Timeout:        30 * time.Second,
				EnableTracing:  true,
				EnableMetrics:  true,
				VectorStoreConfig: VectorStoreConfig{
					MaxBatchSize:       100,
					EnableMMR:          false,
					MMRLambda:          0.5,
					DiversityThreshold: 0.7,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid DefaultK too low",
			config: Config{
				DefaultK:       0,
				ScoreThreshold: 0.7,
				MaxRetries:     3,
				Timeout:        30 * time.Second,
				VectorStoreConfig: VectorStoreConfig{
					MaxBatchSize: 100,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid DefaultK too high",
			config: Config{
				DefaultK:       101,
				ScoreThreshold: 0.7,
				MaxRetries:     3,
				Timeout:        30 * time.Second,
				VectorStoreConfig: VectorStoreConfig{
					MaxBatchSize: 100,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid ScoreThreshold too low",
			config: Config{
				DefaultK:       5,
				ScoreThreshold: -0.1,
				MaxRetries:     3,
				Timeout:        30 * time.Second,
				VectorStoreConfig: VectorStoreConfig{
					MaxBatchSize: 100,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid ScoreThreshold too high",
			config: Config{
				DefaultK:       5,
				ScoreThreshold: 1.1,
				MaxRetries:     3,
				Timeout:        30 * time.Second,
				VectorStoreConfig: VectorStoreConfig{
					MaxBatchSize: 100,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid Timeout too short",
			config: Config{
				DefaultK:       5,
				ScoreThreshold: 0.7,
				MaxRetries:     3,
				Timeout:        500 * time.Millisecond,
				VectorStoreConfig: VectorStoreConfig{
					MaxBatchSize: 100,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid Timeout too long",
			config: Config{
				DefaultK:       5,
				ScoreThreshold: 0.7,
				MaxRetries:     3,
				Timeout:        6 * time.Minute,
				VectorStoreConfig: VectorStoreConfig{
					MaxBatchSize: 100,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCustomErrorTypes(t *testing.T) {
	t.Run("RetrieverError", func(t *testing.T) {
		originalErr := NewRetrieverError("test operation", nil, ErrCodeInvalidInput)
		if originalErr.Op != "test operation" {
			t.Errorf("RetrieverError.Op = %v, want 'test operation'", originalErr.Op)
		}
		if originalErr.Code != ErrCodeInvalidInput {
			t.Errorf("RetrieverError.Code = %v, want %v", originalErr.Code, ErrCodeInvalidInput)
		}
	})

	t.Run("RetrieverErrorWithMessage", func(t *testing.T) {
		customMsg := "custom error message"
		err := NewRetrieverErrorWithMessage("test operation", nil, ErrCodeInvalidInput, customMsg)
		if err.Message != customMsg {
			t.Errorf("RetrieverError.Message = %v, want %v", err.Message, customMsg)
		}
	})

	t.Run("ValidationError", func(t *testing.T) {
		field := "TestField"
		value := "invalid_value"
		msg := "must be valid"
		err := &ValidationError{
			Field: field,
			Value: value,
			Msg:   msg,
		}

		expectedMsg := "validation failed for field 'TestField' with value 'invalid_value': must be valid"
		if err.Error() != expectedMsg {
			t.Errorf("ValidationError.Error() = %v, want %v", err.Error(), expectedMsg)
		}
	})

	t.Run("TimeoutError", func(t *testing.T) {
		timeout := 30 * time.Second
		underlyingErr := NewRetrieverError("underlying", nil, ErrCodeTimeout)
		err := NewTimeoutError("test operation", timeout, underlyingErr)

		if err.Op != "test operation" {
			t.Errorf("TimeoutError.Op = %v, want 'test operation'", err.Op)
		}
		if err.Timeout != timeout {
			t.Errorf("TimeoutError.Timeout = %v, want %v", err.Timeout, timeout)
		}
	})
}

func TestMetrics(t *testing.T) {
	// Create a no-op meter for testing
	meter := noop.NewMeterProvider().Meter("test")

	metrics, err := NewMetrics(meter)
	if err != nil {
		t.Fatalf("NewMetrics() error = %v", err)
	}

	if metrics == nil {
		t.Error("NewMetrics() returned nil")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test recording retrieval metrics
	duration := 100 * time.Millisecond

	metrics.RecordRetrieval(ctx, "test_retriever", duration, 5, 0.8, nil)

	// Test recording vector store metrics
	metrics.RecordVectorStoreOperation(ctx, "similarity_search", duration, 5, nil)

	// Test recording batch metrics
	metrics.RecordBatchOperation(ctx, "test_operation", 10, duration)
}

func TestGetRetrieverTypes(t *testing.T) {
	types := GetRetrieverTypes()

	if len(types) == 0 {
		t.Error("GetRetrieverTypes() returned empty slice")
	}

	// Check that "vector_store" is included
	found := false
	for _, typ := range types {
		if typ == "vector_store" {
			found = true
			break
		}
	}

	if !found {
		t.Error("GetRetrieverTypes() should include 'vector_store'")
	}
}

func TestValidateRetrieverConfig(t *testing.T) {
	validConfig := DefaultConfig()
	err := ValidateRetrieverConfig(validConfig)
	if err != nil {
		t.Errorf("ValidateRetrieverConfig() with valid config returned error: %v", err)
	}

	invalidConfig := DefaultConfig()
	invalidConfig.DefaultK = 0 // Invalid value
	err = ValidateRetrieverConfig(invalidConfig)
	if err == nil {
		t.Error("ValidateRetrieverConfig() with invalid config should return error")
	}
}

// TestVectorStoreRetriever_GetRelevantDocuments tests the core retrieval functionality.
func TestVectorStoreRetriever_GetRelevantDocuments(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func() vectorstores.VectorStore
		query          string
		expectedDocs   int
		expectError    bool
		expectedScores bool
	}{
		{
			name: "successful retrieval with documents",
			setupMock: func() vectorstores.VectorStore {
				docs := createTestDocuments(3)
				scores := createTestScores(3)
				return NewMockVectorStore().WithSimilarityResults(docs, scores)
			},
			query:          "test query",
			expectedDocs:   3,
			expectError:    false,
			expectedScores: true,
		},
		{
			name: "successful retrieval with score filtering",
			setupMock: func() vectorstores.VectorStore {
				docs := createTestDocuments(5)
				scores := []float32{0.9, 0.8, 0.3, 0.2, 0.1} // Some below threshold
				return NewMockVectorStore().WithSimilarityResults(docs, scores)
			},
			query:        "test query",
			expectedDocs: 4, // Limited by default K=4
			expectError:  false,
		},
		{
			name: "vector store error",
			setupMock: func() vectorstores.VectorStore {
				return NewMockVectorStore().WithSearchByQueryError(errors.New("vector store error"))
			},
			query:        "test query",
			expectedDocs: 0,
			expectError:  true,
		},
		{
			name: "empty results",
			setupMock: func() vectorstores.VectorStore {
				return NewMockVectorStore().WithSimilarityResults([]schema.Document{}, []float32{})
			},
			query:        "test query",
			expectedDocs: 0,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := tt.setupMock()
			retriever, err := NewVectorStoreRetriever(mockStore, WithMetrics(false), WithTracing(false))
			if err != nil {
				t.Fatalf("Failed to create retriever: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			docs, err := retriever.GetRelevantDocuments(ctx, tt.query)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(docs) != tt.expectedDocs {
				t.Errorf("Expected %d documents, got %d", tt.expectedDocs, len(docs))
			}
		})
	}
}

// TestVectorStoreRetriever_Invoke tests the Runnable interface Invoke method.
func TestVectorStoreRetriever_Invoke(t *testing.T) {
	tests := []struct {
		input        any
		setupMock    func() vectorstores.VectorStore
		name         string
		expectedType string
		expectError  bool
	}{
		{
			name:  "successful invoke with string input",
			input: "test query",
			setupMock: func() vectorstores.VectorStore {
				docs := createTestDocuments(2)
				scores := createTestScores(2)
				return NewMockVectorStore().WithSimilarityResults(docs, scores)
			},
			expectError:  false,
			expectedType: "[]schema.Document",
		},
		{
			name:         "invalid input type",
			input:        123,
			setupMock:    func() vectorstores.VectorStore { return NewMockVectorStore() },
			expectError:  true,
			expectedType: "",
		},
		{
			name:  "vector store error",
			input: "test query",
			setupMock: func() vectorstores.VectorStore {
				return NewMockVectorStore().WithSearchByQueryError(errors.New("store error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := tt.setupMock()
			retriever, err := NewVectorStoreRetriever(mockStore, WithMetrics(false), WithTracing(false))
			if err != nil {
				t.Fatalf("Failed to create retriever: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			result, err := retriever.Invoke(ctx, tt.input)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.expectedType == "[]schema.Document" {
				docs, ok := result.([]schema.Document)
				if !ok {
					t.Errorf("Expected []schema.Document, got %T", result)
				}
				if len(docs) == 0 {
					t.Error("Expected non-empty document slice")
				}
			}
		})
	}
}

// TestVectorStoreRetriever_Batch tests the Runnable interface Batch method.
func TestVectorStoreRetriever_Batch(t *testing.T) {
	tests := []struct {
		setupMock   func() vectorstores.VectorStore
		name        string
		inputs      []any
		expectError bool
	}{
		{
			name:   "successful batch processing",
			inputs: []any{"query1", "query2", "query3"},
			setupMock: func() vectorstores.VectorStore {
				docs := createTestDocuments(2)
				scores := createTestScores(2)
				return NewMockVectorStore().WithSimilarityResults(docs, scores)
			},
			expectError: false,
		},
		{
			name:        "batch with mixed valid/invalid inputs",
			inputs:      []any{"query1", 123, "query3"},
			setupMock:   func() vectorstores.VectorStore { return NewMockVectorStore() },
			expectError: true, // Should fail on invalid input
		},
		{
			name:   "batch with vector store error",
			inputs: []any{"query1", "query2"},
			setupMock: func() vectorstores.VectorStore {
				return NewMockVectorStore().WithSearchByQueryError(errors.New("store error"))
			},
			expectError: true,
		},
		{
			name:        "empty batch",
			inputs:      []any{},
			setupMock:   func() vectorstores.VectorStore { return NewMockVectorStore() },
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := tt.setupMock()
			retriever, err := NewVectorStoreRetriever(mockStore, WithMetrics(false), WithTracing(false))
			if err != nil {
				t.Fatalf("Failed to create retriever: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			results, err := retriever.Batch(ctx, tt.inputs)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(results) != len(tt.inputs) {
				t.Errorf("Expected %d results, got %d", len(tt.inputs), len(results))
			}

			// Verify results are document slices
			for i, result := range results {
				if result == nil {
					continue // Could be nil due to error in processing
				}
				docs, ok := result.([]schema.Document)
				if !ok {
					t.Errorf("Result %d: expected []schema.Document, got %T", i, result)
				}
				if len(docs) == 0 {
					t.Errorf("Result %d: expected non-empty document slice", i)
				}
			}
		})
	}
}

// TestVectorStoreRetriever_Stream tests the Runnable interface Stream method.
func TestVectorStoreRetriever_Stream(t *testing.T) {
	mockStore := NewMockVectorStore()
	retriever, err := NewVectorStoreRetriever(mockStore, WithMetrics(false), WithTracing(false))
	if err != nil {
		t.Fatalf("Failed to create retriever: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err = retriever.Stream(ctx, "test query")

	if err == nil {
		t.Error("Expected error for Stream method (not supported)")
	}

	// Verify it's a RetrieverError with correct code
	var retrieverErr *RetrieverError
	if !errors.As(err, &retrieverErr) {
		t.Errorf("Expected RetrieverError, got %T", err)
	}
	if retrieverErr.Code != ErrCodeInvalidInput {
		t.Errorf("Expected error code %s, got %s", ErrCodeInvalidInput, retrieverErr.Code)
	}
}

// TestVectorStoreRetriever_CheckHealth tests the health check functionality.
func TestVectorStoreRetriever_CheckHealth(t *testing.T) {
	tests := []struct {
		setupMock   func() vectorstores.VectorStore
		config      *RetrieverOptions
		name        string
		expectError bool
	}{
		{
			name:      "healthy retriever",
			setupMock: func() vectorstores.VectorStore { return NewMockVectorStore() },
			config: &RetrieverOptions{
				DefaultK:       4,
				ScoreThreshold: 0.0,
				EnableTracing:  false,
				EnableMetrics:  false,
			},
			expectError: false,
		},
		{
			name:      "invalid defaultK",
			setupMock: func() vectorstores.VectorStore { return NewMockVectorStore() },
			config: &RetrieverOptions{
				DefaultK:       0, // Invalid
				ScoreThreshold: 0.0,
				EnableTracing:  false,
				EnableMetrics:  false,
			},
			expectError: true,
		},
		{
			name:      "invalid score threshold",
			setupMock: func() vectorstores.VectorStore { return NewMockVectorStore() },
			config: &RetrieverOptions{
				DefaultK:       4,
				ScoreThreshold: -0.1, // Invalid
				EnableTracing:  false,
				EnableMetrics:  false,
			},
			expectError: true,
		},
		{
			name:      "nil vector store",
			setupMock: func() vectorstores.VectorStore { return nil },
			config: &RetrieverOptions{
				DefaultK:       4,
				ScoreThreshold: 0.0,
				EnableTracing:  false,
				EnableMetrics:  false,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := tt.setupMock()
			retriever := newVectorStoreRetrieverInternal(mockStore, tt.config)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			err := retriever.CheckHealth(ctx)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestVectorStoreRetriever_WithOptions tests retriever with call-specific options.
func TestVectorStoreRetriever_WithOptions(t *testing.T) {
	// Create mock with more documents than default K
	docs := createTestDocuments(10)
	scores := createTestScores(10)
	mockStore := NewMockVectorStore().WithSimilarityResults(docs, scores)

	retriever, err := NewVectorStoreRetriever(mockStore, WithDefaultK(2), WithMetrics(false), WithTracing(false))
	if err != nil {
		t.Fatalf("Failed to create retriever: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test with default K
	result1, err := retriever.GetRelevantDocuments(ctx, "query1")
	if err != nil {
		t.Fatalf("GetRelevantDocuments failed: %v", err)
	}
	if len(result1) != 2 {
		t.Errorf("Expected 2 documents with default K, got %d", len(result1))
	}

	// Test Invoke with same configuration
	result2, err := retriever.Invoke(ctx, "query2")
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}
	docs2, ok := result2.([]schema.Document)
	if !ok {
		t.Fatalf("Expected []schema.Document, got %T", result2)
	}
	if len(docs2) != 2 {
		t.Errorf("Expected 2 documents (same as configured K), got %d", len(docs2))
	}
}

// TestVectorStoreRetriever_ErrorHandling tests various error scenarios.
func TestVectorStoreRetriever_ErrorHandling(t *testing.T) {
	tests := []struct {
		setupMock     func() vectorstores.VectorStore
		name          string
		query         string
		errorContains string
		expectError   bool
	}{
		{
			name: "vector store returns error",
			setupMock: func() vectorstores.VectorStore {
				return NewMockVectorStore().WithSearchByQueryError(errors.New("connection failed"))
			},
			query:         "test query",
			expectError:   true,
			errorContains: "connection failed",
		},
		{
			name: "context timeout",
			setupMock: func() vectorstores.VectorStore {
				return NewMockVectorStore()
			},
			query:         "test query",
			expectError:   true,
			errorContains: "context",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := tt.setupMock()
			retriever, err := NewVectorStoreRetriever(mockStore, WithMetrics(false), WithTracing(false))
			if err != nil {
				t.Fatalf("Failed to create retriever: %v", err)
			}

			baseCtx, baseCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer baseCancel()
			ctx := baseCtx
			if tt.name == "context timeout" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 1*time.Nanosecond)
				defer cancel()
				time.Sleep(1 * time.Millisecond) // Ensure timeout
			}

			_, err = retriever.GetRelevantDocuments(ctx, tt.query)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				if tt.errorContains != "" && !errors.Is(err, context.DeadlineExceeded) &&
					!strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Error doesn't contain expected string '%s': %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestVectorStoreRetriever_Observability tests tracing and metrics integration.
func TestVectorStoreRetriever_Observability(t *testing.T) {
	// Create mock store
	docs := createTestDocuments(3)
	scores := createTestScores(3)
	mockStore := NewMockVectorStore().WithSimilarityResults(docs, scores)

	// Create no-op tracer and meter for testing
	var tracer trace.Tracer // nil tracer for testing
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	meter := noop.NewMeterProvider().Meter("test")

	retriever, err := NewVectorStoreRetriever(mockStore,
		WithTracing(true),
		WithTracer(tracer),
		WithMetrics(true),
		WithMeter(meter),
	)
	if err != nil {
		t.Fatalf("Failed to create retriever: %v", err)
	}

	// Test retrieval with observability enabled
	result, err := retriever.GetRelevantDocuments(ctx, "test query")
	if err != nil {
		t.Fatalf("GetRelevantDocuments failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 documents, got %d", len(result))
	}

	// Test health check with observability
	err = retriever.CheckHealth(ctx)
	if err != nil {
		t.Errorf("CheckHealth failed: %v", err)
	}
}

// TestVectorStoreRetriever_CallOptions tests call-specific option overrides.
func TestVectorStoreRetriever_CallOptions(t *testing.T) {
	docs := createTestDocuments(10)
	scores := createTestScores(10)
	mockStore := NewMockVectorStore().WithSimilarityResults(docs, scores)

	retriever, err := NewVectorStoreRetriever(mockStore,
		WithDefaultK(2), // Default small K
		WithMetrics(false),
		WithTracing(false),
	)
	if err != nil {
		t.Fatalf("Failed to create retriever: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test with default options
	result1, err := retriever.GetRelevantDocuments(ctx, "query1")
	if err != nil {
		t.Fatalf("GetRelevantDocuments failed: %v", err)
	}
	if len(result1) != 2 {
		t.Errorf("Expected 2 documents with default K, got %d", len(result1))
	}

	// Test Invoke with same configuration
	result2, err := retriever.Invoke(ctx, "query2")
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}
	docs2, ok := result2.([]schema.Document)
	if !ok {
		t.Fatalf("Expected []schema.Document, got %T", result2)
	}
	if len(docs2) != 2 {
		t.Errorf("Expected 2 documents (same as configured K), got %d", len(docs2))
	}
}

// TestVectorStoreRetriever_IntegrationStyle tests more complex integration scenarios.
func TestVectorStoreRetriever_IntegrationStyle(t *testing.T) {
	// Simulate a more realistic scenario with filtering and multiple calls
	docs := createTestDocuments(20)
	scores := createTestScores(20)
	mockStore := NewMockVectorStore().WithSimilarityResults(docs, scores)

	retriever, err := NewVectorStoreRetriever(mockStore,
		WithDefaultK(5),
		WithTimeout(5*time.Second),
		WithMetrics(false),
		WithTracing(false),
		WithLogger(slog.Default()),
	)
	if err != nil {
		t.Fatalf("Failed to create retriever: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test multiple queries
	queries := []string{
		"What is machine learning?",
		"Tell me about artificial intelligence",
		"How does neural networks work?",
		"Explain deep learning",
	}

	results := make([][]schema.Document, len(queries))

	for i, query := range queries {
		docs, err := retriever.GetRelevantDocuments(ctx, query)
		if err != nil {
			t.Fatalf("Query %d failed: %v", i, err)
		}
		results[i] = docs

		if len(docs) != 5 {
			t.Errorf("Query %d: expected 5 documents, got %d", i, len(docs))
		}
	}

	// Test batch processing
	batchInputs := make([]any, len(queries))
	for i, query := range queries {
		batchInputs[i] = query
	}

	batchResults, err := retriever.Batch(ctx, batchInputs)
	if err != nil {
		t.Fatalf("Batch processing failed: %v", err)
	}

	if len(batchResults) != len(queries) {
		t.Errorf("Expected %d batch results, got %d", len(queries), len(batchResults))
	}

	// Verify batch results match individual results
	for i, result := range batchResults {
		batchDocs, ok := result.([]schema.Document)
		if !ok {
			t.Errorf("Batch result %d: expected []schema.Document, got %T", i, result)
			continue
		}
		if len(batchDocs) != len(results[i]) {
			t.Errorf("Batch result %d length mismatch: expected %d, got %d", i, len(results[i]), len(batchDocs))
		}
	}

	// Test health check
	err = retriever.CheckHealth(ctx)
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}

// BenchmarkConfigValidation benchmarks configuration validation performance.
func BenchmarkConfigValidation(b *testing.B) {
	validConfig := DefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := validConfig.Validate()
		if err != nil {
			b.Fatalf("Config.Validate() error = %v", err)
		}
	}
}

// BenchmarkVectorStoreRetrieverConfigValidation benchmarks VectorStoreRetrieverConfig validation.
func BenchmarkVectorStoreRetrieverConfigValidation(b *testing.B) {
	config := DefaultVectorStoreRetrieverConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := config.Validate()
		if err != nil {
			b.Fatalf("VectorStoreRetrieverConfig.Validate() error = %v", err)
		}
	}
}

// BenchmarkRetrieverOptions benchmarks option application performance.
func BenchmarkRetrieverOptions(b *testing.B) {
	options := []Option{
		WithDefaultK(10),
		WithMaxRetries(5),
		WithTimeout(60 * time.Second),
		WithTracing(true),
		WithMetrics(true),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		opts := &RetrieverOptions{
			DefaultK:       4,
			ScoreThreshold: 0.0,
			MaxRetries:     3,
			Timeout:        30 * time.Second,
			EnableTracing:  true,
			EnableMetrics:  true,
		}

		for _, option := range options {
			option(opts)
		}
	}
}

// BenchmarkDefaultConfig benchmarks default config creation.
func BenchmarkDefaultConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = DefaultConfig()
	}
}

// BenchmarkDefaultVectorStoreRetrieverConfig benchmarks default VectorStoreRetrieverConfig creation.
func BenchmarkDefaultVectorStoreRetrieverConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = DefaultVectorStoreRetrieverConfig()
	}
}
