# Document Ingestion Tutorial

This tutorial shows you how to load documents from files and directories, split them into chunks, and prepare them for RAG pipelines.

## Prerequisites

- Go 1.21 or later
- Beluga AI Framework installed
- Basic understanding of RAG concepts (see [Simple RAG Tutorial](./02-simple-rag.md))

## Step 1: Load Documents from a Directory

First, let's load documents from a directory:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    
    "github.com/lookatitude/beluga-ai/pkg/documentloaders"
)

func main() {
    ctx := context.Background()
    
    // Create a directory loader
    fsys := os.DirFS("./data")
    loader, err := documentloaders.NewDirectoryLoader(fsys,
        documentloaders.WithMaxDepth(2),              // Traverse 2 levels deep
        documentloaders.WithExtensions(".txt", ".md"), // Only text and markdown files
        documentloaders.WithConcurrency(4),           // Use 4 parallel workers
    )
    if err != nil {
        log.Fatalf("Failed to create loader: %v", err)
    }
    
    // Load all documents
    docs, err := loader.Load(ctx)
    if err != nil {
        log.Fatalf("Failed to load documents: %v", err)
    }
    
    fmt.Printf("Loaded %d documents\n", len(docs))
    for i, doc := range docs {
        if i >= 5 { // Show first 5
            fmt.Printf("... and %d more\n", len(docs)-5)
            break
        }
        fmt.Printf("  - %s (%d bytes)\n", 
            doc.Metadata["source"], 
            len(doc.PageContent),
        )
    }
}
```

**Create a test directory:**
```bash
mkdir -p data
echo "This is document 1" > data/doc1.txt
echo "This is document 2" > data/doc2.txt
echo "# Markdown Document" > data/doc3.md
```

## Step 2: Load a Single File

For single files, use `TextLoader`:

```go
loader, err := documentloaders.NewTextLoader("./document.txt")
if err != nil {
    log.Fatalf("Failed to create loader: %v", err)
}

docs, err := loader.Load(ctx)
if err != nil {
    log.Fatalf("Failed to load: %v", err)
}

fmt.Printf("Loaded document: %s\n", docs[0].Metadata["source"])
```

## Step 3: Split Documents into Chunks

Large documents need to be split into smaller chunks:

```go
import "github.com/lookatitude/beluga-ai/pkg/textsplitters"

// Create a splitter
splitter, err := textsplitters.NewRecursiveCharacterTextSplitter(
    textsplitters.WithRecursiveChunkSize(1000),      // 1000 characters per chunk
    textsplitters.WithRecursiveChunkOverlap(200),    // 200 character overlap
)
if err != nil {
    log.Fatalf("Failed to create splitter: %v", err)
}

// Split documents
chunks, err := splitter.SplitDocuments(ctx, docs)
if err != nil {
    log.Fatalf("Failed to split: %v", err)
}

fmt.Printf("Split %d documents into %d chunks\n", len(docs), len(chunks))
```

## Step 4: Complete Pipeline

Combine loading and splitting:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    
    "github.com/lookatitude/beluga-ai/pkg/documentloaders"
    "github.com/lookatitude/beluga-ai/pkg/textsplitters"
)

func main() {
    ctx := context.Background()
    
    // 1. Load documents
    fmt.Println("Loading documents...")
    loader, _ := documentloaders.NewDirectoryLoader(
        os.DirFS("./data"),
        documentloaders.WithExtensions(".txt", ".md"),
    )
    docs, _ := loader.Load(ctx)
    fmt.Printf("✅ Loaded %d documents\n", len(docs))
    
    // 2. Split into chunks
    fmt.Println("Splitting documents...")
    splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter(
        textsplitters.WithRecursiveChunkSize(500),
        textsplitters.WithRecursiveChunkOverlap(50),
    )
    chunks, _ := splitter.SplitDocuments(ctx, docs)
    fmt.Printf("✅ Created %d chunks\n", len(chunks))
    
    // 3. Display chunk information
    for i, chunk := range chunks {
        if i >= 3 {
            fmt.Printf("... and %d more chunks\n", len(chunks)-3)
            break
        }
        fmt.Printf("Chunk %d: %s (from %s)\n",
            i+1,
            chunk.Metadata["chunk_index"],
            chunk.Metadata["source"],
        )
        fmt.Printf("  Preview: %.100s...\n", chunk.PageContent)
    }
}
```

## Step 5: Markdown-Aware Splitting

For markdown documents, use `MarkdownTextSplitter`:

```go
splitter, err := textsplitters.NewMarkdownTextSplitter(
    textsplitters.WithMarkdownChunkSize(500),
    textsplitters.WithMarkdownChunkOverlap(50),
    textsplitters.WithHeadersToSplitOn("#", "##", "###"),
)
if err != nil {
    log.Fatal(err)
}

chunks, err := splitter.SplitDocuments(ctx, markdownDocs)
```

## Step 6: Lazy Loading for Large Datasets

For large datasets, use `LazyLoad()` to stream documents:

```go
ch, err := loader.LazyLoad(ctx)
if err != nil {
    log.Fatal(err)
}

count := 0
for item := range ch {
    switch v := item.(type) {
    case error:
        log.Printf("Error: %v", v)
    case schema.Document:
        count++
        // Process document
        processDocument(v)
    }
}
fmt.Printf("Processed %d documents\n", count)
```

## Step 7: Integration with RAG Pipeline

Complete example integrating with embeddings and vector store:

```go
package main

import (
    "context"
    "log"
    "os"
    
    "github.com/lookatitude/beluga-ai/pkg/documentloaders"
    "github.com/lookatitude/beluga-ai/pkg/embeddings"
    "github.com/lookatitude/beluga-ai/pkg/textsplitters"
    "github.com/lookatitude/beluga-ai/pkg/vectorstores"
)

func main() {
    ctx := context.Background()
    
    // 1. Load documents
    loader, _ := documentloaders.NewDirectoryLoader(
        os.DirFS("./data"),
        documentloaders.WithExtensions(".txt", ".md"),
    )
    docs, _ := loader.Load(ctx)
    
    // 2. Split into chunks
    splitter, _ := textsplitters.NewRecursiveCharacterTextSplitter(
        textsplitters.WithRecursiveChunkSize(1000),
        textsplitters.WithRecursiveChunkOverlap(200),
    )
    chunks, _ := splitter.SplitDocuments(ctx, docs)
    
    // 3. Create embedder
    embedder, _ := embeddings.NewEmbedder(ctx, "mock", embeddings.Config{})
    
    // 4. Create vector store
    store, _ := vectorstores.NewInMemoryStore(ctx,
        vectorstores.WithEmbedder(embedder),
    )
    
    // 5. Add chunks to vector store
    ids, _ := store.AddDocuments(ctx, chunks)
    log.Printf("Added %d chunks to vector store\n", len(ids))
    
    // 6. Query
    query := "What is the main topic?"
    results, _, _ := store.SimilaritySearchByQuery(ctx, query, 5, embedder)
    log.Printf("Found %d relevant chunks\n", len(results))
}
```

## Configuration Options

### Directory Loader Options

- `WithMaxDepth(n)`: Limit directory traversal depth
- `WithExtensions(...exts)`: Filter by file extensions
- `WithConcurrency(n)`: Number of parallel workers
- `WithMaxFileSize(bytes)`: Skip files larger than limit
- `WithFollowSymlinks(bool)`: Enable symlink following

### Text Splitter Options

- `WithRecursiveChunkSize(n)`: Target chunk size in characters
- `WithRecursiveChunkOverlap(n)`: Overlap between chunks
- `WithSeparators(...seps)`: Custom separator hierarchy
- `WithRecursiveLengthFunction(fn)`: Token-based splitting

## Best Practices

1. **Choose appropriate chunk size**: Match your embedding model's context window
2. **Use overlap**: 10-20% overlap preserves context across boundaries
3. **Filter by extension**: Only load relevant file types
4. **Set file size limits**: Prevent loading extremely large files
5. **Use markdown splitter**: For markdown documents, preserve structure

## Next Steps

- Learn about [RAG Concepts](../concepts/rag.md)
- Explore [Document Loading Concepts](../concepts/document-loading.md)
- Read [Text Splitting Concepts](../concepts/text-splitting.md)
- See [Complete Examples](../../examples/documentloaders/) and [RAG Examples](../../examples/rag/)
