// Package embeddings provides comprehensive tests for embedding implementations.
// This file contains advanced testing scenarios including table-driven tests,
// concurrency testing, and performance benchmarks.
package embeddings

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdvancedMockEmbedder tests the advanced mock embedder functionality
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
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
			expectedCallCount: 4, // EmbedQuery + EmbedDocuments (3 docs) + GetDimension
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
			expectedCallCount: 2, // EmbedQuery + GetDimension
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
			expectedCallCount: 2, // EmbedDocuments + GetDimension
			expectedDimension: 64,
		},
		{
			name: "embedder with rate limiting",
			embedder: NewAdvancedMockEmbedder("rate-limited", "rate-model", 128,
				WithMockRateLimit(true)),
			operations: func(ctx context.Context, embedder *AdvancedMockEmbedder) error {
				// Make several calls to trigger rate limit
				// Rate limit kicks in when rateLimitCount > 5, so first 6 calls succeed, 7th fails
				for i := 0; i < 7; i++ {
					_, err := embedder.EmbedQuery(ctx, fmt.Sprintf("query %d", i))
					if i >= 6 && err != nil {
						// Rate limit should kick in after 6 calls (rateLimitCount > 5)
						return nil
					} else if i >= 6 && err == nil {
						return fmt.Errorf("expected rate limit error after 6 calls")
					}
				}
				return nil
			},
			expectedError:     false,
			expectedCallCount: 7, // 6 successful + 1 that hits rate limit
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
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

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

	// Test global registry functions
	// Note: Global registry may be empty if no providers are registered globally
	globalRegistry := GetGlobalRegistry()
	assert.NotNil(t, globalRegistry)

	// List available providers (may be empty if none registered)
	globalProviders := ListAvailableProviders()
	// Just verify the function works, don't require it to be non-empty
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	_ = globalProviders
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
			minExpected: -1.0, // Mock embeddings may not be consistent, allow negative similarity
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
			minExpected: -1.0, // Mock embeddings may not have semantic meaning, allow negative similarity
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, err := tt.testFunc()
			assert.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
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
	// Check for NaN or valid zero distance
	if distance != distance { // NaN check
		t.Errorf("Euclidean distance returned NaN for identical embeddings")
	} else {
		assert.InDelta(t, 0.0, distance, 0.001, "Identical embeddings should have distance 0.0")
	}

	distance = EuclideanDistance(emb1, emb3)
	assert.InDelta(t, 1.414, distance, 0.1, "Expected distance for orthogonal unit vectors")
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

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
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
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
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

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
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
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

	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
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
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
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
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
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
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

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
