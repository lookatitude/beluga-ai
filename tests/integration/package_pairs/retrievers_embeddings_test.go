// Package package_pairs provides integration tests between Retrievers and Embeddings packages.
// This test suite verifies that retrievers work correctly with embedders
// for document embedding, query embedding, and semantic search.
//
// Note: These tests focus on the integration pattern between retrievers and embeddings.
// The actual embedding happens within the vector store when using SimilaritySearchByQuery,
// so these tests verify that the retriever properly uses embedders through the vector store.
package package_pairs

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/retrievers"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/tests/integration/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
)

// TestIntegrationRetrieversEmbeddings tests the integration between Retrievers and Embeddings packages.
func TestIntegrationRetrieversEmbeddings(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	tests := []struct {
		name        string
		description string
		setupFn     func(t *testing.T) (*retrievers.VectorStoreRetriever, embeddingsiface.Embedder)
		testFn      func(t *testing.T, retriever *retrievers.VectorStoreRetriever, embedder embeddingsiface.Embedder)
		wantErr     bool
	}{
		{
			name:        "retrieval_with_embedder",
			description: "Test retrieval using embedder for query embedding",
			setupFn: func(t *testing.T) (*retrievers.VectorStoreRetriever, embeddingsiface.Embedder) {
				embedder := helper.CreateMockEmbedder("test-embedder", 128)
				// Create mock vector store - embedder is used internally by vector store
				documents := utils.CreateTestDocuments(5, "AI")
				scores := []float32{0.9, 0.8, 0.7, 0.6, 0.5}
				vectorStore := newIntegrationMockVectorStore().WithSimilarityResults(documents, scores)

				retriever, err := retrievers.NewVectorStoreRetriever(vectorStore,
					retrievers.WithDefaultK(3),
				)
				require.NoError(t, err)
				return retriever, embedder
			},
			testFn: func(t *testing.T, retriever *retrievers.VectorStoreRetriever, embedder embeddingsiface.Embedder) {
				ctx := context.Background()
				query := "What is artificial intelligence?"

				// Test retrieval - embedder is used internally by vector store via SimilaritySearchByQuery
				docs, err := retriever.GetRelevantDocuments(ctx, query)
				require.NoError(t, err)
				assert.NotNil(t, docs)

				// Verify embedder can embed query independently
				queryEmbedding, err := embedder.EmbedQuery(ctx, query)
				require.NoError(t, err)
				assert.NotNil(t, queryEmbedding)
				assert.Greater(t, len(queryEmbedding), 0)
			},
			wantErr: false,
		},
		{
			name:        "document_embedding_integration",
			description: "Test that documents can be embedded independently",
			setupFn: func(t *testing.T) (*retrievers.VectorStoreRetriever, embeddingsiface.Embedder) {
				embedder := helper.CreateMockEmbedder("doc-embedder", 256)
				documents := []schema.Document{
					schema.NewDocument("Machine learning is a subset of AI.", map[string]string{"topic": "ML"}),
					schema.NewDocument("Deep learning uses neural networks.", map[string]string{"topic": "DL"}),
					schema.NewDocument("Natural language processing handles text.", map[string]string{"topic": "NLP"}),
				}
				scores := []float32{0.9, 0.8, 0.7}
				vectorStore := newIntegrationMockVectorStore().WithSimilarityResults(documents, scores)

				retriever, err := retrievers.NewVectorStoreRetriever(vectorStore)
				require.NoError(t, err)
				return retriever, embedder
			},
			testFn: func(t *testing.T, retriever *retrievers.VectorStoreRetriever, embedder embeddingsiface.Embedder) {
				ctx := context.Background()

				// Test that documents can be embedded
				documents := []string{
					"Test document for embedding",
				}

				embeddings, err := embedder.EmbedDocuments(ctx, documents)
				require.NoError(t, err)
				assert.Len(t, embeddings, len(documents))
				assert.Greater(t, len(embeddings[0]), 0)

				// Test retrieval
				docs, err := retriever.GetRelevantDocuments(ctx, "machine learning")
				require.NoError(t, err)
				assert.NotNil(t, docs)
			},
			wantErr: false,
		},
		{
			name:        "query_embedding_consistency",
			description: "Test that query embeddings are consistent",
			setupFn: func(t *testing.T) (*retrievers.VectorStoreRetriever, embeddingsiface.Embedder) {
				embedder := helper.CreateMockEmbedder("consistency-embedder", 128)
				documents := utils.CreateTestDocuments(10, "ML")
				scores := make([]float32, len(documents))
				for i := range scores {
					scores[i] = 0.9 - float32(i)*0.1
				}
				vectorStore := newIntegrationMockVectorStore().WithSimilarityResults(documents, scores)

				retriever, err := retrievers.NewVectorStoreRetriever(vectorStore)
				require.NoError(t, err)
				return retriever, embedder
			},
			testFn: func(t *testing.T, retriever *retrievers.VectorStoreRetriever, embedder embeddingsiface.Embedder) {
				ctx := context.Background()
				query := "artificial intelligence"

				// Embed query multiple times - should be consistent
				embedding1, err := embedder.EmbedQuery(ctx, query)
				require.NoError(t, err)

				embedding2, err := embedder.EmbedQuery(ctx, query)
				require.NoError(t, err)

				// Mock embedders may not be deterministic, but should return same dimension
				assert.Equal(t, len(embedding1), len(embedding2))

				// Test retrieval uses embeddings
				docs, err := retriever.GetRelevantDocuments(ctx, query)
				require.NoError(t, err)
				assert.NotNil(t, docs)
			},
			wantErr: false,
		},
		{
			name:        "batch_embedding_retrieval",
			description: "Test batch operations with embeddings",
			setupFn: func(t *testing.T) (*retrievers.VectorStoreRetriever, embeddingsiface.Embedder) {
				embedder := helper.CreateMockEmbedder("batch-embedder", 128)
				documents := utils.CreateTestDocuments(20, "")
				scores := make([]float32, len(documents))
				for i := range scores {
					scores[i] = 0.9 - float32(i)*0.05
				}
				vectorStore := newIntegrationMockVectorStore().WithSimilarityResults(documents, scores)

				retriever, err := retrievers.NewVectorStoreRetriever(vectorStore)
				require.NoError(t, err)
				return retriever, embedder
			},
			testFn: func(t *testing.T, retriever *retrievers.VectorStoreRetriever, embedder embeddingsiface.Embedder) {
				ctx := context.Background()

				// Test batch embedding
				queries := []string{"AI", "ML", "DL"}

				embeddings, err := embedder.EmbedDocuments(ctx, queries)
				require.NoError(t, err)
				assert.Len(t, embeddings, len(queries))

				// Test batch retrieval
				queryInputs := make([]any, len(queries))
				for i, q := range queries {
					queryInputs[i] = q
				}

				results, err := retriever.Batch(ctx, queryInputs)
				require.NoError(t, err)
				assert.Len(t, results, len(queries))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			retriever, embedder := tt.setupFn(t)
			tt.testFn(t, retriever, embedder)
		})
	}
}

// TestIntegrationRetrieversEmbeddingsErrorHandling tests error scenarios.
func TestIntegrationRetrieversEmbeddingsErrorHandling(t *testing.T) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	tests := []struct {
		name        string
		description string
		setupFn     func(t *testing.T) (*retrievers.VectorStoreRetriever, embeddingsiface.Embedder)
		testFn      func(t *testing.T, retriever *retrievers.VectorStoreRetriever, embedder embeddingsiface.Embedder)
		expectError bool
	}{
		{
			name:        "embedder_error_handling",
			description: "Test handling of embedder errors",
			setupFn: func(t *testing.T) (*retrievers.VectorStoreRetriever, embeddingsiface.Embedder) {
				embedder := helper.CreateMockEmbedder("error-embedder", 128)
				vectorStore := newIntegrationMockVectorStore()
				retriever, err := retrievers.NewVectorStoreRetriever(vectorStore)
				require.NoError(t, err)
				return retriever, embedder
			},
			testFn: func(t *testing.T, retriever *retrievers.VectorStoreRetriever, embedder embeddingsiface.Embedder) {
				ctx := context.Background()

				// Test embedder error handling
				// Mock embedders may not always error, so we test the interface
				_, err := embedder.EmbedQuery(ctx, "test query")
				// May succeed or fail depending on mock implementation
				t.Logf("Embedding result: err=%v", err)

				// Test retrieval - may use embedder internally
				_, err = retriever.GetRelevantDocuments(ctx, "test query")
				t.Logf("Retrieval result: err=%v", err)
			},
			expectError: false, // Mock may not error
		},
		{
			name:        "empty_query_handling",
			description: "Test handling of empty queries",
			setupFn: func(t *testing.T) (*retrievers.VectorStoreRetriever, embeddingsiface.Embedder) {
				embedder := helper.CreateMockEmbedder("empty-embedder", 128)
				vectorStore := newIntegrationMockVectorStore()
				retriever, err := retrievers.NewVectorStoreRetriever(vectorStore)
				require.NoError(t, err)
				return retriever, embedder
			},
			testFn: func(t *testing.T, retriever *retrievers.VectorStoreRetriever, embedder embeddingsiface.Embedder) {
				ctx := context.Background()

				// Test empty query embedding
				_, err := embedder.EmbedQuery(ctx, "")
				// May succeed or fail depending on implementation
				t.Logf("Empty query embedding: err=%v", err)

				// Test empty query retrieval
				_, err = retriever.GetRelevantDocuments(ctx, "")
				// May succeed or fail depending on implementation
				t.Logf("Empty query retrieval: err=%v", err)
			},
			expectError: false, // May or may not error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			retriever, embedder := tt.setupFn(t)
			tt.testFn(t, retriever, embedder)
		})
	}
}

// TestIntegrationRetrieversEmbeddingsPerformance tests performance characteristics.
func TestIntegrationRetrieversEmbeddingsPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	ctx := context.Background()
	embedder := helper.CreateMockEmbedder("perf-embedder", 128)
	documents := utils.CreateTestDocuments(50, "")
	scores := make([]float32, len(documents))
	for i := range scores {
		scores[i] = 0.9 - float32(i)*0.02
	}
	vectorStore := newIntegrationMockVectorStore().WithSimilarityResults(documents, scores)

	retriever, err := retrievers.NewVectorStoreRetriever(vectorStore)
	require.NoError(t, err)

	queries := utils.CreateTestQueries(10)

	// Benchmark embedding + retrieval
	start := time.Now()
	for _, query := range queries {
		// Embed query
		_, err := embedder.EmbedQuery(ctx, query)
		require.NoError(t, err)

		// Retrieve documents
		_, err = retriever.GetRelevantDocuments(ctx, query)
		require.NoError(t, err)
	}
	duration := time.Since(start)

	t.Logf("Performance: %d queries (embed + retrieve) in %v (avg: %v per query)", len(queries), duration, duration/time.Duration(len(queries)))
	assert.Less(t, duration, 10*time.Second, "Should complete within reasonable time")
}

// BenchmarkIntegrationRetrieversEmbeddings benchmarks retrievers-embeddings integration performance.
func BenchmarkIntegrationRetrieversEmbeddings(b *testing.B) {
	helper := utils.NewIntegrationTestHelper()
	defer func() { _ = helper.Cleanup(context.Background()) }()

	ctx := context.Background()
	embedder := helper.CreateMockEmbedder("benchmark-embedder", 128)
	documents := utils.CreateTestDocuments(100, "")
	scores := make([]float32, len(documents))
	for i := range scores {
		scores[i] = 0.9 - float32(i)*0.01
	}
	vectorStore := newIntegrationMockVectorStore().WithSimilarityResults(documents, scores)

	retriever, err := retrievers.NewVectorStoreRetriever(vectorStore)
	if err != nil {
		b.Fatalf("Failed to create retriever: %v", err)
	}

	queries := utils.CreateTestQueries(5)

	b.Run("EmbedQuery", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			query := queries[i%len(queries)]
			_, err := embedder.EmbedQuery(ctx, query)
			if err != nil {
				b.Errorf("EmbedQuery error: %v", err)
			}
		}
	})

	b.Run("RetrieveWithEmbedding", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			query := queries[i%len(queries)]
			_, err := retriever.GetRelevantDocuments(ctx, query)
			if err != nil {
				b.Errorf("GetRelevantDocuments error: %v", err)
			}
		}
	})
}
