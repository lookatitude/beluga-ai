---
title: "Parallel File Loading"
description: "Efficiently load documents from large directory structures using parallel file walking with worker pools."
---

## Problem

You need to load documents from large directory structures efficiently, but sequential file walking is too slow for directories with thousands of files.

## Solution

Implement parallel file walking that uses multiple goroutines to traverse directories concurrently, processes files in parallel, and collects results efficiently. This works because file system operations can be parallelized, and worker pools balance concurrency with resource usage.

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

1. **Worker pool pattern** — A pool of worker goroutines pulls file paths from a channel and processes them in parallel. This balances concurrency with resource usage, preventing file descriptor exhaustion.

2. **Separate discovery and processing** — File discovery (walking the directory tree) runs in a separate goroutine from file processing. This allows both to happen concurrently, maximizing throughput on large directory trees.

3. **Error isolation** — Errors from individual files do not stop the entire walk. Errors are collected separately and returned alongside successfully loaded documents, allowing callers to decide how to handle partial failures.

4. **Context cancellation** — All goroutines respect context cancellation. If the parent context is cancelled (e.g., timeout), workers and discovery stop promptly.

## Variations

### Rate Limiting

Add rate limiting to prevent overwhelming the file system:

```go
type RateLimitedWalker struct {
    rateLimiter *rate.Limiter
}
```

### Progress Tracking

Track progress of file processing:

```go
type ProgressTracker struct {
    processed int64
    total     int64
}
```

## Related Recipes

- [Corrupt Document Handling](/cookbook/corrupt-doc-handling) — Handle errors gracefully during document loading
- [Document Ingestion Recipes](/cookbook/document-ingestion) — Additional document loading patterns
