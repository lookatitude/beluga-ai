package integration

import (
	"context"
	"fmt"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEmbeddingsVectorStoreIntegration tests the integration between embeddings and vector store operations
func TestEmbeddingsVectorStoreIntegration(t *testing.T) {
	// Setup embedding configuration
	config := &embeddings.Config{
		Mock: &embeddings.MockConfig{
			Dimension: 128,
			Seed:      42, // Deterministic for testing
			Enabled:   true,
		},
	}

	// Create embeddings factory
	factory, err := embeddings.NewEmbedderFactory(config)
	require.NoError(t, err, "Failed to create embeddings factory")

	// Create embedder
	embedder, err := factory.NewEmbedder("mock")
	require.NoError(t, err, "Failed to create embedder")

	ctx := context.Background()

	// Test data simulating documents that would be stored in a vector store
	testDocuments := []string{
		"The cat sat on the mat in the kitchen.",
		"A feline was resting on a rug near the stove.",
		"The dog played in the backyard with a ball.",
		"A canine was running around the garden chasing a toy.",
		"The weather is sunny today in the city.",
		"It's a bright and clear day in downtown.",
	}

	t.Run("GenerateEmbeddingsForVectorStore", func(t *testing.T) {
		// Generate embeddings for all documents
		embeddings, err := embedder.EmbedDocuments(ctx, testDocuments)
		require.NoError(t, err, "Failed to generate document embeddings")
		require.Len(t, embeddings, len(testDocuments), "Should have embeddings for all documents")

		// Verify all embeddings have the expected dimension
		expectedDim := 128
		for i, emb := range embeddings {
			assert.Len(t, emb, expectedDim, fmt.Sprintf("Document %d should have %d dimensions", i, expectedDim))
		}

		t.Logf("Successfully generated %d embeddings with dimension %d", len(embeddings), expectedDim)
	})

	t.Run("QueryEmbeddingForSimilaritySearch", func(t *testing.T) {
		// Generate a query embedding
		query := "Tell me about cats and their resting places"
		queryEmbedding, err := embedder.EmbedQuery(ctx, query)
		require.NoError(t, err, "Failed to generate query embedding")

		// Verify query embedding dimensions
		expectedDim := 128
		assert.Len(t, queryEmbedding, expectedDim, "Query embedding should have expected dimensions")

		t.Logf("Successfully generated query embedding with %d dimensions", len(queryEmbedding))
	})

	t.Run("EmbeddingConsistencyForDuplicateContent", func(t *testing.T) {
		// Test that identical content produces identical embeddings
		content := "This is a test document for consistency verification."

		emb1, err := embedder.EmbedQuery(ctx, content)
		require.NoError(t, err)

		emb2, err := embedder.EmbedQuery(ctx, content)
		require.NoError(t, err)

		// Verify embeddings are identical (mock should be deterministic)
		assert.Equal(t, emb1, emb2, "Identical content should produce identical embeddings")

		t.Log("Verified embedding consistency for duplicate content")
	})

	t.Run("BatchVsIndividualEmbeddingConsistency", func(t *testing.T) {
		// Test that batch embedding produces same results as individual embeddings
		singleDoc := "This is a single test document."

		// Get embedding via batch API
		batchEmbeddings, err := embedder.EmbedDocuments(ctx, []string{singleDoc})
		require.NoError(t, err)
		require.Len(t, batchEmbeddings, 1)

		// Get embedding via individual API
		individualEmbedding, err := embedder.EmbedQuery(ctx, singleDoc)
		require.NoError(t, err)

		// Verify they are identical
		assert.Equal(t, batchEmbeddings[0], individualEmbedding,
			"Batch and individual embedding APIs should produce identical results")

		t.Log("Verified consistency between batch and individual embedding APIs")
	})

	t.Run("VectorStoreWorkflowSimulation", func(t *testing.T) {
		// Simulate a typical vector store workflow:
		// 1. Embed documents for storage
		// 2. Embed query for search
		// 3. Verify dimensions are compatible

		// Step 1: Embed documents (simulating storage preparation)
		docEmbeddings, err := embedder.EmbedDocuments(ctx, testDocuments)
		require.NoError(t, err)

		// Step 2: Embed search query
		searchQuery := "feline resting on floor covering"
		queryEmbedding, err := embedder.EmbedQuery(ctx, searchQuery)
		require.NoError(t, err)

		// Step 3: Verify dimensional compatibility
		for i, docEmb := range docEmbeddings {
			assert.Len(t, docEmb, len(queryEmbedding),
				fmt.Sprintf("Document %d embedding dimension should match query dimension", i))
		}

		// Step 4: Simulate similarity calculation (basic check)
		// In a real vector store, this would use optimized similarity search
		queryDim := len(queryEmbedding)
		for i, docEmb := range docEmbeddings {
			assert.Len(t, docEmb, queryDim,
				fmt.Sprintf("Document %d embedding should be compatible with query", i))
		}

		t.Logf("Successfully simulated vector store workflow with %d documents and query", len(testDocuments))
	})

	t.Run("ConcurrentEmbeddingOperations", func(t *testing.T) {
		// Test concurrent embedding operations (common in vector store scenarios)
		numWorkers := 5
		operationsPerWorker := 10

		type result struct {
			workerID int
			success  bool
			error    error
		}

		results := make(chan result, numWorkers*operationsPerWorker)

		// Start concurrent workers
		for workerID := 0; workerID < numWorkers; workerID++ {
			go func(wid int) {
				// Each worker creates its own embedder instance
				workerEmbedder, err := factory.NewEmbedder("mock")
				if err != nil {
					results <- result{wid, false, err}
					return
				}

				for i := 0; i < operationsPerWorker; i++ {
					doc := fmt.Sprintf("Worker %d document %d for concurrent testing", wid, i)
					_, err := workerEmbedder.EmbedQuery(ctx, doc)
					results <- result{wid, err == nil, err}
				}
			}(workerID)
		}

		// Collect results
		totalOperations := numWorkers * operationsPerWorker
		successCount := 0

		for i := 0; i < totalOperations; i++ {
			res := <-results
			if res.success {
				successCount++
			} else {
				t.Errorf("Worker %d failed: %v", res.workerID, res.error)
			}
		}

		assert.Equal(t, totalOperations, successCount, "All concurrent operations should succeed")
		t.Logf("Successfully completed %d concurrent embedding operations", totalOperations)
	})

	t.Run("HealthCheckIntegration", func(t *testing.T) {
		// Test that embedder health checks work in integration context
		err := factory.CheckHealth(ctx, "mock")
		assert.NoError(t, err, "Embedder health check should pass")

		t.Log("Embedder health check passed")
	})
}
