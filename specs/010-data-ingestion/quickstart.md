# Quickstart: Data Ingestion and Processing

**Feature**: 010-data-ingestion  
**Date**: 2026-01-11

This guide demonstrates how to use the `documentloaders` and `textsplitters` packages to build a RAG data ingestion pipeline.

## Installation

The packages are included in Beluga AI. Import them in your Go code:

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/documentloaders"
    "github.com/lookatitude/beluga-ai/pkg/textsplitters"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)
```

## Quick Examples

### Load a Single Text File

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/pkg/documentloaders"
)

func main() {
    ctx := context.Background()

    // Create a text loader for a single file
    loader, err := documentloaders.NewTextLoader("/path/to/document.txt")
    if err != nil {
        log.Fatalf("Failed to create loader: %v", err)
    }

    // Load the document
    docs, err := loader.Load(ctx)
    if err != nil {
        log.Fatalf("Failed to load document: %v", err)
    }

    // Print document info
    for _, doc := range docs {
        fmt.Printf("Source: %s\n", doc.Metadata["source"])
        fmt.Printf("Content length: %d characters\n", len(doc.PageContent))
    }
}
```

### Load Documents from a Directory

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

    // Create a directory loader with options
    loader, err := documentloaders.NewDirectoryLoader(
        os.DirFS("/path/to/documents"),
        documentloaders.WithMaxDepth(3),              // Limit recursion
        documentloaders.WithExtensions(".txt", ".md"), // Only text and markdown
        documentloaders.WithConcurrency(4),           // 4 parallel workers
        documentloaders.WithMaxFileSize(50*1024*1024), // 50MB limit
    )
    if err != nil {
        log.Fatalf("Failed to create loader: %v", err)
    }

    // Load all matching documents
    docs, err := loader.Load(ctx)
    if err != nil {
        log.Fatalf("Failed to load documents: %v", err)
    }

    fmt.Printf("Loaded %d documents\n", len(docs))
    for i, doc := range docs {
        fmt.Printf("  [%d] %s (%d chars)\n", i, doc.Metadata["source"], len(doc.PageContent))
    }
}
```

### Split Text into Chunks

```go
package main

import (
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/pkg/textsplitters"
)

func main() {
    // Create a recursive character splitter
    splitter, err := textsplitters.NewRecursiveCharacterTextSplitter(
        textsplitters.WithChunkSize(1000),   // ~1000 characters per chunk
        textsplitters.WithChunkOverlap(200), // 200 character overlap
    )
    if err != nil {
        log.Fatalf("Failed to create splitter: %v", err)
    }

    // Split some text
    text := `Long document content here...
    
    This is paragraph one with lots of content.
    
    This is paragraph two with more content.
    
    And so on...`

    chunks, err := splitter.SplitText(text)
    if err != nil {
        log.Fatalf("Failed to split text: %v", err)
    }

    fmt.Printf("Split into %d chunks\n", len(chunks))
    for i, chunk := range chunks {
        fmt.Printf("  Chunk %d: %d characters\n", i, len(chunk))
    }
}
```

### Split Documents (Preserving Metadata)

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

    // Step 1: Load documents
    loader, _ := documentloaders.NewDirectoryLoader(
        os.DirFS("/path/to/docs"),
        documentloaders.WithExtensions(".md"),
    )
    docs, err := loader.Load(ctx)
    if err != nil {
        log.Fatalf("Failed to load: %v", err)
    }

    // Step 2: Split documents
    splitter, _ := textsplitters.NewMarkdownTextSplitter(
        textsplitters.WithChunkSize(1500),
        textsplitters.WithChunkOverlap(100),
    )
    chunks, err := splitter.SplitDocuments(docs)
    if err != nil {
        log.Fatalf("Failed to split: %v", err)
    }

    // Each chunk preserves source metadata + chunk info
    fmt.Printf("Split %d docs into %d chunks\n", len(docs), len(chunks))
    for i, chunk := range chunks[:3] { // Show first 3
        fmt.Printf("  Chunk %d:\n", i)
        fmt.Printf("    Source: %s\n", chunk.Metadata["source"])
        fmt.Printf("    Chunk: %s/%s\n", chunk.Metadata["chunk_index"], chunk.Metadata["chunk_total"])
        fmt.Printf("    Size: %d chars\n", len(chunk.PageContent))
    }
}
```

## Complete RAG Pipeline Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/documentloaders"
    "github.com/lookatitude/beluga-ai/pkg/embeddings"
    "github.com/lookatitude/beluga-ai/pkg/textsplitters"
    "github.com/lookatitude/beluga-ai/pkg/vectorstores"
)

func main() {
    ctx := context.Background()

    // 1. Load documents from knowledge base
    loader, err := documentloaders.NewDirectoryLoader(
        os.DirFS("./knowledge_base"),
        documentloaders.WithExtensions(".txt", ".md"),
        documentloaders.WithConcurrency(8),
    )
    if err != nil {
        log.Fatalf("Loader creation failed: %v", err)
    }

    docs, err := loader.Load(ctx)
    if err != nil {
        log.Fatalf("Document loading failed: %v", err)
    }
    fmt.Printf("✓ Loaded %d documents\n", len(docs))

    // 2. Split documents into chunks
    splitter, err := textsplitters.NewRecursiveCharacterTextSplitter(
        textsplitters.WithChunkSize(1000),
        textsplitters.WithChunkOverlap(200),
    )
    if err != nil {
        log.Fatalf("Splitter creation failed: %v", err)
    }

    chunks, err := splitter.SplitDocuments(docs)
    if err != nil {
        log.Fatalf("Document splitting failed: %v", err)
    }
    fmt.Printf("✓ Split into %d chunks\n", len(chunks))

    // 3. Create embeddings (example with OpenAI)
    embedder, err := embeddings.NewOpenAIEmbedder(
        embeddings.WithModel("text-embedding-3-small"),
    )
    if err != nil {
        log.Fatalf("Embedder creation failed: %v", err)
    }

    // 4. Store in vector database (example with Pinecone)
    store, err := vectorstores.NewPinecone(
        vectorstores.WithIndex("my-index"),
        vectorstores.WithEmbedder(embedder),
    )
    if err != nil {
        log.Fatalf("Vectorstore creation failed: %v", err)
    }

    // 5. Add chunks to vectorstore
    err = store.AddDocuments(ctx, chunks)
    if err != nil {
        log.Fatalf("Document storage failed: %v", err)
    }
    fmt.Printf("✓ Stored %d chunks in vectorstore\n", len(chunks))

    // Pipeline complete - ready for retrieval!
}
```

## Using the Registry

For dynamic loader/splitter selection:

```go
package main

import (
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/pkg/documentloaders"
    "github.com/lookatitude/beluga-ai/pkg/textsplitters"
)

func main() {
    // List available loaders
    loaderRegistry := documentloaders.GetRegistry()
    fmt.Println("Available loaders:", loaderRegistry.List())
    // Output: Available loaders: [text directory]

    // List available splitters
    splitterRegistry := textsplitters.GetRegistry()
    fmt.Println("Available splitters:", splitterRegistry.List())
    // Output: Available splitters: [recursive_character markdown]

    // Create loader by name
    loader, err := loaderRegistry.Create("text", map[string]any{
        "path": "/path/to/file.txt",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created loader: %T\n", loader)

    // Create splitter by name
    splitter, err := splitterRegistry.Create("recursive_character", map[string]any{
        "chunk_size":    1000,
        "chunk_overlap": 200,
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created splitter: %T\n", splitter)
}
```

## Error Handling

Handle errors gracefully with typed errors:

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/pkg/documentloaders"
)

func main() {
    ctx := context.Background()

    loader, _ := documentloaders.NewTextLoader("/nonexistent/file.txt")
    _, err := loader.Load(ctx)
    
    if err != nil {
        // Check for specific error type
        var loaderErr *documentloaders.LoaderError
        if errors.As(err, &loaderErr) {
            switch loaderErr.Code {
            case documentloaders.ErrCodeNotFound:
                fmt.Printf("File not found: %s\n", loaderErr.Path)
            case documentloaders.ErrCodeIOError:
                fmt.Printf("IO error reading %s: %v\n", loaderErr.Path, loaderErr.Err)
            case documentloaders.ErrCodeFileTooLarge:
                fmt.Printf("File too large: %s\n", loaderErr.Path)
            default:
                fmt.Printf("Loader error (%s): %v\n", loaderErr.Code, loaderErr)
            }
        } else {
            log.Fatalf("Unexpected error: %v", err)
        }
    }
}
```

## With OTEL Tracing

Enable observability with OpenTelemetry:

```go
package main

import (
    "context"
    "os"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/sdk/trace"

    "github.com/lookatitude/beluga-ai/pkg/documentloaders"
)

func main() {
    // Setup OTEL exporter (example: Jaeger/OTLP)
    exporter, _ := otlptracegrpc.New(context.Background())
    tp := trace.NewTracerProvider(trace.WithBatcher(exporter))
    otel.SetTracerProvider(tp)
    defer tp.Shutdown(context.Background())

    // Create context with tracing
    ctx := context.Background()

    // All Load() and SplitDocuments() calls now emit traces
    loader, _ := documentloaders.NewDirectoryLoader(
        os.DirFS("./docs"),
        documentloaders.WithExtensions(".txt"),
    )
    
    // This creates a span: documentloaders.directory.Load
    docs, _ := loader.Load(ctx)
    
    // View traces in Jaeger/Tempo/etc.
    _ = docs
}
```

## Next Steps

1. **Integration with embeddings**: See `pkg/embeddings` for creating vector representations
2. **Vector storage**: See `pkg/vectorstores` for persisting embeddings
3. **Retrieval**: See `pkg/retrievers` for similarity search
4. **Custom loaders**: Implement `DocumentLoader` interface and register with the registry
5. **Token-based splitting**: Use `WithLengthFunction` with a tokenizer for accurate token counting
