# Text Splitters

The `textsplitters` package provides interfaces and implementations for splitting text and documents into chunks suitable for embedding and retrieval in RAG pipelines.

## Features

- **RecursiveCharacterTextSplitter**: Recursively splits text using separator hierarchy
- **MarkdownTextSplitter**: Markdown-aware splitting with header boundaries
- **Registry Pattern**: Extensible provider registration system
- **OTEL Integration**: Full observability with tracing and metrics
- **Custom Length Functions**: Support for token-based splitting

## Quick Start

### Recursive Character Splitting

```go
import "github.com/lookatitude/beluga-ai/pkg/textsplitters"

// Create a recursive splitter
splitter, err := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithRecursiveChunkSize(1000),
    textsplitters.WithRecursiveChunkOverlap(200),
    textsplitters.WithSeparators("\n\n", "\n", " ", ""),
)
if err != nil {
    log.Fatal(err)
}

// Split text
ctx := context.Background()
chunks, err := splitter.SplitText(ctx, longText)
if err != nil {
    log.Fatal(err)
}

// Process chunks...
for _, chunk := range chunks {
    fmt.Println(chunk)
}
```

### Markdown Splitting

```go
splitter, err := textsplitters.NewMarkdownTextSplitter(
    textsplitters.WithMarkdownChunkSize(500),
    textsplitters.WithHeadersToSplitOn("#", "##", "###"),
)
if err != nil {
    log.Fatal(err)
}

chunks, err := splitter.SplitText(ctx, markdownText)
```

### Splitting Documents

```go
// Split documents with metadata preservation
chunks, err := splitter.SplitDocuments(ctx, documents)
// Each chunk will have chunk_index and chunk_total metadata
```

### Using the Registry

```go
// Create splitter via registry
registry := textsplitters.GetRegistry()
splitter, err := registry.Create("recursive", map[string]any{
    "chunk_size":    1000,
    "chunk_overlap": 200,
})
```

## Custom Splitter Registration

To register a custom splitter:

```go
import "github.com/lookatitude/beluga-ai/pkg/textsplitters"

func init() {
    registry := textsplitters.GetRegistry()
    registry.Register("my_custom_splitter", func(config map[string]any) (textsplitters.iface.TextSplitter, error) {
        // Extract config
        chunkSize := config["chunk_size"].(int)
        
        // Create and return your custom splitter
        return NewMyCustomSplitter(chunkSize), nil
    })
}
```

## Configuration

### RecursiveCharacterTextSplitter Options

- `WithRecursiveChunkSize(size int)`: Target chunk size
- `WithRecursiveChunkOverlap(overlap int)`: Overlap between chunks
- `WithSeparators(seps ...string)`: Separator hierarchy (default: `["\n\n", "\n", " ", ""]`)
- `WithRecursiveLengthFunction(fn func(string) int)`: Custom length function for token-based splitting

### MarkdownTextSplitter Options

- `WithMarkdownChunkSize(size int)`: Target chunk size
- `WithMarkdownChunkOverlap(overlap int)`: Overlap between chunks
- `WithHeadersToSplitOn(headers ...string)`: Headers that trigger splits

## Separator Hierarchy

The recursive splitter tries separators in order:

1. `"\n\n"` - Paragraph breaks (preferred)
2. `"\n"` - Line breaks
3. `" "` - Word boundaries
4. `""` - Character-level (fallback)

## Error Handling

All splitters return `SplitterError` with specific error codes:

- `ErrCodeInvalidConfig`: Invalid configuration
- `ErrCodeEmptyInput`: Empty text input
- `ErrCodeCancelled`: Context cancellation

## Observability

All splitters emit OTEL traces with attributes:

- `splitter.type`: Splitter type (e.g., "recursive", "markdown")
- `splitter.input_count`: Number of input documents
- `splitter.output_count`: Number of output chunks
- `splitter.duration_ms`: Operation duration

## Testing

See `advanced_test.go` and `registry_test.go` for comprehensive test examples.
