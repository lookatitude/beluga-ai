// Package embeddings provides advanced test utilities and comprehensive mocks for testing embedding implementations.
// This file contains utilities designed to support both unit tests and integration tests.
//
// Test Coverage Exclusions:
//
// The following code paths are intentionally excluded from 100% coverage requirements:
//
// 1. Panic Recovery Paths:
//    - Panic handlers in concurrent test runners (ConcurrentTestRunner)
//    - These paths are difficult to test without causing actual panics in test code
//
// 2. Context Cancellation Edge Cases:
//    - Some context cancellation paths in embedding operations are difficult to reliably test
//    - Race conditions between context cancellation and embedding generation
//
// 3. Error Paths Requiring System Conditions:
//    - Network errors that require actual network failures (provider API calls)
//    - File system errors that require specific OS conditions
//    - Memory exhaustion scenarios during large batch operations
//
// 4. Provider-Specific Untestable Paths:
//    - Provider implementations have paths that require external service failures
//    - These are tested through integration tests rather than unit tests
//    - Cohere provider client initialization (not yet fully implemented)
//
// 5. Test Utility Functions:
//    - Helper functions in test_utils.go that are used by tests but not directly tested
//    - These are validated through their usage in actual test cases
//
// 6. Initialization Code:
//    - Package init() functions and global variable initialization
//    - These are executed automatically and difficult to test in isolation
//
// 7. Provider Mock Limitations:
//    - Some provider-specific error scenarios require actual API responses
//    - Rate limiting behavior that depends on actual API responses
//    - Authentication failures that require actual API validation
//
// All exclusions are documented here to maintain transparency about coverage goals.
// The target is 100% coverage of testable code paths, excluding the above categories.
package embeddings

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// AdvancedMockEmbedder provides a comprehensive mock implementation for testing.
type AdvancedMockEmbedder struct {
	lastHealthCheck time.Time
	errorToReturn   error
	mock.Mock
	modelName         string
	providerName      string
	healthState       string
	embeddings        [][]float32
	embeddingIndex    int
	simulateDelay     time.Duration
	rateLimitCount    int
	callCount         int
	dimension         int
	mu                sync.RWMutex
	shouldError       bool
	simulateRateLimit bool
}

// NewAdvancedMockEmbedder creates a new advanced mock with configurable behavior.
func NewAdvancedMockEmbedder(providerName, modelName string, dimension int, options ...MockEmbedderOption) *AdvancedMockEmbedder {
	mock := &AdvancedMockEmbedder{
		providerName: providerName,
		modelName:    modelName,
		dimension:    dimension,
		embeddings:   make([][]float32, 0),
		healthState:  "healthy",
	}

	// Apply options
	for _, opt := range options {
		opt(mock)
	}

	// Generate default embeddings if none provided
	if len(mock.embeddings) == 0 {
		mock.generateDefaultEmbeddings(10) // Generate 10 default embeddings
	}

	return mock
}

// MockEmbedderOption defines functional options for mock configuration.
type MockEmbedderOption func(*AdvancedMockEmbedder)

// WithMockError configures the mock to return errors.
func WithMockError(shouldError bool, err error) MockEmbedderOption {
	return func(e *AdvancedMockEmbedder) {
		e.shouldError = shouldError
		e.errorToReturn = err
	}
}

// WithMockEmbeddings sets predefined embeddings for the mock.
func WithMockEmbeddings(embeddings [][]float32) MockEmbedderOption {
	return func(e *AdvancedMockEmbedder) {
		e.embeddings = make([][]float32, len(embeddings))
		copy(e.embeddings, embeddings)
	}
}

// WithMockDelay adds artificial delay to mock operations.
func WithMockDelay(delay time.Duration) MockEmbedderOption {
	return func(e *AdvancedMockEmbedder) {
		e.simulateDelay = delay
	}
}

// WithMockRateLimit simulates rate limiting behavior.
func WithMockRateLimit(enabled bool) MockEmbedderOption {
	return func(e *AdvancedMockEmbedder) {
		e.simulateRateLimit = enabled
	}
}

// WithMockErrorCode configures the mock to return a specific EmbeddingError with the given error code.
func WithMockErrorCode(op, code string, underlyingErr error) MockEmbedderOption {
	return func(e *AdvancedMockEmbedder) {
		e.shouldError = true
		e.errorToReturn = NewEmbeddingError(op, code, underlyingErr)
	}
}

// WithMockNetworkError configures the mock to simulate network errors.
func WithMockNetworkError(op string) MockEmbedderOption {
	return WithMockErrorCode(op, ErrCodeNetworkError, errors.New("network connection failed"))
}

// WithMockTimeoutError configures the mock to simulate timeout errors.
func WithMockTimeoutError(op string) MockEmbedderOption {
	return WithMockErrorCode(op, ErrCodeTimeout, errors.New("operation timed out"))
}

// WithMockRateLimitError configures the mock to simulate rate limit errors.
func WithMockRateLimitError(op string) MockEmbedderOption {
	return WithMockErrorCode(op, ErrCodeRateLimit, errors.New("rate limit exceeded"))
}

// WithMockAuthenticationError configures the mock to simulate authentication errors.
func WithMockAuthenticationError(op string) MockEmbedderOption {
	return WithMockErrorCode(op, ErrCodeAuthentication, errors.New("authentication failed"))
}

// WithMockProviderError configures the mock to simulate provider errors.
func WithMockProviderError(op string, underlyingErr error) MockEmbedderOption {
	return WithMockErrorCode(op, ErrCodeProviderError, underlyingErr)
}

// generateDefaultEmbeddings creates random embeddings for testing.
func (e *AdvancedMockEmbedder) generateDefaultEmbeddings(count int) {
	rand.Seed(time.Now().UnixNano())
	e.embeddings = make([][]float32, count)

	for i := 0; i < count; i++ {
		embedding := make([]float32, e.dimension)
		for j := 0; j < e.dimension; j++ {
			embedding[j] = rand.Float32()*2 - 1 // Random values between -1 and 1
		}
		e.embeddings[i] = embedding
	}
}

// Mock implementation methods.
func (e *AdvancedMockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	e.mu.Lock()
	e.callCount++
	shouldError := e.shouldError
	errorToReturn := e.errorToReturn
	simulateRateLimit := e.simulateRateLimit
	rateLimitCount := e.rateLimitCount
	embeddingIndex := e.embeddingIndex
	embeddingsCopy := make([][]float32, len(e.embeddings))
	for i := range e.embeddings {
		embeddingsCopy[i] = make([]float32, len(e.embeddings[i]))
		copy(embeddingsCopy[i], e.embeddings[i])
	}
	dimension := e.dimension
	e.mu.Unlock()

	if e.simulateDelay > 0 {
		time.Sleep(e.simulateDelay)
	}

	e.mu.Lock()
	if simulateRateLimit && rateLimitCount > 5 {
		e.mu.Unlock()
		return nil, errors.New("rate limit exceeded")
	}
	e.rateLimitCount++
	newEmbeddingIndex := embeddingIndex
	e.mu.Unlock()

	if shouldError {
		return nil, errorToReturn
	}

	results := make([][]float32, len(texts))
	for i := range texts {
		e.mu.Lock()
		currentIndex := newEmbeddingIndex
		if currentIndex < len(embeddingsCopy) {
			// Create a copy to avoid shared memory issues
			embedding := make([]float32, len(embeddingsCopy[currentIndex]))
			copy(embedding, embeddingsCopy[currentIndex])
			results[i] = embedding
			newEmbeddingIndex = (currentIndex + 1) % len(embeddingsCopy)
			e.embeddingIndex = newEmbeddingIndex
		} else {
			e.mu.Unlock()
			// Generate random embedding if we run out
			embedding := make([]float32, dimension)
			for j := 0; j < dimension; j++ {
				embedding[j] = rand.Float32()*2 - 1
			}
			results[i] = embedding
			e.mu.Lock()
		}
		e.mu.Unlock()
	}

	return results, nil
}

func (e *AdvancedMockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	e.mu.Lock()
	e.callCount++
	shouldError := e.shouldError
	errorToReturn := e.errorToReturn
	simulateRateLimit := e.simulateRateLimit
	rateLimitCount := e.rateLimitCount
	embeddingIndex := e.embeddingIndex
	embeddingsCopy := make([][]float32, len(e.embeddings))
	for i := range e.embeddings {
		embeddingsCopy[i] = make([]float32, len(e.embeddings[i]))
		copy(embeddingsCopy[i], e.embeddings[i])
	}
	dimension := e.dimension
	e.mu.Unlock()

	if e.simulateDelay > 0 {
		time.Sleep(e.simulateDelay)
	}

	e.mu.Lock()
	if simulateRateLimit && rateLimitCount > 5 {
		e.mu.Unlock()
		return nil, errors.New("rate limit exceeded")
	}
	e.rateLimitCount++
	e.mu.Unlock()

	if shouldError {
		return nil, errorToReturn
	}

	e.mu.Lock()
	var embedding []float32
	if embeddingIndex < len(embeddingsCopy) {
		// Create a copy to avoid shared memory issues
		embedding = make([]float32, len(embeddingsCopy[embeddingIndex]))
		copy(embedding, embeddingsCopy[embeddingIndex])
		e.embeddingIndex = (embeddingIndex + 1) % len(embeddingsCopy)
		e.mu.Unlock()
		return embedding, nil
	}
	e.mu.Unlock()

	// Generate random embedding if we run out
	embedding = make([]float32, dimension)
	for j := 0; j < dimension; j++ {
		embedding[j] = rand.Float32()*2 - 1
	}
	return embedding, nil
}

func (e *AdvancedMockEmbedder) GetDimension(ctx context.Context) (int, error) {
	e.mu.Lock()
	e.callCount++
	e.mu.Unlock()

	if e.shouldError {
		return 0, e.errorToReturn
	}

	return e.dimension, nil
}

func (e *AdvancedMockEmbedder) GetCallCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.callCount
}

func (e *AdvancedMockEmbedder) GetModelName() string {
	return e.modelName
}

func (e *AdvancedMockEmbedder) GetProviderName() string {
	return e.providerName
}

func (e *AdvancedMockEmbedder) ResetRateLimit() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rateLimitCount = 0
}

func (e *AdvancedMockEmbedder) CheckHealth() map[string]any {
	e.lastHealthCheck = time.Now()
	return map[string]any{
		"status":           e.healthState,
		"provider":         e.providerName,
		"model":            e.modelName,
		"dimension":        e.dimension,
		"call_count":       e.callCount,
		"rate_limit_count": e.rateLimitCount,
		"last_checked":     e.lastHealthCheck,
	}
}

// Test data creation helpers

// CreateTestTexts creates a set of test texts for embedding.
func CreateTestTexts(count int) []string {
	texts := make([]string, count)
	for i := 0; i < count; i++ {
		texts[i] = fmt.Sprintf("This is test document %d with some sample content for embedding.", i+1)
	}
	return texts
}

// CreateTestEmbeddings creates a set of test embeddings.
func CreateTestEmbeddings(count, dimension int) [][]float32 {
	rand.Seed(time.Now().UnixNano())
	embeddings := make([][]float32, count)

	for i := 0; i < count; i++ {
		embedding := make([]float32, dimension)
		for j := 0; j < dimension; j++ {
			embedding[j] = rand.Float32()*2 - 1 // Values between -1 and 1
		}
		embeddings[i] = embedding
	}

	return embeddings
}

// CreateTestConfig creates a test embedding configuration.
func CreateTestConfig(provider string) Config {
	config := Config{}

	switch provider {
	case "openai":
		config.OpenAI = &OpenAIConfig{
			APIKey:     "test-api-key",
			Model:      "text-embedding-ada-002",
			BaseURL:    "",
			APIVersion: "",
			Timeout:    30 * time.Second,
			MaxRetries: 3,
			Enabled:    true,
		}
	case "ollama":
		config.Ollama = &OllamaConfig{
			ServerURL:  "http://localhost:11434",
			Model:      "nomic-embed-text",
			Timeout:    30 * time.Second,
			MaxRetries: 3,
			KeepAlive:  "5m",
			Enabled:    true,
		}
	case "mock":
		config.Mock = &MockConfig{
			Dimension:    128,
			Seed:         12345,
			RandomizeNil: false,
			Enabled:      true,
		}
	}

	return config
}

// Assertion helpers

// AssertEmbedding validates an embedding result.
func AssertEmbedding(t *testing.T, embedding []float32, expectedDim int) {
	assert.NotNil(t, embedding)
	assert.Len(t, embedding, expectedDim)

	// Check for valid float values (not NaN or infinite)
	for i, val := range embedding {
		assert.False(t, isNaN(val), "Embedding value at index %d is NaN", i)
		assert.False(t, isInf(val), "Embedding value at index %d is infinite", i)
	}
}

// AssertEmbeddings validates multiple embedding results.
func AssertEmbeddings(t *testing.T, embeddings [][]float32, expectedCount, expectedDim int) {
	t.Helper()
	assert.Len(t, embeddings, expectedCount)

	for i, embedding := range embeddings {
		AssertEmbedding(t, embedding, expectedDim)

		// Check that embeddings are not all zeros (unless intentional)
		hasNonZero := false
		for _, val := range embedding {
			if val != 0.0 {
				hasNonZero = true
				break
			}
		}
		assert.True(t, hasNonZero, "Embedding %d appears to be all zeros", i)
	}
}

// AssertSimilarityScore validates similarity between embeddings.
func AssertSimilarityScore(t *testing.T, emb1, emb2 []float32, minSimilarity float32) {
	t.Helper()
	similarity := CosineSimilarity(emb1, emb2)
	assert.GreaterOrEqual(t, similarity, minSimilarity,
		"Embeddings should have similarity >= %f, got %f", minSimilarity, similarity)
}

// AssertHealthCheck validates health check results.
func AssertHealthCheck(t *testing.T, health map[string]any, expectedStatus string) {
	t.Helper()
	assert.Contains(t, health, "status")
	assert.Equal(t, expectedStatus, health["status"])
	assert.Contains(t, health, "provider")
	assert.Contains(t, health, "model")
	assert.Contains(t, health, "dimension")
}

// AssertErrorType validates error types and codes.
func AssertErrorType(t *testing.T, err error, expectedCode string) {
	t.Helper()
	require.Error(t, err)
	var embErr *iface.EmbeddingError
	if assert.ErrorAs(t, err, &embErr) {
		assert.Equal(t, expectedCode, embErr.Code)
	}
}

// Helper functions

// CosineSimilarity calculates cosine similarity between two embeddings.
func CosineSimilarity(a, b []float32) float32 {
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

	return dotProduct / (sqrt(normA) * sqrt(normB))
}

// EuclideanDistance calculates Euclidean distance between two embeddings.
func EuclideanDistance(a, b []float32) float32 {
	if len(a) != len(b) {
		return float32(^uint(0) >> 1) // Max float32 value
	}

	var sum float32
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	return sqrt(sum)
}

// Helper functions for float operations.
func isNaN(f float32) bool {
	return f != f
}

func isInf(f float32) bool {
	// Use constants instead of division by zero
	const maxFloat32 = 3.40282347e+38
	return f > maxFloat32 || f < -maxFloat32
}

func sqrt(f float32) float32 {
	if f < 0 {
		return 0
	}
	if f == 0 {
		return 0
	}
	// Simple approximation for testing purposes
	x := f
	for i := 0; i < 10; i++ {
		x = (x + f/x) / 2
	}
	return x
}

// Performance testing helpers

// ConcurrentTestRunner runs embedding tests concurrently for performance testing.
type ConcurrentTestRunner struct {
	testFunc      func() error
	NumGoroutines int
	TestDuration  time.Duration
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

// RunLoadTest executes a load test scenario on embedder.
func RunLoadTest(t *testing.T, embedder *AdvancedMockEmbedder, numOperations, concurrency int) {
	t.Helper()
	var wg sync.WaitGroup
	errChan := make(chan error, numOperations)

	semaphore := make(chan struct{}, concurrency)
	texts := CreateTestTexts(5) // Reuse test texts

	for i := 0; i < numOperations; i++ {
		wg.Add(1)
		go func(opID int) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			ctx := context.Background()

			if opID%2 == 0 {
				// Test EmbedDocuments
				_, err := embedder.EmbedDocuments(ctx, texts)
				if err != nil {
					errChan <- err
				}
			} else {
				// Test EmbedQuery
				_, err := embedder.EmbedQuery(ctx, texts[0])
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
		require.NoError(t, err)
	}

	// Verify expected call count
	assert.Equal(t, numOperations, embedder.GetCallCount())
}

// Integration test helpers

// IntegrationTestHelper provides utilities for integration testing.
type IntegrationTestHelper struct {
	embedders map[string]*AdvancedMockEmbedder
	registry  *ProviderRegistry
}

func NewIntegrationTestHelper() *IntegrationTestHelper {
	return &IntegrationTestHelper{
		embedders: make(map[string]*AdvancedMockEmbedder),
		registry:  NewProviderRegistry(),
	}
}

func (h *IntegrationTestHelper) AddEmbedder(name string, embedder *AdvancedMockEmbedder) {
	h.embedders[name] = embedder
}

func (h *IntegrationTestHelper) GetEmbedder(name string) *AdvancedMockEmbedder {
	return h.embedders[name]
}

func (h *IntegrationTestHelper) GetRegistry() *ProviderRegistry {
	return h.registry
}

func (h *IntegrationTestHelper) Reset() {
	for _, embedder := range h.embedders {
		embedder.callCount = 0
		embedder.embeddingIndex = 0
		embedder.rateLimitCount = 0
	}
}

// EmbeddingBenchmark provides benchmarking utilities.
type EmbeddingBenchmark struct {
	embedder iface.Embedder
	texts    []string
}

func NewEmbeddingBenchmark(embedder iface.Embedder, textCount int) *EmbeddingBenchmark {
	return &EmbeddingBenchmark{
		embedder: embedder,
		texts:    CreateTestTexts(textCount),
	}
}

func (b *EmbeddingBenchmark) BenchmarkSingleEmbedding(iterations int) (time.Duration, error) {
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < iterations; i++ {
		_, err := b.embedder.EmbedQuery(ctx, b.texts[i%len(b.texts)])
		if err != nil {
			return 0, err
		}
	}

	return time.Since(start), nil
}

func (b *EmbeddingBenchmark) BenchmarkBatchEmbedding(batchSize, iterations int) (time.Duration, error) {
	ctx := context.Background()

	start := time.Now()
	for i := 0; i < iterations; i++ {
		batchTexts := b.texts[:batchSize]
		_, err := b.embedder.EmbedDocuments(ctx, batchTexts)
		if err != nil {
			return 0, err
		}
	}

	return time.Since(start), nil
}

// EmbeddingQualityTester provides utilities for testing embedding quality.
type EmbeddingQualityTester struct {
	embedder iface.Embedder
}

func NewEmbeddingQualityTester(embedder iface.Embedder) *EmbeddingQualityTester {
	return &EmbeddingQualityTester{embedder: embedder}
}

func (q *EmbeddingQualityTester) TestSimilarityConsistency(ctx context.Context, text string, iterations int) (float32, error) {
	embeddings := make([][]float32, iterations)

	// Generate multiple embeddings for the same text
	for i := 0; i < iterations; i++ {
		emb, err := q.embedder.EmbedQuery(ctx, text)
		if err != nil {
			return 0, err
		}
		embeddings[i] = emb
	}

	// Calculate average similarity between all pairs
	var totalSimilarity float32
	var pairCount int

	for i := 0; i < len(embeddings); i++ {
		for j := i + 1; j < len(embeddings); j++ {
			similarity := CosineSimilarity(embeddings[i], embeddings[j])
			totalSimilarity += similarity
			pairCount++
		}
	}

	if pairCount == 0 {
		return 1.0, nil // Single embedding, perfect consistency
	}

	return totalSimilarity / float32(pairCount), nil
}

func (q *EmbeddingQualityTester) TestSemanticSimilarity(ctx context.Context, similarTexts []string) (float32, error) {
	if len(similarTexts) < 2 {
		return 0, errors.New("need at least 2 texts to test semantic similarity")
	}

	embeddings := make([][]float32, len(similarTexts))

	// Generate embeddings for all texts
	for i, text := range similarTexts {
		emb, err := q.embedder.EmbedQuery(ctx, text)
		if err != nil {
			return 0, err
		}
		embeddings[i] = emb
	}

	// Calculate average similarity
	var totalSimilarity float32
	var pairCount int

	for i := 0; i < len(embeddings); i++ {
		for j := i + 1; j < len(embeddings); j++ {
			similarity := CosineSimilarity(embeddings[i], embeddings[j])
			totalSimilarity += similarity
			pairCount++
		}
	}

	return totalSimilarity / float32(pairCount), nil
}
