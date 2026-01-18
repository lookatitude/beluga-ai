// Package retrievers provides comprehensive tests for retriever implementations.
// This file contains advanced testing scenarios including table-driven tests,
// concurrency testing, and performance benchmarks.
package retrievers

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdvancedMockRetriever tests the advanced mock retriever functionality.
func TestAdvancedMockRetriever(t *testing.T) {
	tests := []struct {
		name              string
		retriever         *AdvancedMockRetriever
		query             string
		expectedError     bool
		expectedCallCount int
		expectedMinDocs   int
		expectedMaxDocs   int
	}{
		{
			name:              "successful_retrieval",
			retriever:         NewAdvancedMockRetriever("test-retriever", "vector_store"),
			query:             "What is machine learning?",
			expectedError:     false,
			expectedCallCount: 1,
			expectedMinDocs:   1,
			expectedMaxDocs:   5,
		},
		{
			name: "retriever_with_preloaded_documents",
			retriever: NewAdvancedMockRetriever("preloaded-retriever", "vector_store",
				WithMockDocuments(CreateTestRetrievalDocuments(8)),
				WithMockDefaultK(3)),
			query:             "artificial intelligence concepts",
			expectedError:     false,
			expectedCallCount: 1,
			expectedMinDocs:   1,
			expectedMaxDocs:   3,
		},
		{
			name: "retriever_with_error",
			retriever: NewAdvancedMockRetriever("error-retriever", "vector_store",
				WithMockError(true, errors.New("retrieval service unavailable"))),
			query:             "test query",
			expectedError:     true,
			expectedCallCount: 1,
		},
		{
			name: "retriever_with_delay",
			retriever: NewAdvancedMockRetriever("delay-retriever", "vector_store",
				WithMockDelay(25*time.Millisecond)),
			query:             "delayed query",
			expectedError:     false,
			expectedCallCount: 1,
			expectedMinDocs:   1,
			expectedMaxDocs:   5,
		},
		{
			name: "retriever_with_score_threshold",
			retriever: NewAdvancedMockRetriever("threshold-retriever", "vector_store",
				WithMockDocuments(CreateTestRetrievalDocuments(10)),
				WithMockScores([]float32{0.9, 0.8, 0.6, 0.4, 0.3, 0.2, 0.1, 0.05, 0.02, 0.01}),
				WithScoreThreshold(0.5)),
			query:             "high relevance query",
			expectedError:     false,
			expectedCallCount: 1,
			expectedMinDocs:   2, // Should return docs with scores >= 0.5 (0.9, 0.8, 0.6)
			expectedMaxDocs:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			start := time.Now()
			documents, err := tt.retriever.GetRelevantDocuments(ctx, tt.query)
			duration := time.Since(start)

			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				AssertRetrievalResults(t, documents, tt.expectedMinDocs, tt.expectedMaxDocs)

				// Verify delay was respected if configured
				if tt.retriever.simulateDelay > 0 {
					assert.GreaterOrEqual(t, duration, tt.retriever.simulateDelay)
				}
			}

			// Verify call count
			assert.Equal(t, tt.expectedCallCount, tt.retriever.GetCallCount())

			// Test health check
			health := tt.retriever.CheckHealth()
			AssertRetrieverHealth(t, health, "healthy")
		})
	}
}

// TestRetrieverScenarios tests real-world retrieval scenarios.
func TestRetrieverScenarios(t *testing.T) {
	tests := []struct {
		scenario func(t *testing.T, retriever core.Retriever)
		name     string
	}{
		{
			name: "multi_query_retrieval",
			scenario: func(t *testing.T, retriever core.Retriever) {
				t.Helper()
				runner := NewRetrieverScenarioRunner(retriever)
				queries := CreateTestRetrievalQueries(5)

				results, err := runner.RunMultiQueryScenario(context.Background(), queries)
				require.NoError(t, err)
				assert.Len(t, results, len(queries))

				// Verify each query returned some results
				for i, queryResults := range results {
					assert.LessOrEqual(t, len(queryResults), 10, "Query %d should not return too many results", i+1)

					// Verify document quality
					for j, doc := range queryResults {
						assert.NotEmpty(t, doc.GetContent(), "Query %d, Document %d should have content", i+1, j+1)
					}
				}
			},
		},
		{
			name: "relevance_testing",
			scenario: func(t *testing.T, retriever core.Retriever) {
				t.Helper()
				// Create query-document pairs for relevance testing
				pairs := []QueryDocumentPair{
					{
						Query: "machine learning algorithms",
						ExpectedDoc: schema.NewDocument("Machine learning algorithms are computational methods...",
							map[string]string{"topic": "ML"}),
						ShouldBeRelevant: true,
						MinScore:         0.3,
					},
					{
						Query: "deep learning neural networks",
						ExpectedDoc: schema.NewDocument("Neural networks with multiple layers form the basis of deep learning...",
							map[string]string{"topic": "DL"}),
						ShouldBeRelevant: true,
						MinScore:         0.4,
					},
				}

				runner := NewRetrieverScenarioRunner(retriever)
				err := runner.RunRelevanceTestScenario(context.Background(), pairs)
				// Note: Mock retriever may not implement actual relevance, so we test the interface
				// The mock generates documents based on the query, so it may not match exact expected documents
				// We just verify the interface works without errors
				require.NoError(t, err)
			},
		},
		{
			name: "ranking_consistency",
			scenario: func(t *testing.T, retriever core.Retriever) {
				t.Helper()
				ctx := context.Background()
				query := "artificial intelligence applications"

				// Perform same query multiple times
				results := make([][]schema.Document, 3)
				for i := 0; i < 3; i++ {
					docs, err := retriever.GetRelevantDocuments(ctx, query)
					require.NoError(t, err)
					results[i] = docs
				}

				// Verify consistent results (for deterministic retrievers)
				if len(results) > 1 {
					assert.Len(t, results[1], len(results[0]),
						"Retrieval results should be consistent")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use mock retriever with documents for scenario testing
			retriever := NewAdvancedMockRetriever("scenario-test", "vector_store",
				WithMockDocuments(CreateTestRetrievalDocuments(10)))

			tt.scenario(t, retriever)
		})
	}
}

// TestRetrieverConfiguration tests different retriever configurations.
func TestRetrieverConfiguration(t *testing.T) {
	tests := []struct {
		setup    func() *AdvancedMockRetriever
		validate func(t *testing.T, retriever *AdvancedMockRetriever, results []schema.Document)
		name     string
	}{
		{
			name: "different_k_values",
			setup: func() *AdvancedMockRetriever {
				return NewAdvancedMockRetriever("k-test", "vector_store",
					WithMockDocuments(CreateTestRetrievalDocuments(20)),
					WithMockDefaultK(7))
			},
			validate: func(t *testing.T, retriever *AdvancedMockRetriever, results []schema.Document) {
				assert.LessOrEqual(t, len(results), 7, "Should respect k limit")
				assert.GreaterOrEqual(t, len(results), 1, "Should return some results")
			},
		},
		{
			name: "score_threshold_filtering",
			setup: func() *AdvancedMockRetriever {
				scores := []float32{0.95, 0.87, 0.65, 0.43, 0.21, 0.12}
				return NewAdvancedMockRetriever("threshold-test", "vector_store",
					WithMockDocuments(CreateTestRetrievalDocuments(6)),
					WithMockScores(scores),
					WithScoreThreshold(0.5))
			},
			validate: func(t *testing.T, retriever *AdvancedMockRetriever, results []schema.Document) {
				// Should filter out documents with scores < 0.5 (last 3 documents)
				assert.LessOrEqual(t, len(results), 3, "Should filter by score threshold")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retriever := tt.setup()
			ctx := context.Background()

			results, err := retriever.GetRelevantDocuments(ctx, "test query")
			require.NoError(t, err)

			tt.validate(t, retriever, results)
		})
	}
}

// TestRetrieverPerformance tests retriever performance characteristics.
func TestRetrieverPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	tests := []struct {
		name           string
		documentsCount int
		queries        int
		maxDuration    time.Duration
	}{
		{
			name:           "small_collection",
			documentsCount: 50,
			queries:        10,
			maxDuration:    1 * time.Second,
		},
		{
			name:           "medium_collection",
			documentsCount: 200,
			queries:        20,
			maxDuration:    3 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retriever := NewAdvancedMockRetriever("perf-test", "vector_store",
				WithMockDocuments(CreateTestRetrievalDocuments(tt.documentsCount)))

			queries := CreateTestRetrievalQueries(tt.queries)
			ctx := context.Background()

			start := time.Now()

			for i, query := range queries {
				_, err := retriever.GetRelevantDocuments(ctx, query)
				require.NoError(t, err, "Query %d failed", i+1)
			}

			duration := time.Since(start)
			assert.LessOrEqual(t, duration, tt.maxDuration,
				"Performance test should complete within %v, took %v", tt.maxDuration, duration)

			t.Logf("Retriever performance: %d documents, %d queries in %v (avg: %v per query)",
				tt.documentsCount, tt.queries, duration, duration/time.Duration(tt.queries))
		})
	}
}

// TestConcurrencyAdvanced tests concurrent retriever operations.
func TestConcurrencyAdvanced(t *testing.T) {
	retriever := NewAdvancedMockRetriever("concurrent-test", "vector_store",
		WithMockDocuments(CreateTestRetrievalDocuments(50)))

	const numGoroutines = 8
	const queriesPerGoroutine = 5

	t.Run("concurrent_retrieval_operations", func(t *testing.T) {
		var wg sync.WaitGroup
		errChan := make(chan error, numGoroutines*queriesPerGoroutine)
		queries := CreateTestRetrievalQueries(10)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < queriesPerGoroutine; j++ {
					ctx := context.Background()
					query := queries[(goroutineID*queriesPerGoroutine+j)%len(queries)]

					_, err := retriever.GetRelevantDocuments(ctx, query)
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
			t.Errorf("Concurrent retrieval error: %v", err)
		}

		// Verify total operations
		expectedOps := numGoroutines * queriesPerGoroutine
		assert.Equal(t, expectedOps, retriever.GetCallCount())
	})
}

// TestLoadTesting performs load testing on retriever components.
func TestLoadTesting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load tests in short mode")
	}

	retriever := NewAdvancedMockRetriever("load-test", "vector_store",
		WithMockDocuments(CreateTestRetrievalDocuments(100)))

	const numOperations = 100
	const concurrency = 10

	t.Run("retriever_load_test", func(t *testing.T) {
		RunLoadTest(t, retriever, numOperations, concurrency)

		// Verify health after load test
		health := retriever.CheckHealth()
		AssertRetrieverHealth(t, health, "healthy")
		assert.Equal(t, numOperations, health["call_count"])
	})
}

// TestRetrieverIntegrationTestHelper tests the integration test helper.
func TestRetrieverIntegrationTestHelper(t *testing.T) {
	helper := NewIntegrationTestHelper()

	// Add retrievers and dependencies
	vectorRetriever := NewAdvancedMockRetriever("vector-retriever", "vector_store")
	keywordRetriever := NewAdvancedMockRetriever("keyword-retriever", "keyword")

	helper.AddRetriever("vector", vectorRetriever)
	helper.AddRetriever("keyword", keywordRetriever)

	// Test retrieval
	assert.Equal(t, vectorRetriever, helper.GetRetriever("vector"))
	assert.Equal(t, keywordRetriever, helper.GetRetriever("keyword"))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Test operations
	_, err := vectorRetriever.GetRelevantDocuments(ctx, "test query")
	require.NoError(t, err)

	// Test reset
	helper.Reset()

	// Verify reset worked
	assert.Equal(t, 0, vectorRetriever.GetCallCount())
	assert.Equal(t, 0, keywordRetriever.GetCallCount())
}

// TestRetrieverHelperFunctions tests utility functions.
func TestRetrieverHelperFunctions(t *testing.T) {
	t.Run("relevance_scoring", func(t *testing.T) {
		query := "machine learning algorithms"
		docContent := "Machine learning algorithms are powerful computational methods for data analysis"

		score := CalculateRelevanceScore(query, docContent)
		assert.GreaterOrEqual(t, score, float32(0.0), "Relevance score should be non-negative")
		assert.LessOrEqual(t, score, float32(1.0), "Relevance score should be <= 1.0")

		// Test with completely unrelated content
		unrelatedContent := "The weather is sunny today and birds are singing"
		unrelatedScore := CalculateRelevanceScore(query, unrelatedContent)
		assert.LessOrEqual(t, unrelatedScore, score, "Unrelated content should have lower score")
	})

	t.Run("document_ranking", func(t *testing.T) {
		query := "artificial intelligence"
		documents := []schema.Document{
			schema.NewDocument("AI is fascinating technology", map[string]string{"topic": "AI"}),
			schema.NewDocument("Unrelated content about cooking", map[string]string{"topic": "cooking"}),
			schema.NewDocument("Artificial intelligence revolutionizes industries", map[string]string{"topic": "AI"}),
		}

		rankedDocs := RankDocuments(query, documents)
		assert.Len(t, rankedDocs, len(documents))

		// Verify ranking order (scores should be descending)
		for i := 1; i < len(rankedDocs); i++ {
			assert.GreaterOrEqual(t, rankedDocs[i-1].Score, rankedDocs[i].Score,
				"Documents should be ranked in descending order of relevance")
		}

		// Verify ranks are assigned correctly
		for i, docScore := range rankedDocs {
			assert.Equal(t, i+1, docScore.Rank, "Rank should match position")
		}
	})

	t.Run("query_tokenization", func(t *testing.T) {
		query := "machine learning, deep learning!"
		tokens := tokenizeSimple(query)

		expected := []string{"machine", "learning", "deep", "learning"}
		assert.Equal(t, expected, tokens, "Tokenization should split on delimiters")

		// Test edge cases
		emptyQuery := ""
		emptyTokens := tokenizeSimple(emptyQuery)
		assert.Empty(t, emptyTokens, "Empty query should produce no tokens")

		singleWord := "AI"
		singleTokens := tokenizeSimple(singleWord)
		assert.Equal(t, []string{"AI"}, singleTokens, "Single word should produce one token")
	})
}

// TestErrorHandling tests comprehensive error handling scenarios.
func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() core.Retriever
		operation func(retriever core.Retriever) error
		errorCode string
	}{
		{
			name: "retrieval_service_error",
			setup: func() core.Retriever {
				return NewAdvancedMockRetriever("error-retriever", "vector_store",
					WithMockError(true, errors.New("vector store connection failed")))
			},
			operation: func(retriever core.Retriever) error {
				ctx := context.Background()
				_, err := retriever.GetRelevantDocuments(ctx, "test query")
				return err
			},
		},
		{
			name: "timeout_error",
			setup: func() core.Retriever {
				return NewAdvancedMockRetriever("timeout-retriever", "vector_store",
					WithMockDelay(5*time.Second))
			},
			operation: func(retriever core.Retriever) error {
				// Use a short timeout to trigger timeout error
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()
				_, err := retriever.GetRelevantDocuments(ctx, "timeout test")
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retriever := tt.setup()
			err := tt.operation(retriever)

			require.Error(t, err)
			// For timeout_error, verify it's a context deadline exceeded error
			if tt.name == "timeout_error" {
				assert.ErrorContains(t, err, "context deadline exceeded")
			}
		})
	}
}

// BenchmarkRetrieverOperations benchmarks retriever operation performance.
func BenchmarkRetrieverOperations(b *testing.B) {
	retriever := NewAdvancedMockRetriever("benchmark", "vector_store",
		WithMockDocuments(CreateTestRetrievalDocuments(500)))

	queries := CreateTestRetrievalQueries(50)

	b.Run("GetRelevantDocuments", func(b *testing.B) {
		ctx := context.Background()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			query := queries[i%len(queries)]
			_, err := retriever.GetRelevantDocuments(ctx, query)
			if err != nil {
				b.Errorf("GetRelevantDocuments error: %v", err)
			}
		}
	})

	b.Run("HealthCheck", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			health := retriever.CheckHealth()
			if health == nil {
				b.Error("Health check should not return nil")
			}
		}
	})
}

// BenchmarkRetrieverBenchmark tests the benchmark helper utility.
func BenchmarkRetrieverBenchmark(b *testing.B) {
	retriever := NewAdvancedMockRetriever("benchmark-helper", "vector_store",
		WithMockDocuments(CreateTestRetrievalDocuments(200)))

	helper := NewBenchmarkHelper(retriever, 20)

	b.Run("BenchmarkRetrieval", func(b *testing.B) {
		_, err := helper.BenchmarkRetrieval(b.N)
		if err != nil {
			b.Errorf("BenchmarkRetrieval error: %v", err)
		}
	})

	b.Run("BenchmarkBatchRetrieval", func(b *testing.B) {
		_, err := helper.BenchmarkBatchRetrieval(5, b.N)
		if err != nil {
			b.Errorf("BenchmarkBatchRetrieval error: %v", err)
		}
	})
}

// BenchmarkHelperFunctions benchmarks helper function performance.
func BenchmarkHelperFunctions(b *testing.B) {
	query := "machine learning artificial intelligence"
	document := "Machine learning is a subset of artificial intelligence that enables computers to learn and improve from experience"

	b.Run("CalculateRelevanceScore", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			score := CalculateRelevanceScore(query, document)
			if score < 0 {
				b.Error("Relevance score should be non-negative")
			}
		}
	})

	b.Run("RankDocuments", func(b *testing.B) {
		documents := CreateTestRetrievalDocuments(10)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ranked := RankDocuments(query, documents)
			if len(ranked) != len(documents) {
				b.Error("Ranking should preserve document count")
			}
		}
	})
}

// TestRetrieversFactoryFunctions tests factory functions in retrievers.go.
func TestRetrieversFactoryFunctions(t *testing.T) {
	ctx := context.Background()
	mockVectorStore := NewMockVectorStore()

	tests := []struct {
		name        string
		setupFn     func() (*VectorStoreRetriever, error)
		validateFn  func(t *testing.T, retriever *VectorStoreRetriever, err error)
		description string
		wantErr     bool
	}{
		{
			name:        "new_vector_store_retriever_with_defaults",
			description: "Test creating retriever with default options",
			setupFn: func() (*VectorStoreRetriever, error) {
				return NewVectorStoreRetriever(mockVectorStore)
			},
			validateFn: func(t *testing.T, retriever *VectorStoreRetriever, err error) {
				require.NoError(t, err)
				assert.NotNil(t, retriever)
				assert.Equal(t, 4, retriever.defaultK)
				assert.Equal(t, float32(0.0), retriever.scoreThreshold)
				assert.Equal(t, 3, retriever.maxRetries)
				assert.Equal(t, 30*time.Second, retriever.timeout)
			},
			wantErr: false,
		},
		{
			name:        "new_vector_store_retriever_with_custom_options",
			description: "Test creating retriever with custom options",
			setupFn: func() (*VectorStoreRetriever, error) {
				return NewVectorStoreRetriever(mockVectorStore,
					WithDefaultK(10),
					WithMaxRetries(5),
					WithTimeout(60*time.Second),
				)
			},
			validateFn: func(t *testing.T, retriever *VectorStoreRetriever, err error) {
				require.NoError(t, err)
				assert.NotNil(t, retriever)
				assert.Equal(t, 10, retriever.defaultK)
				assert.Equal(t, float32(0.0), retriever.scoreThreshold) // Default value
				assert.Equal(t, 5, retriever.maxRetries)
				assert.Equal(t, 60*time.Second, retriever.timeout)
			},
			wantErr: false,
		},
		{
			name:        "new_vector_store_retriever_invalid_k_too_low",
			description: "Test creating retriever with invalid k (too low)",
			setupFn: func() (*VectorStoreRetriever, error) {
				return NewVectorStoreRetriever(mockVectorStore, WithDefaultK(0))
			},
			validateFn: func(t *testing.T, retriever *VectorStoreRetriever, err error) {
				require.Error(t, err)
				assert.Nil(t, retriever)
				AssertErrorType(t, err, ErrCodeInvalidConfig)
			},
			wantErr: true,
		},
		{
			name:        "new_vector_store_retriever_invalid_k_too_high",
			description: "Test creating retriever with invalid k (too high)",
			setupFn: func() (*VectorStoreRetriever, error) {
				return NewVectorStoreRetriever(mockVectorStore, WithDefaultK(101))
			},
			validateFn: func(t *testing.T, retriever *VectorStoreRetriever, err error) {
				require.Error(t, err)
				assert.Nil(t, retriever)
				AssertErrorType(t, err, ErrCodeInvalidConfig)
			},
			wantErr: true,
		},
		{
			name:        "new_vector_store_retriever_from_config",
			description: "Test creating retriever from config struct",
			setupFn: func() (*VectorStoreRetriever, error) {
				config := VectorStoreRetrieverConfig{
					K:              5,
					ScoreThreshold: 0.6,
					Timeout:        45 * time.Second,
				}
				config.ApplyDefaults()
				return NewVectorStoreRetrieverFromConfig(mockVectorStore, config)
			},
			validateFn: func(t *testing.T, retriever *VectorStoreRetriever, err error) {
				require.NoError(t, err)
				assert.NotNil(t, retriever)
				assert.Equal(t, 5, retriever.defaultK)
				assert.Equal(t, float32(0.6), retriever.scoreThreshold)
				assert.Equal(t, 45*time.Second, retriever.timeout)
			},
			wantErr: false,
		},
		{
			name:        "new_vector_store_retriever_from_config_invalid",
			description: "Test creating retriever from invalid config",
			setupFn: func() (*VectorStoreRetriever, error) {
				config := VectorStoreRetrieverConfig{
					K:              101, // Invalid - exceeds max
					ScoreThreshold: 0.6,
				}
				// Apply defaults first (which won't fix K=101)
				config.ApplyDefaults()
				// Validate should catch the error
				if err := config.Validate(); err != nil {
					return nil, err
				}
				return NewVectorStoreRetrieverFromConfig(mockVectorStore, config)
			},
			validateFn: func(t *testing.T, retriever *VectorStoreRetriever, err error) {
				require.Error(t, err)
				assert.Nil(t, retriever)
			},
			wantErr: true,
		},
		{
			name:        "get_retriever_types",
			description: "Test getting available retriever types",
			setupFn: func() (*VectorStoreRetriever, error) {
				types := GetRetrieverTypes()
				assert.Contains(t, types, "vector_store")
				return nil, nil
			},
			validateFn: func(t *testing.T, retriever *VectorStoreRetriever, err error) {
				// Function returns void, validation done in setupFn
			},
			wantErr: false,
		},
		{
			name:        "validate_retriever_config",
			description: "Test validating retriever configuration",
			setupFn: func() (*VectorStoreRetriever, error) {
				config := DefaultConfig()
				err := ValidateRetrieverConfig(config)
				if err != nil {
					return nil, err
				}
				return nil, nil
			},
			validateFn: func(t *testing.T, retriever *VectorStoreRetriever, err error) {
				require.NoError(t, err)
			},
			wantErr: false,
		},
		{
			name:        "validate_retriever_config_invalid",
			description: "Test validating invalid retriever configuration",
			setupFn: func() (*VectorStoreRetriever, error) {
				config := Config{
					DefaultK: 0, // Invalid
				}
				err := ValidateRetrieverConfig(config)
				return nil, err
			},
			validateFn: func(t *testing.T, retriever *VectorStoreRetriever, err error) {
				require.Error(t, err)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			retriever, err := tt.setupFn()
			tt.validateFn(t, retriever, err)

			// Test retriever functionality if created successfully
			if retriever != nil && !tt.wantErr {
				// Test health check
				healthErr := retriever.CheckHealth(ctx)
				assert.NoError(t, healthErr)

				// Test retrieval with mock vector store
				docs, err := retriever.GetRelevantDocuments(ctx, "test query")
				// May succeed or fail depending on mock implementation
				t.Logf("Retrieval test: docs=%d, err=%v", len(docs), err)
			}
		})
	}
}

// TestErrorHandlingAdvanced provides comprehensive error handling tests.
func TestErrorHandlingAdvanced(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		checkFn     func(t *testing.T, err error)
		description string
	}{
		{
			name:        "retriever_error_with_code",
			description: "Test RetrieverError with specific code",
			err:         NewRetrieverError("test_op", errors.New("underlying error"), ErrCodeRetrievalFailed),
			checkFn: func(t *testing.T, err error) {
				require.Error(t, err)
				var retErr *RetrieverError
				require.ErrorAs(t, err, &retErr)
				assert.Equal(t, "test_op", retErr.Op)
				assert.Equal(t, ErrCodeRetrievalFailed, retErr.Code)
				assert.NotNil(t, retErr.Err)
				assert.Contains(t, err.Error(), "retriever test_op")
			},
		},
		{
			name:        "retriever_error_with_message",
			description: "Test RetrieverError with custom message",
			err:         NewRetrieverErrorWithMessage("test_op", errors.New("underlying"), ErrCodeInvalidConfig, "custom message"),
			checkFn: func(t *testing.T, err error) {
				require.Error(t, err)
				var retErr *RetrieverError
				require.ErrorAs(t, err, &retErr)
				assert.Equal(t, "test_op", retErr.Op)
				assert.Equal(t, ErrCodeInvalidConfig, retErr.Code)
				assert.Equal(t, "custom message", retErr.Message)
				assert.Contains(t, err.Error(), "custom message")
			},
		},
		{
			name:        "retriever_error_unwrap",
			description: "Test RetrieverError unwrapping",
			err:         NewRetrieverError("test_op", errors.New("wrapped error"), ErrCodeNetworkError),
			checkFn: func(t *testing.T, err error) {
				require.Error(t, err)
				var retErr *RetrieverError
				require.ErrorAs(t, err, &retErr)
				unwrapped := retErr.Unwrap()
				assert.NotNil(t, unwrapped)
				assert.Contains(t, unwrapped.Error(), "wrapped error")
			},
		},
		{
			name:        "validation_error",
			description: "Test ValidationError",
			err: &ValidationError{
				Field: "DefaultK",
				Value: 0,
				Msg:   "must be between 1 and 100",
			},
			checkFn: func(t *testing.T, err error) {
				require.Error(t, err)
				var valErr *ValidationError
				require.ErrorAs(t, err, &valErr)
				assert.Equal(t, "DefaultK", valErr.Field)
				assert.Equal(t, 0, valErr.Value)
				assert.Contains(t, err.Error(), "validation failed")
				assert.Contains(t, err.Error(), "DefaultK")
			},
		},
		{
			name:        "timeout_error",
			description: "Test TimeoutError",
			err:         NewTimeoutError("test_op", 30*time.Second, errors.New("operation timed out")),
			checkFn: func(t *testing.T, err error) {
				require.Error(t, err)
				var timeoutErr *TimeoutError
				require.ErrorAs(t, err, &timeoutErr)
				assert.Equal(t, "test_op", timeoutErr.Op)
				assert.Equal(t, 30*time.Second, timeoutErr.Timeout)
				assert.Contains(t, err.Error(), "timed out")
				assert.Contains(t, err.Error(), "30s")
				unwrapped := timeoutErr.Unwrap()
				assert.NotNil(t, unwrapped)
			},
		},
		{
			name:        "all_error_codes",
			description: "Test all error code constants",
			err:         nil,
			checkFn: func(t *testing.T, err error) {
				// Verify all error codes are defined
				errorCodes := []string{
					ErrCodeInvalidConfig,
					ErrCodeInvalidInput,
					ErrCodeRetrievalFailed,
					ErrCodeEmbeddingFailed,
					ErrCodeVectorStoreError,
					ErrCodeTimeout,
					ErrCodeRateLimit,
					ErrCodeNetworkError,
					ErrCodeQueryGenerationFailed,
				}
				for _, code := range errorCodes {
					assert.NotEmpty(t, code, "Error code should not be empty")
					err := NewRetrieverError("test", nil, code)
					assert.Equal(t, code, err.Code)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			tt.checkFn(t, tt.err)
		})
	}
}
