# Document Loaders Examples

This directory contains examples demonstrating how to use the `documentloaders` package to load documents from various sources.

## Examples

### Basic Usage (`basic/main.go`)

Demonstrates fundamental document loading operations:
- Loading documents from a directory using `RecursiveDirectoryLoader`
- Loading a single text file using `TextLoader`
- Using the registry pattern to create loaders dynamically

**Run:**
```bash
go run examples/documentloaders/basic/main.go
```

### Advanced Directory Loading (`directory/main.go`)

Shows advanced configuration options for `RecursiveDirectoryLoader`:
- Depth limiting (`WithMaxDepth`)
- Extension filtering (`WithExtensions`)
- Concurrency control (`WithConcurrency`)
- File size limits (`WithMaxFileSize`)
- Symlink following (`WithFollowSymlinks`)
- Lazy loading for streaming large datasets (`LazyLoad`)
- Combining multiple options

**Run:**
```bash
go run examples/documentloaders/directory/main.go
```

## Key Concepts

### Directory Loader Options

- **MaxDepth**: Limits how deep the loader will traverse into subdirectories
- **Extensions**: Filters files by extension (e.g., `.txt`, `.md`, `.go`)
- **Concurrency**: Controls the number of parallel workers for file loading
- **MaxFileSize**: Skips files larger than the specified size (in bytes)
- **FollowSymlinks**: Enables following symbolic links with cycle detection

### Lazy Loading

For large datasets, use `LazyLoad()` instead of `Load()` to stream documents one at a time:

```go
ch, err := loader.LazyLoad(ctx)
if err != nil {
    log.Fatal(err)
}

for item := range ch {
    switch v := item.(type) {
    case error:
        // Handle error
    case schema.Document:
        // Process document
    }
}
```

### Registry Pattern

Create loaders dynamically using the registry:

```go
registry := documentloaders.GetRegistry()
loader, err := registry.Create("directory", map[string]any{
    "max_depth":  2,
    "extensions": []string{".txt", ".md"},
})
```

## Integration with RAG Pipeline

Document loaders are typically the first step in a RAG (Retrieval-Augmented Generation) pipeline:

1. **Load**: Use document loaders to read documents from files/directories
2. **Split**: Use text splitters to chunk documents into smaller pieces
3. **Embed**: Generate embeddings for each chunk
4. **Store**: Store embeddings in a vector database
5. **Retrieve**: Query the vector database for relevant chunks
6. **Generate**: Use retrieved chunks as context for LLM generation

See `examples/rag/with_loaders/main.go` for a complete example.
