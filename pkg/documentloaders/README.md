# Document Loaders

The `documentloaders` package provides interfaces and implementations for loading documents from various sources (directories, files, etc.) into the Beluga AI Framework's document format.

## Features

- **RecursiveDirectoryLoader**: Recursively loads documents from directory structures
- **TextLoader**: Loads a single text file
- **Registry Pattern**: Extensible provider registration system
- **OTEL Integration**: Full observability with tracing and metrics
- **Lazy Loading**: Streaming support via `LazyLoad()` method

## Quick Start

### Loading from a Directory

```go
import (
    "io/fs"
    "os"
    "github.com/lookatitude/beluga-ai/pkg/documentloaders"
)

// Create a directory loader
fsys := os.DirFS("/path/to/documents")
loader, err := documentloaders.NewDirectoryLoader(fsys,
    documentloaders.WithMaxDepth(5),
    documentloaders.WithExtensions(".txt", ".md"),
    documentloaders.WithConcurrency(4),
)
if err != nil {
    log.Fatal(err)
}

// Load all documents
ctx := context.Background()
docs, err := loader.Load(ctx)
if err != nil {
    log.Fatal(err)
}

// Process documents...
for _, doc := range docs {
    fmt.Println(doc.PageContent)
}
```

### Loading a Single File

```go
loader, err := documentloaders.NewTextLoader("/path/to/file.txt")
if err != nil {
    log.Fatal(err)
}

docs, err := loader.Load(ctx)
```

### Using the Registry

```go
// Create loader via registry
registry := documentloaders.GetRegistry()
loader, err := registry.Create("directory", map[string]any{
    "path":       "/path/to/documents",
    "max_depth":  5,
    "extensions": []string{".txt", ".md"},
})
```

## Custom Loader Registration

To register a custom loader:

```go
import "github.com/lookatitude/beluga-ai/pkg/documentloaders"

func init() {
    registry := documentloaders.GetRegistry()
    registry.Register("my_custom_loader", func(config map[string]any) (documentloaders.iface.DocumentLoader, error) {
        // Extract config
        path := config["path"].(string)
        
        // Create and return your custom loader
        return NewMyCustomLoader(path), nil
    })
}
```

## Configuration

### DirectoryLoader Options

- `WithMaxDepth(depth int)`: Maximum recursion depth (0 = root only)
- `WithExtensions(exts ...string)`: Filter by file extensions
- `WithConcurrency(n int)`: Number of concurrent workers
- `WithMaxFileSize(size int64)`: Maximum file size in bytes
- `WithFollowSymlinks(follow bool)`: Enable/disable symlink following

### TextLoader Options

- `WithMaxFileSize(size int64)`: Maximum file size in bytes

## Error Handling

All loaders return `LoaderError` with specific error codes:

- `ErrCodeIOError`: File system errors
- `ErrCodeNotFound`: File/directory not found
- `ErrCodeInvalidConfig`: Invalid configuration
- `ErrCodeFileTooLarge`: File exceeds size limit
- `ErrCodeBinaryFile`: Binary file detected

## Observability

All loaders emit OTEL traces with attributes:

- `loader.type`: Loader type (e.g., "directory", "text")
- `loader.documents_count`: Number of documents loaded
- `loader.duration_ms`: Operation duration
- `loader.files_skipped`: Number of files skipped

## Testing

See `advanced_test.go` and `registry_test.go` for comprehensive test examples.
