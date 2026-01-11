# Text Splitting Concepts

This document explains how text splitting works in Beluga AI, including splitters, chunking strategies, and best practices for preparing documents for RAG pipelines.

## Overview

Text splitters divide large documents into smaller chunks that fit within embedding model context windows and improve retrieval quality. Effective splitting balances chunk size, overlap, and semantic boundaries.

## Core Interface

### TextSplitter Interface

All text splitters implement the `TextSplitter` interface:

```go
type TextSplitter interface {
    retrievers.iface.Splitter
    
    // SplitText splits a single text string into chunks
    SplitText(ctx context.Context, text string) ([]string, error)
    
    // SplitDocuments splits existing documents into smaller documents
    SplitDocuments(ctx context.Context, documents []schema.Document) ([]schema.Document, error)
    
    // CreateDocuments creates documents from raw text with metadata
    CreateDocuments(ctx context.Context, texts []string, metadatas []map[string]any) ([]schema.Document, error)
}
```

## Available Splitters

### RecursiveCharacterTextSplitter

Uses a hierarchy of separators to split text while respecting natural boundaries.

**Separator Hierarchy:**
1. Paragraphs (`\n\n`)
2. Lines (`\n`)
3. Words (` `)
4. Characters (fallback)

**Example:**
```go
import (
    "github.com/lookatitude/beluga-ai/pkg/textsplitters"
)

splitter, err := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithRecursiveChunkSize(1000),      // 1000 characters per chunk
    textsplitters.WithRecursiveChunkOverlap(200),    // 200 character overlap
    textsplitters.WithSeparators("\n\n", "\n", " "), // Custom separators
)
if err != nil {
    log.Fatal(err)
}

// Split text
chunks, err := splitter.SplitText(ctx, longText)
if err != nil {
    log.Fatal(err)
}
```

**Configuration Options:**

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `WithRecursiveChunkSize` | `int` | `1000` | Target chunk size in characters |
| `WithRecursiveChunkOverlap` | `int` | `200` | Overlap between chunks |
| `WithRecursiveLengthFunction` | `func(string) int` | `len()` | Custom length function (for tokens) |
| `WithSeparators` | `[]string` | `["\n\n", "\n", " ", ""]` | Separator hierarchy |

### MarkdownTextSplitter

Respects markdown structure when splitting, preserving code blocks and splitting at headers.

**Features:**
- Splits at header boundaries (`#`, `##`, etc.)
- Preserves code blocks intact
- Maintains table structure
- Handles nested markdown elements

**Example:**
```go
splitter, err := textsplitters.NewMarkdownTextSplitter(
    textsplitters.WithMarkdownChunkSize(500),
    textsplitters.WithMarkdownChunkOverlap(50),
    textsplitters.WithHeadersToSplitOn("#", "##", "###"),
)
if err != nil {
    log.Fatal(err)
}

chunks, err := splitter.SplitText(ctx, markdownText)
```

**Configuration Options:**

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `WithMarkdownChunkSize` | `int` | `1000` | Target chunk size |
| `WithMarkdownChunkOverlap` | `int` | `200` | Overlap between chunks |
| `WithMarkdownLengthFunction` | `func(string) int` | `len()` | Custom length function |
| `WithHeadersToSplitOn` | `[]string` | `["#", "##", "###", ...]` | Headers that trigger splits |
| `WithReturnEachLine` | `bool` | `false` | Return each line as separate chunk |

## Chunk Size and Overlap

### Choosing Chunk Size

Chunk size should match your embedding model's context window:

- **Small models** (e.g., `text-embedding-ada-002`): 512-1000 characters
- **Medium models**: 1000-2000 characters
- **Large models**: 2000-4000 characters

```go
// For OpenAI ada-002 (512 token context)
splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithRecursiveChunkSize(500), // ~500 characters ≈ 125 tokens
)

// For larger models
splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithRecursiveChunkSize(2000), // 2000 characters
)
```

### Overlap Strategy

Overlap preserves context across chunk boundaries:

- **10-20% of chunk size**: Good general rule
- **More overlap**: Better for technical documents
- **Less overlap**: Better for narrative text

```go
// 20% overlap
splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithRecursiveChunkSize(1000),
    textsplitters.WithRecursiveChunkOverlap(200), // 20% overlap
)
```

## Token-Based Splitting

For token-aware splitting, provide a custom length function:

```go
// Simple tokenizer (word count)
tokenizer := func(text string) int {
    return len(strings.Fields(text))
}

// Or use a real tokenizer library
import "github.com/tiktoken-go/tokenizer"
tokenizer := func(text string) int {
    enc, _ := tokenizer.Get("cl100k_base")
    tokens, _ := enc.Encode(text)
    return len(tokens)
}

splitter, err := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithRecursiveLengthFunction(tokenizer),
    textsplitters.WithRecursiveChunkSize(100), // 100 tokens
    textsplitters.WithRecursiveChunkOverlap(20), // 20 token overlap
)
```

## Splitting Documents

### SplitDocuments

Splits existing `schema.Document` objects while preserving metadata:

```go
docs, _ := loader.Load(ctx)

splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter()
chunks, _ := splitter.SplitDocuments(ctx, docs)

// Each chunk inherits source document metadata plus:
// - chunk_index: 0-based index
// - chunk_total: total chunks from source
for _, chunk := range chunks {
    fmt.Printf("Chunk %s/%s from %s\n",
        chunk.Metadata["chunk_index"],
        chunk.Metadata["chunk_total"],
        chunk.Metadata["source"],
    )
}
```

### CreateDocuments

Creates documents from raw text with optional metadata:

```go
texts := []string{
    "First document text...",
    "Second document text...",
}

metadatas := []map[string]any{
    {"source": "doc1.txt", "author": "Alice"},
    {"source": "doc2.txt", "author": "Bob"},
}

splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter()
docs, _ := splitter.CreateDocuments(ctx, texts, metadatas)
```

## Best Practices

### 1. Match Splitter to Document Type

```go
// For markdown documents
splitter, _ := textsplitters.NewMarkdownTextSplitter(
    textsplitters.WithMarkdownChunkSize(500),
)

// For general text
splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithRecursiveChunkSize(1000),
)
```

### 2. Customize Separators

```go
// For code files
splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithSeparators("\n\n", "\n", " ", ""),
)

// For prose
splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithSeparators("\n\n", "\n", ". ", " ", ""),
)
```

### 3. Use Appropriate Overlap

```go
// Technical documentation (more overlap)
splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithRecursiveChunkSize(1000),
    textsplitters.WithRecursiveChunkOverlap(250), // 25% overlap
)

// Narrative text (less overlap)
splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithRecursiveChunkSize(1000),
    textsplitters.WithRecursiveChunkOverlap(100), // 10% overlap
)
```

### 4. Handle Edge Cases

```go
// Empty text
chunks, err := splitter.SplitText(ctx, "")
if err != nil {
    // Handle error (empty text returns error)
}

// Very short text
chunks, _ := splitter.SplitText(ctx, "Short")
// Returns single chunk even if smaller than chunk size
```

## Integration with Document Loaders

Text splitters work seamlessly with document loaders:

```go
// 1. Load documents
loader, _ := documentloaders.NewDirectoryLoader(fsys)
docs, _ := loader.Load(ctx)

// 2. Split into chunks
splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithRecursiveChunkSize(1000),
    textsplitters.WithRecursiveChunkOverlap(200),
)
chunks, _ := splitter.SplitDocuments(ctx, docs)

// 3. Process chunks
for _, chunk := range chunks {
    // Embed, store, etc.
}
```

## Registry Pattern

Create splitters dynamically using the registry:

```go
registry := textsplitters.GetRegistry()

// List available splitters
splitters := registry.List()
fmt.Printf("Available: %v\n", splitters)

// Create splitter via registry
splitter, err := registry.Create("recursive", map[string]any{
    "chunk_size": 1000,
    "chunk_overlap": 200,
})
```

## Error Handling

Text splitters use custom error types:

```go
type SplitterError struct {
    Op    string // Operation name
    Code  string // Error code
    Msg   string // Error message
    Err   error  // Underlying error
}
```

**Common Error Codes:**
- `ErrCodeInvalidConfig`: Invalid configuration (e.g., overlap > chunk size)
- `ErrCodeEmptyText`: Empty input text
- `ErrCodeCancelled`: Context cancelled

## Observability

All splitters integrate with OpenTelemetry:

- **Traces**: Spans for split operations
- **Metrics**: Chunk count, split duration, input/output sizes
- **Attributes**: Splitter type, chunk size, overlap settings

## Extending Splitters

Create custom splitters by implementing the `TextSplitter` interface:

```go
type CustomSplitter struct {
    // Your fields
}

func (s *CustomSplitter) SplitText(ctx context.Context, text string) ([]string, error) {
    // Implementation
}

func (s *CustomSplitter) SplitDocuments(ctx context.Context, docs []schema.Document) ([]schema.Document, error) {
    // Implementation
}

func (s *CustomSplitter) CreateDocuments(ctx context.Context, texts []string, metadatas []map[string]any) ([]schema.Document, error) {
    // Implementation
}

// Register in registry
registry.Register("custom", func(config map[string]any) (iface.TextSplitter, error) {
    return NewCustomSplitter(config), nil
})
```

## Related Documentation

- [Document Loading Concepts](./document-loading.md)
- [RAG Concepts](./rag.md)
- [Text Splitters Package](../../pkg/textsplitters/README.md)
- [Examples](../../examples/textsplitters/)
