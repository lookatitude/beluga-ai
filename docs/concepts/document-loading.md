# Document Loading Concepts

This document explains how document loading works in Beluga AI, including loaders, configuration options, and best practices for ingesting documents into RAG pipelines.

## Overview

Document loaders read documents from various sources (files, directories, databases, APIs) and convert them into `schema.Document` objects that can be processed by text splitters and vector stores.

## Core Interface

### DocumentLoader Interface

All document loaders implement the `DocumentLoader` interface:

```go
type DocumentLoader interface {
    core.Loader
    
    // Load reads all documents from the configured source
    Load(ctx context.Context) ([]schema.Document, error)
    
    // LazyLoad provides streaming document loading
    LazyLoad(ctx context.Context) (<-chan any, error)
}
```

## Available Loaders

### RecursiveDirectoryLoader

Loads documents recursively from a directory structure.

**Features:**
- Recursive traversal with depth limits
- Extension filtering
- Concurrent file loading
- File size limits
- Symlink following with cycle detection
- Binary file detection

**Example:**
```go
import (
    "os"
    "github.com/lookatitude/beluga-ai/pkg/documentloaders"
)

// Create loader
fsys := os.DirFS("./data")
loader, err := documentloaders.NewDirectoryLoader(fsys,
    documentloaders.WithMaxDepth(3),              // Max 3 levels deep
    documentloaders.WithExtensions(".txt", ".md"), // Only text and markdown
    documentloaders.WithConcurrency(4),          // 4 parallel workers
    documentloaders.WithMaxFileSize(10*1024*1024), // Max 10MB per file
    documentloaders.WithFollowSymlinks(true),     // Follow symlinks
)
if err != nil {
    log.Fatal(err)
}

// Load all documents
docs, err := loader.Load(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Loaded %d documents\n", len(docs))
```

**Configuration Options:**

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `WithMaxDepth` | `int` | `0` (unlimited) | Maximum directory depth to traverse |
| `WithExtensions` | `[]string` | `[]` (all files) | File extensions to include |
| `WithConcurrency` | `int` | `GOMAXPROCS` | Number of parallel workers |
| `WithMaxFileSize` | `int64` | `0` (unlimited) | Maximum file size in bytes |
| `WithFollowSymlinks` | `bool` | `true` | Enable symlink following with cycle detection |

### TextLoader

Loads a single text file.

**Example:**
```go
loader, err := documentloaders.NewTextLoader("./document.txt",
    documentloaders.WithMaxFileSize(5*1024*1024), // Max 5MB
)
if err != nil {
    log.Fatal(err)
}

docs, err := loader.Load(ctx)
if err != nil {
    log.Fatal(err)
}
```

## Lazy Loading

For large datasets, use `LazyLoad()` to stream documents one at a time:

```go
ch, err := loader.LazyLoad(ctx)
if err != nil {
    log.Fatal(err)
}

for item := range ch {
    switch v := item.(type) {
    case error:
        log.Printf("Error: %v", v)
    case schema.Document:
        // Process document
        processDocument(v)
    }
}
```

**Benefits:**
- Lower memory usage for large datasets
- Process documents as they become available
- Better for streaming pipelines

## Registry Pattern

Create loaders dynamically using the registry:

```go
registry := documentloaders.GetRegistry()

// List available loaders
loaders := registry.List()
fmt.Printf("Available: %v\n", loaders)

// Create loader via registry
loader, err := registry.Create("directory", map[string]any{
    "max_depth":  2,
    "extensions": []string{".txt", ".md"},
    "concurrency": 4,
})
if err != nil {
    log.Fatal(err)
}
```

## Metadata

All loaded documents include metadata:

- `source`: File path or source identifier
- `file_size`: File size in bytes
- `modified_at`: File modification timestamp (ISO 8601)
- `loader_type`: Type of loader used ("directory", "text", etc.)

**Example:**
```go
doc := docs[0]
fmt.Printf("Source: %s\n", doc.Metadata["source"])
fmt.Printf("Size: %s bytes\n", doc.Metadata["file_size"])
fmt.Printf("Modified: %s\n", doc.Metadata["modified_at"])
```

## Best Practices

### 1. Choose Appropriate Depth Limits

```go
// For shallow directory structures
loader, _ := documentloaders.NewDirectoryLoader(fsys,
    documentloaders.WithMaxDepth(1), // Only current directory
)

// For deep hierarchies
loader, _ := documentloaders.NewDirectoryLoader(fsys,
    documentloaders.WithMaxDepth(5), // Allow deeper traversal
)
```

### 2. Filter by Extension

```go
// Only load relevant file types
loader, _ := documentloaders.NewDirectoryLoader(fsys,
    documentloaders.WithExtensions(".txt", ".md", ".rst"),
)
```

### 3. Set File Size Limits

```go
// Prevent loading extremely large files
loader, _ := documentloaders.NewDirectoryLoader(fsys,
    documentloaders.WithMaxFileSize(10*1024*1024), // 10MB limit
)
```

### 4. Configure Concurrency

```go
// For I/O-bound workloads, use more workers
loader, _ := documentloaders.NewDirectoryLoader(fsys,
    documentloaders.WithConcurrency(8), // 8 parallel workers
)

// For CPU-bound processing, use fewer workers
loader, _ := documentloaders.NewDirectoryLoader(fsys,
    documentloaders.WithConcurrency(2), // 2 workers
)
```

### 5. Handle Errors Gracefully

```go
docs, err := loader.Load(ctx)
if err != nil {
    // Check error type
    if loaderErr, ok := err.(*documentloaders.LoaderError); ok {
        switch loaderErr.Code {
        case documentloaders.ErrCodeIOError:
            log.Printf("I/O error: %v", loaderErr)
        case documentloaders.ErrCodeFileTooLarge:
            log.Printf("File too large: %v", loaderErr)
        case documentloaders.ErrCodeBinaryFile:
            log.Printf("Binary file skipped: %v", loaderErr)
        default:
            log.Printf("Error: %v", loaderErr)
        }
    }
    return err
}
```

## Integration with RAG Pipeline

Document loaders are typically the first step in a RAG pipeline:

```go
// 1. Load documents
loader, _ := documentloaders.NewDirectoryLoader(fsys)
docs, _ := loader.Load(ctx)

// 2. Split into chunks
splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter()
chunks, _ := splitter.SplitDocuments(ctx, docs)

// 3. Generate embeddings
embeddings, _ := embedder.EmbedDocuments(ctx, extractTexts(chunks))

// 4. Store in vector database
store.AddDocuments(ctx, chunks, vectorstores.WithEmbedder(embedder))
```

## Error Handling

Document loaders use custom error types with specific error codes:

```go
type LoaderError struct {
    Op    string // Operation name
    Code  string // Error code
    Path  string // File path (if applicable)
    Msg   string // Error message
    Err   error  // Underlying error
}
```

**Common Error Codes:**
- `ErrCodeIOError`: File I/O errors
- `ErrCodeFileTooLarge`: File exceeds MaxFileSize
- `ErrCodeBinaryFile`: Binary content detected
- `ErrCodeInvalidConfig`: Invalid configuration
- `ErrCodeCancelled`: Context cancelled

## Observability

All loaders integrate with OpenTelemetry:

- **Traces**: Spans for `Load()` and `LazyLoad()` operations
- **Metrics**: Document count, load duration, file count
- **Attributes**: Loader type, max depth, concurrency settings

## Extending Loaders

Create custom loaders by implementing the `DocumentLoader` interface:

```go
type CustomLoader struct {
    // Your fields
}

func (l *CustomLoader) Load(ctx context.Context) ([]schema.Document, error) {
    // Implementation
}

func (l *CustomLoader) LazyLoad(ctx context.Context) (<-chan any, error) {
    // Implementation
}

// Register in registry
registry.Register("custom", func(config map[string]any) (iface.DocumentLoader, error) {
    return NewCustomLoader(config), nil
})
```

## Related Documentation

- [Text Splitting Concepts](./text-splitting.md)
- [RAG Concepts](./rag.md)
- [Document Loaders Package](https://github.com/lookatitude/beluga-ai/tree/main/pkg/documentloaders/README.md)
- [Examples](https://github.com/lookatitude/beluga-ai/tree/main/examples/documentloaders/)
