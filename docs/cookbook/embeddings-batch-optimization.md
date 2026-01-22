---
title: "Batch Embedding Optimization"
package: "embeddings"
category: "optimization"
complexity: "intermediate"
---

# Batch Embedding Optimization

## Problem

You need to embed large numbers of documents efficiently, but making individual API calls for each document is slow and expensive. You want to batch embeddings to reduce API calls and improve throughput.

## Solution

Implement intelligent batching that groups documents into optimal batch sizes, handles rate limits, and processes batches concurrently. This works because most embedding providers support batch operations, and batching reduces API overhead while staying within provider limits.

## Code Example
```go
package main

import (
    "context"
    "fmt"
    "log"
    "sync"
    "time"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
    
    "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

var tracer = otel.Tracer("beluga.embeddings.batch")

// BatchEmbedder optimizes embedding operations with batching
type BatchEmbedder struct {
    embedder      iface.Embedder
    batchSize     int
    maxConcurrent int
    rateLimiter   *RateLimiter
}

// RateLimiter limits API call rate
type RateLimiter struct {
    tokens    chan struct{}
    refillRate time.Duration
}

// NewRateLimiter creates a rate limiter
func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
    rl := &RateLimiter{
        tokens:    make(chan struct{}, maxTokens),
        refillRate: refillRate,
    }

    // Fill initial tokens
    for i := 0; i < maxTokens; i++ {
        rl.tokens <- struct{}{}
    }
    
    // Start refill goroutine
    go rl.refill()
    
    return rl
}

func (rl *RateLimiter) refill() {
    ticker := time.NewTicker(rl.refillRate)
    for range ticker.C {
        select {
        case rl.tokens <- struct{}{}:
        default:
        }
    }
}

func (rl *RateLimiter) Wait(ctx context.Context) error {
    select {
    case <-rl.tokens:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

// NewBatchEmbedder creates a new batch embedder
func NewBatchEmbedder(embedder iface.Embedder, batchSize, maxConcurrent int) *BatchEmbedder {
    return &BatchEmbedder{
        embedder:      embedder,
        batchSize:     batchSize,
        maxConcurrent: maxConcurrent,
        rateLimiter:   NewRateLimiter(maxConcurrent, 100*time.Millisecond),
    }
}

// EmbedDocumentsBatch embeds documents in optimized batches
func (be *BatchEmbedder) EmbedDocumentsBatch(ctx context.Context, documents []schema.Document) ([][]float32, error) {
    ctx, span := tracer.Start(ctx, "batch_embedder.embed_documents")
    defer span.End()
    
    span.SetAttributes(
        attribute.Int("document_count", len(documents)),
        attribute.Int("batch_size", be.batchSize),
    )
    
    // Extract texts
    texts := make([]string, len(documents))
    for i, doc := range documents {
        texts[i] = doc.GetContent()
    }
    
    // Create batches
    batches := be.createBatches(texts)
    span.SetAttributes(attribute.Int("batch_count", len(batches)))
    
    // Process batches concurrently
    results := make([][]float32, len(texts))
    var wg sync.WaitGroup
    semaphore := make(chan struct{}, be.maxConcurrent)
    errCh := make(chan error, len(batches))

    for i, batch := range batches {
        wg.Add(1)
        go func(batchIndex int, batchTexts []string, startIdx int) {
            defer wg.Done()
            
            // Acquire semaphore
            semaphore <- struct{}{}
            defer func() { <-semaphore }()
            
            // Wait for rate limit
            if err := be.rateLimiter.Wait(ctx); err != nil {
                errCh <- err
                return
            }
            
            // Embed batch
            batchEmbeddings, err := be.embedBatch(ctx, batchTexts)
            if err != nil {
                errCh <- err
                return
            }
            
            // Store results
            for j, embedding := range batchEmbeddings {
                results[startIdx+j] = embedding
            }
        }(i, batch, i*be.batchSize)
    }
    
    wg.Wait()
    close(errCh)
    
    // Check for errors
    if len(errCh) > 0 {
        err := <-errCh
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return nil, fmt.Errorf("batch embedding failed: %w", err)
    }
    
    span.SetStatus(trace.StatusOK, "all batches embedded successfully")
    return results, nil
}

// createBatches splits texts into batches
func (be *BatchEmbedder) createBatches(texts []string) [][]string {
    batches := make([][]string, 0)
    for i := 0; i < len(texts); i += be.batchSize {
        end := i + be.batchSize
        if end > len(texts) {
            end = len(texts)
        }
        batches = append(batches, texts[i:end])
    }
    return batches
}

// embedBatch embeds a single batch
func (be *BatchEmbedder) embedBatch(ctx context.Context, texts []string) ([][]float32, error) {
    ctx, span := tracer.Start(ctx, "batch_embedder.embed_batch")
    defer span.End()
    
    span.SetAttributes(attribute.Int("batch_text_count", len(texts)))
    
    embeddings, err := be.embedder.EmbedDocuments(ctx, texts)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return nil, err
    }
    
    span.SetStatus(trace.StatusOK, "batch embedded")
    return embeddings, nil
}

func main() {
    ctx := context.Background()

    // Create embedder
    // embedder := yourEmbedder
    
    // Create batch embedder
    batchEmbedder := NewBatchEmbedder(embedder, 100, 5) // Batch size 100, 5 concurrent
    
    // Embed documents
    documents := []schema.Document{
        schema.NewDocument("Document 1", nil),
        schema.NewDocument("Document 2", nil),
        // ... more documents
    }
    
    embeddings, err := batchEmbedder.EmbedDocumentsBatch(ctx, documents)
    if err != nil {
        log.Fatalf("Failed to embed: %v", err)
    }
    fmt.Printf("Embedded %d documents\n", len(embeddings))
}
```

## Explanation

Let's break down what's happening:

1. **Intelligent batching** - Notice how we split documents into optimal batch sizes. Most providers have limits (e.g., 100 documents per batch), so we respect those limits while maximizing throughput.

2. **Concurrent processing** - We process multiple batches concurrently using a semaphore to limit concurrency. This prevents overwhelming the API while still utilizing parallelism.

3. **Rate limiting** - We use a token bucket rate limiter to respect API rate limits. This prevents hitting rate limit errors while maintaining good throughput.

```go
**Key insight:** Balance batch size, concurrency, and rate limits. Too large batches hit provider limits, too much concurrency hits rate limits, and too small batches waste API calls.

## Testing

```
Here's how to test this solution:
```go
func TestBatchEmbedder_ProcessesBatches(t *testing.T) {
    mockEmbedder := &MockEmbedder{}
    batchEmbedder := NewBatchEmbedder(mockEmbedder, 10, 2)
    
    documents := make([]schema.Document, 25)
    for i := 0; i < 25; i++ {
        documents[i] = schema.NewDocument(fmt.Sprintf("doc %d", i), nil)
    }
    
    embeddings, err := batchEmbedder.EmbedDocumentsBatch(context.Background(), documents)
    require.NoError(t, err)
    require.Len(t, embeddings, 25)
}

## Variations

### Adaptive Batch Sizing

Adjust batch size based on document length:
func (be *BatchEmbedder) calculateOptimalBatchSize(texts []string) int {
    // Calculate based on total character count
}
```

### Retry on Batch Failure

Retry failed batches with exponential backoff:
```go
func (be *BatchEmbedder) embedBatchWithRetry(ctx context.Context, texts []string) ([][]float32, error) {
    // Retry logic
}
```

## Related Recipes

- **[Embeddings Metadata-aware Embedding Clusters](./embeddings-metadata-aware-clusters.md)** - Cluster embeddings with metadata
- **[Vectorstores Advanced Meta-filtering](./vectorstores-advanced-meta-filtering.md)** - Filter with metadata
- **[Embeddings Package Guide](../package_design_patterns.md)** - For a deeper understanding of embeddings
