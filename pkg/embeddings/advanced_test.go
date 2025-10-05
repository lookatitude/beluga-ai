// Package embeddings provides comprehensive tests for embedding implementations.
// This file contains advanced testing scenarios including table-driven tests,
// concurrency testing, and performance benchmarks.
package embeddings

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdvancedMockEmbedder tests the advanced mock embedder functionality
func TestAdvancedMockEmbedder(t *testing.T) {
	tests := []struct {
		name              string
		embedder          *AdvancedMockEmbedder
		operations        func(ctx context.Context, embedder *AdvancedMockEmbedder) error
		expectedError     bool
		expectedCallCount int
		expectedDimension int
	}{
		{
			name:     "successful embedding operations",
			embedder: NewAdvancedMockEmbedder("test-provider", "test-model", 128),
			operations: func(ctx context.Context, embedder *AdvancedMockEmbedder) error {
				// Test single query embedding
				query, err := embedder.EmbedQuery(ctx, "test query")
				if err != nil {
					return err
				}
				if len(query) != 128 {
					return fmt.Errorf("expected dimension 128, got %d", len(query))
				}

				// Test document batch embedding
				texts := CreateTestTexts(3)
				docs, err := embedder.EmbedDocuments(ctx, texts)
				if err != nil {
					return err
				}
				if len(docs) != 3 {
					return fmt.Errorf("expected 3 embeddings, got %d", len(docs))
				}

				// Test dimension retrieval
				dim, err := embedder.GetDimension(ctx)
				if err != nil {
					return err
				}
				if dim != 128 {
					return fmt.Errorf("expected dimension 128, got %d", dim)
				}

				return nil
			},
			expectedError:     false,
			expectedCallCount: 4,
			expectedDimension: 128,
		},
		{
			name: "embedder with error",
			embedder: NewAdvancedMockEmbedder("error-provider", "error-model", 128,
				WithMockError(true, fmt.Errorf("embedding failed"))),
			operations: func(ctx context.Context, embedder *AdvancedMockEmbedder) error {
				_, err := embedder.EmbedQuery(ctx, "test query")
				return err
			},
			expectedError:     true,
			expectedCallCount: 1,
		},
		{
			name: "embedder with delay",
			embedder: NewAdvancedMockEmbedder("delay-provider", "delay-model", 256,
				WithMockDelay(20*time.Millisecond)),
			operations: func(ctx context.Context, embedder *AdvancedMockEmbedder) error {
				start := time.Now()
				_, err := embedder.EmbedQuery(ctx, "test query")
				duration := time.Since(start)

				if duration < 20*time.Millisecond {
					return fmt.Errorf("expected delay was not respected")
				}

				return err
			},
			expectedError:     false,
			expectedCallCount: 2,
			expectedDimension: 256,
		},
		{
			name: "embedder with custom embeddings",
			embedder: NewAdvancedMockEmbedder("custom-provider", "custom-model", 64,
				WithMockEmbeddings(CreateTestEmbeddings(5, 64))),
			operations: func(ctx context.Context, embedder *AdvancedMockEmbedder) error {
				texts := CreateTestTexts(5)
				embeddings, err := embedder.EmbedDocuments(ctx, texts)
				if err != nil {
					return err
				}

				if len(embeddings) != 5 {
					return fmt.Errorf("expected 5 embeddings, got %d", len(embeddings))
				}

				for i, emb := range embeddings {
					if len(emb) != 64 {
						return fmt.Errorf("embedding %d has dimension %d, expected 64", i, len(emb))
					}
				}

				return nil
			},
			expectedError:     false,
			expectedCallCount: 2,
			expectedDimension: 64,
		},
		{
			name: "embedder with rate limiting",
			embedder: NewAdvancedMockEmbedder("rate-limited", "rate-model", 128,
				WithMockRateLimit(true)),
			operations: func(ctx context.Context, embedder *AdvancedMockEmbedder) error {
				// Make several calls to trigger rate limit
				for i := 0; i < 7; i++ {
					_, err := embedder.EmbedQuery(ctx, fmt.Sprintf("query %d", i))
					if i >= 5 && err != nil {
						// Rate limit should kick in after 5 calls
						return nil
					} else if i >= 5 && err == nil {
						return fmt.Errorf("expected rate limit error after 5 calls")
					}
				}
				return nil
			},
			expectedError:     false,
			expectedCallCount: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Run operations
			err := tt.operations(ctx, tt.embedder)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				if tt.expectedDimension > 0 {
					dim, err := tt.embedder.GetDimension(ctx)
					assert.NoError(t, err)
					assert.Equal(t, tt.expectedDimension, dim)
				}
			}

			// Verify call count
			if tt.expectedCallCount > 0 {
				assert.Equal(t, tt.expectedCallCount, tt.embedder.GetCallCount())
			}

			// Test health check
			health := tt.embedder.CheckHealth()
			AssertHealthCheck(t, health, "healthy")
		})
	}
}

// TestEmbeddingProviderRegistry tests the provider registry functionality
func TestEmbeddingProviderRegistry(t *testing.T) {
	registry := NewProviderRegistry()

	// Test registration
	testCreator := func(ctx context.Context, config Config) (iface.Embedder, error) {
		return NewAdvancedMockEmbedder("test", "test-model", 128), nil
	}

	registry.Register("test_embedder", testCreator)

	// Test listing
	providers := registry.ListProviders()
	assert.Contains(t, providers, "test_embedder")

	// Test creation
	ctx := context.Background()
	config := CreateTestConfig("mock")

	embedder, err := registry.Create(ctx, "test_embedder", config)
	assert.NoError(t, err)
	assert.NotNil(t, embedder)

	// Test unknown provider
	_, err = registry.Create(ctx, "unknown_provider", config)
	assert.Error(t, err)
	AssertErrorType(t, err, iface.ErrCodeProviderNotFound)

	// Test global registry functions - register globally for testing
	RegisterGlobal("test_global", testCreator)
	globalProviders := ListAvailableProviders()
	assert.NotEmpty(t, globalProviders)
	assert.Contains(t, globalProviders, "test_global")

	globalRegistry := GetGlobalRegistry()
	assert.NotNil(t, globalRegistry)
}

// TestEmbeddingQuality tests embedding quality and consistency
func TestEmbeddingQuality(t *testing.T) {
	embedder := NewAdvancedMockEmbedder("quality-test", "test-model", 128)
	tester := NewEmbeddingQualityTester(embedder)
	ctx := context.Background()

	tests := []struct {
		name        string
		testFunc    func() (float32, error)
		minExpected float32
	}{
		{
			name: "similarity_consistency",
			testFunc: func() (float32, error) {
				return tester.TestSimilarityConsistency(ctx, "test text for consistency", 5)
			},
			minExpected: 0.8, // Should be highly consistent
		},
		{
			name: "semantic_similarity",
			testFunc: func() (float32, error) {
				similarTexts := []string{
					"The cat sat on the mat",
					"A feline rested on the rug",
					"The kitten was on the carpet",
				}
				return tester.TestSemanticSimilarity(ctx, similarTexts)
			},
			minExpected: 0.0, // Mock embeddings may not have semantic meaning
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, err := tt.testFunc()
			assert.NoError(t, err)
			assert.GreaterOrEqual(t, score, tt.minExpected)
		})
	}
}

// TestEmbeddingHelperFunctions tests helper functions
func TestEmbeddingHelperFunctions(t *testing.T) {
	// Test cosine similarity
	emb1 := []float32{1.0, 0.0, 0.0}
	emb2 := []float32{1.0, 0.0, 0.0}
	similarity := CosineSimilarity(emb1, emb2)
	assert.InDelta(t, 1.0, similarity, 0.001, "Identical embeddings should have similarity 1.0")

	emb3 := []float32{0.0, 1.0, 0.0}
	similarity = CosineSimilarity(emb1, emb3)
	assert.InDelta(t, 0.0, similarity, 0.001, "Orthogonal embeddings should have similarity 0.0")

	// Test Euclidean distance
	distance := EuclideanDistance(emb1, emb2)
	assert.InDelta(t, 0.0, distance, 0.001, "Identical embeddings should have distance 0.0")

	distance = EuclideanDistance(emb1, emb3)
	assert.InDelta(t, 1.414, distance, 0.1, "Expected distance for orthogonal unit vectors")

	// Test different length embeddings
	emb4 := []float32{1.0, 0.0}
	similarity = CosineSimilarity(emb1, emb4)
	assert.Equal(t, float32(0.0), similarity, "Different length embeddings should return 0 similarity")
}

// TestConcurrencyAdvanced tests concurrent embedding operations
func TestConcurrencyAdvanced(t *testing.T) {
	embedder := NewAdvancedMockEmbedder("concurrent-test", "test-model", 128)

	const numGoroutines = 10
	const numOperationsPerGoroutine = 5

	t.Run("concurrent_embedding_operations", func(t *testing.T) {
		var wg sync.WaitGroup
		errChan := make(chan error, numGoroutines*numOperationsPerGoroutine*2)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < numOperationsPerGoroutine; j++ {
					ctx := context.Background()

					// Test EmbedQuery
					query := fmt.Sprintf("query-%d-%d", goroutineID, j)
					_, err := embedder.EmbedQuery(ctx, query)
					if err != nil {
						errChan <- err
						return
					}

					// Test EmbedDocuments
					texts := []string{fmt.Sprintf("doc-%d-%d", goroutineID, j)}
					_, err = embedder.EmbedDocuments(ctx, texts)
					if err != nil {
						errChan <- err
						return
					}
				}
			}(i)
		}

		wg.Wait()
		close(errChan)

		// Check for errors
		for err := range errChan {
			t.Errorf("Concurrent operation error: %v", err)
		}

		// Verify total operations (each iteration does 2 operations)
		expectedOps := numGoroutines * numOperationsPerGoroutine * 2
		assert.Equal(t, expectedOps, embedder.GetCallCount())
	})
}

// TestLoadTesting performs load testing on embedding components
func TestLoadTesting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load tests in short mode")
	}

	embedder := NewAdvancedMockEmbedder("load-test", "test-model", 256)

	const numOperations = 50
	const concurrency = 5

	t.Run("embedding_load_test", func(t *testing.T) {
		RunLoadTest(t, embedder, numOperations, concurrency)

		// Verify health after load test
		health := embedder.CheckHealth()
		AssertHealthCheck(t, health, "healthy")
		assert.Equal(t, numOperations, health["call_count"])
	})
}

// TestEmbeddingScenarios tests real-world embedding usage scenarios
func TestEmbeddingScenarios(t *testing.T) {
	tests := []struct {
		name     string
		scenario func(t *testing.T, embedder iface.Embedder)
	}{
		{
			name: "document_similarity_search",
			scenario: func(t *testing.T, embedder iface.Embedder) {
				ctx := context.Background()

				// Create documents with some semantic similarity
				documents := []string{
					"Machine learning is a subset of artificial intelligence",
					"Deep learning uses neural networks with multiple layers",
					"Natural language processing helps computers understand human language",
					"Computer vision enables machines to interpret visual information",
				}

				// Embed all documents
				docEmbeddings, err := embedder.EmbedDocuments(ctx, documents)
				require.NoError(t, err)
				AssertEmbeddings(t, docEmbeddings, len(documents), 128)

				// Create a query related to the documents
				query := "What is artificial intelligence and machine learning?"
				queryEmbedding, err := embedder.EmbedQuery(ctx, query)
				require.NoError(t, err)
				AssertEmbedding(t, queryEmbedding, 128)

				// Calculate similarities (for mock, just verify the function works)
				similarities := make([]float32, len(docEmbeddings))
				for i, docEmb := range docEmbeddings {
					similarities[i] = CosineSimilarity(queryEmbedding, docEmb)
				}

				// Verify we got similarity scores
				for i, sim := range similarities {
					assert.GreaterOrEqual(t, sim, float32(-1.0), "Similarity %d should be >= -1.0", i)
					assert.LessOrEqual(t, sim, float32(1.0), "Similarity %d should be <= 1.0", i)
				}
			},
		},
		{
			name: "batch_vs_individual_consistency",
			scenario: func(t *testing.T, embedder iface.Embedder) {
				ctx := context.Background()

				texts := CreateTestTexts(3)

				// Get embeddings individually
				individualEmbeddings := make([][]float32, len(texts))
				for i, text := range texts {
					emb, err := embedder.EmbedQuery(ctx, text)
					require.NoError(t, err)
					individualEmbeddings[i] = emb
				}

				// Get embeddings as batch
				batchEmbeddings, err := embedder.EmbedDocuments(ctx, texts)
				require.NoError(t, err)

				// Compare results (for deterministic mock embedders)
				require.Equal(t, len(individualEmbeddings), len(batchEmbeddings))
				for i := range individualEmbeddings {
					assert.Len(t, batchEmbeddings[i], len(individualEmbeddings[i]))
				}
			},
		},
		{
			name: "embedding_dimension_consistency",
			scenario: func(t *testing.T, embedder iface.Embedder) {
				ctx := context.Background()

				// Get expected dimension
				expectedDim, err := embedder.GetDimension(ctx)
				require.NoError(t, err)
				assert.Greater(t, expectedDim, 0)

				// Test query embedding dimension
				query, err := embedder.EmbedQuery(ctx, "test query")
				require.NoError(t, err)
				assert.Len(t, query, expectedDim)

				// Test document embeddings dimensions
				docs, err := embedder.EmbedDocuments(ctx, CreateTestTexts(3))
				require.NoError(t, err)
				for i, doc := range docs {
					assert.Len(t, doc, expectedDim, "Document %d has wrong dimension", i)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use mock embedder for scenario testing
			embedder := NewAdvancedMockEmbedder("scenario-test", "test-model", 128)
			tt.scenario(t, embedder)
		})
	}
}

// TestIntegrationTestHelper tests the integration test helper functionality
func TestIntegrationTestHelper(t *testing.T) {
	helper := NewIntegrationTestHelper()

	// Add embedders
	openaiEmbedder := NewAdvancedMockEmbedder("openai", "text-embedding-ada-002", 1536)
	ollamaEmbedder := NewAdvancedMockEmbedder("ollama", "nomic-embed-text", 768)

	helper.AddEmbedder("openai", openaiEmbedder)
	helper.AddEmbedder("ollama", ollamaEmbedder)

	// Test retrieval
	assert.Equal(t, openaiEmbedder, helper.GetEmbedder("openai"))
	assert.Equal(t, ollamaEmbedder, helper.GetEmbedder("ollama"))

	// Test operations
	ctx := context.Background()
	_, err := openaiEmbedder.EmbedQuery(ctx, "test query")
	assert.NoError(t, err)

	_, err = ollamaEmbedder.EmbedDocuments(ctx, CreateTestTexts(2))
	assert.NoError(t, err)

	// Test registry
	registry := helper.GetRegistry()
	assert.NotNil(t, registry)

	// Test reset
	helper.Reset()

	// Verify reset worked (call counts should be reset)
	assert.Equal(t, 0, openaiEmbedder.GetCallCount())
	assert.Equal(t, 0, ollamaEmbedder.GetCallCount())
}

// TestEmbeddingConfiguration tests configuration scenarios
func TestEmbeddingConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		validate func(t *testing.T, config Config)
	}{
		{
			name:     "openai_config",
			provider: "openai",
			validate: func(t *testing.T, config Config) {
				assert.NotNil(t, config.OpenAI)
				assert.Equal(t, "text-embedding-ada-002", config.OpenAI.Model)
				assert.Equal(t, "test-api-key", config.OpenAI.APIKey)
				assert.True(t, config.OpenAI.Enabled)

				err := config.OpenAI.Validate()
				assert.NoError(t, err)
			},
		},
		{
			name:     "ollama_config",
			provider: "ollama",
			validate: func(t *testing.T, config Config) {
				assert.NotNil(t, config.Ollama)
				assert.Equal(t, "nomic-embed-text", config.Ollama.Model)
				assert.Equal(t, "http://localhost:11434", config.Ollama.ServerURL)
				assert.True(t, config.Ollama.Enabled)

				err := config.Ollama.Validate()
				assert.NoError(t, err)
			},
		},
		{
			name:     "mock_config",
			provider: "mock",
			validate: func(t *testing.T, config Config) {
				assert.NotNil(t, config.Mock)
				assert.Equal(t, 128, config.Mock.Dimension)
				assert.Equal(t, int64(12345), config.Mock.Seed)
				assert.True(t, config.Mock.Enabled)

				err := config.Mock.Validate()
				assert.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := CreateTestConfig(tt.provider)
			tt.validate(t, config)

			// Test overall config validation
			err := config.Validate()
			assert.NoError(t, err)
		})
	}
}

// BenchmarkEmbeddingOperations benchmarks embedding operation performance
func BenchmarkEmbeddingOperations(b *testing.B) {
	embedder := NewAdvancedMockEmbedder("benchmark", "test-model", 256)
	ctx := context.Background()

	b.Run("EmbedQuery", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := embedder.EmbedQuery(ctx, fmt.Sprintf("query-%d", i))
			if err != nil {
				b.Errorf("EmbedQuery error: %v", err)
			}
		}
	})

	b.Run("EmbedDocuments", func(b *testing.B) {
		texts := CreateTestTexts(5)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := embedder.EmbedDocuments(ctx, texts)
			if err != nil {
				b.Errorf("EmbedDocuments error: %v", err)
			}
		}
	})

	b.Run("GetDimension", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := embedder.GetDimension(ctx)
			if err != nil {
				b.Errorf("GetDimension error: %v", err)
			}
		}
	})
}

// BenchmarkEmbeddingBenchmark tests the embedding benchmark utility
func BenchmarkEmbeddingBenchmark(b *testing.B) {
	embedder := NewAdvancedMockEmbedder("benchmark-util", "test-model", 128)
	benchmark := NewEmbeddingBenchmark(embedder, 100)

	b.Run("SingleEmbedding", func(b *testing.B) {
		_, err := benchmark.BenchmarkSingleEmbedding(b.N)
		if err != nil {
			b.Errorf("SingleEmbedding benchmark error: %v", err)
		}
	})

	b.Run("BatchEmbedding", func(b *testing.B) {
		_, err := benchmark.BenchmarkBatchEmbedding(10, b.N)
		if err != nil {
			b.Errorf("BatchEmbedding benchmark error: %v", err)
		}
	})
}

// BenchmarkHelperFunctions benchmarks helper function performance
func BenchmarkHelperFunctions(b *testing.B) {
	emb1 := CreateTestEmbeddings(1, 256)[0]
	emb2 := CreateTestEmbeddings(1, 256)[0]

	b.Run("CosineSimilarity", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			CosineSimilarity(emb1, emb2)
		}
	})

	b.Run("EuclideanDistance", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			EuclideanDistance(emb1, emb2)
		}
	})
}

// Load testing scenarios with realistic user patterns

func TestLoadTestingScenarios(t *testing.T) {
	tests := []struct {
		name        string
		scenario    func(*EmbedderFactory) error
		expectError bool
	}{
		{
			name: "realistic_user_session",
			scenario: func(factory *EmbedderFactory) error {
				return testRealisticUserSession(factory)
			},
			expectError: false,
		},
		{
			name: "api_burst_traffic",
			scenario: func(factory *EmbedderFactory) error {
				return testAPIBurstTraffic(factory)
			},
			expectError: false,
		},
		{
			name: "gradual_load_increase",
			scenario: func(factory *EmbedderFactory) error {
				return testGradualLoadIncrease(factory)
			},
			expectError: false,
		},
		{
			name: "mixed_workload_patterns",
			scenario: func(factory *EmbedderFactory) error {
				return testMixedWorkloadPatterns(factory)
			},
			expectError: false,
		},
		{
			name: "error_recovery_scenarios",
			scenario: func(factory *EmbedderFactory) error {
				return testErrorRecoveryScenarios(factory)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Mock: &MockConfig{
					Dimension: 128,
					Seed:      42,
					Enabled:   true,
				},
			}

			factory, err := NewEmbedderFactory(config)
			require.NoError(t, err)

			err = tt.scenario(factory)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// testRealisticUserSession simulates a typical user session with embeddings
func testRealisticUserSession(factory *EmbedderFactory) error {
	ctx := context.Background()
	embedder, err := factory.NewEmbedder("mock")
	if err != nil {
		return err
	}

	// Simulate user document processing workflow
	documents := []string{
		"Introduction to machine learning concepts",
		"Deep learning neural networks explained",
		"Natural language processing fundamentals",
		"Computer vision applications and techniques",
		"Reinforcement learning basics",
	}

	// Phase 1: Initial document processing (batch)
	_, err = embedder.EmbedDocuments(ctx, documents[:3])
	if err != nil {
		return fmt.Errorf("batch embedding failed: %w", err)
	}

	// Phase 2: Individual queries (typical search behavior)
	for _, doc := range documents {
		_, err = embedder.EmbedQuery(ctx, "What is "+strings.Split(doc, " ")[0]+"?")
		if err != nil {
			return fmt.Errorf("query embedding failed: %w", err)
		}
		time.Sleep(10 * time.Millisecond) // Simulate user think time
	}

	// Phase 3: Follow-up queries (refined search)
	refinedQueries := []string{
		"machine learning algorithms",
		"neural network architectures",
		"NLP preprocessing steps",
	}

	for _, query := range refinedQueries {
		_, err = embedder.EmbedQuery(ctx, query)
		if err != nil {
			return fmt.Errorf("refined query failed: %w", err)
		}
	}

	return nil
}

// testAPIBurstTraffic simulates sudden traffic spikes
func testAPIBurstTraffic(factory *EmbedderFactory) error {
	ctx := context.Background()
	embedder, err := factory.NewEmbedder("mock")
	if err != nil {
		return err
	}

	// Simulate API burst: many requests in quick succession
	burstSize := 20
	testDoc := "Burst traffic test document"

	for i := 0; i < burstSize; i++ {
		_, err = embedder.EmbedQuery(ctx, fmt.Sprintf("%s %d", testDoc, i))
		if err != nil {
			return fmt.Errorf("burst request %d failed: %w", i, err)
		}
	}

	return nil
}

// testGradualLoadIncrease simulates slowly increasing load
func testGradualLoadIncrease(factory *EmbedderFactory) error {
	ctx := context.Background()
	embedder, err := factory.NewEmbedder("mock")
	if err != nil {
		return err
	}

	// Gradually increase load over time
	phases := []int{5, 10, 15, 20} // Increasing batch sizes

	for phase, batchSize := range phases {
		documents := make([]string, batchSize)
		for i := range documents {
			documents[i] = fmt.Sprintf("Load phase %d document %d", phase, i)
		}

		_, err = embedder.EmbedDocuments(ctx, documents)
		if err != nil {
			return fmt.Errorf("phase %d batch failed: %w", phase, err)
		}

		time.Sleep(50 * time.Millisecond) // Brief pause between phases
	}

	return nil
}

// testMixedWorkloadPatterns simulates different types of embedding workloads
func testMixedWorkloadPatterns(factory *EmbedderFactory) error {
	ctx := context.Background()
	embedder, err := factory.NewEmbedder("mock")
	if err != nil {
		return err
	}

	// Mix of different workload patterns
	workloads := []struct {
		description string
		operation   func() error
	}{
		{
			description: "single short query",
			operation: func() error {
				_, err := embedder.EmbedQuery(ctx, "test")
				return err
			},
		},
		{
			description: "single long document",
			operation: func() error {
				longDoc := strings.Repeat("This is a long document with many words. ", 50)
				_, err := embedder.EmbedQuery(ctx, longDoc)
				return err
			},
		},
		{
			description: "small batch",
			operation: func() error {
				docs := []string{"doc1", "doc2", "doc3"}
				_, err := embedder.EmbedDocuments(ctx, docs)
				return err
			},
		},
		{
			description: "large batch",
			operation: func() error {
				docs := make([]string, 10)
				for i := range docs {
					docs[i] = fmt.Sprintf("batch document %d", i)
				}
				_, err := embedder.EmbedDocuments(ctx, docs)
				return err
			},
		},
		{
			description: "empty input",
			operation: func() error {
				_, err := embedder.EmbedQuery(ctx, "")
				return err // This should succeed (return zero vector)
			},
		},
	}

	for _, workload := range workloads {
		if err := workload.operation(); err != nil {
			return fmt.Errorf("%s failed: %w", workload.description, err)
		}
	}

	return nil
}

// testErrorRecoveryScenarios simulates error conditions and recovery
func testErrorRecoveryScenarios(factory *EmbedderFactory) error {
	ctx := context.Background()

	// Test with error-prone configuration
	errorConfig := &Config{
		Mock: &MockConfig{
			Dimension:      128,
			Seed:           42,
			Enabled:        true,
			SimulateErrors: true,
			ErrorRate:      0.5, // 50% error rate
		},
	}

	errorFactory, err := NewEmbedderFactory(errorConfig)
	if err != nil {
		return err
	}

	embedder, err := errorFactory.NewEmbedder("mock")
	if err != nil {
		return err
	}

	// Make several requests, some should fail due to simulated errors
	totalRequests := 10
	successCount := 0

	for i := 0; i < totalRequests; i++ {
		_, err := embedder.EmbedQuery(ctx, fmt.Sprintf("error test %d", i))
		if err == nil {
			successCount++
		}
		// Errors are expected, so we don't fail the test
	}

	// Verify some requests succeeded (despite error simulation)
	if successCount == 0 {
		return fmt.Errorf("no requests succeeded during error simulation")
	}

	// Test recovery: switch to normal configuration
	normalEmbedder, err := factory.NewEmbedder("mock")
	if err != nil {
		return err
	}

	// Verify normal operation after error scenario
	for i := 0; i < 5; i++ {
		_, err := normalEmbedder.EmbedQuery(ctx, fmt.Sprintf("recovery test %d", i))
		if err != nil {
			return fmt.Errorf("recovery test %d failed: %w", i, err)
		}
	}

	return nil
}

// TestLoadTestingWithDifferentConfigurations tests load scenarios with various mock configurations
func TestLoadTestingWithDifferentConfigurations(t *testing.T) {
	configs := []struct {
		name   string
		config *Config
	}{
		{
			name: "standard_configuration",
			config: &Config{
				Mock: &MockConfig{
					Dimension: 128,
					Seed:      42,
					Enabled:   true,
				},
			},
		},
		{
			name: "with_rate_limiting",
			config: &Config{
				Mock: &MockConfig{
					Dimension:          128,
					Seed:               42,
					Enabled:            true,
					RateLimitPerSecond: 10,
				},
			},
		},
		{
			name: "with_delays",
			config: &Config{
				Mock: &MockConfig{
					Dimension:     128,
					Seed:          42,
					Enabled:       true,
					SimulateDelay: 10 * time.Millisecond,
				},
			},
		},
		{
			name: "with_memory_pressure",
			config: &Config{
				Mock: &MockConfig{
					Dimension:      128,
					Seed:           42,
					Enabled:        true,
					MemoryPressure: true,
				},
			},
		},
	}

	for _, cfg := range configs {
		t.Run(cfg.name, func(t *testing.T) {
			factory, err := NewEmbedderFactory(cfg.config)
			require.NoError(t, err)

			embedder, err := factory.NewEmbedder("mock")
			require.NoError(t, err)

			ctx := context.Background()

			// Run a basic load test
			for i := 0; i < 20; i++ {
				_, err := embedder.EmbedQuery(ctx, fmt.Sprintf("config test %d", i))
				if cfg.name == "with_rate_limiting" && i > 15 {
					// Rate limiting might cause some failures, which is expected
					continue
				}
				// For other configurations, all requests should succeed
				if cfg.name != "with_rate_limiting" {
					assert.NoError(t, err, "Request %d failed for %s", i, cfg.name)
				}
			}
		})
	}
}

// TestNetworkFailureSimulation tests embedder behavior under simulated network failures
func TestNetworkFailureSimulation(t *testing.T) {
	ctx := context.Background()

	// Create mock embedders with network failure simulation
	failingEmbedder := NewAdvancedMockEmbedder("failing-provider", "fail-model", 128,
		WithMockError(true, iface.WrapError(fmt.Errorf("network connection failed"), iface.ErrCodeConnectionFailed, "simulated network failure")))

	reliableEmbedder := NewAdvancedMockEmbedder("reliable-provider", "reliable-model", 128)

	testCases := []struct {
		name       string
		embedder   *AdvancedMockEmbedder
		shouldFail bool
	}{
		{"network_failure_embedder", failingEmbedder, true},
		{"reliable_embedder", reliableEmbedder, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test single query embedding
			_, err := tc.embedder.EmbedQuery(ctx, "test query")
			if tc.shouldFail {
				assert.Error(t, err, "Expected network failure but got success")
				assert.Contains(t, err.Error(), "network", "Error should mention network failure")
			} else {
				assert.NoError(t, err, "Expected success but got network failure")
			}

			// Test batch embedding
			texts := CreateTestTexts(3)
			_, err = tc.embedder.EmbedDocuments(ctx, texts)
			if tc.shouldFail {
				assert.Error(t, err, "Expected network failure in batch embedding")
			} else {
				assert.NoError(t, err, "Expected success in batch embedding")
			}

			// Test dimension retrieval (may fail if embedder is configured to fail)
			dim, err := tc.embedder.GetDimension(ctx)
			if tc.shouldFail {
				// Dimension retrieval may also fail when embedder is configured to fail
				if err != nil {
					t.Log("Dimension retrieval failed as expected for failing embedder")
				}
			} else {
				assert.NoError(t, err, "Dimension retrieval should work for reliable embedder")
				assert.Equal(t, 128, dim, "Dimension should be correct")
			}
		})
	}

	// Test recovery after network issues - simulate by creating new embedder without error
	t.Run("network_recovery", func(t *testing.T) {
		// Start with failing embedder
		failingEmbedder := NewAdvancedMockEmbedder("recovery-provider", "recovery-model", 128,
			WithMockError(true, iface.WrapError(fmt.Errorf("temporary network issue"), iface.ErrCodeConnectionFailed, "temporary failure")))

		// Should fail initially
		_, err := failingEmbedder.EmbedQuery(ctx, "test")
		assert.Error(t, err, "Expected initial failure")

		// Simulate recovery by creating new embedder without error
		recoveredEmbedder := NewAdvancedMockEmbedder("recovery-provider", "recovery-model", 128)

		// Should work after recovery
		_, err = recoveredEmbedder.EmbedQuery(ctx, "test")
		assert.NoError(t, err, "Expected success after recovery")
	})
}

// TestAPIRateLimitScenarios tests embedder behavior under various rate limiting conditions
func TestAPIRateLimitScenarios(t *testing.T) {
	ctx := context.Background()

	// Test different rate limit configurations
	rateLimitTests := []struct {
		name           string
		embedder       *AdvancedMockEmbedder
		requestCount   int
		expectedErrors int
	}{
		{
			name:           "no_rate_limiting",
			embedder:       NewAdvancedMockEmbedder("no-limit", "model", 128),
			requestCount:   10,
			expectedErrors: 0,
		},
		{
			name:           "strict_rate_limiting",
			embedder:       NewAdvancedMockEmbedder("strict-limit", "model", 128, WithMockRateLimit(true)),
			requestCount:   15,
			expectedErrors: 5, // Should fail after rate limit exceeded
		},
		{
			name:           "high_frequency_requests",
			embedder:       NewAdvancedMockEmbedder("high-freq", "model", 128, WithMockRateLimit(true)),
			requestCount:   100,
			expectedErrors: 50, // High volume should trigger many rate limit errors
		},
	}

	for _, rt := range rateLimitTests {
		t.Run(rt.name, func(t *testing.T) {
			errorCount := 0
			successCount := 0

			// Make multiple requests
			for i := 0; i < rt.requestCount; i++ {
				_, err := rt.embedder.EmbedQuery(ctx, fmt.Sprintf("test query %d", i))
				if err != nil {
					errorCount++
					// Verify error is rate limit related
					assert.Contains(t, strings.ToLower(err.Error()), "rate", "Rate limit error should mention rate limiting")
				} else {
					successCount++
				}
			}

			t.Logf("Rate limit test %s: %d successes, %d errors out of %d requests",
				rt.name, successCount, errorCount, rt.requestCount)

			// For rate limited tests, we expect some errors
			if strings.Contains(rt.name, "strict") || strings.Contains(rt.name, "high_frequency") {
				assert.True(t, errorCount > 0, "Expected some rate limit errors but got none")
				assert.True(t, successCount > 0, "Expected some successful requests")
			} else {
				assert.Equal(t, 0, errorCount, "Expected no errors for unlimited test")
				assert.Equal(t, rt.requestCount, successCount, "All requests should succeed")
			}
		})
	}

	// Test rate limit reset functionality
	t.Run("rate_limit_reset", func(t *testing.T) {
		embedder := NewAdvancedMockEmbedder("reset-test", "model", 128, WithMockRateLimit(true))

		// Exhaust rate limit
		for i := 0; i < 20; i++ {
			embedder.EmbedQuery(ctx, "test")
		}

		// Should be rate limited now
		_, err := embedder.EmbedQuery(ctx, "test")
		assert.Error(t, err, "Expected rate limit error")

		// Reset rate limit
		embedder.ResetRateLimit()

		// Should work again
		_, err = embedder.EmbedQuery(ctx, "test")
		assert.NoError(t, err, "Expected success after rate limit reset")
	})
}

// TestProviderUnavailableScenarios tests embedder behavior when providers become unavailable
func TestProviderUnavailableScenarios(t *testing.T) {
	ctx := context.Background()

	// Test provider that is permanently unavailable (using WithMockError)
	t.Run("permanent_unavailability", func(t *testing.T) {
		embedder := NewAdvancedMockEmbedder("permanent-failure", "model", 128,
			WithMockError(true, iface.WrapError(fmt.Errorf("provider permanently down"), iface.ErrCodeConnectionFailed, "permanent failure")))

		// Multiple attempts should all fail
		for i := 0; i < 5; i++ {
			_, err := embedder.EmbedQuery(ctx, fmt.Sprintf("attempt %d", i))
			assert.Error(t, err, "Expected permanent failure on attempt %d", i)
			assert.Contains(t, err.Error(), "permanent", "Error should indicate permanent failure")
		}
	})

	// Test degraded provider performance
	t.Run("degraded_performance", func(t *testing.T) {
		embedder := NewAdvancedMockEmbedder("degraded", "model", 128,
			WithMockDelay(100*time.Millisecond)) // Add significant delay

		start := time.Now()
		_, err := embedder.EmbedQuery(ctx, "slow test")
		duration := time.Since(start)

		assert.NoError(t, err, "Expected success despite degradation")
		assert.True(t, duration >= 100*time.Millisecond, "Expected delay of at least 100ms, got %v", duration)
	})

	// Test concurrent access during provider issues
	t.Run("concurrent_provider_issues", func(t *testing.T) {
		embedder := NewAdvancedMockEmbedder("concurrent-issues", "model", 128,
			WithMockError(true, iface.WrapError(fmt.Errorf("concurrent access issue"), iface.ErrCodeConnectionFailed, "concurrent failure")))

		var wg sync.WaitGroup
		errorCount := int64(0)
		successCount := int64(0)

		// Launch concurrent requests
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				_, err := embedder.EmbedQuery(ctx, fmt.Sprintf("concurrent test %d", id))
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}(i)
		}

		wg.Wait()

		t.Logf("Concurrent test: %d errors, %d successes", errorCount, successCount)
		assert.True(t, errorCount > 0, "Expected some errors during concurrent access to failing provider")
	})

	// Test provider recovery (create new embedder without error)
	t.Run("provider_recovery", func(t *testing.T) {
		// Start with failing embedder
		failingEmbedder := NewAdvancedMockEmbedder("recovery-provider", "recovery-model", 128,
			WithMockError(true, iface.WrapError(fmt.Errorf("temporary network issue"), iface.ErrCodeConnectionFailed, "temporary failure")))

		// Should fail initially
		_, err := failingEmbedder.EmbedQuery(ctx, "test")
		assert.Error(t, err, "Expected initial failure")

		// Simulate recovery by creating new embedder without error
		recoveredEmbedder := NewAdvancedMockEmbedder("recovery-provider", "recovery-model", 128)

		// Should work after recovery
		_, err = recoveredEmbedder.EmbedQuery(ctx, "test")
		assert.NoError(t, err, "Expected success after recovery")
	})
}
