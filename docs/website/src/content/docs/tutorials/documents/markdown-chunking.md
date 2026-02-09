---
title: Markdown-Aware Chunking
description: Split Markdown documents into semantically coherent chunks that respect heading hierarchy, preserve code blocks, and maintain context for RAG retrieval.
---

Splitting text by character count is a recipe for broken context. If a heading is separated from its paragraph or a code block is cut in half, retrieval quality degrades. The `rag/splitter` package provides a `MarkdownSplitter` that understands document structure and keeps chunks coherent.

## What You Will Build

A Markdown splitting pipeline that divides documents based on heading hierarchy, preserves code block integrity, prepends parent heading context to each chunk, and falls back to recursive character splitting for oversized sections.

## Prerequisites

- Familiarity with `schema.Document` and the RAG pipeline
- Understanding of why chunking matters for embedding quality

## Core Concepts

### TextSplitter Interface

All splitters implement the same interface:

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

### MarkdownSplitter

The `MarkdownSplitter` splits on heading boundaries (`#`, `##`, `###`, etc.), keeping each section as a coherent chunk:

```go
s := splitter.NewMarkdownSplitter(
    splitter.WithMarkdownChunkSize(1000),
    splitter.WithMarkdownChunkOverlap(0),
    splitter.WithPreserveHeaders(true),
)
```

## Step 1: Initialize the Markdown Splitter

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

With `PreserveHeaders` enabled, each chunk includes its parent heading context:

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

Use `SplitDocuments` to split a collection while preserving and augmenting metadata:

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

Create splitters via the registry for configuration-driven pipelines:

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

When a single Markdown section exceeds the chunk size, the `MarkdownSplitter` automatically falls back to the `RecursiveSplitter` for that section, splitting on paragraph breaks, then line breaks, then spaces:

```go
// A section with 5000 characters and a chunkSize of 1000
// will be sub-split into ~5 chunks using recursive character splitting.
s := splitter.NewMarkdownSplitter(
    splitter.WithMarkdownChunkSize(1000),
    splitter.WithMarkdownChunkOverlap(100), // Overlap for recursive fallback.
)
```

## Step 6: Combine with Recursive Splitting

For documents that are not Markdown, use the `RecursiveSplitter` directly:

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

The recursive splitter tries separators in order: paragraph breaks (`\n\n`), line breaks (`\n`), spaces (` `), and finally character-level splitting.

## Verification

1. Take a Markdown file with multiple headings and code blocks.
2. Split with `PreserveHeaders` enabled.
3. Verify each chunk starts with its heading context.
4. Verify no code blocks are split mid-block (unless they exceed the chunk size).
5. Check that `SplitDocuments` preserves metadata and adds chunk indices.

## Next Steps

- [Semantic Splitting](/tutorials/documents/semantic-splitting) -- Split by meaning rather than structure
- [Directory and PDF Scraper](/tutorials/documents/pdf-scraper) -- Load files to split
