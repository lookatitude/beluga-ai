---
title: "Parallel File Walkers"
package: "documentloaders"
category: "performance"
complexity: "intermediate"
---

# Parallel File Walkers

## Problem

You need to load documents from large directory structures efficiently, but sequential file walking is too slow for directories with thousands of files.

## Solution

Implement parallel file walking that uses multiple goroutines to traverse directories concurrently, processes files in parallel, and collects results efficiently. This works because file system operations can be parallelized, and you can use worker pools to process files concurrently.

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
    "go.opentelemetry.io/otel/trace"
    
    "github.com/lookatitude/beluga-ai/pkg/documentloaders"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

var tracer = otel.Tracer("beluga.documentloaders.parallel_walker")

// ParallelFileWalker walks directories in parallel
type ParallelFileWalker struct {
    workers      int
    extensions   []string
    maxDepth     int
    fileCh       chan string
    resultCh     chan schema.Document
    errorCh      chan error
}

// NewParallelFileWalker creates a new parallel walker
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

// Walk walks directory in parallel
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
    
    // Wait for workers
    wg.Wait()
    close(pfw.resultCh)
    close(pfw.errorCh)
    
    // Wait for collection
    collectWg.Wait()
    
    span.SetAttributes(
        attribute.Int("document_count", len(documents)),
        attribute.Int("error_count", len(errors)),
    )
    span.SetStatus(trace.StatusOK, "parallel walk completed")
    
    return documents, errors, nil
}

// discoverFiles discovers files and sends to file channel
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

// worker processes files from file channel
func (pfw *ParallelFileWalker) worker(ctx context.Context) {
    for {
        select {
        case filePath, ok := <-pfw.fileCh:
            if !ok {
                return
            }
            
            // Load document
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

// loadDocument loads a single document
func (pfw *ParallelFileWalker) loadDocument(ctx context.Context, filePath string) (schema.Document, error) {
    data, err := os.ReadFile(filePath)
    if err != nil {
        return nil, err
    }
    
    return schema.NewDocument(string(data), map[string]string{
        "source": filePath,
    }), nil
}

// matchesExtension checks if file matches allowed extensions
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

// getDepth calculates directory depth
func (pfw *ParallelFileWalker) getDepth(rootPath, path string) int {
    rel, _ := filepath.Rel(rootPath, path)
    return len(filepath.SplitList(rel))
}

func main() {
    ctx := context.Background()

    // Create parallel walker
    walker := NewParallelFileWalker(10, []string{".md", ".txt"}, 3)
    
    // Walk directory
    docs, errors, err := walker.Walk(ctx, "./docs")
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }
    fmt.Printf("Loaded %d documents, %d errors\n", len(docs), len(errors))
}
```

## Explanation

Let's break down what's happening:

1. **Worker pool pattern** - Notice how we use a worker pool to process files concurrently. Workers pull files from a channel and process them in parallel.

2. **Separate discovery and processing** - We separate file discovery (walking) from file processing. This allows both to happen concurrently, maximizing throughput.

3. **Error isolation** - Errors from individual files don't stop the entire walk. We collect errors separately and continue processing other files.

```go
**Key insight:** Parallelize file discovery and processing. Use worker pools to balance parallelism with resource usage.

## Testing

```
Here's how to test this solution:
```go
func TestParallelFileWalker_WalksDirectory(t *testing.T) {
    walker := NewParallelFileWalker(5, []string{".txt"}, 2)
    
    docs, errors, err := walker.Walk(context.Background(), "./testdata")
    require.NoError(t, err)
    require.Greater(t, len(docs), 0)
}

## Variations

### Rate Limiting

Add rate limiting to prevent overwhelming the file system:
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

- **[Documentloaders Robust Error Handling](./documentloaders-robust-error-handling-corrupt-docs.md)** - Handle errors gracefully
- **[Document Ingestion Recipes](./document-ingestion-recipes.md)** - Additional document loading patterns
- **[Documentloaders Package Guide](../package_design_patterns.md)** - For a deeper understanding of document loaders
