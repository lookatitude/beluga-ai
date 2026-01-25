# Lazy-loading Large Data Lakes

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll implement lazy-loading patterns to process massive datasets without exhausting your server's memory. You'll learn how to use the `LazyLoad` iterator interface to stream documents to a vector store one by one.

## Learning Objectives
- ✅ Understand Eager vs. Lazy loading
- ✅ Use the `LazyLoad` iterator interface
- ✅ Implement parallel loading with rate limiting
- ✅ Streaming documents to a Vector Store

## Introduction
Welcome, colleague! Loading 10,000 documents into a single Go slice is fine, but loading a million will crash your server. Let's look at how to use lazy loading to process massive data lakes with a flat memory footprint, keeping our ingestion pipelines lean and stable.

## Prerequisites

- [Directory & PDF Scraper](./docloaders-directory-pdf-scraper.md)

## The Problem: Memory Exhaustion

`loader.Load()` returns `[]Document`. If you have 1 million documents, each 10KB, that's 10GB of RAM!

## Step 1: Using the Lazy Loader

Instead of `Load()`, use `LazyLoad()` which returns a channel or an iterator.
```go
package main

import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/documentloaders"
)

func main() {
    loader := documentloaders.NewDirectoryLoader(os.DirFS("/data/lake"))
    
    // LazyLoad returns a channel of documents
    docChan, err := loader.LazyLoad(context.Background())
    
    for doc := range docChan {
        // Process ONE document at a time
        // RAM usage stays low!
        processAndIndex(doc)
    }
}
```

## Step 2: Parallel Processing

Speed up the process using a worker pool.
```go
func processInParallel(docChan <-chan schema.Document) {
    for i := 0; i < 10; i++ { // 10 workers
        go func() {
            for doc := range docChan {
                // Embed and save to DB
            }
        }()
    }
}
```

## Step 3: Rate Limiting

Avoid overwhelming your Embedding API (e.g., OpenAI rate limits).
```text
import "golang.org/x/time/rate"
go
limiter := rate.NewLimiter(rate.Limit(50), 1) // 50 docs per second

for doc := range docChan {
    limiter.Wait(ctx)
    embedder.Embed(doc)
}
```

## Step 4: Checkpointing (Advanced)

If the process crashes after 500,000 files, you don't want to restart.
```
// Implementation logic:
// 1. Store processed file paths in a tiny SQLite DB or Redis.
// 2. In MetadataFunc, check if file is already processed.
// 3. Skip if exists.

## Verification

1. Point the loader at a directory with 10,000 small files.
2. Monitor RAM usage of your Go process.
3. Compare `Load()` (RAM spikes) vs `LazyLoad()` (RAM stays flat).

## Next Steps

- **[Semantic Splitting](./textsplitters-semantic-splitting.md)** - Better chunking for large data.
- **[Production pgvector Sharding](../providers/vectorstores-pgvector-sharding.md)** - Store the results.
