# Text Splitters Examples

This directory contains examples demonstrating how to use the `textsplitters` package to split documents into chunks for RAG pipelines.

## Examples

### Basic Usage (`basic/main.go`)

Demonstrates fundamental text splitting operations:
- Using `RecursiveCharacterTextSplitter` for general text
- Using `MarkdownTextSplitter` for markdown documents
- Configuring chunk size and overlap
- Using the registry pattern

**Run:**
```bash
go run examples/textsplitters/basic/main.go
```

### Token-Based Splitting (`token_based/main.go`)

Shows how to use custom length functions for token-aware splitting:
- Implementing a custom tokenizer function
- Configuring splitters with token-based chunking
- Handling different tokenization strategies

**Run:**
```bash
go run examples/textsplitters/token_based/main.go
```

## Key Concepts

### Recursive Character Text Splitter

The `RecursiveCharacterTextSplitter` uses a hierarchy of separators to split text:
1. Paragraphs (`\n\n`)
2. Lines (`\n`)
3. Words (` `)
4. Characters (fallback)

This ensures chunks respect natural text boundaries while maintaining size limits.

**Example:**
```go
splitter, err := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithChunkSize(1000),
    textsplitters.WithChunkOverlap(200),
)
chunks, err := splitter.SplitText(ctx, longText)
```

### Markdown Text Splitter

The `MarkdownTextSplitter` respects markdown structure:
- Splits at header boundaries (`#`, `##`, etc.)
- Preserves code blocks intact
- Maintains table structure
- Handles nested markdown elements

**Example:**
```go
splitter, err := textsplitters.NewMarkdownTextSplitter(
    textsplitters.WithMarkdownChunkSize(500),
    textsplitters.WithHeadersToSplitOn("#", "##", "###"),
)
chunks, err := splitter.SplitText(ctx, markdownText)
```

### Chunk Size and Overlap

- **ChunkSize**: Maximum size of each chunk (in characters or tokens)
- **ChunkOverlap**: Number of characters/tokens to overlap between chunks

Overlap helps maintain context across chunk boundaries, which is important for retrieval quality.

### Custom Length Functions

For token-based splitting, provide a custom length function:

```go
tokenizer := func(text string) int {
    // Implement your tokenization logic
    return len(strings.Fields(text)) // Simple word count example
}

splitter, err := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithLengthFunction(tokenizer),
    textsplitters.WithChunkSize(100), // 100 tokens
)
```

## Integration with Document Loaders

Text splitters work seamlessly with document loaders:

```go
// Load documents
loader, _ := documentloaders.NewDirectoryLoader(fsys)
docs, _ := loader.Load(ctx)

// Split documents
splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter()
chunks, _ := splitter.SplitDocuments(ctx, docs)
```

See `examples/rag/with_loaders/main.go` for a complete RAG pipeline example.

## Best Practices

1. **Chunk Size**: Choose based on your embedding model's context window (typically 512-2048 tokens)
2. **Overlap**: Use 10-20% of chunk size for good context preservation
3. **Separators**: Customize separators based on your document type
4. **Markdown**: Use `MarkdownTextSplitter` for markdown documents to preserve structure
5. **Metadata**: Splitters preserve and enhance document metadata with chunk indices
