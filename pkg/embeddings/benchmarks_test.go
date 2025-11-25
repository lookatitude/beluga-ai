package embeddings

import (
	"context"
	"fmt"
	"strings"
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
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
