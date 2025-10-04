// Package retrievers provides comprehensive tests for retriever implementations.
// This file contains advanced testing scenarios including table-driven tests,
// concurrency testing, and performance benchmarks.
package retrievers

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdvancedMockRetriever tests the advanced mock retriever functionality
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
				WithMockError(true, fmt.Errorf("retrieval service unavailable"))),
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
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
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

// TestRetrieverScenarios tests real-world retrieval scenarios
func TestRetrieverScenarios(t *testing.T) {
	tests := []struct {
		name     string
		scenario func(t *testing.T, retriever core.Retriever)
	}{
		{
			name: "multi_query_retrieval",
			scenario: func(t *testing.T, retriever core.Retriever) {
				runner := NewRetrieverScenarioRunner(retriever)
				queries := CreateTestRetrievalQueries(5)

				results, err := runner.RunMultiQueryScenario(context.Background(), queries)
				assert.NoError(t, err)
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
				assert.NoError(t, err)
			},
		},
		{
			name: "ranking_consistency",
			scenario: func(t *testing.T, retriever core.Retriever) {
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
					assert.Equal(t, len(results[0]), len(results[1]),
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

// TestRetrieverConfiguration tests different retriever configurations
func TestRetrieverConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *AdvancedMockRetriever
		validate func(t *testing.T, retriever *AdvancedMockRetriever, results []schema.Document)
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
			assert.NoError(t, err)

			tt.validate(t, retriever, results)
		})
	}
}

// TestRetrieverPerformance tests retriever performance characteristics
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

// TestConcurrencyAdvanced tests concurrent retriever operations
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

// TestLoadTesting performs load testing on retriever components
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

// TestRetrieverIntegrationTestHelper tests the integration test helper
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

	// Test operations
	ctx := context.Background()
	_, err := vectorRetriever.GetRelevantDocuments(ctx, "test query")
	assert.NoError(t, err)

	// Test reset
	helper.Reset()

	// Verify reset worked
	assert.Equal(t, 0, vectorRetriever.GetCallCount())
	assert.Equal(t, 0, keywordRetriever.GetCallCount())
}

// TestRetrieverHelperFunctions tests utility functions
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

// TestErrorHandling tests comprehensive error handling scenarios
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
					WithMockError(true, fmt.Errorf("vector store connection failed")))
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

			assert.Error(t, err)
		})
	}
}

// BenchmarkRetrieverOperations benchmarks retriever operation performance
func BenchmarkRetrieverOperations(b *testing.B) {
	retriever := NewAdvancedMockRetriever("benchmark", "vector_store",
		WithMockDocuments(CreateTestRetrievalDocuments(500)))

	ctx := context.Background()
	queries := CreateTestRetrievalQueries(50)

	b.Run("GetRelevantDocuments", func(b *testing.B) {
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

// BenchmarkRetrieverBenchmark tests the benchmark helper utility
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

// BenchmarkHelperFunctions benchmarks helper function performance
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
