---
title: "Document Ingestion Recipes"
description: "Common patterns and recipes for loading, splitting, and processing documents in Beluga AI."
---

Common patterns and recipes for loading and processing documents in Beluga AI.

## Problem

You need to load documents from various sources, split them into chunks, and prepare them for embedding and storage in a RAG pipeline. Different file types, directory structures, and processing strategies require flexible ingestion patterns.

## Solution

Use Beluga AI's document loaders and text splitters to build composable ingestion pipelines. Combine directory loading, extension filtering, concurrency, and type-aware splitting to handle diverse document sources efficiently.

## Code Example

### Basic Directory Loading

Load all text files from a directory:

```go
loader, err := documentloaders.NewDirectoryLoader(
    os.DirFS("./data"),
    documentloaders.WithExtensions(".txt", ".md"),
)
if err != nil {
    log.Fatalf("Failed to create loader: %v", err)
}
docs, err := loader.Load(ctx)
if err != nil {
    log.Fatalf("Failed to load: %v", err)
}
```

### Recursive Loading with Depth Limit

Limit directory traversal depth:

```go
loader, err := documentloaders.NewDirectoryLoader(
    os.DirFS("./docs"),
    documentloaders.WithMaxDepth(3),
    documentloaders.WithExtensions(".md"),
)
if err != nil {
    log.Fatalf("Failed to create loader: %v", err)
}
docs, err := loader.Load(ctx)
if err != nil {
    log.Fatalf("Failed to load: %v", err)
}
```

### Filtering by File Size

Skip files larger than a threshold:

```go
loader, err := documentloaders.NewDirectoryLoader(
    os.DirFS("./data"),
    documentloaders.WithMaxFileSize(10*1024*1024), // 10MB limit
)
if err != nil {
    log.Fatalf("Failed to create loader: %v", err)
}
docs, err := loader.Load(ctx)
if err != nil {
    log.Fatalf("Failed to load: %v", err)
}
```

### Concurrent Loading

Use multiple workers for faster loading:

```go
loader, err := documentloaders.NewDirectoryLoader(
    os.DirFS("./large_dataset"),
    documentloaders.WithConcurrency(8),
)
if err != nil {
    log.Fatalf("Failed to create loader: %v", err)
}
docs, err := loader.Load(ctx)
if err != nil {
    log.Fatalf("Failed to load: %v", err)
}
```

### Lazy Loading for Large Datasets

Stream documents one at a time:

```go
ch, err := loader.LazyLoad(ctx)
if err != nil {
    log.Fatalf("Failed to lazy load: %v", err)
}

var docs []schema.Document
for item := range ch {
    switch v := item.(type) {
    case schema.Document:
        docs = append(docs, v)
    case error:
        log.Printf("Error: %v", v)
    }
}
```

### Basic Text Splitting

Split documents into chunks:

```go
splitter, err := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithRecursiveChunkSize(1000),
    textsplitters.WithRecursiveChunkOverlap(200),
)
if err != nil {
    log.Fatalf("Failed to create splitter: %v", err)
}
chunks, err := splitter.SplitDocuments(ctx, docs)
if err != nil {
    log.Fatalf("Failed to split: %v", err)
}
```

### Markdown-Aware Splitting

Preserve markdown structure:

```go
splitter, err := textsplitters.NewMarkdownTextSplitter(
    textsplitters.WithMarkdownChunkSize(500),
    textsplitters.WithHeadersToSplitOn("#", "##", "###"),
)
if err != nil {
    log.Fatalf("Failed to create splitter: %v", err)
}
chunks, err := splitter.SplitDocuments(ctx, markdownDocs)
if err != nil {
    log.Fatalf("Failed to split: %v", err)
}
```

### Token-Based Splitting

Use token counting for accurate chunk sizing:

```go
tokenizer := func(text string) int {
    return len(strings.Fields(text)) // Simple word count
}

splitter, err := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithRecursiveLengthFunction(tokenizer),
    textsplitters.WithRecursiveChunkSize(100),
    textsplitters.WithRecursiveChunkOverlap(20),
)
if err != nil {
    log.Fatalf("Failed to create splitter: %v", err)
}
chunks, err := splitter.SplitDocuments(ctx, docs)
if err != nil {
    log.Fatalf("Failed to split: %v", err)
}
```

### Complete Ingestion Pipeline

Load, split, and prepare for RAG:

```go
// 1. Load
loader, err := documentloaders.NewDirectoryLoader(
    os.DirFS("./data"),
    documentloaders.WithExtensions(".txt", ".md"),
    documentloaders.WithMaxDepth(2),
)
if err != nil {
    log.Fatalf("Failed to create loader: %v", err)
}
docs, err := loader.Load(ctx)
if err != nil {
    log.Fatalf("Failed to load: %v", err)
}

// 2. Split
splitter, err := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithRecursiveChunkSize(1000),
    textsplitters.WithRecursiveChunkOverlap(200),
)
if err != nil {
    log.Fatalf("Failed to create splitter: %v", err)
}
chunks, err := splitter.SplitDocuments(ctx, docs)
if err != nil {
    log.Fatalf("Failed to split: %v", err)
}

// 3. Ready for embedding and storage
```

### Loading Multiple Sources

Combine documents from different sources:

```go
var allDocs []schema.Document

// Load from directory
dirLoader, err := documentloaders.NewDirectoryLoader(os.DirFS("./docs"))
if err != nil {
    log.Fatalf("Failed to create dir loader: %v", err)
}
dirDocs, err := dirLoader.Load(ctx)
if err != nil {
    log.Fatalf("Failed to load dir: %v", err)
}
allDocs = append(allDocs, dirDocs...)

// Load single file
fileLoader, err := documentloaders.NewTextLoader("./important.txt")
if err != nil {
    log.Fatalf("Failed to create file loader: %v", err)
}
fileDocs, err := fileLoader.Load(ctx)
if err != nil {
    log.Fatalf("Failed to load file: %v", err)
}
allDocs = append(allDocs, fileDocs...)

// Process all together
splitter, err := textsplitters.NewRecursiveCharacterTextSplitter()
if err != nil {
    log.Fatalf("Failed to create splitter: %v", err)
}
chunks, err := splitter.SplitDocuments(ctx, allDocs)
if err != nil {
    log.Fatalf("Failed to split: %v", err)
}
```

## Explanation

These recipes demonstrate the core document ingestion patterns:

1. **Directory loading** — Use `NewDirectoryLoader` with functional options to control file discovery. Options like `WithExtensions`, `WithMaxDepth`, and `WithMaxFileSize` allow fine-grained control over which files are loaded.

2. **Concurrent loading** — For large datasets, `WithConcurrency` distributes file loading across multiple goroutines. This significantly improves throughput for I/O-bound loading.

3. **Type-aware splitting** — Different document types benefit from different splitting strategies. Markdown documents should use `MarkdownTextSplitter` to preserve heading structure, while plain text uses `RecursiveCharacterTextSplitter`.

4. **Token-based sizing** — When embedding models have token limits, use a length function that counts tokens rather than characters for accurate chunk sizing.

## Variations

### Error Handling Pattern

Handle errors gracefully during loading:

```go
docs, err := loader.Load(ctx)
if err != nil {
    if loaderErr, ok := err.(*documentloaders.LoaderError); ok {
        switch loaderErr.Code {
        case documentloaders.ErrCodeFileTooLarge:
            log.Printf("Skipped large file: %s", loaderErr.Path)
        case documentloaders.ErrCodeBinaryFile:
            log.Printf("Skipped binary file: %s", loaderErr.Path)
        default:
            log.Printf("Error loading: %v", loaderErr)
        }
    }
    return err
}
```

### Processing Different File Types

Handle different document types with appropriate splitters:

```go
var textDocs, markdownDocs []schema.Document
for _, doc := range docs {
    if strings.HasSuffix(doc.Metadata["source"], ".md") {
        markdownDocs = append(markdownDocs, doc)
    } else {
        textDocs = append(textDocs, doc)
    }
}

textSplitter, err := textsplitters.NewRecursiveCharacterTextSplitter()
if err != nil {
    log.Fatalf("Failed to create text splitter: %v", err)
}
textChunks, err := textSplitter.SplitDocuments(ctx, textDocs)
if err != nil {
    log.Fatalf("Failed to split text docs: %v", err)
}

markdownSplitter, err := textsplitters.NewMarkdownTextSplitter()
if err != nil {
    log.Fatalf("Failed to create markdown splitter: %v", err)
}
markdownChunks, err := markdownSplitter.SplitDocuments(ctx, markdownDocs)
if err != nil {
    log.Fatalf("Failed to split markdown docs: %v", err)
}
```

### Chunk Metadata Preservation

Access chunk metadata after splitting:

```go
chunks, err := splitter.SplitDocuments(ctx, docs)
if err != nil {
    log.Fatalf("Failed to split: %v", err)
}

for _, chunk := range chunks {
    source := chunk.Metadata["source"]
    index := chunk.Metadata["chunk_index"]
    total := chunk.Metadata["chunk_total"]

    fmt.Printf("Chunk %s/%s from %s\n", index, total, source)
}
```

### Batch Processing

Process documents in batches:

```go
const batchSize = 100
for i := 0; i < len(docs); i += batchSize {
    end := i + batchSize
    if end > len(docs) {
        end = len(docs)
    }

    batch := docs[i:end]
    chunks, err := splitter.SplitDocuments(ctx, batch)
    if err != nil {
        log.Printf("Failed to split batch %d: %v", i/batchSize, err)
        continue
    }

    processBatch(chunks)
}
```

### Registry Pattern

Create loaders dynamically using the registry:

```go
registry := documentloaders.GetRegistry()

loader, err := registry.Create("directory", map[string]any{
    "max_depth":   2,
    "extensions":  []string{".txt", ".md"},
    "concurrency": 4,
})
if err != nil {
    log.Fatalf("Failed to create loader: %v", err)
}
docs, err := loader.Load(ctx)
if err != nil {
    log.Fatalf("Failed to load: %v", err)
}
```

## Related Recipes

- [Parallel File Loading](/cookbook/parallel-file-loading) — Parallel directory traversal with worker pools
- [Corrupt Document Handling](/cookbook/corrupt-doc-handling) — Graceful error handling for corrupt documents
- [Sentence-Aware Splitting](/cookbook/sentence-splitting) — Sentence-boundary-aware text splitting
- [Code Splitting](/cookbook/code-splitting) — Tree-sitter-based code splitting
