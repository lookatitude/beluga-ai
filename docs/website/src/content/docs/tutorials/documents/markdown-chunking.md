---
title: Markdown-Aware Chunking
description: Split Markdown documents into semantically coherent chunks that respect heading hierarchy, preserve code blocks, and maintain context for RAG retrieval.
---

Splitting text by character count is a recipe for broken context. If a heading is separated from its paragraph or a code block is cut in half, retrieval quality degrades because the embedding for that chunk no longer represents a coherent unit of meaning. The `rag/splitter` package provides a `MarkdownSplitter` that understands document structure and keeps chunks coherent by splitting on heading boundaries rather than arbitrary character positions.

## What You Will Build

A Markdown splitting pipeline that divides documents based on heading hierarchy, preserves code block integrity, prepends parent heading context to each chunk, and falls back to recursive character splitting for oversized sections.

## Prerequisites

- Familiarity with `schema.Document` and the RAG pipeline
- Understanding of why chunking matters for embedding quality

## Core Concepts

### TextSplitter Interface

All splitters implement the same interface. This uniform contract means you can swap between `MarkdownSplitter`, `RecursiveSplitter`, and custom splitters without changing your pipeline code. The `SplitDocuments` method handles the additional work of preserving and augmenting document metadata through the splitting process.

```go
import "github.com/lookatitude/beluga-ai/rag/splitter"

type TextSplitter interface {
    Split(ctx context.Context, text string) ([]string, error)
    SplitDocuments(ctx context.Context, docs []schema.Document) ([]schema.Document, error)
}
```

### The Problem with Naive Splitting

If you split a Markdown file every 1,000 characters:

- A heading might be separated from its content
- A code block might be cut mid-function
- A list item might lose its context
- Retrieval returns fragments without meaning

Each of these fragments produces a poor embedding because the text does not represent a complete thought. The retriever then returns chunks that confuse the model rather than helping it answer the query.

### MarkdownSplitter

The `MarkdownSplitter` splits on heading boundaries (`#`, `##`, `###`, etc.), keeping each section as a coherent chunk. The `PreserveHeaders` option prepends parent headings to each chunk, giving the embedding model the hierarchical context it needs to understand where the chunk fits in the document. Without header preservation, a chunk containing "Run `go test`" loses the context that it belongs to the "Testing" section of the "Installation Guide."

```go
s := splitter.NewMarkdownSplitter(
    splitter.WithMarkdownChunkSize(1000),
    splitter.WithMarkdownChunkOverlap(0),
    splitter.WithPreserveHeaders(true),
)
```

## Step 1: Initialize the Markdown Splitter

Configure the splitter with functional options. The chunk size sets the upper bound -- sections smaller than this are kept whole, while larger sections trigger the recursive fallback. An overlap of 0 is typical for structural splitting because heading boundaries already provide clear semantic breaks; overlap is more useful for character-based splitting where the break point is arbitrary.

```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/rag/splitter"
)

func main() {
    ctx := context.Background()

    s := splitter.NewMarkdownSplitter(
        splitter.WithMarkdownChunkSize(1500),    // Max chunk size in characters.
        splitter.WithMarkdownChunkOverlap(0),     // No overlap between chunks.
        splitter.WithPreserveHeaders(true),       // Prepend parent headers to each chunk.
    )

    _ = s
    _ = ctx
}
```

## Step 2: Split a Markdown Document

```go
func splitMarkdown(ctx context.Context) {
    s := splitter.NewMarkdownSplitter(
        splitter.WithPreserveHeaders(true),
    )

    text := `# Project Documentation

## Installation

Run the following command to install:

` + "```bash" + `
go get github.com/example/project
` + "```" + `

## Configuration

Create a config file at ~/.config/project.yaml:

` + "```yaml" + `
database:
  host: localhost
  port: 5432
` + "```" + `

## Usage

### Basic Usage

Import the package and create a new instance:

` + "```go" + `
import "github.com/example/project"

func main() {
    p := project.New()
    p.Run()
}
` + "```" + `

### Advanced Usage

For advanced scenarios, configure options:

` + "```go" + `
p := project.New(
    project.WithTimeout(30 * time.Second),
    project.WithRetries(3),
)
` + "```" + `
`

    chunks, err := s.Split(ctx, text)
    if err != nil {
        fmt.Printf("split error: %v\n", err)
        return
    }

    for i, chunk := range chunks {
        fmt.Printf("--- Chunk %d ---\n%s\n\n", i, chunk)
    }
}
```

With `PreserveHeaders` enabled, each chunk includes its parent heading context. This means a chunk from the "Basic Usage" subsection includes both the `# Project Documentation` and `## Usage` headings, giving the embedding model full hierarchical context:

```
--- Chunk 0 ---
# Project Documentation
## Installation
Run the following command to install:
```bash
go get github.com/example/project
```

--- Chunk 1 ---
# Project Documentation
## Configuration
Create a config file at ~/.config/project.yaml:
...
```

## Step 3: Split Documents (Preserving Metadata)

Use `SplitDocuments` to split a collection while preserving and augmenting metadata. Each chunk document inherits its parent's metadata and receives additional fields (`parent_id`, `chunk_index`, `chunk_total`) that enable reconstruction of the original document order and provide useful signals for retrieval ranking -- for example, a retriever could boost chunks from the same parent document to provide more coherent context.

```go
import "github.com/lookatitude/beluga-ai/schema"

func splitDocuments(ctx context.Context, docs []schema.Document) {
    s := splitter.NewMarkdownSplitter(
        splitter.WithMarkdownChunkSize(1000),
        splitter.WithPreserveHeaders(true),
    )

    chunks, err := s.SplitDocuments(ctx, docs)
    if err != nil {
        fmt.Printf("split error: %v\n", err)
        return
    }

    for _, chunk := range chunks {
        fmt.Printf("ID: %s | Parent: %v | Index: %v/%v | Size: %d\n",
            chunk.ID,
            chunk.Metadata["parent_id"],
            chunk.Metadata["chunk_index"],
            chunk.Metadata["chunk_total"],
            len(chunk.Content),
        )
    }
}
```

Each chunk document includes metadata:

| Key | Description |
|-----|-------------|
| `parent_id` | ID of the original document |
| `chunk_index` | Zero-based index within the parent |
| `chunk_total` | Total number of chunks from the parent |

## Step 4: Use the Registry

Create splitters via the registry for configuration-driven pipelines. This follows the standard Beluga AI registry pattern (`Register()` + `New()` + `List()`), enabling splitter selection from configuration files or environment variables without hardcoding the splitter type.

```go
import "github.com/lookatitude/beluga-ai/config"

func createFromRegistry() {
    s, err := splitter.New("markdown", config.ProviderConfig{
        Options: map[string]any{
            "chunk_size":       1000,
            "chunk_overlap":    0,
            "preserve_headers": true,
        },
    })
    if err != nil {
        fmt.Printf("error: %v\n", err)
        return
    }

    // Use the splitter...
    _ = s
}
```

## Step 5: Handle Oversized Sections

When a single Markdown section exceeds the chunk size, the `MarkdownSplitter` automatically falls back to the `RecursiveSplitter` for that section, splitting on paragraph breaks, then line breaks, then spaces. This two-level approach preserves structural coherence at the section level while still respecting the chunk size limit for unusually long sections. The overlap parameter becomes relevant during recursive fallback, providing continuity between sub-chunks of the same section.

```go
// A section with 5000 characters and a chunkSize of 1000
// will be sub-split into ~5 chunks using recursive character splitting.
s := splitter.NewMarkdownSplitter(
    splitter.WithMarkdownChunkSize(1000),
    splitter.WithMarkdownChunkOverlap(100), // Overlap for recursive fallback.
)
```

## Step 6: Combine with Recursive Splitting

For documents that are not Markdown, use the `RecursiveSplitter` directly. The recursive splitter tries separators in order: paragraph breaks (`\n\n`), line breaks (`\n`), spaces (` `), and finally character-level splitting. This priority order ensures the cleanest possible break points are tried first -- paragraph breaks produce the most coherent chunks, while character-level splitting is the last resort for text without any natural break points.

```go
func splitPlainText(ctx context.Context, text string) {
    s := splitter.NewRecursiveSplitter(
        splitter.WithChunkSize(1000),
        splitter.WithChunkOverlap(200),
    )

    chunks, err := s.Split(ctx, text)
    if err != nil {
        fmt.Printf("error: %v\n", err)
        return
    }

    fmt.Printf("Split into %d chunks\n", len(chunks))
}
```

## Verification

1. Take a Markdown file with multiple headings and code blocks.
2. Split with `PreserveHeaders` enabled.
3. Verify each chunk starts with its heading context.
4. Verify no code blocks are split mid-block (unless they exceed the chunk size).
5. Check that `SplitDocuments` preserves metadata and adds chunk indices.

## Next Steps

- [Semantic Splitting](/tutorials/documents/semantic-splitting) -- Split by meaning rather than structure
- [Directory and PDF Scraper](/tutorials/documents/pdf-scraper) -- Load files to split
