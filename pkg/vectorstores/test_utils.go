// Package vectorstores provides advanced test utilities and comprehensive mocks for testing vector store implementations.
// This file contains utilities designed to support both unit tests and integration tests.
package vectorstores

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/schema"
	vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// AdvancedMockVectorStore provides a comprehensive mock implementation for testing
type AdvancedMockVectorStore struct {
	mock.Mock

	// Configuration
	name      string
	callCount int
	mu        sync.RWMutex

	// Configurable behavior
	shouldError      bool
	errorToReturn    error
	simulateDelay    time.Duration
	simulateCapacity int

	// Storage
	documents   []schema.Document
	embeddings  [][]float32
	documentIDs []string
	nextID      int

	// Search behavior
	defaultK        int
	scoreThreshold  float32
	metadataFilters map[string]interface{}

	// Health check data
	healthState     string
	lastHealthCheck time.Time
}

// NewAdvancedMockVectorStore creates a new advanced mock with configurable behavior
func NewAdvancedMockVectorStore(name string, options ...MockVectorStoreOption) *AdvancedMockVectorStore {
	mock := &AdvancedMockVectorStore{
		name:             name,
		documents:        make([]schema.Document, 0),
		embeddings:       make([][]float32, 0),
		documentIDs:      make([]string, 0),
		nextID:           1,
		defaultK:         5,
		scoreThreshold:   0.0,
		metadataFilters:  make(map[string]interface{}),
		simulateCapacity: 10000, // Default capacity
		healthState:      "healthy",
	}

	// Apply options
	for _, opt := range options {
		opt(mock)
	}

	return mock
}

// MockVectorStoreOption defines functional options for mock configuration
type MockVectorStoreOption func(*AdvancedMockVectorStore)

// WithMockError configures the mock to return errors
func WithMockError(shouldError bool, err error) MockVectorStoreOption {
	return func(v *AdvancedMockVectorStore) {
		v.shouldError = shouldError
		v.errorToReturn = err
	}
}

// WithMockDelay adds artificial delay to mock operations
func WithMockDelay(delay time.Duration) MockVectorStoreOption {
	return func(v *AdvancedMockVectorStore) {
		v.simulateDelay = delay
	}
}

// WithMockCapacity sets the storage capacity limit
func WithMockCapacity(capacity int) MockVectorStoreOption {
	return func(v *AdvancedMockVectorStore) {
		v.simulateCapacity = capacity
	}
}

// WithPreloadedDocuments preloads documents into the mock
func WithPreloadedDocuments(documents []schema.Document, embeddings [][]float32) MockVectorStoreOption {
	return func(v *AdvancedMockVectorStore) {
		v.documents = make([]schema.Document, len(documents))
		copy(v.documents, documents)

		v.embeddings = make([][]float32, len(embeddings))
		for i, emb := range embeddings {
			v.embeddings[i] = make([]float32, len(emb))
			copy(v.embeddings[i], emb)
		}

		v.documentIDs = make([]string, len(documents))
		for i := range documents {
			v.documentIDs[i] = fmt.Sprintf("doc_%d", i+1)
		}
		v.nextID = len(documents) + 1
	}
}

// WithDefaultK sets the default number of results to return
func WithDefaultK(k int) MockVectorStoreOption {
	return func(v *AdvancedMockVectorStore) {
		v.defaultK = k
	}
}

// Mock implementation methods
func (v *AdvancedMockVectorStore) AddDocuments(ctx context.Context, documents []schema.Document, opts ...vectorstoresiface.Option) ([]string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.callCount++

	if v.simulateDelay > 0 {
		time.Sleep(v.simulateDelay)
	}

	if v.shouldError {
		return nil, v.errorToReturn
	}

	if len(v.documents)+len(documents) > v.simulateCapacity {
		return nil, fmt.Errorf("capacity exceeded: cannot store %d more documents", len(documents))
	}

	// Parse options
	config := &vectorstoresiface.Config{}
	for _, opt := range opts {
		opt(config)
	}

	ids := make([]string, len(documents))
	for i, doc := range documents {
		id := fmt.Sprintf("doc_%d", v.nextID)
		v.nextID++

		ids[i] = id
		v.documentIDs = append(v.documentIDs, id)
		v.documents = append(v.documents, doc)

		// Generate random embedding if embedder is available
		if config.Embedder != nil {
			embedding, err := config.Embedder.EmbedQuery(ctx, doc.GetContent())
			if err != nil {
				return nil, fmt.Errorf("failed to embed document: %w", err)
			}
			v.embeddings = append(v.embeddings, embedding)
		} else {
			// Generate random embedding for testing
			embedding := generateRandomEmbedding(128)
			v.embeddings = append(v.embeddings, embedding)
		}
	}

	return ids, nil
}

func (v *AdvancedMockVectorStore) DeleteDocuments(ctx context.Context, ids []string, opts ...vectorstoresiface.Option) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.callCount++

	if v.simulateDelay > 0 {
		time.Sleep(v.simulateDelay)
	}

	if v.shouldError {
		return v.errorToReturn
	}

	// Find and remove documents by ID
	for _, id := range ids {
		for i, docID := range v.documentIDs {
			if docID == id {
				// Remove from all slices
				v.documents = append(v.documents[:i], v.documents[i+1:]...)
				v.embeddings = append(v.embeddings[:i], v.embeddings[i+1:]...)
				v.documentIDs = append(v.documentIDs[:i], v.documentIDs[i+1:]...)
				break
			}
		}
	}

	return nil
}

func (v *AdvancedMockVectorStore) SimilaritySearch(ctx context.Context, queryVector []float32, k int, opts ...vectorstoresiface.Option) ([]schema.Document, []float32, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	v.callCount++

	if v.simulateDelay > 0 {
		time.Sleep(v.simulateDelay)
	}

	if v.shouldError {
		return nil, nil, v.errorToReturn
	}

	if len(v.embeddings) == 0 {
		return []schema.Document{}, []float32{}, nil
	}

	// Parse options
	config := &vectorstoresiface.Config{}
	for _, opt := range opts {
		opt(config)
	}

	scoreThreshold := v.scoreThreshold
	if config.ScoreThreshold > 0 {
		scoreThreshold = config.ScoreThreshold
	}

	// Calculate similarities
	type docScore struct {
		doc   schema.Document
		score float32
		index int
	}

	scores := make([]docScore, 0, len(v.embeddings))
	for i, embedding := range v.embeddings {
		similarity := cosineSimilarity(queryVector, embedding)
		if similarity >= scoreThreshold {
			scores = append(scores, docScore{
				doc:   v.documents[i],
				score: similarity,
				index: i,
			})
		}
	}

	// Sort by score (descending)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// Limit results
	if k > len(scores) {
		k = len(scores)
	}
	if k < 0 {
		k = len(scores)
	}

	results := make([]schema.Document, k)
	resultScores := make([]float32, k)
	for i := 0; i < k; i++ {
		results[i] = scores[i].doc
		resultScores[i] = scores[i].score
	}

	return results, resultScores, nil
}

func (v *AdvancedMockVectorStore) SimilaritySearchByQuery(ctx context.Context, query string, k int, embedder vectorstoresiface.Embedder, opts ...vectorstoresiface.Option) ([]schema.Document, []float32, error) {
	v.callCount++

	if v.simulateDelay > 0 {
		time.Sleep(v.simulateDelay)
	}

	if v.shouldError {
		return nil, nil, v.errorToReturn
	}

	// Generate embedding for query
	queryVector, err := embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to embed query: %w", err)
	}

	return v.SimilaritySearch(ctx, queryVector, k, opts...)
}

func (v *AdvancedMockVectorStore) AsRetriever(opts ...vectorstoresiface.Option) vectorstoresiface.Retriever {
	return &AdvancedMockRetriever{
		vectorStore: v,
		k:           v.defaultK,
	}
}

func (v *AdvancedMockVectorStore) GetName() string {
	return v.name
}

func (v *AdvancedMockVectorStore) GetCallCount() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.callCount
}

func (v *AdvancedMockVectorStore) GetDocumentCount() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.documents)
}

func (v *AdvancedMockVectorStore) GetDocuments() []schema.Document {
	v.mu.RLock()
	defer v.mu.RUnlock()
	docs := make([]schema.Document, len(v.documents))
	copy(docs, v.documents)
	return docs
}

func (v *AdvancedMockVectorStore) CheckHealth() map[string]interface{} {
	v.lastHealthCheck = time.Now()
	return map[string]interface{}{
		"status":         v.healthState,
		"name":           v.name,
		"document_count": len(v.documents),
		"capacity":       v.simulateCapacity,
		"call_count":     v.callCount,
		"last_checked":   v.lastHealthCheck,
	}
}

// AdvancedMockRetriever implements the Retriever interface for testing
type AdvancedMockRetriever struct {
	vectorStore vectorstoresiface.VectorStore
	k           int
}

func (r *AdvancedMockRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
	// This is a simplified implementation - in reality would need an embedder
	// For mock purposes, just return first k documents
	return []schema.Document{}, nil
}

// AdvancedMockEmbedder for vector store testing
type AdvancedMockEmbedder struct {
	dimension int
	mu        sync.RWMutex
}

func NewAdvancedMockEmbedder(dimension int) *AdvancedMockEmbedder {
	return &AdvancedMockEmbedder{dimension: dimension}
}

func (e *AdvancedMockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i := range texts {
		embeddings[i] = generateRandomEmbedding(e.dimension)
	}
	return embeddings, nil
}

func (e *AdvancedMockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return generateRandomEmbedding(e.dimension), nil
}

// Test data creation helpers

// CreateTestDocuments creates a set of test documents
func CreateTestDocuments(count int) []schema.Document {
	documents := make([]schema.Document, count)
	for i := 0; i < count; i++ {
		content := fmt.Sprintf("This is test document %d with unique content for testing vector storage and retrieval.", i+1)
		metadata := map[string]string{
			"id":       fmt.Sprintf("test_doc_%d", i+1),
			"category": fmt.Sprintf("category_%d", (i%3)+1),
			"priority": fmt.Sprintf("%d", (i%5)+1),
		}
		documents[i] = schema.NewDocument(content, metadata)
	}
	return documents
}

// CreateTestEmbeddings creates test embeddings for documents
func CreateTestEmbeddings(count, dimension int) [][]float32 {
	embeddings := make([][]float32, count)
	for i := 0; i < count; i++ {
		embeddings[i] = generateRandomEmbedding(dimension)
	}
	return embeddings
}

// CreateTestVectorStoreConfig creates a test vector store configuration
func CreateTestVectorStoreConfig() vectorstoresiface.Config {
	return vectorstoresiface.Config{
		SearchK:        5,
		ScoreThreshold: 0.0,
		MetadataFilters: map[string]interface{}{
			"category": "test",
		},
		ProviderConfig: map[string]interface{}{
			"dimension": 128,
		},
	}
}

// Helper functions

func generateRandomEmbedding(dimension int) []float32 {
	embedding := make([]float32, dimension)
	for i := 0; i < dimension; i++ {
		embedding[i] = rand.Float32()*2 - 1 // Random values between -1 and 1
	}
	return embedding
}

func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float32

	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// Assertion helpers

// AssertVectorStoreResults validates vector store search results
func AssertVectorStoreResults(t *testing.T, documents []schema.Document, scores []float32, expectedMinCount int, expectedMaxScore float32) {
	assert.GreaterOrEqual(t, len(documents), expectedMinCount, "Should return at least %d documents", expectedMinCount)
	assert.Equal(t, len(documents), len(scores), "Documents and scores should have same length")

	// Verify scores are in descending order
	for i := 1; i < len(scores); i++ {
		assert.GreaterOrEqual(t, scores[i-1], scores[i], "Scores should be in descending order")
	}

	// Verify scores are within valid range
	for i, score := range scores {
		assert.GreaterOrEqual(t, score, float32(-1.0), "Score %d should be >= -1.0", i)
		assert.LessOrEqual(t, score, expectedMaxScore, "Score %d should be <= %.2f", i, expectedMaxScore)
	}

	// Verify documents are not nil
	for i, doc := range documents {
		assert.NotNil(t, doc, "Document %d should not be nil", i)
		assert.NotEmpty(t, doc.GetContent(), "Document %d should have content", i)
	}
}

// AssertDocumentStorage validates document storage operations
func AssertDocumentStorage(t *testing.T, ids []string, expectedCount int) {
	assert.Len(t, ids, expectedCount, "Should return %d document IDs", expectedCount)

	// Verify all IDs are unique and non-empty
	idSet := make(map[string]bool)
	for i, id := range ids {
		assert.NotEmpty(t, id, "ID %d should not be empty", i)
		assert.False(t, idSet[id], "ID %s should be unique", id)
		idSet[id] = true
	}
}

// AssertHealthCheck validates health check results
func AssertHealthCheck(t *testing.T, health map[string]interface{}, expectedStatus string) {
	assert.Contains(t, health, "status")
	assert.Equal(t, expectedStatus, health["status"])
	assert.Contains(t, health, "name")
	assert.Contains(t, health, "document_count")
	assert.Contains(t, health, "call_count")
}

// AssertErrorType validates error types and codes
func AssertErrorType(t *testing.T, err error, expectedCode string) {
	assert.Error(t, err)
	var vsErr *vectorstoresiface.VectorStoreError
	if assert.ErrorAs(t, err, &vsErr) {
		assert.Equal(t, expectedCode, vsErr.Code)
	}
}

// Performance testing helpers

// ConcurrentTestRunner runs vector store tests concurrently for performance testing
type ConcurrentTestRunner struct {
	NumGoroutines int
	TestDuration  time.Duration
	testFunc      func() error
}

func NewConcurrentTestRunner(numGoroutines int, duration time.Duration, testFunc func() error) *ConcurrentTestRunner {
	return &ConcurrentTestRunner{
		NumGoroutines: numGoroutines,
		TestDuration:  duration,
		testFunc:      testFunc,
	}
}

func (r *ConcurrentTestRunner) Run() error {
	var wg sync.WaitGroup
	errChan := make(chan error, r.NumGoroutines)
	stopChan := make(chan struct{})

	// Start timer
	timer := time.AfterFunc(r.TestDuration, func() {
		close(stopChan)
	})
	defer timer.Stop()

	// Start worker goroutines
	for i := 0; i < r.NumGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopChan:
					return
				default:
					if err := r.testFunc(); err != nil {
						errChan <- err
						return
					}
				}
			}
		}()
	}

	// Wait for completion
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// RunLoadTest executes a load test scenario on vector store
func RunLoadTest(t *testing.T, vectorStore *AdvancedMockVectorStore, numOperations int, concurrency int) {
	var wg sync.WaitGroup
	errChan := make(chan error, numOperations)

	semaphore := make(chan struct{}, concurrency)
	testDocs := CreateTestDocuments(5) // Reuse test documents

	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(opID int) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			ctx := context.Background()

			if opID%3 == 0 {
				// Test AddDocuments
				_, err := vectorStore.AddDocuments(ctx, []schema.Document{testDocs[opID%len(testDocs)]})
				if err != nil {
					errChan <- err
				}
			} else if opID%3 == 1 {
				// Test SimilaritySearch
				queryVector := generateRandomEmbedding(128)
				_, _, err := vectorStore.SimilaritySearch(ctx, queryVector, 3)
				if err != nil {
					errChan <- err
				}
			} else {
				// Test SimilaritySearchByQuery (requires embedder)
				embedder := NewAdvancedMockEmbedder(128)
				_, _, err := vectorStore.SimilaritySearchByQuery(ctx, "test query", 3, embedder)
				if err != nil {
					errChan <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Verify no errors occurred
	for err := range errChan {
		assert.NoError(t, err)
	}

	// Verify expected call count
	assert.Equal(t, numOperations, vectorStore.GetCallCount())
}

// Integration test helpers

// IntegrationTestHelper provides utilities for integration testing
type IntegrationTestHelper struct {
	vectorStores map[string]*AdvancedMockVectorStore
	embedders    map[string]*AdvancedMockEmbedder
}

func NewIntegrationTestHelper() *IntegrationTestHelper {
	return &IntegrationTestHelper{
		vectorStores: make(map[string]*AdvancedMockVectorStore),
		embedders:    make(map[string]*AdvancedMockEmbedder),
	}
}

func (h *IntegrationTestHelper) AddVectorStore(name string, store *AdvancedMockVectorStore) {
	h.vectorStores[name] = store
}

func (h *IntegrationTestHelper) AddEmbedder(name string, embedder *AdvancedMockEmbedder) {
	h.embedders[name] = embedder
}

func (h *IntegrationTestHelper) GetVectorStore(name string) *AdvancedMockVectorStore {
	return h.vectorStores[name]
}

func (h *IntegrationTestHelper) GetEmbedder(name string) *AdvancedMockEmbedder {
	return h.embedders[name]
}

func (h *IntegrationTestHelper) Reset() {
	for _, store := range h.vectorStores {
		store.documents = store.documents[:0]
		store.embeddings = store.embeddings[:0]
		store.documentIDs = store.documentIDs[:0]
		store.callCount = 0
		store.nextID = 1
	}
}

// VectorStoreScenarioRunner runs common vector store scenarios
type VectorStoreScenarioRunner struct {
	vectorStore vectorstoresiface.VectorStore
	embedder    vectorstoresiface.Embedder
}

func NewVectorStoreScenarioRunner(store vectorstoresiface.VectorStore, embedder vectorstoresiface.Embedder) *VectorStoreScenarioRunner {
	return &VectorStoreScenarioRunner{
		vectorStore: store,
		embedder:    embedder,
	}
}

func (r *VectorStoreScenarioRunner) RunDocumentIngestionScenario(ctx context.Context, documentCount int) error {
	documents := CreateTestDocuments(documentCount)

	// Add documents in batches
	batchSize := 10
	for i := 0; i < len(documents); i += batchSize {
		end := i + batchSize
		if end > len(documents) {
			end = len(documents)
		}

		batch := documents[i:end]
		_, err := r.vectorStore.AddDocuments(ctx, batch, vectorstoresiface.WithEmbedder(r.embedder))
		if err != nil {
			return fmt.Errorf("failed to add document batch: %w", err)
		}
	}

	return nil
}

func (r *VectorStoreScenarioRunner) RunSimilaritySearchScenario(ctx context.Context, queries []string, k int) ([][]schema.Document, error) {
	results := make([][]schema.Document, len(queries))

	for i, query := range queries {
		docs, _, err := r.vectorStore.SimilaritySearchByQuery(ctx, query, k, r.embedder)
		if err != nil {
			return nil, fmt.Errorf("failed to search for query '%s': %w", query, err)
		}
		results[i] = docs
	}

	return results, nil
}

func (r *VectorStoreScenarioRunner) RunDocumentDeletionScenario(ctx context.Context, idsToDelete []string) error {
	return r.vectorStore.DeleteDocuments(ctx, idsToDelete)
}

// BenchmarkHelper provides benchmarking utilities for vector stores
type BenchmarkHelper struct {
	vectorStore vectorstoresiface.VectorStore
	embedder    vectorstoresiface.Embedder
	documents   []schema.Document
}

func NewBenchmarkHelper(store vectorstoresiface.VectorStore, embedder vectorstoresiface.Embedder, docCount int) *BenchmarkHelper {
	return &BenchmarkHelper{
		vectorStore: store,
		embedder:    embedder,
		documents:   CreateTestDocuments(docCount),
	}
}

func (b *BenchmarkHelper) BenchmarkAddDocuments(batchSize, iterations int) (time.Duration, error) {
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < iterations; i++ {
		batchDocs := b.documents[:batchSize]
		_, err := b.vectorStore.AddDocuments(ctx, batchDocs, vectorstoresiface.WithEmbedder(b.embedder))
		if err != nil {
			return 0, err
		}
	}

	return time.Since(start), nil
}

func (b *BenchmarkHelper) BenchmarkSimilaritySearch(k, iterations int) (time.Duration, error) {
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < iterations; i++ {
		query := fmt.Sprintf("test query %d", i)
		_, _, err := b.vectorStore.SimilaritySearchByQuery(ctx, query, k, b.embedder)
		if err != nil {
			return 0, err
		}
	}

	return time.Since(start), nil
}
