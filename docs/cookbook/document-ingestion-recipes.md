# Document Ingestion Recipes

Common patterns and recipes for loading and processing documents in Beluga AI.

## Basic Directory Loading

Load all text files from a directory:

```go
loader, _ := documentloaders.NewDirectoryLoader(
    os.DirFS("./data"),
    documentloaders.WithExtensions(".txt", ".md"),
)
docs, _ := loader.Load(ctx)
```

## Recursive Loading with Depth Limit

Limit directory traversal depth:

```go
loader, _ := documentloaders.NewDirectoryLoader(
    os.DirFS("./docs"),
    documentloaders.WithMaxDepth(3), // Only 3 levels deep
    documentloaders.WithExtensions(".md"),
)
docs, _ := loader.Load(ctx)
```

## Filtering by File Size

Skip files larger than a threshold:

```go
loader, _ := documentloaders.NewDirectoryLoader(
    os.DirFS("./data"),
    documentloaders.WithMaxFileSize(10*1024*1024), // 10MB limit
)
docs, _ := loader.Load(ctx)
```

## Concurrent Loading

Use multiple workers for faster loading:

```go
loader, _ := documentloaders.NewDirectoryLoader(
    os.DirFS("./large_dataset"),
    documentloaders.WithConcurrency(8), // 8 parallel workers
)
docs, _ := loader.Load(ctx)
```

## Lazy Loading for Large Datasets

Stream documents one at a time:

```go
ch, _ := loader.LazyLoad(ctx)

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

## Basic Text Splitting

Split documents into chunks:

```go
splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithRecursiveChunkSize(1000),
    textsplitters.WithRecursiveChunkOverlap(200),
)
chunks, _ := splitter.SplitDocuments(ctx, docs)
```

## Markdown-Aware Splitting

Preserve markdown structure:

```go
splitter, _ := textsplitters.NewMarkdownTextSplitter(
    textsplitters.WithMarkdownChunkSize(500),
    textsplitters.WithHeadersToSplitOn("#", "##", "###"),
)
chunks, _ := splitter.SplitDocuments(ctx, markdownDocs)
```

## Token-Based Splitting

Use token counting for accurate chunk sizing:

```go
tokenizer := func(text string) int {
    // Use your tokenizer (e.g., tiktoken)
    return len(strings.Fields(text)) // Simple word count
}

splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithRecursiveLengthFunction(tokenizer),
    textsplitters.WithRecursiveChunkSize(100), // 100 tokens
    textsplitters.WithRecursiveChunkOverlap(20), // 20 token overlap
)
chunks, _ := splitter.SplitDocuments(ctx, docs)
```

## Complete Ingestion Pipeline

Load, split, and prepare for RAG:

```go
// 1. Load
loader, _ := documentloaders.NewDirectoryLoader(
    os.DirFS("./data"),
    documentloaders.WithExtensions(".txt", ".md"),
    documentloaders.WithMaxDepth(2),
)
docs, _ := loader.Load(ctx)

// 2. Split
splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithRecursiveChunkSize(1000),
    textsplitters.WithRecursiveChunkOverlap(200),
)
chunks, _ := splitter.SplitDocuments(ctx, docs)

// 3. Ready for embedding and storage
```

## Loading Multiple Sources

Combine documents from different sources:

```go
var allDocs []schema.Document

// Load from directory
dirLoader, _ := documentloaders.NewDirectoryLoader(os.DirFS("./docs"))
dirDocs, _ := dirLoader.Load(ctx)
allDocs = append(allDocs, dirDocs...)

// Load single file
fileLoader, _ := documentloaders.NewTextLoader("./important.txt")
fileDocs, _ := fileLoader.Load(ctx)
allDocs = append(allDocs, fileDocs...)

// Process all together
splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter()
chunks, _ := splitter.SplitDocuments(ctx, allDocs)
```

## Error Handling Pattern

Handle errors gracefully:

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

## Registry Pattern

Create loaders dynamically:

```go
registry := documentloaders.GetRegistry()

// Create loader from config
loader, _ := registry.Create("directory", map[string]any{
    "max_depth":  2,
    "extensions": []string{".txt", ".md"},
    "concurrency": 4,
})
docs, _ := loader.Load(ctx)
```

## Processing Different File Types

Handle different document types appropriately:

```go
// Load all files
loader, _ := documentloaders.NewDirectoryLoader(os.DirFS("./data"))
docs, _ := loader.Load(ctx)

// Separate by type
var textDocs, markdownDocs []schema.Document
for _, doc := range docs {
    if strings.HasSuffix(doc.Metadata["source"], ".md") {
        markdownDocs = append(markdownDocs, doc)
    } else {
        textDocs = append(textDocs, doc)
    }
}

// Use appropriate splitters
textSplitter, _ := textsplitters.NewRecursiveCharacterTextSplitter()
textChunks, _ := textSplitter.SplitDocuments(ctx, textDocs)

markdownSplitter, _ := textsplitters.NewMarkdownTextSplitter()
markdownChunks, _ := markdownSplitter.SplitDocuments(ctx, markdownDocs)
```

## Chunk Metadata Preservation

Access chunk metadata:

```go
chunks, _ := splitter.SplitDocuments(ctx, docs)

for _, chunk := range chunks {
    source := chunk.Metadata["source"]
    index := chunk.Metadata["chunk_index"]
    total := chunk.Metadata["chunk_total"]
    
    fmt.Printf("Chunk %s/%s from %s\n", index, total, source)
}
```

## Batch Processing

Process documents in batches:

```go
const batchSize = 100
for i := 0; i < len(docs); i += batchSize {
    end := i + batchSize
    if end > len(docs) {
        end = len(docs)
    }
    
    batch := docs[i:end]
    chunks, _ := splitter.SplitDocuments(ctx, batch)
    
    // Process batch
    processBatch(chunks)
}
```

## Related Recipes

- [RAG Recipes](./rag-recipes.md) - Complete RAG pipeline patterns
- [Integration Recipes](./integration-recipes.md) - Integration patterns
