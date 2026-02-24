---
title: "Parallel File Loading"
description: "Recipe for loading documents from large directories in Go using parallel worker pools with bounded concurrency and file descriptor management."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, parallel file loading, Go worker pool, concurrent document loading, bounded concurrency, RAG performance, file walking"
---

## Problem

You need to load documents from large directory structures efficiently, but sequential file walking is too slow for directories with thousands of files.

Sequential file loading is strictly I/O-bound: the CPU sits idle while waiting for each file read to complete. For a directory with 10,000 files where each read takes 5ms, sequential loading takes 50 seconds. Parallel loading with 10 workers reduces this to roughly 5 seconds. The key challenge is balancing concurrency against resource limits: too many concurrent reads can exhaust file descriptors (typically limited to 1024 on Linux) or saturate disk I/O bandwidth on spinning disks.

## Solution

Implement parallel file walking that uses a producer-consumer pattern: one goroutine discovers files by walking the directory tree, and a pool of worker goroutines processes files concurrently. The worker pool size controls parallelism, and channels coordinate between discovery and processing. This architecture separates concerns (discovery vs. processing) and scales well because adding workers increases throughput linearly until the I/O subsystem saturates.

## Code Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "sync"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"

    "github.com/lookatitude/beluga-ai/schema"
)

var tracer = otel.Tracer("beluga.documentloaders.parallel_walker")

// ParallelFileWalker walks directories in parallel.
type ParallelFileWalker struct {
    workers    int
    extensions []string
    maxDepth   int
    fileCh     chan string
    resultCh   chan schema.Document
    errorCh    chan error
}

// NewParallelFileWalker creates a new parallel walker.
func NewParallelFileWalker(workers int, extensions []string, maxDepth int) *ParallelFileWalker {
    return &ParallelFileWalker{
        workers:    workers,
        extensions: extensions,
        maxDepth:   maxDepth,
        fileCh:     make(chan string, workers*2),
        resultCh:   make(chan schema.Document, workers*2),
        errorCh:    make(chan error, workers),
    }
}

// Walk walks a directory in parallel and returns loaded documents and any errors.
func (pfw *ParallelFileWalker) Walk(ctx context.Context, rootPath string) ([]schema.Document, []error, error) {
    ctx, span := tracer.Start(ctx, "parallel_walker.walk")
    defer span.End()

    span.SetAttributes(
        attribute.String("root_path", rootPath),
        attribute.Int("workers", pfw.workers),
        attribute.Int("max_depth", pfw.maxDepth),
    )

    var wg sync.WaitGroup
    documents := []schema.Document{}
    errors := []error{}

    // Start worker goroutines
    for i := 0; i < pfw.workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            pfw.worker(ctx)
        }()
    }

    // Start file discovery
    discoveryWg := sync.WaitGroup{}
    discoveryWg.Add(1)
    go func() {
        defer discoveryWg.Done()
        defer close(pfw.fileCh)
        pfw.discoverFiles(ctx, rootPath)
    }()

    // Collect results
    collectWg := sync.WaitGroup{}
    collectWg.Add(1)
    go func() {
        defer collectWg.Done()
        for {
            select {
            case doc, ok := <-pfw.resultCh:
                if !ok {
                    return
                }
                documents = append(documents, doc)
            case err, ok := <-pfw.errorCh:
                if !ok {
                    return
                }
                errors = append(errors, err)
            case <-ctx.Done():
                return
            }
        }
    }()

    // Wait for workers, then close result channels
    wg.Wait()
    close(pfw.resultCh)
    close(pfw.errorCh)

    // Wait for collection
    collectWg.Wait()

    span.SetAttributes(
        attribute.Int("document_count", len(documents)),
        attribute.Int("error_count", len(errors)),
    )

    return documents, errors, nil
}

// discoverFiles discovers files and sends paths to the file channel.
func (pfw *ParallelFileWalker) discoverFiles(ctx context.Context, rootPath string) {
    err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            pfw.errorCh <- err
            return nil // Continue walking
        }

        // Check depth
        if pfw.maxDepth > 0 {
            depth := pfw.getDepth(rootPath, path)
            if depth > pfw.maxDepth {
                if info.IsDir() {
                    return filepath.SkipDir
                }
                return nil
            }
        }

        // Check if file matches extensions
        if !info.IsDir() && pfw.matchesExtension(path) {
            select {
            case pfw.fileCh <- path:
            case <-ctx.Done():
                return ctx.Err()
            }
        }

        return nil
    })

    if err != nil {
        pfw.errorCh <- err
    }
}

// worker processes files from the file channel.
func (pfw *ParallelFileWalker) worker(ctx context.Context) {
    for {
        select {
        case filePath, ok := <-pfw.fileCh:
            if !ok {
                return
            }

            doc, err := pfw.loadDocument(ctx, filePath)
            if err != nil {
                pfw.errorCh <- fmt.Errorf("failed to load %s: %w", filePath, err)
                continue
            }

            select {
            case pfw.resultCh <- doc:
            case <-ctx.Done():
                return
            }

        case <-ctx.Done():
            return
        }
    }
}

// loadDocument loads a single document from disk.
func (pfw *ParallelFileWalker) loadDocument(ctx context.Context, filePath string) (schema.Document, error) {
    data, err := os.ReadFile(filePath)
    if err != nil {
        return nil, err
    }

    return schema.NewDocument(string(data), map[string]string{
        "source": filePath,
    }), nil
}

// matchesExtension checks if a file matches allowed extensions.
func (pfw *ParallelFileWalker) matchesExtension(path string) bool {
    if len(pfw.extensions) == 0 {
        return true
    }

    ext := filepath.Ext(path)
    for _, allowedExt := range pfw.extensions {
        if ext == allowedExt || ext == "."+allowedExt {
            return true
        }
    }
    return false
}

// getDepth calculates directory depth relative to root.
func (pfw *ParallelFileWalker) getDepth(rootPath, path string) int {
    rel, _ := filepath.Rel(rootPath, path)
    return len(filepath.SplitList(rel))
}

func main() {
    ctx := context.Background()

    walker := NewParallelFileWalker(10, []string{".md", ".txt"}, 3)

    docs, errors, err := walker.Walk(ctx, "./docs")
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }
    fmt.Printf("Loaded %d documents, %d errors\n", len(docs), len(errors))
}
```

## Explanation

1. **Worker pool pattern** -- A pool of worker goroutines pulls file paths from a shared channel and processes them in parallel. The pool size directly controls concurrency: 10 workers means at most 10 files are being read simultaneously. This balances throughput against resource usage, preventing file descriptor exhaustion that would occur with unbounded goroutine creation.

2. **Separate discovery and processing** -- File discovery (walking the directory tree) runs in a dedicated goroutine, feeding paths into a channel that workers consume. This decouples the speed of directory traversal from file processing. The buffered channel (`workers*2` capacity) provides backpressure: if workers are busy, discovery naturally slows down rather than accumulating unbounded work.

3. **Error isolation** -- Errors from individual files do not stop the entire walk. Each error is collected separately via the error channel and returned alongside successfully loaded documents. This allows callers to decide how to handle partial failures (e.g., log and continue, or retry specific files).

4. **Context cancellation** -- All goroutines check `ctx.Done()` in their select statements. If the parent context is cancelled (e.g., timeout, user cancellation), workers, discovery, and collection all stop promptly without leaving orphaned goroutines.

## Variations

### Rate Limiting

Add rate limiting to prevent overwhelming the file system on shared infrastructure:

```go
type RateLimitedWalker struct {
    rateLimiter *rate.Limiter
}
```

### Progress Tracking

Track progress of file processing for user feedback:

```go
type ProgressTracker struct {
    processed int64
    total     int64
}
```

## Related Recipes

- [Corrupt Document Handling](/docs/cookbook/corrupt-doc-handling) -- Handle errors gracefully during document loading
- [Document Ingestion Recipes](/docs/cookbook/document-ingestion) -- Additional document loading patterns
