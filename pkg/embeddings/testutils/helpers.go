package testutils

import (
	"context"
	"math/rand"
	"strings"
	"time"
)

// Note: TestConfig functions moved to avoid import cycles.
// Use the config creation functions directly in test files instead.

// TestDocuments returns a set of test documents for embedding tests
func TestDocuments() []string {
	return []string{
		"The quick brown fox jumps over the lazy dog",
		"Machine learning is a subset of artificial intelligence",
		"Natural language processing helps computers understand human language",
		"Vector databases store high-dimensional vectors efficiently",
		"Semantic search finds content based on meaning rather than keywords",
	}
}

// TestQueries returns a set of test queries for embedding tests
func TestQueries() []string {
	return []string{
		"What is machine learning?",
		"How do vector databases work?",
		"What are embeddings?",
		"Tell me about natural language processing",
	}
}

// RandomDocuments generates random test documents
func RandomDocuments(count, minWords, maxWords int) []string {
	rand.Seed(time.Now().UnixNano())
	documents := make([]string, count)

	words := []string{
		"the", "quick", "brown", "fox", "jumps", "over", "lazy", "dog",
		"machine", "learning", "artificial", "intelligence", "natural",
		"language", "processing", "vector", "database", "semantic",
		"search", "embedding", "computer", "science", "algorithm",
		"neural", "network", "deep", "learning", "data", "analysis",
	}

	for i := 0; i < count; i++ {
		wordCount := minWords + rand.Intn(maxWords-minWords+1)
		docWords := make([]string, wordCount)
		for j := 0; j < wordCount; j++ {
			docWords[j] = words[rand.Intn(len(words))]
		}
		documents[i] = strings.Join(docWords, " ")
	}

	return documents
}

// TestContext returns a context with timeout for testing
func TestContext() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	return ctx
}

// TestContextWithCancel returns a cancellable context for testing
func TestContextWithCancel() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// PerformanceMetrics holds performance test results
type PerformanceMetrics struct {
	QueriesPerSecond    float64
	DocumentsPerSecond  float64
	AverageLatency      time.Duration
	P95Latency          time.Duration
	ErrorRate           float64
}

// Note: Interface testing and performance measurement functions
// moved to avoid import cycles. Use directly in test files.

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Note: For actual benchmarking, use the standard testing.B in your test files
