// Package embeddings provides advanced test utilities and comprehensive mocks for testing embedding implementations.
// This file contains utilities designed to support both unit tests and integration tests.
package embeddings

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// AdvancedMockEmbedder provides a comprehensive mock implementation for testing
type AdvancedMockEmbedder struct {
	mock.Mock

	// Configuration
	modelName    string
	providerName string
	dimension    int
	callCount    int
	mu           sync.RWMutex

	// Configurable behavior
	shouldError       bool
	errorToReturn     error
	embeddings        [][]float32
	embeddingIndex    int
	simulateDelay     time.Duration
	simulateRateLimit bool
	rateLimitCount    int

	// Health check data
	healthState     string
	lastHealthCheck time.Time
}

// NewAdvancedMockEmbedder creates a new advanced mock with configurable behavior
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

	// Don't generate default embeddings - use deterministic generation instead

	return mock
}

// MockEmbedderOption defines functional options for mock configuration
type MockEmbedderOption func(*AdvancedMockEmbedder)

// WithMockError configures the mock to return errors
func WithMockError(shouldError bool, err error) MockEmbedderOption {
	return func(e *AdvancedMockEmbedder) {
		e.shouldError = shouldError
		e.errorToReturn = err
	}
}

// WithMockEmbeddings sets predefined embeddings for the mock
func WithMockEmbeddings(embeddings [][]float32) MockEmbedderOption {
	return func(e *AdvancedMockEmbedder) {
		e.embeddings = make([][]float32, len(embeddings))
		copy(e.embeddings, embeddings)
	}
}

// WithMockDelay adds artificial delay to mock operations
func WithMockDelay(delay time.Duration) MockEmbedderOption {
	return func(e *AdvancedMockEmbedder) {
		e.simulateDelay = delay
	}
}

// WithMockRateLimit simulates rate limiting behavior
func WithMockRateLimit(enabled bool) MockEmbedderOption {
	return func(e *AdvancedMockEmbedder) {
		e.simulateRateLimit = enabled
	}
}

// generateDefaultEmbeddings creates random embeddings for testing
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

// Mock implementation methods
func (e *AdvancedMockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	e.mu.Lock()
	e.callCount++
	e.mu.Unlock()

	if e.simulateDelay > 0 {
		time.Sleep(e.simulateDelay)
	}

	if e.simulateRateLimit && e.rateLimitCount >= 5 {
		return nil, fmt.Errorf("rate limit exceeded")
	}
	e.rateLimitCount++

	if e.shouldError {
		return nil, e.errorToReturn
	}

	results := make([][]float32, len(texts))
	for i := range texts {
		if e.embeddingIndex < len(e.embeddings) {
			// Create a copy to avoid shared memory issues
			embedding := make([]float32, len(e.embeddings[e.embeddingIndex]))
			copy(embedding, e.embeddings[e.embeddingIndex])
			results[i] = embedding
			e.embeddingIndex = (e.embeddingIndex + 1) % len(e.embeddings)
		} else {
			// Generate random embedding if we run out
			embedding := make([]float32, e.dimension)
			for j := 0; j < e.dimension; j++ {
				embedding[j] = rand.Float32()*2 - 1
			}
			results[i] = embedding
		}
	}

	return results, nil
}

func (e *AdvancedMockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	e.mu.Lock()
	e.callCount++
	e.mu.Unlock()

	if e.simulateDelay > 0 {
		time.Sleep(e.simulateDelay)
	}

	if e.simulateRateLimit && e.rateLimitCount >= 5 {
		return nil, fmt.Errorf("rate limit exceeded")
	}
	e.rateLimitCount++

	if e.shouldError {
		return nil, e.errorToReturn
	}

	if e.embeddingIndex < len(e.embeddings) {
		// Create a copy to avoid shared memory issues
		embedding := make([]float32, len(e.embeddings[e.embeddingIndex]))
		copy(embedding, e.embeddings[e.embeddingIndex])
		e.embeddingIndex = (e.embeddingIndex + 1) % len(e.embeddings)
		return embedding, nil
	}

	// Generate deterministic embedding based on text hash for consistency
	embedding := make([]float32, e.dimension)
	// Use text hash as seed for deterministic but varied embeddings
	textHash := hashString(text)
	r := rand.New(rand.NewSource(int64(textHash)))
	for j := 0; j < e.dimension; j++ {
		embedding[j] = float32(r.Float64()*2 - 1) // Values between -1 and 1
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

func (e *AdvancedMockEmbedder) CheckHealth() map[string]interface{} {
	e.lastHealthCheck = time.Now()
	return map[string]interface{}{
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

// CreateTestTexts creates a set of test texts for embedding
func CreateTestTexts(count int) []string {
	texts := make([]string, count)
	for i := 0; i < count; i++ {
		texts[i] = fmt.Sprintf("This is test document %d with some sample content for embedding.", i+1)
	}
	return texts
}

// CreateTestEmbeddings creates a set of test embeddings
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

// CreateTestConfig creates a test embedding configuration
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

// AssertEmbedding validates an embedding result
func AssertEmbedding(t *testing.T, embedding []float32, expectedDim int) {
	assert.NotNil(t, embedding)
	assert.Len(t, embedding, expectedDim)

	// Check for valid float values (not NaN or infinite)
	for i, val := range embedding {
		assert.False(t, isNaN(val), "Embedding value at index %d is NaN", i)
		assert.False(t, isInf(val), "Embedding value at index %d is infinite", i)
	}
}

// AssertEmbeddings validates multiple embedding results
func AssertEmbeddings(t *testing.T, embeddings [][]float32, expectedCount, expectedDim int) {
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

// AssertSimilarityScore validates similarity between embeddings
func AssertSimilarityScore(t *testing.T, emb1, emb2 []float32, minSimilarity float32) {
	similarity := CosineSimilarity(emb1, emb2)
	assert.GreaterOrEqual(t, similarity, minSimilarity,
		"Embeddings should have similarity >= %f, got %f", minSimilarity, similarity)
}

// AssertHealthCheck validates health check results
func AssertHealthCheck(t *testing.T, health map[string]interface{}, expectedStatus string) {
	assert.Contains(t, health, "status")
	assert.Equal(t, expectedStatus, health["status"])
	assert.Contains(t, health, "provider")
	assert.Contains(t, health, "model")
	assert.Contains(t, health, "dimension")
}

// AssertErrorType validates error types and codes
func AssertErrorType(t *testing.T, err error, expectedCode string) {
	assert.Error(t, err)
	var embErr *iface.EmbeddingError
	if assert.ErrorAs(t, err, &embErr) {
		assert.Equal(t, expectedCode, embErr.Code)
	}
}

// Helper functions

// CosineSimilarity calculates cosine similarity between two embeddings
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

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// hashString creates a simple hash of a string for deterministic seeding
func hashString(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

// EuclideanDistance calculates Euclidean distance between two embeddings
func EuclideanDistance(a, b []float32) float32 {
	if len(a) != len(b) {
		return float32(^uint(0) >> 1) // Max float32 value
	}

	var sum float32
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	return float32(math.Sqrt(float64(sum)))
}

// Helper functions for float operations
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
	// Simple approximation for testing purposes
	x := f
	for i := 0; i < 10; i++ {
		x = (x + f/x) / 2
	}
	return x
}

// Performance testing helpers

// ConcurrentTestRunner runs embedding tests concurrently for performance testing
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

// RunLoadTest executes a load test scenario on embedder
func RunLoadTest(t *testing.T, embedder *AdvancedMockEmbedder, numOperations int, concurrency int) {
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
		assert.NoError(t, err)
	}

	// Verify expected call count
	assert.Equal(t, numOperations, embedder.GetCallCount())
}

// Integration test helpers

// IntegrationTestHelper provides utilities for integration testing
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

// EmbeddingBenchmark provides benchmarking utilities
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

// EmbeddingQualityTester provides utilities for testing embedding quality
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
		return 0, fmt.Errorf("need at least 2 texts to test semantic similarity")
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

// ConcurrentLoadTestResult holds results from concurrent load testing
type ConcurrentLoadTestResult struct {
	TotalOperations int64
	TotalErrors     int64
	Duration        time.Duration
	OpsPerSecond    float64
	ErrorRate       float64
}

// ConcurrentLoadTestConfig configures concurrent load testing parameters
type ConcurrentLoadTestConfig struct {
	NumWorkers          int
	OperationsPerWorker int
	TestDocuments       []string
	Timeout             time.Duration
}

// RunConcurrentLoadTest executes concurrent embedding operations under load
func RunConcurrentLoadTest(ctx context.Context, embedder iface.Embedder, config ConcurrentLoadTestConfig) (*ConcurrentLoadTestResult, error) {
	if config.NumWorkers <= 0 {
		config.NumWorkers = 5
	}
	if config.OperationsPerWorker <= 0 {
		config.OperationsPerWorker = 100
	}
	if len(config.TestDocuments) == 0 {
		config.TestDocuments = []string{
			"This is a test document for concurrent load testing.",
			"Another document with different content for variety.",
			"Short text.",
			"A longer document that contains more words and provides better testing coverage for the embedding system under concurrent load conditions.",
		}
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	result := &ConcurrentLoadTestResult{}
	start := time.Now()

	// Channel to collect results from workers
	results := make(chan WorkerResult, config.NumWorkers)

	// Start workers
	for i := 0; i < config.NumWorkers; i++ {
		go func(workerID int) {
			workerResult := runWorkerLoadTest(ctx, embedder, workerID, config)
			results <- workerResult
		}(i)
	}

	// Collect results
	for i := 0; i < config.NumWorkers; i++ {
		workerResult := <-results
		result.TotalOperations += workerResult.Operations
		result.TotalErrors += workerResult.Errors
	}

	result.Duration = time.Since(start)
	result.OpsPerSecond = float64(result.TotalOperations) / result.Duration.Seconds()
	if result.TotalOperations > 0 {
		result.ErrorRate = float64(result.TotalErrors) / float64(result.TotalOperations)
	}

	return result, nil
}

// WorkerResult holds results from a single worker in load testing
type WorkerResult struct {
	WorkerID   int
	Operations int64
	Errors     int64
}

// runWorkerLoadTest executes load testing operations for a single worker
func runWorkerLoadTest(ctx context.Context, embedder iface.Embedder, workerID int, config ConcurrentLoadTestConfig) WorkerResult {
	result := WorkerResult{WorkerID: workerID}
	operationCount := int64(0)

	for operationCount < int64(config.OperationsPerWorker) {
		select {
		case <-ctx.Done():
			return result
		default:
			// Select document based on operation count for deterministic cycling
			docIndex := int(operationCount) % len(config.TestDocuments)
			document := config.TestDocuments[docIndex]

			// Alternate between single query and batch operations
			if operationCount%2 == 0 {
				// Single query operation
				_, err := embedder.EmbedQuery(ctx, document)
				operationCount++
				if err != nil {
					result.Errors++
				}
			} else {
				// Batch operation with 2-3 documents
				batchSize := 2 + int(operationCount%2) // Alternate between 2 and 3
				if docIndex+batchSize > len(config.TestDocuments) {
					batchSize = len(config.TestDocuments) - docIndex
				}
				if batchSize < 1 {
					batchSize = 1
				}

				documents := config.TestDocuments[docIndex : docIndex+batchSize]
				_, err := embedder.EmbedDocuments(ctx, documents)
				operationCount++
				if err != nil {
					result.Errors++
				}
			}

			result.Operations = operationCount
		}
	}

	return result
}

// LoadPattern represents different load testing patterns
type LoadPattern int

const (
	ConstantLoad LoadPattern = iota
	BurstLoad
	RampUpLoad
	RandomLoad
)

// LoadTestScenario defines a complete load testing scenario
type LoadTestScenario struct {
	Name         string
	Pattern      LoadPattern
	Duration     time.Duration
	Concurrency  int
	TargetOpsSec int
	WarmupPeriod time.Duration
}

// RunLoadTestScenario executes a comprehensive load testing scenario
func RunLoadTestScenario(ctx context.Context, embedder iface.Embedder, scenario LoadTestScenario, progressCallback func(*ConcurrentLoadTestResult)) error {
	ctx, cancel := context.WithTimeout(ctx, scenario.Duration)
	defer cancel()

	start := time.Now()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	totalResult := &ConcurrentLoadTestResult{}

	for {
		select {
		case <-ctx.Done():
			if progressCallback != nil {
				progressCallback(totalResult)
			}
			return nil
		case <-ticker.C:
			// Calculate current concurrency based on pattern
			elapsed := time.Since(start)
			currentConcurrency := calculateConcurrencyForPattern(scenario, elapsed)

			// Run a short burst of operations
			config := ConcurrentLoadTestConfig{
				NumWorkers:          currentConcurrency,
				OperationsPerWorker: 10, // Short burst
				Timeout:             100 * time.Millisecond,
			}

			result, err := RunConcurrentLoadTest(ctx, embedder, config)
			if err != nil {
				continue // Skip this measurement if it fails
			}

			// Accumulate results
			totalResult.TotalOperations += result.TotalOperations
			totalResult.TotalErrors += result.TotalErrors
			totalResult.Duration = time.Since(start)
			totalResult.OpsPerSecond = float64(totalResult.TotalOperations) / totalResult.Duration.Seconds()
			if totalResult.TotalOperations > 0 {
				totalResult.ErrorRate = float64(totalResult.TotalErrors) / float64(totalResult.TotalOperations)
			}

			if progressCallback != nil {
				progressCallback(totalResult)
			}
		}
	}
}

// calculateConcurrencyForPattern determines concurrency level based on load pattern
func calculateConcurrencyForPattern(scenario LoadTestScenario, elapsed time.Duration) int {
	progress := float64(elapsed) / float64(scenario.Duration)
	if progress > 1.0 {
		progress = 1.0
	}

	switch scenario.Pattern {
	case ConstantLoad:
		return scenario.Concurrency
	case BurstLoad:
		// Alternate between high and low concurrency
		if int(elapsed.Seconds())%10 < 5 {
			return scenario.Concurrency * 2
		}
		return scenario.Concurrency / 2
	case RampUpLoad:
		// Linear ramp up from 1 to target concurrency
		return 1 + int(float64(scenario.Concurrency-1)*progress)
	case RandomLoad:
		// Random concurrency between 1 and target
		return 1 + rand.Intn(scenario.Concurrency)
	default:
		return scenario.Concurrency
	}
}
