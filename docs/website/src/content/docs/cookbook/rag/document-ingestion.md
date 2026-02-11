---
title: "Document Ingestion Recipes"
description: "Common patterns for loading, splitting, and processing documents in Beluga AI RAG pipelines, from basic directory loading to complete ingestion pipelines."
---

Document ingestion is the first stage of any RAG pipeline: getting documents from their source format into chunks that can be embedded and stored. The quality of ingestion directly affects retrieval quality downstream. Poorly split documents produce poor embeddings; missing metadata means no filtering capability; sequential loading wastes time on large collections.

## Problem

You need to load documents from various sources, split them into chunks, and prepare them for embedding and storage in a RAG pipeline. Different file types, directory structures, and processing strategies require flexible ingestion patterns.

## Solution

Use Beluga AI's document loaders and text splitters to build composable ingestion pipelines. The framework provides `DocumentLoader` and `TextSplitter` interfaces with multiple implementations, each configurable via the `WithX()` functional options pattern. Combine directory loading, extension filtering, concurrency, and type-aware splitting to handle diverse document sources efficiently.

## Code Example

### Basic Directory Loading

Load all text files from a directory. The `WithExtensions` option filters by file type, avoiding binary files that would produce garbage chunks:

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

Limit directory traversal depth to avoid accidentally traversing deep dependency trees (e.g., `node_modules` or `.git` directories):

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

Skip files larger than a threshold. Very large files (logs, data dumps) can overwhelm the splitter and produce too many chunks. Setting a size limit keeps ingestion predictable:

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

Use multiple workers for faster loading. Document loading is I/O-bound, so concurrency significantly improves throughput on large directories by overlapping disk reads:

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

Stream documents one at a time to avoid loading the entire collection into memory at once. This is important when the total document size exceeds available RAM:

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

Split documents into chunks. The `RecursiveCharacterTextSplitter` tries increasingly fine-grained separators (paragraphs, then sentences, then words) to find the best split point near the target size:

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

Preserve markdown structure by splitting at heading boundaries. This ensures each chunk corresponds to a logical section, making retrieval results more coherent and preserving the document's organizational hierarchy in the chunk metadata:

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

Use token counting for accurate chunk sizing. Character counts are a poor proxy for tokens because different words have different tokenization lengths. When your embedding model has a strict token limit, counting tokens directly prevents truncation errors:

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

Load, split, and prepare for RAG. This three-stage pipeline (load, split, embed/store) is the standard pattern for document ingestion:

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

Combine documents from different sources into a single collection for uniform splitting and embedding. This is common when your knowledge base spans multiple directories, individual files, or external sources:

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

These recipes demonstrate the core document ingestion patterns in Beluga AI:

1. **Directory loading with functional options** -- Use `NewDirectoryLoader` with `WithX()` options to control file discovery. Options like `WithExtensions`, `WithMaxDepth`, and `WithMaxFileSize` allow fine-grained control over which files are loaded. This follows Beluga AI's standard functional options pattern for configuration.

2. **Concurrent loading** -- For large datasets, `WithConcurrency` distributes file loading across multiple goroutines. This significantly improves throughput for I/O-bound loading because Go's goroutine scheduler efficiently multiplexes concurrent reads.

3. **Type-aware splitting** -- Different document types benefit from different splitting strategies. Markdown documents should use `MarkdownTextSplitter` to preserve heading structure (which becomes metadata), while plain text uses `RecursiveCharacterTextSplitter`. Code files benefit from the language-aware `CodeSplitter`. Choosing the right splitter for each content type is one of the highest-leverage decisions in a RAG pipeline.

4. **Token-based sizing** -- When embedding models have token limits (most do), use a length function that counts tokens rather than characters for accurate chunk sizing. This prevents silent truncation where the embedding model only sees part of each chunk, degrading embedding quality without any error signal.

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

Process documents in batches to control memory usage:

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

Create loaders dynamically using the registry, which allows selecting the loader type from configuration without hardcoding the implementation:

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

- [Parallel File Loading](/cookbook/parallel-file-loading) -- Parallel directory traversal with worker pools
- [Corrupt Document Handling](/cookbook/corrupt-doc-handling) -- Graceful error handling for corrupt documents
- [Sentence-Aware Splitting](/cookbook/sentence-splitting) -- Sentence-boundary-aware text splitting
- [Code Splitting](/cookbook/code-splitting) -- Tree-sitter-based code splitting
