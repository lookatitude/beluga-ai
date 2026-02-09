---
title: Lazy-Loading Large Document Collections
description: Process massive document collections with constant memory usage using the RAG loader pipeline and streaming patterns.
---

Loading 10,000 documents into a single Go slice is manageable. Loading a million documents at once will exhaust memory. The `rag/loader` package supports pipeline-based document loading with transformers that process documents incrementally, keeping memory usage flat regardless of collection size.

## What You Will Build

A document ingestion pipeline that loads files from a directory, applies metadata enrichment and content transformations, and streams results to a vector store. You will compare eager loading versus pipeline-based loading and implement parallel processing with rate limiting.

## Prerequisites

- Familiarity with the `schema.Document` type
- Basic understanding of the RAG pipeline

## Core Concepts

### DocumentLoader Interface

Every loader implements the `DocumentLoader` interface:

```go
import "github.com/lookatitude/beluga-ai/rag/loader"

type DocumentLoader interface {
    Load(ctx context.Context, source string) ([]schema.Document, error)
}
```

### LoaderPipeline

The `LoaderPipeline` chains loaders and transformers. Each transformer processes documents incrementally:

```go
pipeline := loader.NewPipeline(
    loader.WithLoader(textLoader),
    loader.WithTransformer(metadataEnricher),
)
```

### Document Structure

Documents carry content, metadata, and optional embeddings:

```go
import "github.com/lookatitude/beluga-ai/schema"

doc := schema.Document{
    ID:       "doc-001",
    Content:  "Document text content...",
    Metadata: map[string]any{"source": "report.txt", "author": "team"},
}
```

## Step 1: Basic Document Loading

Load documents using a registered loader:

```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/rag/loader"

    // Register built-in loaders.
    _ "github.com/lookatitude/beluga-ai/rag/loader" // registers "text", "json", "csv", "markdown"
)

func main() {
    ctx := context.Background()

    // Create a text loader via the registry.
    textLoader, err := loader.New("text", config.ProviderConfig{})
    if err != nil {
        fmt.Printf("loader creation failed: %v\n", err)
        return
    }

    docs, err := textLoader.Load(ctx, "/path/to/document.txt")
    if err != nil {
        fmt.Printf("load failed: %v\n", err)
        return
    }

    for _, doc := range docs {
        fmt.Printf("Loaded: %s (%d chars)\n", doc.ID, len(doc.Content))
    }
}
```

## Step 2: Build a Loading Pipeline

Chain loaders with transformers for metadata enrichment:

```go
func buildPipeline() *loader.LoaderPipeline {
    textLoader, err := loader.New("text", config.ProviderConfig{})
    if err != nil {
        panic(err)
    }

    // Add metadata enrichment transformer.
    enricher := loader.TransformerFunc(
        func(ctx context.Context, doc schema.Document) (schema.Document, error) {
            if doc.Metadata == nil {
                doc.Metadata = make(map[string]any)
            }
            doc.Metadata["ingested_at"] = time.Now().UTC().Format(time.RFC3339)
            doc.Metadata["word_count"] = len(strings.Fields(doc.Content))
            return doc, nil
        },
    )

    return loader.NewPipeline(
        loader.WithLoader(textLoader),
        loader.WithTransformer(enricher),
    )
}
```

## Step 3: Process Files in a Directory

Iterate over files in a directory and load each through the pipeline:

```go
import (
    "io/fs"
    "os"
    "path/filepath"
    "strings"
    "time"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/rag/loader"
    "github.com/lookatitude/beluga-ai/schema"
)

func loadDirectory(ctx context.Context, pipeline *loader.LoaderPipeline, dirPath string) ([]schema.Document, error) {
    var allDocs []schema.Document

    err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }
        if d.IsDir() {
            return nil
        }

        // Filter by extension.
        ext := strings.ToLower(filepath.Ext(path))
        if ext != ".txt" && ext != ".md" {
            return nil
        }

        docs, err := pipeline.Load(ctx, path)
        if err != nil {
            return fmt.Errorf("loading %s: %w", path, err)
        }

        allDocs = append(allDocs, docs...)
        return nil
    })

    return allDocs, err
}
```

## Step 4: Parallel Processing with Worker Pool

Speed up loading by processing files concurrently with a bounded worker pool:

```go
import "sync"

func loadDirectoryParallel(ctx context.Context, dirPath string, workers int) ([]schema.Document, error) {
    // Collect file paths.
    var paths []string
    err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }
        if !d.IsDir() {
            paths = append(paths, path)
        }
        return nil
    })
    if err != nil {
        return nil, err
    }

    // Process with worker pool.
    pathCh := make(chan string, len(paths))
    for _, p := range paths {
        pathCh <- p
    }
    close(pathCh)

    var mu sync.Mutex
    var allDocs []schema.Document
    var wg sync.WaitGroup

    textLoader, err := loader.New("text", config.ProviderConfig{})
    if err != nil {
        return nil, err
    }

    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for path := range pathCh {
                docs, err := textLoader.Load(ctx, path)
                if err != nil {
                    continue // Log and skip failed files.
                }
                mu.Lock()
                allDocs = append(allDocs, docs...)
                mu.Unlock()
            }
        }()
    }

    wg.Wait()
    return allDocs, nil
}
```

## Step 5: Rate-Limited Embedding

When indexing documents to a vector store, respect embedding API rate limits:

```go
import "golang.org/x/time/rate"

func indexWithRateLimit(ctx context.Context, docs []schema.Document, embedFn func(context.Context, schema.Document) error) error {
    limiter := rate.NewLimiter(rate.Limit(50), 1) // 50 documents per second.

    for _, doc := range docs {
        if err := limiter.Wait(ctx); err != nil {
            return fmt.Errorf("rate limiter: %w", err)
        }
        if err := embedFn(ctx, doc); err != nil {
            return fmt.Errorf("embed %s: %w", doc.ID, err)
        }
    }
    return nil
}
```

## Step 6: Checkpointing for Crash Recovery

For large ingestion jobs, track processed files to resume after crashes:

```go
type Checkpoint struct {
    mu        sync.Mutex
    processed map[string]bool
}

func NewCheckpoint() *Checkpoint {
    return &Checkpoint{processed: make(map[string]bool)}
}

func (c *Checkpoint) IsProcessed(path string) bool {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.processed[path]
}

func (c *Checkpoint) MarkProcessed(path string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.processed[path] = true
}
```

Use the checkpoint in your loading loop:

```go
func loadWithCheckpoint(ctx context.Context, dirPath string, checkpoint *Checkpoint) {
    filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
        if err != nil || d.IsDir() {
            return err
        }
        if checkpoint.IsProcessed(path) {
            return nil // Skip already processed files.
        }

        // ... load and process ...

        checkpoint.MarkProcessed(path)
        return nil
    })
}
```

## Verification

1. Point the loader at a directory with 1,000 small text files.
2. Load using the pipeline and verify all documents are returned with metadata.
3. Compare memory usage between loading all files at once versus processing in batches.
4. Test the parallel loader and verify it completes faster than sequential loading.

## Next Steps

- [Directory and PDF Scraper](/tutorials/documents/pdf-scraper) -- Load PDFs and other binary formats
- [Semantic Splitting](/tutorials/documents/semantic-splitting) -- Split loaded documents by semantic boundaries
