//go:build integration

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/testutils"
)

// TestEmbedderFactory_Integration tests the embedder factory with real providers
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
func TestEmbedderFactory_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test with mock provider (always available)
	t.Run("mock_provider", func(t *testing.T) {
		testMockProviderIntegration(t)
	})

	// Test OpenAI if API key is available
	if os.Getenv("OPENAI_API_KEY") != "" {
		t.Run("openai_provider", func(t *testing.T) {
			testOpenAIProviderIntegration(t)
		})
	} else {
		t.Log("Skipping OpenAI integration test: OPENAI_API_KEY not set")
	}

	// Test Ollama if server is available
	if os.Getenv("OLLAMA_SERVER_URL") != "" || isOllamaServerAvailable() {
		t.Run("ollama_provider", func(t *testing.T) {
			testOllamaProviderIntegration(t)
		})
	} else {
		t.Log("Skipping Ollama integration test: OLLAMA_SERVER_URL not set and server not detected")
	}
}

func testMockProviderIntegration(t *testing.T) {
	config := testutils.TestConfigWithProvider("mock")

	factory, err := embeddings.NewEmbedderFactory(config)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	ctx := testutils.TestContext()

	// Test embedder creation
	embedder, err := factory.NewEmbedder("mock")
	if err != nil {
		t.Fatalf("Failed to create mock embedder: %v", err)
	}

	// Test interface compliance
	if err := testutils.AssertEmbedderInterface(embedder); err != nil {
		t.Errorf("Interface compliance failed: %v", err)
	}

	// Test health check
	if err := factory.CheckHealth(ctx, "mock"); err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	// Test deterministic output
	documents := testutils.TestDocuments()
	embeddings1, err := embedder.EmbedDocuments(ctx, documents)
	if err != nil {
		t.Fatalf("First embedding failed: %v", err)
	}

	embeddings2, err := embedder.EmbedDocuments(ctx, documents)
	if err != nil {
		t.Fatalf("Second embedding failed: %v", err)
	}

	// Should be identical (deterministic)
	for i := range embeddings1 {
		for j := range embeddings1[i] {
			if embeddings1[i][j] != embeddings2[i][j] {
				t.Errorf("Embeddings not deterministic at [%d][%d]: %f != %f", i, j, embeddings1[i][j], embeddings2[i][j])
			}
		}
	}
}

func testOpenAIProviderIntegration(t *testing.T) {
	config := &embeddings.Config{
		OpenAI: &embeddings.OpenAIConfig{
			APIKey:     os.Getenv("OPENAI_API_KEY"),
			Model:      getEnvOrDefault("OPENAI_MODEL", "text-embedding-ada-002"),
			Timeout:    60 * time.Second, // Longer timeout for real API
			MaxRetries: 2,
			Enabled:    true,
		},
	}

	factory, err := embeddings.NewEmbedderFactory(config)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	ctx := testutils.TestContext()

	// Test embedder creation
	embedder, err := factory.NewEmbedder("openai")
	if err != nil {
		t.Fatalf("Failed to create OpenAI embedder: %v", err)
	}

	// Test interface compliance
	if err := testutils.AssertEmbedderInterface(embedder); err != nil {
		t.Errorf("Interface compliance failed: %v", err)
	}

	// Test health check
	if err := factory.CheckHealth(ctx, "openai"); err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	// Test with real documents
	documents := testutils.TestDocuments()[:3] // Limit to reduce API costs
	_, err = embedder.EmbedDocuments(ctx, documents)
	if err != nil {
		t.Errorf("Document embedding failed: %v", err)
	}

	// Test with real query
	query := testutils.TestQueries()[0]
	_, err = embedder.EmbedQuery(ctx, query)
	if err != nil {
		t.Errorf("Query embedding failed: %v", err)
	}

	// Test dimension
	dimension, err := embedder.GetDimension(ctx)
	if err != nil {
		t.Errorf("GetDimension failed: %v", err)
	}
	if dimension <= 0 {
		t.Errorf("Invalid dimension: %d", dimension)
	}

	t.Logf("OpenAI integration test passed with dimension: %d", dimension)
}

func testOllamaProviderIntegration(t *testing.T) {
	serverURL := getEnvOrDefault("OLLAMA_SERVER_URL", "http://localhost:11434")

	config := &embeddings.Config{
		Ollama: &embeddings.OllamaConfig{
			ServerURL:  serverURL,
			Model:      getEnvOrDefault("OLLAMA_MODEL", "nomic-embed-text"),
			Timeout:    120 * time.Second, // Longer timeout for local model
			MaxRetries: 1,
			Enabled:    true,
		},
	}

	factory, err := embeddings.NewEmbedderFactory(config)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	ctx := testutils.TestContext()

	// Test embedder creation
	embedder, err := factory.NewEmbedder("ollama")
	if err != nil {
		t.Fatalf("Failed to create Ollama embedder: %v", err)
	}

	// Test interface compliance (may be more lenient for Ollama)
	if err := testutils.AssertEmbedderInterface(embedder); err != nil {
		t.Errorf("Interface compliance failed: %v", err)
	}

	// Test health check
	if err := factory.CheckHealth(ctx, "ollama"); err != nil {
		t.Errorf("Health check failed: %v", err)
	}

	// Test with real documents
	documents := testutils.TestDocuments()[:2] // Smaller batch for local model
	_, err = embedder.EmbedDocuments(ctx, documents)
	if err != nil {
		t.Errorf("Document embedding failed: %v", err)
	}

	// Test with real query
	query := testutils.TestQueries()[0]
	_, err = embedder.EmbedQuery(ctx, query)
	if err != nil {
		t.Errorf("Query embedding failed: %v", err)
	}

	// Test dimension (may be unknown for Ollama)
	dimension, err := embedder.GetDimension(ctx)
	if err != nil {
		t.Logf("GetDimension failed (expected for some Ollama models): %v", err)
	} else if dimension > 0 {
		t.Logf("Ollama dimension: %d", dimension)
	} else {
		t.Log("Ollama dimension unknown (expected)")
	}

	t.Log("Ollama integration test passed")
}
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()

// TestPerformance_Integration tests performance with real providers
func TestPerformance_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	config := testutils.TestConfigWithProvider("mock") // Use mock for consistent performance testing

	factory, err := embeddings.NewEmbedderFactory(config)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	embedder, err := factory.NewEmbedder("mock")
	if err != nil {
		t.Fatalf("Failed to create embedder: %v", err)
	}

	ctx := testutils.TestContext()

	// Run performance measurement
	metrics, err := testutils.MeasurePerformance(embedder, 5*time.Second)
	if err != nil {
		t.Fatalf("Performance measurement failed: %v", err)
	}

	t.Logf("Performance metrics: Queries/sec: %.2f, Documents/sec: %.2f",
		metrics.QueriesPerSecond, metrics.DocumentsPerSecond)

	// Basic sanity checks
	if metrics.QueriesPerSecond <= 0 {
		t.Error("Queries per second should be positive")
	}
	if metrics.DocumentsPerSecond <= 0 {
		t.Error("Documents per second should be positive")
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	}
}

// TestConcurrentAccess_Integration tests concurrent access with real providers
func TestConcurrentAccess_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	config := testutils.TestConfigWithProvider("mock")

	factory, err := embeddings.NewEmbedderFactory(config)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	ctx := testutils.TestContext()
	documents := testutils.TestDocuments()
	queries := testutils.TestQueries()

	// Test concurrent embedder creation and usage
	done := make(chan bool, 20)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Create embedder
			embedder, err := factory.NewEmbedder("mock")
			if err != nil {
				t.Errorf("Goroutine %d: failed to create embedder: %v", id, err)
				return
			}

			// Embed documents
			_, err = embedder.EmbedDocuments(ctx, documents)
			if err != nil {
				t.Errorf("Goroutine %d: failed to embed documents: %v", id, err)
			}

			// Embed queries
			for _, query := range queries {
				_, err = embedder.EmbedQuery(ctx, query)
				if err != nil {
					t.Errorf("Goroutine %d: failed to embed query %q: %v", id, query, err)
				}
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Helper functions

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func isOllamaServerAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5s)
	defer cancel()
	// Simple check - in a real implementation, you might try to connect
	// For now, just check if we can create a basic HTTP client
	return true // Assume available, let the test fail if not
}

// TestLoadTest_Integration performs load testing with real providers
func TestLoadTest_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	config := testutils.TestConfigWithProvider("mock")

	factory, err := embeddings.NewEmbedderFactory(config)
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}

	ctx := testutils.TestContext()
	documents := testutils.RandomDocuments(1000, 5, 20) // 1000 documents

	embedder, err := factory.NewEmbedder("mock")
	if err != nil {
		t.Fatalf("Failed to create embedder: %v", err)
	}

	start := time.Now()

	// Process all documents in batches
	batchSize := 50
	totalProcessed := 0

	for i := 0; i < len(documents); i += batchSize {
		end := i + batchSize
		if end > len(documents) {
			end = len(documents)
		}

		batch := documents[i:end]
		_, err := embedder.EmbedDocuments(ctx, batch)
		if err != nil {
			t.Errorf("Batch %d-%d failed: %v", i, end, err)
			continue
		}

		totalProcessed += len(batch)
	}

	duration := time.Since(start)
	documentsPerSecond := float64(totalProcessed) / duration.Seconds()

	t.Logf("Load test completed: %d documents processed in %v (%.2f docs/sec)",
		totalProcessed, duration, documentsPerSecond)

	if totalProcessed == 0 {
		t.Error("No documents were processed successfully")
	}
}
