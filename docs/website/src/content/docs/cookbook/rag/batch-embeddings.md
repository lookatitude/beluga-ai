---
title: "Batch Embedding Optimization"
description: "Optimize embedding operations with intelligent batching, concurrency control, and rate limiting."
---

## Problem

You need to embed large numbers of documents efficiently, but making individual API calls for each document is slow and expensive. You want to batch embeddings to reduce API calls and improve throughput.

## Solution

Implement intelligent batching that groups documents into optimal batch sizes, handles rate limits, and processes batches concurrently. Most embedding providers support batch operations, and batching reduces API overhead while staying within provider limits.

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

1. **Intelligent batching** — Documents are split into optimal batch sizes. Most providers have limits (e.g., 100 documents per batch), so the batcher respects those limits while maximizing throughput.

2. **Concurrent processing** — Multiple batches are processed concurrently using a semaphore to limit parallelism. This prevents overwhelming the API while still utilizing available bandwidth.

3. **Rate limiting** — A token bucket rate limiter respects API rate limits. This prevents hitting rate limit errors while maintaining good throughput.

4. **Result ordering** — Despite concurrent processing, results maintain their original ordering by writing directly to indexed positions in the results slice.

## Variations

### Adaptive Batch Sizing

Adjust batch size based on document length:

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

- [Metadata-Aware Clustering](/cookbook/metadata-clustering) — Cluster embeddings with metadata constraints
- [Advanced Metadata Filtering](/cookbook/meta-filtering) — Filter vector store results with metadata
