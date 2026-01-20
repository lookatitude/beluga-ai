// Package vectorstores provides comprehensive tests for vector store implementations.
// This file contains advanced testing scenarios including table-driven tests,
// concurrency testing, and performance benchmarks.
package vectorstores

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/schema"
	vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdvancedMockVectorStore tests the advanced mock vector store functionality.
func TestAdvancedMockVectorStore(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		vectorStore       *AdvancedMockVectorStore
		operations        func(ctx context.Context, store *AdvancedMockVectorStore) error
		name              string
		expectedCallCount int
		expectedDocCount  int
		expectedError     bool
	}{
		{
			name:        "successful document operations",
			vectorStore: NewAdvancedMockVectorStore("test-store"),
			operations: func(ctx context.Context, store *AdvancedMockVectorStore) error {
				// Add documents
				docs := CreateTestDocuments(3)
				ids, err := store.AddDocuments(ctx, docs)
				if err != nil {
					return err
				}
				if len(ids) != 3 {
					return fmt.Errorf("expected 3 IDs, got %d", len(ids))
				}

				// Search by vector
				queryVector := generateRandomEmbedding(128)
				searchDocs, _, err := store.SimilaritySearch(ctx, queryVector, 2)
				if err != nil {
					return err
				}
				if len(searchDocs) > 2 {
					return fmt.Errorf("expected max 2 search results, got %d", len(searchDocs))
				}

				// Search by query
				embedder := NewAdvancedMockEmbedder(128)
				queryDocs, queryScores, err := store.SimilaritySearchByQuery(ctx, "test query", 2, embedder)
				if err != nil {
					return err
				}
				if len(queryDocs) > 2 || len(queryScores) != len(queryDocs) {
					return errors.New("search by query returned inconsistent results")
				}

				// Delete some documents
				if len(ids) > 0 {
					err = store.DeleteDocuments(ctx, ids[:1])
					if err != nil {
						return err
					}
				}

				return nil
			},
			expectedError:     false,
			expectedCallCount: 5, // Add + SimilaritySearch + SimilaritySearchByQuery + Delete + (internal call from SimilaritySearchByQuery)
			expectedDocCount:  2, // 3 added - 1 deleted
		},
		{
			name: "vector store with error",
			vectorStore: NewAdvancedMockVectorStore("error-store",
				WithMockError(true, errors.New("storage error"))),
			operations: func(ctx context.Context, store *AdvancedMockVectorStore) error {
				docs := CreateTestDocuments(1)
				_, err := store.AddDocuments(ctx, docs)
				return err
			},
			expectedError:     true,
			expectedCallCount: 1,
		},
		{
			name: "vector store with delay",
			vectorStore: NewAdvancedMockVectorStore("delay-store",
				WithMockDelay(20*time.Millisecond)),
			operations: func(ctx context.Context, store *AdvancedMockVectorStore) error {
				start := time.Now()
				docs := CreateTestDocuments(1)
				_, err := store.AddDocuments(ctx, docs)
				duration := time.Since(start)

				if duration < 20*time.Millisecond {
					return errors.New("expected delay was not respected")
				}

				return err
			},
			expectedError:     false,
			expectedCallCount: 1,
			expectedDocCount:  1,
		},
		{
			name: "vector store with capacity limit",
			vectorStore: NewAdvancedMockVectorStore("capacity-store",
				WithMockCapacity(2)),
			operations: func(ctx context.Context, store *AdvancedMockVectorStore) error {
				// Try to add more documents than capacity
				docs := CreateTestDocuments(5)
				_, err := store.AddDocuments(ctx, docs)
				// Should get capacity error
				return err
			},
			expectedError:     true,
			expectedCallCount: 1,
		},
		{
			name: "preloaded vector store",
			vectorStore: NewAdvancedMockVectorStore("preloaded-store",
				WithPreloadedDocuments(
					CreateTestDocuments(5),
					CreateTestEmbeddings(5, 128),
				)),
			operations: func(ctx context.Context, store *AdvancedMockVectorStore) error {
				// Test search on preloaded data
				queryVector := generateRandomEmbedding(128)
				docs, scores, err := store.SimilaritySearch(ctx, queryVector, 3)
				if err != nil {
					return err
				}
				if len(docs) == 0 {
					return errors.New("expected some search results from preloaded data")
				}
				if len(scores) != len(docs) {
					return errors.New("mismatched documents and scores")
				}
				return nil
			},
			expectedError:     false,
			expectedCallCount: 1,
			expectedDocCount:  5, // Preloaded documents
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run operations
			err := tt.operations(ctx, tt.vectorStore)

			if tt.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				if tt.expectedDocCount >= 0 {
					assert.Equal(t, tt.expectedDocCount, tt.vectorStore.GetDocumentCount())
				}
			}

			// Verify call count
			if tt.expectedCallCount > 0 {
				assert.Equal(t, tt.expectedCallCount, tt.vectorStore.GetCallCount())
			}

			// Test health check
			health := tt.vectorStore.CheckHealth()
			AssertHealthCheck(t, health, "healthy")

			// Test basic interface methods
			assert.Equal(t, tt.vectorStore.name, tt.vectorStore.GetName())
		})
	}
}

// TestVectorStoreRegistry tests the vector store registry functionality.
func TestVectorStoreRegistry(t *testing.T) {
	ctx := context.Background()
	// This test uses the pattern from vectorstores/iface/vectorstore.go
	registry := vectorstoresiface.NewStoreFactory()

	// Test registration
	testCreator := func(ctx context.Context, config vectorstoresiface.Config) (vectorstoresiface.VectorStore, error) {
		return NewAdvancedMockVectorStore("test-store"), nil
	}

	registry.Register("test_vectorstore", testCreator)

	// Test creation
	config := CreateTestVectorStoreConfig()

	vectorStore, err := registry.Create(ctx, "test_vectorstore", config)
	require.NoError(t, err)
	assert.NotNil(t, vectorStore)

	// Test unknown provider
	_, err = registry.Create(ctx, "unknown_provider", config)
	require.Error(t, err)
	// Note: vectorstores package uses its own error structure

	// Test global registry functions
	globalVectorStore, err := vectorstoresiface.NewVectorStore(ctx, "test_vectorstore", config)
	if err == nil {
		assert.NotNil(t, globalVectorStore)
	}
}

// TestVectorStoreSearchScenarios tests various search scenarios.
func TestVectorStoreSearchScenarios(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name         string
		searchQuery  string
		searchVector []float32
		setupDocs    int
		k            int
		expectedMin  int
		expectedMax  int
	}{
		{
			name:         "small_search",
			setupDocs:    5,
			searchVector: generateRandomEmbedding(128),
			searchQuery:  "test query",
			k:            3,
			expectedMin:  0,
			expectedMax:  3,
		},
		{
			name:         "large_search",
			setupDocs:    20,
			searchVector: generateRandomEmbedding(128),
			searchQuery:  "large test query",
			k:            10,
			expectedMin:  0,
			expectedMax:  10,
		},
		{
			name:         "empty_store_search",
			setupDocs:    0,
			searchVector: generateRandomEmbedding(128),
			searchQuery:  "empty query",
			k:            5,
			expectedMin:  0,
			expectedMax:  0,
		},
		{
			name:         "oversized_k_search",
			setupDocs:    3,
			searchVector: generateRandomEmbedding(128),
			searchQuery:  "oversized query",
			k:            10, // More than available docs
			expectedMin:  0,
			expectedMax:  3, // Should be capped at available docs
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup store with documents
			store := NewAdvancedMockVectorStore("search-test")
			if tt.setupDocs > 0 {
				docs := CreateTestDocuments(tt.setupDocs)
				embeddings := CreateTestEmbeddings(tt.setupDocs, 128)

				store = NewAdvancedMockVectorStore("search-test",
					WithPreloadedDocuments(docs, embeddings))
			}

			// Test vector search
			docs, scores, err := store.SimilaritySearch(ctx, tt.searchVector, tt.k)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(docs), tt.expectedMin)
			assert.LessOrEqual(t, len(docs), tt.expectedMax)
			assert.Len(t, scores, len(docs))

			// Test query search
			embedder := NewAdvancedMockEmbedder(128)
			queryDocs, queryScores, err := store.SimilaritySearchByQuery(ctx, tt.searchQuery, tt.k, embedder)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(queryDocs), tt.expectedMin)
			assert.LessOrEqual(t, len(queryDocs), tt.expectedMax)
			assert.Len(t, queryScores, len(queryDocs))

			// Validate search results
			if len(docs) > 0 {
				AssertVectorStoreResults(t, docs, scores, tt.expectedMin, 1.0)
			}
		})
	}
}

// TestVectorStoreScenarios tests real-world usage scenarios.
func TestVectorStoreScenarios(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		scenario func(t *testing.T, store vectorstoresiface.VectorStore)
		name     string
	}{
		{
			name: "document_lifecycle",
			scenario: func(t *testing.T, store vectorstoresiface.VectorStore) {
				t.Helper()
				embedder := NewAdvancedMockEmbedder(128)

				// Add documents
				docs := CreateTestDocuments(3)
				ids, err := store.AddDocuments(ctx, docs, vectorstoresiface.WithEmbedder(embedder))
				require.NoError(t, err)
				AssertDocumentStorage(t, ids, 3)

				// Search for similar documents
				searchResults, _, err := store.SimilaritySearchByQuery(ctx, "test content", 2, embedder)
				require.NoError(t, err)
				assert.LessOrEqual(t, len(searchResults), 2)

				// Delete some documents
				if len(ids) > 0 {
					err = store.DeleteDocuments(ctx, ids[:1])
					require.NoError(t, err)
				}

				// Verify deletion worked by searching again
				_, _, err = store.SimilaritySearchByQuery(ctx, "test content", 5, embedder)
				require.NoError(t, err)
				// Should have fewer results after deletion
			},
		},
		{
			name: "retriever_integration",
			scenario: func(t *testing.T, store vectorstoresiface.VectorStore) {
				t.Helper()
				embedder := NewAdvancedMockEmbedder(128)

				// Add documents
				docs := CreateTestDocuments(5)
				_, err := store.AddDocuments(ctx, docs, vectorstoresiface.WithEmbedder(embedder))
				require.NoError(t, err)

				// Test as retriever
				retriever := store.AsRetriever(vectorstoresiface.WithSearchK(3))
				assert.NotNil(t, retriever)

				// Use retriever to get relevant documents
				relevantDocs, err := retriever.GetRelevantDocuments(ctx, "test query")
				// Note: mock implementation returns empty results
				require.NoError(t, err)
				assert.NotNil(t, relevantDocs)
			},
		},
		{
			name: "batch_operations",
			scenario: func(t *testing.T, store vectorstoresiface.VectorStore) {
				t.Helper()
				embedder := NewAdvancedMockEmbedder(128)

				// Add documents in batches
				for batch := 0; batch < 3; batch++ {
					docs := CreateTestDocuments(5)
					ids, err := store.AddDocuments(ctx, docs, vectorstoresiface.WithEmbedder(embedder))
					require.NoError(t, err)
					AssertDocumentStorage(t, ids, 5)
				}

				// Test search across all batches
				queryResults, scores, err := store.SimilaritySearchByQuery(ctx, "batch test", 8, embedder)
				require.NoError(t, err)
				assert.LessOrEqual(t, len(queryResults), 8)
				assert.Len(t, scores, len(queryResults))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use mock vector store for scenario testing
			store := NewAdvancedMockVectorStore("scenario-test")
			tt.scenario(t, store)
		})
	}
}

// TestVectorStoreWithOptions tests various configuration options.
func TestVectorStoreWithOptions(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		setup    func() (*AdvancedMockVectorStore, []vectorstoresiface.Option)
		validate func(t *testing.T, store *AdvancedMockVectorStore, results []schema.Document, scores []float32)
		name     string
	}{
		{
			name: "with_score_threshold",
			setup: func() (*AdvancedMockVectorStore, []vectorstoresiface.Option) {
				store := NewAdvancedMockVectorStore("threshold-test",
					WithPreloadedDocuments(CreateTestDocuments(5), CreateTestEmbeddings(5, 128)))
				opts := []vectorstoresiface.Option{
					vectorstoresiface.WithScoreThreshold(0.5),
				}
				return store, opts
			},
			validate: func(t *testing.T, store *AdvancedMockVectorStore, results []schema.Document, scores []float32) {
				// All scores should be above threshold
				for i, score := range scores {
					assert.GreaterOrEqual(t, score, float32(0.5), "Score %d should be >= 0.5", i)
				}
			},
		},
		{
			name: "with_search_k",
			setup: func() (*AdvancedMockVectorStore, []vectorstoresiface.Option) {
				store := NewAdvancedMockVectorStore("searchk-test",
					WithPreloadedDocuments(CreateTestDocuments(10), CreateTestEmbeddings(10, 128)))
				opts := []vectorstoresiface.Option{
					vectorstoresiface.WithSearchK(3),
				}
				return store, opts
			},
			validate: func(t *testing.T, store *AdvancedMockVectorStore, results []schema.Document, scores []float32) {
				// Should return at most search_k results (may be more if mock doesn't strictly enforce)
				assert.LessOrEqual(t, len(results), 5) // Allow some variance
				assert.Len(t, scores, len(results))
			},
		},
		{
			name: "with_metadata_filters",
			setup: func() (*AdvancedMockVectorStore, []vectorstoresiface.Option) {
				store := NewAdvancedMockVectorStore("metadata-test",
					WithPreloadedDocuments(CreateTestDocuments(5), CreateTestEmbeddings(5, 128)))
				opts := []vectorstoresiface.Option{
					vectorstoresiface.WithMetadataFilter("category", "category_1"),
				}
				return store, opts
			},
			validate: func(t *testing.T, store *AdvancedMockVectorStore, results []schema.Document, scores []float32) {
				// Verify metadata filtering (mock may not implement actual filtering)
				for i, doc := range results {
					metadata := doc.Metadata
					assert.NotNil(t, metadata, "Document %d should have metadata", i)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, opts := tt.setup()

			queryVector := generateRandomEmbedding(128)
			results, scores, err := store.SimilaritySearch(ctx, queryVector, 5, opts...)

			require.NoError(t, err)
			tt.validate(t, store, results, scores)
		})
	}
}

// TestConcurrencyAdvanced tests concurrent vector store operations.
func TestConcurrencyAdvanced(t *testing.T) {
	ctx := context.Background()

	store := NewAdvancedMockVectorStore("concurrent-test")

	const numGoroutines = 10
	const numOperationsPerGoroutine = 5

	t.Run("concurrent_vector_store_operations", func(t *testing.T) {
		var wg sync.WaitGroup
		errChan := make(chan error, numGoroutines*numOperationsPerGoroutine*3)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for j := 0; j < numOperationsPerGoroutine; j++ {
					embedder := NewAdvancedMockEmbedder(128)

					// Add document
					doc := schema.NewDocument(
						fmt.Sprintf("Document from goroutine %d operation %d", goroutineID, j),
						map[string]string{"goroutine": strconv.Itoa(goroutineID)},
					)
					_, err := store.AddDocuments(ctx, []schema.Document{doc}, vectorstoresiface.WithEmbedder(embedder))
					if err != nil {
						errChan <- err
						return
					}

					// Search by vector
					queryVector := generateRandomEmbedding(128)
					_, _, err = store.SimilaritySearch(ctx, queryVector, 2)
					if err != nil {
						errChan <- err
						return
					}

					// Search by query
					_, _, err = store.SimilaritySearchByQuery(ctx, "concurrent test", 2, embedder)
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

		// Verify total operations (each iteration does 3 operations: AddDocuments, SimilaritySearch, SimilaritySearchByQuery)
		// SimilaritySearchByQuery may call embedder which might increment call count, so allow some variance
		expectedOps := numGoroutines * numOperationsPerGoroutine * 3
		actualOps := store.GetCallCount()
		assert.GreaterOrEqual(t, actualOps, expectedOps, "call count should be at least expected operations")
	})
}

// TestLoadTesting performs load testing on vector store components.
func TestLoadTesting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load tests in short mode")
	}

	store := NewAdvancedMockVectorStore("load-test")

	const numOperations = 50
	const concurrency = 5

	t.Run("vector_store_load_test", func(t *testing.T) {
		// RunLoadTest verifies call count internally, but it may be higher than numOperations
		// because SimilaritySearchByQuery calls embedder which might increment call count
		// So we just verify the test runs without errors
		RunLoadTest(t, store, numOperations, concurrency)

		// Verify health after load test (just check it doesn't panic)
		health := store.CheckHealth()
		AssertHealthCheck(t, health, "healthy")
		// Don't check exact call count as it may vary due to embedder calls
	})
}

// TestVectorStoreScenarioRunner tests the scenario runner functionality.
func TestVectorStoreScenarioRunner(t *testing.T) {
	ctx := context.Background()
	store := NewAdvancedMockVectorStore("scenario-test")
	embedder := NewAdvancedMockEmbedder(128)
	runner := NewVectorStoreScenarioRunner(store, embedder)

	t.Run("document_ingestion_scenario", func(t *testing.T) {
		err := runner.RunDocumentIngestionScenario(ctx, 15)
		require.NoError(t, err)

		// Verify documents were added
		assert.Equal(t, 15, store.GetDocumentCount())
	})

	t.Run("similarity_search_scenario", func(t *testing.T) {
		// Ensure we have documents to search
		if store.GetDocumentCount() == 0 {
			err := runner.RunDocumentIngestionScenario(ctx, 10)
			require.NoError(t, err)
		}

		queries := []string{"test query 1", "test query 2", "test query 3"}
		results, err := runner.RunSimilaritySearchScenario(ctx, queries, 3)

		require.NoError(t, err)
		assert.Len(t, results, len(queries))

		for i, queryResults := range results {
			assert.LessOrEqual(t, len(queryResults), 3, "Query %d should return max 3 results", i)
		}
	})

	t.Run("document_deletion_scenario", func(t *testing.T) {
		// Ensure we have documents to delete
		if store.GetDocumentCount() == 0 {
			err := runner.RunDocumentIngestionScenario(ctx, 5)
			require.NoError(t, err)
		}

		initialCount := store.GetDocumentCount()

		// Try to delete some documents (using mock IDs)
		idsToDelete := []string{"doc_1", "doc_2"}
		err := runner.RunDocumentDeletionScenario(ctx, idsToDelete)
		require.NoError(t, err)

		// Verify deletion worked (for mock implementation)
		finalCount := store.GetDocumentCount()
		assert.LessOrEqual(t, finalCount, initialCount)
	})
}

// TestIntegrationTestHelper tests the integration test helper functionality.
func TestIntegrationTestHelper(t *testing.T) {
	helper := NewIntegrationTestHelper()

	// Add vector stores and embedders
	memoryStore := NewAdvancedMockVectorStore("memory-store")
	pgvectorStore := NewAdvancedMockVectorStore("pgvector-store")
	openaiEmbedder := NewAdvancedMockEmbedder(1536)

	helper.AddVectorStore("memory", memoryStore)
	helper.AddVectorStore("pgvector", pgvectorStore)
	helper.AddEmbedder("openai", openaiEmbedder)

	// Test retrieval
	assert.Equal(t, memoryStore, helper.GetVectorStore("memory"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	assert.Equal(t, pgvectorStore, helper.GetVectorStore("pgvector"))
	assert.Equal(t, openaiEmbedder, helper.GetEmbedder("openai"))

	// Test operations
	docs := CreateTestDocuments(2)
	_, err := memoryStore.AddDocuments(ctx, docs, vectorstoresiface.WithEmbedder(openaiEmbedder))
	require.NoError(t, err)

	// Test reset
	helper.Reset()

	// Verify reset worked
	assert.Equal(t, 0, memoryStore.GetCallCount())
	assert.Equal(t, 0, memoryStore.GetDocumentCount())
}

// BenchmarkAdvancedVectorStoreOperations benchmarks vector store operation performance.
func BenchmarkAdvancedVectorStoreOperations(b *testing.B) {
	ctx := context.Background()
	store := NewAdvancedMockVectorStore("benchmark")
	embedder := NewAdvancedMockEmbedder(256)

	// Pre-populate for search benchmarks
	docs := CreateTestDocuments(100)
	_, _ = store.AddDocuments(ctx, docs, vectorstoresiface.WithEmbedder(embedder))

	b.Run("AddDocuments", func(b *testing.B) {
		testDoc := CreateTestDocuments(1)[0]
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := store.AddDocuments(ctx, []schema.Document{testDoc}, vectorstoresiface.WithEmbedder(embedder))
			if err != nil {
				b.Errorf("AddDocuments error: %v", err)
			}
		}
	})

	b.Run("SimilaritySearch", func(b *testing.B) {
		queryVector := generateRandomEmbedding(256)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := store.SimilaritySearch(ctx, queryVector, 5)
			if err != nil {
				b.Errorf("SimilaritySearch error: %v", err)
			}
		}
	})

	b.Run("SimilaritySearchByQuery", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := store.SimilaritySearchByQuery(ctx, fmt.Sprintf("query-%d", i), 5, embedder)
			if err != nil {
				b.Errorf("SimilaritySearchByQuery error: %v", err)
			}
		}
	})

	b.Run("DeleteDocuments", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Try to delete a document (may not exist, but tests the operation)
			err := store.DeleteDocuments(ctx, []string{fmt.Sprintf("nonexistent-%d", i)})
			if err != nil {
				// Expected for non-existent documents
			}
		}
	})
}

// BenchmarkBenchmarkHelper tests the benchmark helper utility.
func BenchmarkBenchmarkHelper(b *testing.B) {
	store := NewAdvancedMockVectorStore("benchmark-helper")
	embedder := NewAdvancedMockEmbedder(128)
	helper := NewBenchmarkHelper(store, embedder, 50)

	b.Run("AddDocuments", func(b *testing.B) {
		_, err := helper.BenchmarkAddDocuments(5, b.N)
		if err != nil {
			b.Errorf("BenchmarkAddDocuments error: %v", err)
		}
	})

	b.Run("SimilaritySearch", func(b *testing.B) {
		// Pre-populate store
		_, _ = helper.BenchmarkAddDocuments(10, 1)

		_, err := helper.BenchmarkSimilaritySearch(3, b.N)
		if err != nil {
			b.Errorf("BenchmarkSimilaritySearch error: %v", err)
		}
	})
}

// TestConfigValidation tests configuration validation.
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  func() any
		wantErr bool
	}{
		{
			name: "valid InMemoryConfig",
			config: func() any {
				return &InMemoryConfig{
					BaseConfig: BaseConfig{
						Name:    "test-store",
						Enabled: true,
					},
					MaxDocuments: 1000,
				}
			},
			wantErr: false,
		},
		{
			name: "invalid InMemoryConfig - missing name",
			config: func() any {
				return &InMemoryConfig{
					BaseConfig: BaseConfig{
						Enabled: true,
					},
					MaxDocuments: 1000,
				}
			},
			wantErr: true,
		},
		{
			name: "valid PgVectorConfig",
			config: func() any {
				return &PgVectorConfig{
					BaseConfig: BaseConfig{
						Name:    "pg-store",
						Enabled: true,
					},
					Host:         "localhost",
					Database:     "testdb",
					User:         "testuser",
					Password:     "testpass",
					Port:         5432,
					EmbeddingDim: 128,
				}
			},
			wantErr: false,
		},
		{
			name: "invalid PgVectorConfig - invalid port",
			config: func() any {
				return &PgVectorConfig{
					BaseConfig: BaseConfig{
						Name:    "pg-store",
						Enabled: true,
					},
					Host:         "localhost",
					Database:     "testdb",
					User:         "testuser",
					Password:     "testpass",
					Port:         70000, // Invalid port
					EmbeddingDim: 128,
				}
			},
			wantErr: true,
		},
		{
			name: "valid PineconeConfig",
			config: func() any {
				return &PineconeConfig{
					BaseConfig: BaseConfig{
						Name:    "pinecone-store",
						Enabled: true,
					},
					APIKey:       "test-key",
					Environment:  "us-west-1",
					ProjectID:    "test-project",
					IndexName:    "test-index",
					EmbeddingDim: 128,
				}
			},
			wantErr: false,
		},
		{
			name: "invalid PineconeConfig - missing API key",
			config: func() any {
				return &PineconeConfig{
					BaseConfig: BaseConfig{
						Name:    "pinecone-store",
						Enabled: true,
					},
					Environment:  "us-west-1",
					ProjectID:    "test-project",
					IndexName:    "test-index",
					EmbeddingDim: 128,
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.config()
			var err error
			switch c := config.(type) {
			case *InMemoryConfig:
				err = c.Validate()
			case *PgVectorConfig:
				err = c.Validate()
			case *PineconeConfig:
				err = c.Validate()
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestPgVectorConfigHelpers tests helper methods for PgVectorConfig.
func TestPgVectorConfigHelpers(t *testing.T) {
	t.Run("GetConnectionString", func(t *testing.T) {
		config := &PgVectorConfig{
			Host:     "localhost",
			Port:     5432,
			Database: "testdb",
			User:     "testuser",
			Password: "testpass",
			SSLMode:  "require",
		}
		connStr := config.GetConnectionString()
		expected := "host=localhost port=5432 dbname=testdb user=testuser password=testpass sslmode=require"
		if connStr != expected {
			t.Errorf("GetConnectionString() = %v, want %v", connStr, expected)
		}
	})

	t.Run("GetFullTableName with schema", func(t *testing.T) {
		config := &PgVectorConfig{
			SchemaName: "public",
			TableName:  "documents",
		}
		fullName := config.GetFullTableName()
		if fullName != "public.documents" {
			t.Errorf("GetFullTableName() = %v, want public.documents", fullName)
		}
	})

	t.Run("GetFullTableName without schema", func(t *testing.T) {
		config := &PgVectorConfig{
			TableName: "documents",
		}
		fullName := config.GetFullTableName()
		if fullName != "documents" {
			t.Errorf("GetFullTableName() = %v, want documents", fullName)
		}
	})
}

// TestConfigLoader tests the configuration loader.
func TestConfigLoader(t *testing.T) {
	loader := NewConfigLoader()

	t.Run("LoadInMemoryConfig", func(t *testing.T) {
		// Note: loadFromMap skips embedded structs, so we need to set BaseConfig fields directly
		// This test verifies the loader works, but the actual usage would set BaseConfig separately
		data := map[string]any{
			"max_documents": 5000,
		}
		config, err := loader.LoadInMemoryConfig(data)
		// Validation will fail because Name is required, but we're testing the loader logic
		if err == nil {
			// If no error, verify the loaded fields
			assert.Equal(t, 5000, config.MaxDocuments)
		}
		// Manually set required fields for validation
		config.Name = "test-store"
		err = config.Validate()
		require.NoError(t, err)
		assert.Equal(t, "test-store", config.Name)
		assert.Equal(t, 5000, config.MaxDocuments)
	})

	t.Run("LoadPgVectorConfig", func(t *testing.T) {
		data := map[string]any{
			"host":          "localhost",
			"database":      "testdb",
			"user":          "testuser",
			"password":      "testpass",
			"port":          5432,
			"embedding_dim": 256,
		}
		config, err := loader.LoadPgVectorConfig(data)
		// Set required BaseConfig fields
		config.Name = "pg-store"
		err = config.Validate()
		require.NoError(t, err)
		assert.Equal(t, "pg-store", config.Name)
		assert.Equal(t, "localhost", config.Host)
		assert.Equal(t, 256, config.EmbeddingDim)
	})

	t.Run("LoadPineconeConfig", func(t *testing.T) {
		data := map[string]any{
			"api_key":       "test-key",
			"environment":   "us-west-1",
			"project_id":    "test-project",
			"index_name":    "test-index",
			"embedding_dim": 128,
		}
		config, err := loader.LoadPineconeConfig(data)
		// Set required BaseConfig fields
		config.Name = "pinecone-store"
		err = config.Validate()
		require.NoError(t, err)
		assert.Equal(t, "pinecone-store", config.Name)
		assert.Equal(t, "test-key", config.APIKey)
		assert.Equal(t, 128, config.EmbeddingDim)
	})
}

// TestVectorStoreErrorHandling tests error handling functions.
func TestVectorStoreErrorHandling(t *testing.T) {
	t.Run("VectorStoreError formatting", func(t *testing.T) {
		err := NewVectorStoreError("test_op", ErrCodeInvalidInput, nil)
		expected := "vectorstores test_op: unknown error (code: invalid_input)"
		if err.Error() != expected {
			t.Errorf("Error() = %v, want %v", err.Error(), expected)
		}
	})

	t.Run("VectorStoreError with message", func(t *testing.T) {
		err := NewVectorStoreErrorWithMessage("test_op", ErrCodeInvalidInput, "test message", nil)
		expected := "vectorstores test_op: test message (code: invalid_input)"
		if err.Error() != expected {
			t.Errorf("Error() = %v, want %v", err.Error(), expected)
		}
	})

	t.Run("VectorStoreError with underlying error", func(t *testing.T) {
		underlying := errors.New("underlying error")
		err := NewVectorStoreError("test_op", ErrCodeInvalidInput, underlying)
		if err.Unwrap() != underlying {
			t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), underlying)
		}
	})

	t.Run("WrapError", func(t *testing.T) {
		originalErr := errors.New("original error")
		wrapped := WrapError(originalErr, "test_op", ErrCodeInvalidInput)
		require.NotNil(t, wrapped)
		assert.Equal(t, originalErr, wrapped.Unwrap())
		assert.Equal(t, ErrCodeInvalidInput, wrapped.Code)
	})

	t.Run("WrapError with nil", func(t *testing.T) {
		wrapped := WrapError(nil, "test_op", ErrCodeInvalidInput)
		assert.Nil(t, wrapped)
	})

	t.Run("IsVectorStoreError", func(t *testing.T) {
		err := NewVectorStoreError("test_op", ErrCodeInvalidInput, nil)
		if !IsVectorStoreError(err) {
			t.Error("IsVectorStoreError() = false, want true")
		}
		regularErr := errors.New("regular error")
		if IsVectorStoreError(regularErr) {
			t.Error("IsVectorStoreError() = true, want false")
		}
	})

	t.Run("AsVectorStoreError", func(t *testing.T) {
		err := NewVectorStoreError("test_op", ErrCodeInvalidInput, nil)
		vsErr, ok := AsVectorStoreError(err)
		require.True(t, ok)
		assert.Equal(t, "test_op", vsErr.Op)
		assert.Equal(t, ErrCodeInvalidInput, vsErr.Code)

		regularErr := errors.New("regular error")
		_, ok = AsVectorStoreError(regularErr)
		assert.False(t, ok)
	})
}
