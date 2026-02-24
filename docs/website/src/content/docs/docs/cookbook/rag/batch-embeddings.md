---
title: "Batch Embedding Optimization"
description: "Recipe for optimizing embedding throughput in Go with intelligent batching, concurrency control, and rate limiting to stay within provider limits."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, batch embeddings, Go embedding optimization, concurrency control, rate limiting, embedding throughput, RAG recipe"
---

## Problem

You need to embed large numbers of documents efficiently, but making individual API calls for each document is slow and expensive. You want to batch embeddings to reduce API calls and improve throughput.

Embedding APIs have significant per-request overhead: TLS handshake, request serialization, server-side batch setup, and response deserialization. When embedding 10,000 documents one at a time, this overhead dominates total latency -- often 50-100ms per call regardless of input size. Batching amortizes this overhead across many inputs, reducing total wall-clock time by an order of magnitude. Additionally, embedding providers often price by token count rather than API calls, so batching doesn't increase cost but dramatically improves throughput.

## Solution

Implement intelligent batching that groups documents into optimal batch sizes, handles rate limits, and processes batches concurrently. Most embedding providers support batch operations (e.g., OpenAI allows up to 2048 inputs per batch), and batching reduces API overhead while staying within provider limits. The combination of batching and concurrent processing creates a pipeline that saturates available bandwidth without overwhelming the provider.

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

    "github.com/lookatitude/beluga-ai/rag/embedding"
    "github.com/lookatitude/beluga-ai/schema"
)

var tracer = otel.Tracer("beluga.embeddings.batch")

// BatchEmbedder optimizes embedding operations with batching.
type BatchEmbedder struct {
    embedder      embedding.Embedder
    batchSize     int
    maxConcurrent int
    rateLimiter   *RateLimiter
}

// RateLimiter limits API call rate using a token bucket.
type RateLimiter struct {
    tokens     chan struct{}
    refillRate time.Duration
}

// NewRateLimiter creates a rate limiter.
func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
    rl := &RateLimiter{
        tokens:     make(chan struct{}, maxTokens),
        refillRate: refillRate,
    }

    for i := 0; i < maxTokens; i++ {
        rl.tokens <- struct{}{}
    }

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

// NewBatchEmbedder creates a new batch embedder.
func NewBatchEmbedder(embedder embedding.Embedder, batchSize, maxConcurrent int) *BatchEmbedder {
    return &BatchEmbedder{
        embedder:      embedder,
        batchSize:     batchSize,
        maxConcurrent: maxConcurrent,
        rateLimiter:   NewRateLimiter(maxConcurrent, 100*time.Millisecond),
    }
}

// EmbedDocumentsBatch embeds documents in optimized batches.
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

            semaphore <- struct{}{}
            defer func() { <-semaphore }()

            if err := be.rateLimiter.Wait(ctx); err != nil {
                errCh <- err
                return
            }

            batchEmbeddings, err := be.embedBatch(ctx, batchTexts)
            if err != nil {
                errCh <- err
                return
            }

            for j, emb := range batchEmbeddings {
                results[startIdx+j] = emb
            }
        }(i, batch, i*be.batchSize)
    }

    wg.Wait()
    close(errCh)

    if len(errCh) > 0 {
        err := <-errCh
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return nil, fmt.Errorf("batch embedding failed: %w", err)
    }

    span.SetStatus(trace.StatusOK, "all batches embedded successfully")
    return results, nil
}

// createBatches splits texts into batches.
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

// embedBatch embeds a single batch.
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

    // embedder := your embedding.Embedder instance
    batchEmbedder := NewBatchEmbedder(embedder, 100, 5)

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

1. **Intelligent batching** -- Documents are split into optimal batch sizes matching provider limits. Most providers have per-request limits (e.g., 100 documents per batch for OpenAI), so the batcher respects those limits while maximizing throughput per API call. This reduces total API calls from N to N/batch_size.

2. **Concurrent processing** -- Multiple batches are processed concurrently using a semaphore to limit parallelism. The semaphore pattern prevents overwhelming the API with too many simultaneous requests while still utilizing available bandwidth. The concurrency level should match the provider's rate limit headroom.

3. **Rate limiting** -- A token bucket rate limiter respects API rate limits by controlling how frequently new requests can be issued. The refill rate determines sustained throughput, while the bucket size allows short bursts. This prevents 429 (rate limit) errors while maintaining good throughput.

4. **Result ordering** -- Despite concurrent processing, results maintain their original ordering by writing directly to indexed positions in the results slice. Each goroutine writes to a disjoint range of the output slice, so no synchronization is needed for writes. This avoids a post-processing sort step.

## Variations

### Adaptive Batch Sizing

Adjust batch size based on document length to stay within token limits:

```go
func (be *BatchEmbedder) calculateOptimalBatchSize(texts []string) int {
    // Calculate based on total character count to stay within token limits
}
```

### Retry on Batch Failure

Retry failed batches with exponential backoff:

```go
func (be *BatchEmbedder) embedBatchWithRetry(ctx context.Context, texts []string) ([][]float32, error) {
    // Retry logic with exponential backoff
}
```

## Related Recipes

- [Metadata-Aware Clustering](/docs/cookbook/metadata-clustering) -- Cluster embeddings with metadata constraints
- [Advanced Metadata Filtering](/docs/cookbook/meta-filtering) -- Filter vector store results with metadata
