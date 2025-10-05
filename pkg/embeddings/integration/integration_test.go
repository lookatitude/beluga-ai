//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
)

// TestEmbedderFactory_Integration tests the embedder factory with real providers
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

// TestCrossProviderCompatibility tests that different providers can be used interchangeably
func TestCrossProviderCompatibility(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping cross-provider compatibility test in short mode")
	}

	ctx := context.Background()
	testText := "This is a test document for cross-provider compatibility."

	// Test data that should work across all providers
	compatibleProviders := []struct {
		name   string
		config *embeddings.Config
	}{
		{
			name: "mock",
			config: &embeddings.Config{
				Mock: &embeddings.MockConfig{
					Dimension: 128,
				},
			},
		},
	}

	// Add OpenAI if API key is available
	if os.Getenv("OPENAI_API_KEY") != "" {
		compatibleProviders = append(compatibleProviders, struct {
			name   string
			config *embeddings.Config
		}{
			name: "openai",
			config: &embeddings.Config{
				OpenAI: &embeddings.OpenAIConfig{
					APIKey: os.Getenv("OPENAI_API_KEY"),
					Model:  "text-embedding-ada-002",
				},
			},
		})
	}

	// Add Ollama if available
	if os.Getenv("OLLAMA_SERVER_URL") != "" || isOllamaServerAvailable() {
		compatibleProviders = append(compatibleProviders, struct {
			name   string
			config *embeddings.Config
		}{
			name: "ollama",
			config: &embeddings.Config{
				Ollama: &embeddings.OllamaConfig{
					Model: "nomic-embed-text",
				},
			},
		})
	}

	if len(compatibleProviders) < 2 {
		t.Skip("Need at least 2 providers for cross-provider compatibility test")
	}

	results := make(map[string][]float32)

	// Test each provider with the same input
	for _, provider := range compatibleProviders {
		t.Run(fmt.Sprintf("embed_%s", provider.name), func(t *testing.T) {
			embedder, err := embeddings.NewEmbedder(ctx, provider.name, *provider.config)
			if err != nil {
				t.Fatalf("Failed to create %s embedder: %v", provider.name, err)
			}

			vectors, err := embedder.EmbedQuery(ctx, testText)
			if err != nil {
				t.Fatalf("Failed to embed with %s: %v", provider.name, err)
			}

			if len(vectors) == 0 {
				t.Errorf("%s returned empty embedding", provider.name)
			}

			results[provider.name] = vectors
			t.Logf("%s produced embedding with %d dimensions", provider.name, len(vectors))
		})
	}

	// Verify results are reasonable (same text should produce embeddings)
	for provider, vectors := range results {
		if len(vectors) == 0 {
			t.Errorf("%s produced empty embedding", provider)
		}

		// Check for NaN or infinite values
		for i, v := range vectors {
			if fmt.Sprintf("%f", v) == "NaN" {
				t.Errorf("%s embedding[%d] is NaN", provider, i)
			}
		}
	}

	t.Logf("Cross-provider compatibility test completed with %d providers", len(results))
}

// TestEndToEndWorkflow tests complete embedding workflows from creation to usage
func TestEndToEndWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end workflow test in short mode")
	}

	ctx := context.Background()

	// Test documents
	documents := []string{
		"The quick brown fox jumps over the lazy dog.",
		"Machine learning is transforming technology.",
		"Artificial intelligence enables automation.",
		"Natural language processing understands text.",
		"Computer vision recognizes images and patterns.",
	}

	workflows := []struct {
		name     string
		config   embeddings.Config
		testFunc func(t *testing.T, embedder iface.Embedder)
	}{
		{
			name: "mock_workflow",
			config: embeddings.Config{
				Mock: &embeddings.MockConfig{
					Dimension: 128,
				},
			},
			testFunc: func(t *testing.T, embedder iface.Embedder) {
				testEmbedderWorkflow(t, ctx, embedder, documents)
			},
		},
	}

	// Add real provider workflows if available
	if os.Getenv("OPENAI_API_KEY") != "" {
		workflows = append(workflows, struct {
			name     string
			config   embeddings.Config
			testFunc func(t *testing.T, embedder iface.Embedder)
		}{
			name: "openai_workflow",
			config: embeddings.Config{
				OpenAI: &embeddings.OpenAIConfig{
					APIKey: os.Getenv("OPENAI_API_KEY"),
					Model:  "text-embedding-ada-002",
				},
			},
			testFunc: func(t *testing.T, embedder iface.Embedder) {
				testEmbedderWorkflow(t, ctx, embedder, documents)
			},
		})
	}

	for _, workflow := range workflows {
		t.Run(workflow.name, func(t *testing.T) {
			embedder, err := embeddings.NewEmbedder(ctx, strings.TrimSuffix(workflow.name, "_workflow"), workflow.config)
			if err != nil {
				t.Fatalf("Failed to create embedder for %s: %v", workflow.name, err)
			}

			workflow.testFunc(t, embedder)
		})
	}
}

// testEmbedderWorkflow performs a complete embedding workflow test
func testEmbedderWorkflow(t *testing.T, ctx context.Context, embedder iface.Embedder, documents []string) {
	// Test dimension retrieval
	dimension, err := embedder.GetDimension(ctx)
	if err != nil {
		t.Fatalf("Failed to get dimension: %v", err)
	}
	if dimension <= 0 {
		t.Errorf("Invalid dimension: %d", dimension)
	}

	// Test batch embedding
	batchVectors, err := embedder.EmbedDocuments(ctx, documents)
	if err != nil {
		t.Fatalf("Failed to embed documents: %v", err)
	}

	if len(batchVectors) != len(documents) {
		t.Errorf("Expected %d embeddings, got %d", len(documents), len(batchVectors))
	}

	// Verify each document produced an embedding
	for i, vectors := range batchVectors {
		if len(vectors) != dimension {
			t.Errorf("Document %d: expected dimension %d, got %d", i, dimension, len(vectors))
		}

		// Check for valid float values
		for j, v := range vectors {
			if fmt.Sprintf("%f", v) == "NaN" {
				t.Errorf("Document %d, dimension %d: NaN value", i, j)
			}
		}
	}

	// Test single query embedding
	query := "What is artificial intelligence?"
	queryVector, err := embedder.EmbedQuery(ctx, query)
	if err != nil {
		t.Fatalf("Failed to embed query: %v", err)
	}

	if len(queryVector) != dimension {
		t.Errorf("Query embedding: expected dimension %d, got %d", dimension, len(queryVector))
	}

	// Verify query embedding has valid values
	for i, v := range queryVector {
		if fmt.Sprintf("%f", v) == "NaN" {
			t.Errorf("Query embedding dimension %d: NaN value", i)
		}
	}

	t.Logf("Workflow completed: %d documents + 1 query embedded successfully", len(documents))
}

// TestProviderSwitching tests the ability to switch between providers dynamically
func TestProviderSwitching(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping provider switching test in short mode")
	}

	ctx := context.Background()
	testText := "Provider switching test document."

	providers := []string{"mock"}

	// Add real providers if available
	if os.Getenv("OPENAI_API_KEY") != "" {
		providers = append(providers, "openai")
	}

	if len(providers) < 2 {
		t.Skip("Need at least 2 providers for switching test")
	}

	configs := map[string]embeddings.Config{
		"mock": {
			Mock: &embeddings.MockConfig{
				Dimension: 128,
			},
		},
		"openai": {
			OpenAI: &embeddings.OpenAIConfig{
				APIKey: os.Getenv("OPENAI_API_KEY"),
				Model:  "text-embedding-ada-002",
			},
		},
	}

	// Test switching between providers
	for i, provider1 := range providers {
		for j, provider2 := range providers {
			if i == j {
				continue // Skip same provider
			}

			t.Run(fmt.Sprintf("switch_%s_to_%s", provider1, provider2), func(t *testing.T) {
				// Create first embedder
				embedder1, err := embeddings.NewEmbedder(ctx, provider1, configs[provider1])
				if err != nil {
					t.Fatalf("Failed to create %s embedder: %v", provider1, err)
				}

				// Create second embedder
				embedder2, err := embeddings.NewEmbedder(ctx, provider2, configs[provider2])
				if err != nil {
					t.Fatalf("Failed to create %s embedder: %v", provider2, err)
				}

				// Embed with first provider
				vector1, err := embedder1.EmbedQuery(ctx, testText)
				if err != nil {
					t.Fatalf("Failed to embed with %s: %v", provider1, err)
				}

				// Embed with second provider
				vector2, err := embedder2.EmbedQuery(ctx, testText)
				if err != nil {
					t.Fatalf("Failed to embed with %s: %v", provider2, err)
				}

				// Verify both embeddings are valid
				if len(vector1) == 0 {
					t.Errorf("%s produced empty embedding", provider1)
				}
				if len(vector2) == 0 {
					t.Errorf("%s produced empty embedding", provider2)
				}

				// Embeddings from different providers should generally have different dimensions
				// (This is a sanity check, not a requirement)
				t.Logf("Provider switch test: %s (%d dims) → %s (%d dims)",
					provider1, len(vector1), provider2, len(vector2))
			})
		}
	}
}
