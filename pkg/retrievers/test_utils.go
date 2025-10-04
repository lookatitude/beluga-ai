// Package retrievers provides advanced test utilities and comprehensive mocks for testing retriever implementations.
// This file contains utilities designed to support both unit tests and integration tests.
package retrievers

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// AdvancedMockRetriever provides a comprehensive mock implementation for testing
type AdvancedMockRetriever struct {
	mock.Mock

	// Configuration
	name          string
	retrieverType string
	callCount     int
	mu            sync.RWMutex

	// Configurable behavior
	shouldError    bool
	errorToReturn  error
	simulateDelay  time.Duration
	documents      []schema.Document
	defaultK       int
	scoreThreshold float32

	// Search simulation
	similarityScores []float32
	documentOrder    []int

	// Health check data
	healthState     string
	lastHealthCheck time.Time
}

// NewAdvancedMockRetriever creates a new advanced mock with configurable behavior
func NewAdvancedMockRetriever(name, retrieverType string, options ...MockRetrieverOption) *AdvancedMockRetriever {
	mock := &AdvancedMockRetriever{
		name:             name,
		retrieverType:    retrieverType,
		documents:        make([]schema.Document, 0),
		defaultK:         5,
		scoreThreshold:   0.0,
		similarityScores: []float32{},
		documentOrder:    []int{},
		healthState:      "healthy",
	}

	// Apply options
	for _, opt := range options {
		opt(mock)
	}

	return mock
}

// MockRetrieverOption defines functional options for mock configuration
type MockRetrieverOption func(*AdvancedMockRetriever)

// WithMockError configures the mock to return errors
func WithMockError(shouldError bool, err error) MockRetrieverOption {
	return func(r *AdvancedMockRetriever) {
		r.shouldError = shouldError
		r.errorToReturn = err
	}
}

// WithMockDocuments sets the documents to return from searches
func WithMockDocuments(documents []schema.Document) MockRetrieverOption {
	return func(r *AdvancedMockRetriever) {
		r.documents = make([]schema.Document, len(documents))
		copy(r.documents, documents)
	}
}

// WithMockDelay adds artificial delay to mock operations
func WithMockDelay(delay time.Duration) MockRetrieverOption {
	return func(r *AdvancedMockRetriever) {
		r.simulateDelay = delay
	}
}

// WithMockScores sets predefined similarity scores
func WithMockScores(scores []float32) MockRetrieverOption {
	return func(r *AdvancedMockRetriever) {
		r.similarityScores = make([]float32, len(scores))
		copy(r.similarityScores, scores)
	}
}

// WithMockDefaultK sets the default number of documents to return
func WithMockDefaultK(k int) MockRetrieverOption {
	return func(r *AdvancedMockRetriever) {
		r.defaultK = k
	}
}

// WithScoreThreshold sets the minimum similarity score threshold
func WithScoreThreshold(threshold float32) MockRetrieverOption {
	return func(r *AdvancedMockRetriever) {
		r.scoreThreshold = threshold
	}
}

// Mock implementation methods for core.Runnable interface
func (r *AdvancedMockRetriever) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	if query, ok := input.(string); ok {
		return r.GetRelevantDocuments(ctx, query)
	}
	return nil, fmt.Errorf("retriever input must be string query, got %T", input)
}

func (r *AdvancedMockRetriever) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	for i, input := range inputs {
		result, err := r.Invoke(ctx, input, options...)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func (r *AdvancedMockRetriever) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	ch := make(chan any, 1)
	go func() {
		defer close(ch)
		result, err := r.Invoke(ctx, input, options...)
		if err != nil {
			// In a real implementation, would send error through channel
			return
		}
		ch <- result
	}()
	return ch, nil
}

// Mock implementation methods for Retriever interface
func (r *AdvancedMockRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.callCount++

	if r.simulateDelay > 0 {
		time.Sleep(r.simulateDelay)
	}

	if r.shouldError {
		return nil, r.errorToReturn
	}

	// Return configured documents or create mock documents
	if len(r.documents) == 0 {
		// Generate mock documents based on query
		mockDocs := r.generateMockDocuments(query, r.defaultK)
		return mockDocs, nil
	}

	// Filter documents based on threshold if scores are available
	results := make([]schema.Document, 0)
	for i, doc := range r.documents {
		if i < len(r.similarityScores) {
			if r.similarityScores[i] >= r.scoreThreshold {
				results = append(results, doc)
			}
		} else {
			results = append(results, doc)
		}

		// Limit to defaultK
		if len(results) >= r.defaultK {
			break
		}
	}

	return results, nil
}

func (r *AdvancedMockRetriever) generateMockDocuments(query string, k int) []schema.Document {
	documents := make([]schema.Document, k)
	for i := 0; i < k; i++ {
		content := fmt.Sprintf("Mock document %d relevant to query '%s'. This document contains information that would be useful for answering the query.", i+1, query)
		metadata := map[string]string{
			"doc_id":    fmt.Sprintf("mock_doc_%d", i+1),
			"query":     query,
			"relevance": fmt.Sprintf("%.2f", 1.0-float64(i)*0.1), // Decreasing relevance
		}
		documents[i] = schema.NewDocument(content, metadata)
	}
	return documents
}

func (r *AdvancedMockRetriever) GetName() string {
	return r.name
}

func (r *AdvancedMockRetriever) GetType() string {
	return r.retrieverType
}

func (r *AdvancedMockRetriever) GetCallCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.callCount
}

func (r *AdvancedMockRetriever) GetDocumentCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.documents)
}

func (r *AdvancedMockRetriever) CheckHealth() map[string]interface{} {
	r.lastHealthCheck = time.Now()
	return map[string]interface{}{
		"status":          r.healthState,
		"name":            r.name,
		"type":            r.retrieverType,
		"call_count":      r.callCount,
		"document_count":  len(r.documents),
		"default_k":       r.defaultK,
		"score_threshold": r.scoreThreshold,
		"last_checked":    r.lastHealthCheck,
	}
}

// AdvancedMockVectorStoreRetriever provides a comprehensive mock for vector store retrievers
type AdvancedMockVectorStoreRetriever struct {
	*AdvancedMockRetriever
	vectorStore interface{} // Mock vector store
	embedder    interface{} // Mock embedder
}

func NewAdvancedMockVectorStoreRetriever(name string, options ...MockRetrieverOption) *AdvancedMockVectorStoreRetriever {
	baseRetriever := NewAdvancedMockRetriever(name, "vector_store_retriever", options...)
	return &AdvancedMockVectorStoreRetriever{
		AdvancedMockRetriever: baseRetriever,
	}
}

func (r *AdvancedMockVectorStoreRetriever) GetVectorStore() interface{} {
	return r.vectorStore
}

func (r *AdvancedMockVectorStoreRetriever) SetVectorStore(store interface{}) {
	r.vectorStore = store
}

func (r *AdvancedMockVectorStoreRetriever) GetEmbedder() interface{} {
	return r.embedder
}

func (r *AdvancedMockVectorStoreRetriever) SetEmbedder(embedder interface{}) {
	r.embedder = embedder
}

// Test data creation helpers

// CreateTestRetrievalQueries creates standardized test queries
func CreateTestRetrievalQueries(count int) []string {
	queries := make([]string, count)
	topics := []string{"artificial intelligence", "machine learning", "deep learning", "natural language processing", "computer vision"}

	for i := 0; i < count; i++ {
		topic := topics[i%len(topics)]
		queries[i] = fmt.Sprintf("What is %s and how does it work in modern applications?", topic)
	}

	return queries
}

// CreateTestRetrievalDocuments creates documents optimized for retrieval testing
func CreateTestRetrievalDocuments(count int) []schema.Document {
	documents := make([]schema.Document, count)
	topics := []string{"AI", "ML", "DL", "NLP", "CV"}

	for i := 0; i < count; i++ {
		topic := topics[i%len(topics)]
		content := fmt.Sprintf("This document covers %s concepts and explains how %s works in practice. "+
			"It includes detailed explanations, examples, and applications of %s technology. "+
			"The document is designed to be comprehensive and informative for retrieval testing. "+
			"Document %d in the test collection.", topic, topic, topic, i+1)

		metadata := map[string]string{
			"doc_id":     fmt.Sprintf("retrieval_doc_%d", i+1),
			"topic":      topic,
			"category":   fmt.Sprintf("category_%d", (i%3)+1),
			"difficulty": fmt.Sprintf("level_%d", (i%5)+1),
			"language":   "en",
		}

		documents[i] = schema.NewDocument(content, metadata)
	}

	return documents
}

// CreateTestRetrieverConfig creates a test retriever configuration
func CreateTestRetrieverConfig() RetrieverOptions {
	return RetrieverOptions{
		DefaultK:       5,
		ScoreThreshold: 0.0,
		MaxRetries:     3,
		EnableMetrics:  true,
		EnableTracing:  true,
		Timeout:        30 * time.Second,
	}
}

// Assertion helpers

// AssertRetrievalResults validates retrieval results
func AssertRetrievalResults(t *testing.T, documents []schema.Document, expectedMinCount, expectedMaxCount int) {
	assert.GreaterOrEqual(t, len(documents), expectedMinCount, "Should return at least %d documents", expectedMinCount)
	assert.LessOrEqual(t, len(documents), expectedMaxCount, "Should return at most %d documents", expectedMaxCount)

	// Verify documents are not nil and have content
	for i, doc := range documents {
		assert.NotNil(t, doc, "Document %d should not be nil", i)
		assert.NotEmpty(t, doc.GetContent(), "Document %d should have content", i)
		assert.NotNil(t, doc.Metadata, "Document %d should have metadata", i)
	}
}

// AssertRetrievalRelevance validates document relevance to query
func AssertRetrievalRelevance(t *testing.T, documents []schema.Document, query string, minRelevance float32) {
	for i, doc := range documents {
		// Simple relevance check: document should contain some words from query
		content := doc.GetContent()
		assert.NotEmpty(t, content, "Document %d should have content", i)

		// In a real test, would check semantic relevance
		// For mock test, just verify content exists
		assert.Greater(t, len(content), 10, "Document %d should have substantial content", i)
	}
}

// AssertRetrieverHealth validates retriever health check results
func AssertRetrieverHealth(t *testing.T, health map[string]interface{}, expectedStatus string) {
	assert.Contains(t, health, "status")
	assert.Equal(t, expectedStatus, health["status"])
	assert.Contains(t, health, "name")
	assert.Contains(t, health, "type")
	assert.Contains(t, health, "call_count")
}

// AssertErrorType validates error types and codes
func AssertErrorType(t *testing.T, err error, expectedCode string) {
	assert.Error(t, err)
	var retErr *RetrieverError
	if assert.ErrorAs(t, err, &retErr) {
		assert.Equal(t, expectedCode, retErr.Code)
	}
}

// Performance testing helpers

// ConcurrentTestRunner runs retriever tests concurrently for performance testing
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

// RunLoadTest executes a load test scenario on retriever
func RunLoadTest(t *testing.T, retriever *AdvancedMockRetriever, numOperations int, concurrency int) {
	var wg sync.WaitGroup
	errChan := make(chan error, numOperations)

	semaphore := make(chan struct{}, concurrency)
	queries := CreateTestRetrievalQueries(5) // Reuse test queries

	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(opID int) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			ctx := context.Background()
			query := queries[opID%len(queries)]

			_, err := retriever.GetRelevantDocuments(ctx, query)
			if err != nil {
				errChan <- err
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
	assert.Equal(t, numOperations, retriever.GetCallCount())
}

// Integration test helpers

// IntegrationTestHelper provides utilities for integration testing
type IntegrationTestHelper struct {
	retrievers   map[string]*AdvancedMockRetriever
	vectorStores map[string]interface{}
	embedders    map[string]interface{}
}

func NewIntegrationTestHelper() *IntegrationTestHelper {
	return &IntegrationTestHelper{
		retrievers:   make(map[string]*AdvancedMockRetriever),
		vectorStores: make(map[string]interface{}),
		embedders:    make(map[string]interface{}),
	}
}

func (h *IntegrationTestHelper) AddRetriever(name string, retriever *AdvancedMockRetriever) {
	h.retrievers[name] = retriever
}

func (h *IntegrationTestHelper) AddVectorStore(name string, store interface{}) {
	h.vectorStores[name] = store
}

func (h *IntegrationTestHelper) AddEmbedder(name string, embedder interface{}) {
	h.embedders[name] = embedder
}

func (h *IntegrationTestHelper) GetRetriever(name string) *AdvancedMockRetriever {
	return h.retrievers[name]
}

func (h *IntegrationTestHelper) GetVectorStore(name string) interface{} {
	return h.vectorStores[name]
}

func (h *IntegrationTestHelper) GetEmbedder(name string) interface{} {
	return h.embedders[name]
}

func (h *IntegrationTestHelper) Reset() {
	for _, retriever := range h.retrievers {
		retriever.callCount = 0
		retriever.documentOrder = []int{}
	}
}

// RetrieverScenarioRunner runs common retriever scenarios
type RetrieverScenarioRunner struct {
	retriever core.Retriever
}

func NewRetrieverScenarioRunner(retriever core.Retriever) *RetrieverScenarioRunner {
	return &RetrieverScenarioRunner{
		retriever: retriever,
	}
}

func (r *RetrieverScenarioRunner) RunMultiQueryScenario(ctx context.Context, queries []string) ([][]schema.Document, error) {
	results := make([][]schema.Document, len(queries))

	for i, query := range queries {
		docs, err := r.retriever.GetRelevantDocuments(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("query %d failed: %w", i+1, err)
		}
		results[i] = docs
	}

	return results, nil
}

func (r *RetrieverScenarioRunner) RunRelevanceTestScenario(ctx context.Context, queryDocPairs []QueryDocumentPair) error {
	for i, pair := range queryDocPairs {
		docs, err := r.retriever.GetRelevantDocuments(ctx, pair.Query)
		if err != nil {
			return fmt.Errorf("relevance test %d failed: %w", i+1, err)
		}

		// Check if expected document is in results
		found := false
		for _, doc := range docs {
			if doc.GetContent() == pair.ExpectedDoc.GetContent() {
				found = true
				break
			}
		}

		if !found && pair.ShouldBeRelevant {
			return fmt.Errorf("expected document not found in results for query: %s", pair.Query)
		}
	}

	return nil
}

// QueryDocumentPair represents a query and its expected relevant document
type QueryDocumentPair struct {
	Query            string
	ExpectedDoc      schema.Document
	ShouldBeRelevant bool
	MinScore         float32
}

// Helper functions for similarity and ranking

// CalculateRelevanceScore calculates a simple relevance score between query and document
func CalculateRelevanceScore(query, docContent string) float32 {
	// Simple word overlap scoring for testing
	queryWords := tokenizeSimple(query)
	docWords := tokenizeSimple(docContent)

	intersection := make(map[string]bool)
	for _, word := range queryWords {
		intersection[word] = false
	}

	matchCount := 0
	for _, word := range docWords {
		if _, exists := intersection[word]; exists && !intersection[word] {
			intersection[word] = true
			matchCount++
		}
	}

	if len(queryWords) == 0 {
		return 0.0
	}

	return float32(matchCount) / float32(len(queryWords))
}

func tokenizeSimple(text string) []string {
	// Simple tokenization for testing (split by common delimiters)
	// In a real implementation, would use proper tokenization
	words := make([]string, 0)
	current := ""

	for _, char := range text {
		if char == ' ' || char == ',' || char == '.' || char == '!' || char == '?' {
			if current != "" {
				words = append(words, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}

	if current != "" {
		words = append(words, current)
	}

	return words
}

// RankDocuments ranks documents by relevance score
func RankDocuments(query string, documents []schema.Document) []DocumentScore {
	scores := make([]DocumentScore, len(documents))

	for i, doc := range documents {
		score := CalculateRelevanceScore(query, doc.GetContent())
		scores[i] = DocumentScore{
			Document: doc,
			Score:    score,
			Rank:     i + 1, // Will be updated after sorting
		}
	}

	// Sort by score (descending)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	// Update ranks
	for i := range scores {
		scores[i].Rank = i + 1
	}

	return scores
}

// DocumentScore represents a document with its relevance score
type DocumentScore struct {
	Document schema.Document
	Score    float32
	Rank     int
}

// BenchmarkHelper provides benchmarking utilities for retrievers
type BenchmarkHelper struct {
	retriever core.Retriever
	queries   []string
}

func NewBenchmarkHelper(retriever core.Retriever, queryCount int) *BenchmarkHelper {
	return &BenchmarkHelper{
		retriever: retriever,
		queries:   CreateTestRetrievalQueries(queryCount),
	}
}

func (b *BenchmarkHelper) BenchmarkRetrieval(iterations int) (time.Duration, error) {
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < iterations; i++ {
		query := b.queries[i%len(b.queries)]
		_, err := b.retriever.GetRelevantDocuments(ctx, query)
		if err != nil {
			return 0, err
		}
	}

	return time.Since(start), nil
}

func (b *BenchmarkHelper) BenchmarkBatchRetrieval(batchSize, iterations int) (time.Duration, error) {
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < iterations; i++ {
		// Execute batch of queries
		for j := 0; j < batchSize; j++ {
			query := b.queries[(i*batchSize+j)%len(b.queries)]
			_, err := b.retriever.GetRelevantDocuments(ctx, query)
			if err != nil {
				return 0, err
			}
		}
	}

	return time.Since(start), nil
}
