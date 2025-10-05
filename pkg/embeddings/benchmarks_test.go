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
)

func BenchmarkNewEmbedderFactory(b *testing.B) {
	config := &Config{
		Mock: &MockConfig{
			Dimension: 128,
			Enabled:   true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewEmbedderFactory(config)
	}
}

func BenchmarkEmbedderFactory_NewEmbedder(b *testing.B) {
	config := &Config{
		Mock: &MockConfig{
			Dimension: 128,
			Enabled:   true,
		},
	}

	factory, err := NewEmbedderFactory(config)
	if err != nil {
		b.Fatalf("Failed to create factory: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = factory.NewEmbedder("mock")
	}
}

func BenchmarkConfig_Validate(b *testing.B) {
	config := &Config{
		Mock: &MockConfig{
			Dimension: 128,
			Enabled:   true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.Validate()
	}
}

func BenchmarkConfig_SetDefaults(b *testing.B) {
	config := &Config{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config.SetDefaults()
	}
}

// Provider-specific benchmarks
func BenchmarkMockEmbedder_EmbedQuery(b *testing.B) {
	config := &Config{
		Mock: &MockConfig{
			Dimension: 128,
			Enabled:   true,
		},
	}
	factory, err := NewEmbedderFactory(config)
	if err != nil {
		b.Fatalf("Failed to create factory: %v", err)
	}

	embedder, err := factory.NewEmbedder("mock")
	if err != nil {
		b.Fatalf("Failed to create embedder: %v", err)
	}

	ctx := context.Background()
	query := "This is a benchmark query for testing embedding performance"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = embedder.EmbedQuery(ctx, query)
	}
}

func BenchmarkMockEmbedder_EmbedDocuments_SmallBatch(b *testing.B) {
	benchmarkMockEmbedderEmbedDocuments(b, 5)
}

func BenchmarkMockEmbedder_EmbedDocuments_MediumBatch(b *testing.B) {
	benchmarkMockEmbedderEmbedDocuments(b, 20)
}

func BenchmarkMockEmbedder_EmbedDocuments_LargeBatch(b *testing.B) {
	benchmarkMockEmbedderEmbedDocuments(b, 100)
}

func benchmarkMockEmbedderEmbedDocuments(b *testing.B, batchSize int) {
	config := &Config{
		Mock: &MockConfig{
			Dimension: 128,
			Enabled:   true,
		},
	}
	factory, err := NewEmbedderFactory(config)
	if err != nil {
		b.Fatalf("Failed to create factory: %v", err)
	}

	embedder, err := factory.NewEmbedder("mock")
	if err != nil {
		b.Fatalf("Failed to create embedder: %v", err)
	}

	ctx := context.Background()
	documents := make([]string, batchSize)
	for i := range documents {
		documents[i] = fmt.Sprintf("This is test document number %d for benchmarking purposes", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = embedder.EmbedDocuments(ctx, documents)
	}
}

// Memory benchmarks
func BenchmarkMockEmbedder_EmbedDocuments_Memory(b *testing.B) {
	config := &Config{
		Mock: &MockConfig{
			Dimension: 128,
			Enabled:   true,
		},
	}
	factory, err := NewEmbedderFactory(config)
	if err != nil {
		b.Fatalf("Failed to create factory: %v", err)
	}

	embedder, err := factory.NewEmbedder("mock")
	if err != nil {
		b.Fatalf("Failed to create embedder: %v", err)
	}

	ctx := context.Background()
	documents := make([]string, 50)
	for i := range documents {
		documents[i] = fmt.Sprintf("This is a longer test document number %d with more content for memory benchmarking purposes and testing allocation patterns", i)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = embedder.EmbedDocuments(ctx, documents)
	}
}

// Concurrent benchmarks
func BenchmarkMockEmbedder_ConcurrentEmbeddings(b *testing.B) {
	config := &Config{
		Mock: &MockConfig{
			Dimension: 128,
			Enabled:   true,
		},
	}
	factory, err := NewEmbedderFactory(config)
	if err != nil {
		b.Fatalf("Failed to create factory: %v", err)
	}

	ctx := context.Background()
	documents := []string{"Test document 1", "Test document 2", "Test document 3"}
	query := "Benchmark query for concurrent testing"

	b.RunParallel(func(pb *testing.PB) {
		embedder, err := factory.NewEmbedder("mock")
		if err != nil {
			b.Fatalf("Failed to create embedder: %v", err)
		}

		for pb.Next() {
			// Alternate between documents and query embedding
			if b.N%2 == 0 {
				_, _ = embedder.EmbedDocuments(ctx, documents)
			} else {
				_, _ = embedder.EmbedQuery(ctx, query)
			}
		}
	})
}

// Dimension benchmarks
func BenchmarkMockEmbedder_DifferentDimensions(b *testing.B) {
	dimensions := []int{64, 128, 256, 512, 1024}

	for _, dim := range dimensions {
		b.Run(fmt.Sprintf("Dim%d", dim), func(b *testing.B) {
			config := &Config{
				Mock: &MockConfig{
					Dimension: dim,
					Seed:      42,
					Enabled:   true,
				},
			}

			factory, err := NewEmbedderFactory(config)
			if err != nil {
				b.Fatalf("Failed to create factory: %v", err)
			}

			embedder, err := factory.NewEmbedder("mock")
			if err != nil {
				b.Fatalf("Failed to create embedder: %v", err)
			}

			ctx := context.Background()
			documents := make([]string, 10)
			for i := range documents {
				documents[i] = fmt.Sprintf("Test document %d for dimension %d", i, dim)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = embedder.EmbedDocuments(ctx, documents)
			}
		})
	}
}

// Factory operation benchmarks
func BenchmarkEmbedderFactory_GetAvailableProviders(b *testing.B) {
	config := &Config{
		Mock: &MockConfig{
			Dimension: 128,
			Enabled:   true,
		},
	}

	factory, err := NewEmbedderFactory(config)
	if err != nil {
		b.Fatalf("Failed to create factory: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = factory.GetAvailableProviders()
	}
}

func BenchmarkEmbedderFactory_CheckHealth(b *testing.B) {
	config := &Config{
		Mock: &MockConfig{
			Dimension: 128,
			Enabled:   true,
		},
	}

	factory, err := NewEmbedderFactory(config)
	if err != nil {
		b.Fatalf("Failed to create factory: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = factory.CheckHealth(ctx, "mock")
	}
}

// Performance comparison benchmarks
func BenchmarkEmbedderFactory_ProviderCreation(b *testing.B) {
	config := &Config{
		Mock: &MockConfig{
			Dimension: 128,
			Enabled:   true,
		},
	}

	factory, err := NewEmbedderFactory(config)
	if err != nil {
		b.Fatalf("Failed to create factory: %v", err)
	}

	providers := []string{"mock"} // Only test mock for consistent benchmarking

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, provider := range providers {
			_, _ = factory.NewEmbedder(provider)
		}
	}
}

// Load testing benchmarks
func BenchmarkMockEmbedder_LoadTest_SmallDocuments(b *testing.B) {
	benchmarkLoadTest(b, 100, 5, 10) // 100 docs, 5-10 words each
}

func BenchmarkMockEmbedder_LoadTest_MediumDocuments(b *testing.B) {
	benchmarkLoadTest(b, 1000, 20, 50) // 1000 docs, 20-50 words each
}

func BenchmarkMockEmbedder_LoadTest_LargeDocuments(b *testing.B) {
	benchmarkLoadTest(b, 100, 100, 200) // 100 docs, 100-200 words each
}

func benchmarkLoadTest(b *testing.B, numDocs, minWords, maxWords int) {
	config := &Config{
		Mock: &MockConfig{
			Dimension: 128,
			Enabled:   true,
		},
	}
	factory, err := NewEmbedderFactory(config)
	if err != nil {
		b.Fatalf("Failed to create factory: %v", err)
	}

	embedder, err := factory.NewEmbedder("mock")
	if err != nil {
		b.Fatalf("Failed to create embedder: %v", err)
	}

	ctx := context.Background()
	documents := make([]string, numDocs)
	for i := range documents {
		words := minWords + (i % (maxWords - minWords + 1))
		docWords := make([]string, words)
		for j := range docWords {
			docWords[j] = fmt.Sprintf("word%d", j)
		}
		documents[i] = strings.Join(docWords, " ")
	}

	b.ResetTimer()
	b.ReportAllocs()

	start := time.Now()
	totalProcessed := 0

	for i := 0; i < b.N; i++ {
		for _, doc := range documents {
			_, err := embedder.EmbedQuery(ctx, doc)
			if err != nil {
				b.Fatalf("Embedding failed: %v", err)
			}
			totalProcessed++
		}
	}

	duration := time.Since(start)
	b.ReportMetric(float64(totalProcessed)/duration.Seconds(), "embeddings/sec")
}

// Throughput benchmarks
func BenchmarkMockEmbedder_Throughput(b *testing.B) {
	config := &Config{
		Mock: &MockConfig{
			Dimension: 128,
			Seed:      42,
			Enabled:   true,
		},
	}

	factory, err := NewEmbedderFactory(config)
	if err != nil {
		b.Fatalf("Failed to create factory: %v", err)
	}

	embedder, err := factory.NewEmbedder("mock")
	if err != nil {
		b.Fatalf("Failed to create embedder: %v", err)
	}

	ctx := context.Background()

	// Pre-generate test data
	smallDocs := make([]string, 100)
	for i := range smallDocs {
		smallDocs[i] = fmt.Sprintf("Small test document number %d with short content", i)
	}
	mediumDocs := make([]string, 50)
	for i := range mediumDocs {
		mediumDocs[i] = fmt.Sprintf("Medium test document number %d with medium length content that contains more words and provides better testing for performance metrics", i)
	}
	largeDocs := make([]string, 10)
	for i := range largeDocs {
		largeDocs[i] = strings.Repeat(fmt.Sprintf("Large test document number %d with very long content that repeats many times to simulate real-world large documents ", i), 20)
	}

	b.Run("SmallDocuments_10", func(b *testing.B) {
		benchmarkThroughput(b, embedder, ctx, smallDocs[:10])
	})

	b.Run("SmallDocuments_50", func(b *testing.B) {
		benchmarkThroughput(b, embedder, ctx, smallDocs[:50])
	})

	b.Run("MediumDocuments_10", func(b *testing.B) {
		benchmarkThroughput(b, embedder, ctx, mediumDocs[:10])
	})

	b.Run("LargeDocuments_5", func(b *testing.B) {
		benchmarkThroughput(b, embedder, ctx, largeDocs[:5])
	})
}

func benchmarkThroughput(b *testing.B, embedder iface.Embedder, ctx context.Context, documents []string) {
	b.ResetTimer()
	b.ReportAllocs()

	totalTokens := 0
	for _, doc := range documents {
		// Rough token estimation (words / 0.75 for English text)
		totalTokens += len(strings.Fields(doc))
	}

	start := time.Now()
	operations := 0

	for i := 0; i < b.N; i++ {
		for _, doc := range documents {
			_, err := embedder.EmbedQuery(ctx, doc)
			if err != nil {
				b.Fatalf("Embedding failed: %v", err)
			}
			operations++
		}
	}

	duration := time.Since(start)
	b.ReportMetric(float64(operations)/duration.Seconds(), "ops/sec")
	b.ReportMetric(float64(totalTokens*b.N)/duration.Seconds(), "tokens/sec")
}

// Load testing benchmarks - simulate realistic concurrent user patterns

func BenchmarkLoadTest_ConcurrentUsers(b *testing.B) {
	config := &Config{
		Mock: &MockConfig{
			Dimension: 128,
			Seed:      42,
			Enabled:   true,
		},
	}

	factory, err := NewEmbedderFactory(config)
	if err != nil {
		b.Fatalf("Failed to create factory: %v", err)
	}

	ctx := context.Background()
	testDocuments := []string{
		"This is a short document for testing.",
		"This is a longer document that contains more text and should be processed correctly by the embedding system.",
		"A very short doc.",
		"This document contains multiple sentences. It has more content than the short ones. This should provide a good test case for the embedding algorithm.",
		"Single sentence document.",
	}

	// Test different concurrency levels
	concurrencyLevels := []int{1, 5, 10, 20}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(b *testing.B) {
			benchmarkConcurrentLoad(b, factory, ctx, testDocuments, concurrency)
		})
	}
}

func benchmarkConcurrentLoad(b *testing.B, factory *EmbedderFactory, ctx context.Context, documents []string, concurrency int) {
	b.ResetTimer()
	b.ReportAllocs()

	totalOperations := int64(0)
	totalErrors := int64(0)

	// Use a mutex for thread-safe updates to shared counters
	var mu sync.Mutex

	b.RunParallel(func(pb *testing.PB) {
		// Each goroutine gets its own embedder instance
		embedder, err := factory.NewEmbedder("mock")
		if err != nil {
			b.Errorf("Failed to create embedder: %v", err)
			return
		}

		localOps := int64(0)
		localErrors := int64(0)

		for pb.Next() {
			// Cycle through documents to simulate load
			docIndex := int(localOps) % len(documents)
			doc := documents[docIndex]

			_, err := embedder.EmbedQuery(ctx, doc)
			if err != nil {
				localErrors++
			}
			localOps++
		}

		// Update shared counters with mutex protection
		mu.Lock()
		totalOperations += localOps
		totalErrors += localErrors
		mu.Unlock()
	})

	b.ReportMetric(float64(totalOperations)/b.Elapsed().Seconds(), "ops/sec")
	b.ReportMetric(float64(totalErrors), "errors")
}

func BenchmarkLoadTest_SustainedLoad(b *testing.B) {
	config := &Config{
		Mock: &MockConfig{
			Dimension: 128,
			Seed:      42,
			Enabled:   true,
		},
	}

	factory, err := NewEmbedderFactory(config)
	if err != nil {
		b.Fatalf("Failed to create factory: %v", err)
	}

	ctx := context.Background()

	// Generate a large set of test documents
	testDocuments := make([]string, 1000)
	for i := range testDocuments {
		testDocuments[i] = fmt.Sprintf("Test document number %d with some content for load testing purposes.", i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	embedder, err := factory.NewEmbedder("mock")
	if err != nil {
		b.Fatalf("Failed to create embedder: %v", err)
	}

	operations := 0
	start := time.Now()

	for i := 0; i < b.N; i++ {
		// Cycle through documents to simulate sustained load
		docIndex := i % len(testDocuments)
		doc := testDocuments[docIndex]

		_, err := embedder.EmbedQuery(ctx, doc)
		if err != nil {
			b.Fatalf("Embedding failed: %v", err)
		}
		operations++
	}

	duration := time.Since(start)
	b.ReportMetric(float64(operations)/duration.Seconds(), "ops/sec")
	b.ReportMetric(duration.Seconds()/float64(operations)*1000, "ms/op")
}

func BenchmarkLoadTest_BurstTraffic(b *testing.B) {
	config := &Config{
		Mock: &MockConfig{
			Dimension: 128,
			Seed:      42,
			Enabled:   true,
		},
	}

	factory, err := NewEmbedderFactory(config)
	if err != nil {
		b.Fatalf("Failed to create factory: %v", err)
	}

	ctx := context.Background()
	testDocuments := []string{
		"Short burst request 1.",
		"Short burst request 2.",
		"Short burst request 3.",
		"Short burst request 4.",
		"Short burst request 5.",
	}

	// Simulate burst traffic patterns
	burstSizes := []int{10, 50, 100}

	for _, burstSize := range burstSizes {
		b.Run(fmt.Sprintf("Burst_%d", burstSize), func(b *testing.B) {
			benchmarkBurstLoad(b, factory, ctx, testDocuments, burstSize)
		})
	}
}

func benchmarkBurstLoad(b *testing.B, factory *EmbedderFactory, ctx context.Context, documents []string, burstSize int) {
	b.ResetTimer()
	b.ReportAllocs()

	embedder, err := factory.NewEmbedder("mock")
	if err != nil {
		b.Fatalf("Failed to create embedder: %v", err)
	}

	operations := 0

	for i := 0; i < b.N; i++ {
		// Simulate burst: process multiple documents in quick succession
		for j := 0; j < burstSize; j++ {
			docIndex := j % len(documents)
			doc := documents[docIndex]

			_, err := embedder.EmbedQuery(ctx, doc)
			if err != nil {
				b.Fatalf("Embedding failed: %v", err)
			}
			operations++
		}
	}

	b.ReportMetric(float64(operations)/b.Elapsed().Seconds(), "ops/sec")
	b.ReportMetric(float64(burstSize), "burst_size")
}

// Performance regression detection benchmarks

func BenchmarkPerformanceRegressionDetection(b *testing.B) {
	config := &Config{
		Mock: &MockConfig{
			Dimension: 128,
			Seed:      42,
			Enabled:   true,
		},
	}

	factory, err := NewEmbedderFactory(config)
	if err != nil {
		b.Fatalf("Failed to create factory: %v", err)
	}

	ctx := context.Background()
	embedder, err := factory.NewEmbedder("mock")
	if err != nil {
		b.Fatalf("Failed to create embedder: %v", err)
	}

	// Performance baselines (operations per second)
	baselines := map[string]float64{
		"single_query":     10000, // 10k ops/sec baseline
		"batch_documents":  5000,  // 5k ops/sec baseline
		"mixed_operations": 8000,  // 8k ops/sec baseline
	}

	b.Run("SingleQueryRegression", func(b *testing.B) {
		testDoc := "This is a test document for performance regression detection."

		b.ResetTimer()
		b.ReportAllocs()

		operations := 0
		start := time.Now()

		for i := 0; i < b.N; i++ {
			_, err := embedder.EmbedQuery(ctx, testDoc)
			if err != nil {
				b.Fatalf("Embedding failed: %v", err)
			}
			operations++
		}

		duration := time.Since(start)
		opsPerSec := float64(operations) / duration.Seconds()

		b.ReportMetric(opsPerSec, "ops/sec")

		// Check for performance regression
		if baseline, exists := baselines["single_query"]; exists && opsPerSec < baseline*0.8 {
			b.Errorf("PERFORMANCE REGRESSION: ops/sec (%.0f) is 20%% below baseline (%.0f)",
				opsPerSec, baseline)
		}
	})

	b.Run("BatchDocumentsRegression", func(b *testing.B) {
		documents := []string{
			"First test document for batch processing.",
			"Second document with different content.",
			"Third document to test batch performance.",
			"Fourth document in the batch test.",
			"Fifth and final document for regression testing.",
		}

		b.ResetTimer()
		b.ReportAllocs()

		operations := 0
		totalDocuments := 0
		start := time.Now()

		for i := 0; i < b.N; i++ {
			_, err := embedder.EmbedDocuments(ctx, documents)
			if err != nil {
				b.Fatalf("Batch embedding failed: %v", err)
			}
			operations++
			totalDocuments += len(documents)
		}

		duration := time.Since(start)
		docsPerSec := float64(totalDocuments) / duration.Seconds()

		b.ReportMetric(docsPerSec, "docs/sec")

		// Check for performance regression
		if baseline, exists := baselines["batch_documents"]; exists && docsPerSec < baseline*0.8 {
			b.Errorf("PERFORMANCE REGRESSION: docs/sec (%.0f) is 20%% below baseline (%.0f)",
				docsPerSec, baseline)
		}
	})

	b.Run("MemoryRegressionDetection", func(b *testing.B) {
		testDoc := "Memory usage test document for regression detection."

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := embedder.EmbedQuery(ctx, testDoc)
			if err != nil {
				b.Fatalf("Embedding failed: %v", err)
			}
		}

		// Memory regression detection is handled by b.ReportAllocs()
		// Significant increases in allocations would indicate memory regression
	})

	b.Run("ConcurrentRegressionDetection", func(b *testing.B) {
		// Test concurrent performance regression
		numWorkers := 10
		operationsPerWorker := 100

		b.ResetTimer()

		var wg sync.WaitGroup
		errorCount := int64(0)

		for w := 0; w < numWorkers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < operationsPerWorker; i++ {
					doc := fmt.Sprintf("Concurrent test document %d", i)
					_, err := embedder.EmbedQuery(ctx, doc)
					if err != nil {
						atomic.AddInt64(&errorCount, 1)
					}
				}
			}()
		}

		wg.Wait()

		totalOperations := numWorkers * operationsPerWorker
		opsPerSec := float64(totalOperations) / b.Elapsed().Seconds()

		b.ReportMetric(opsPerSec, "ops/sec")
		b.ReportMetric(float64(errorCount), "errors")

		// Check for concurrent performance regression
		if baseline, exists := baselines["mixed_operations"]; exists && opsPerSec < baseline*0.7 {
			b.Errorf("CONCURRENT PERFORMANCE REGRESSION: ops/sec (%.0f) is 30%% below baseline (%.0f)",
				opsPerSec, baseline)
		}
	})
}

// Helper function for atomic operations (since Go's atomic doesn't have AddInt64)
func atomicAddInt64(addr *int64, delta int64) {
	// Simple implementation - in real concurrent code, use sync/atomic
	*addr += delta
}
