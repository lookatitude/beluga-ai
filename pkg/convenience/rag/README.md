# RAG Package

The rag package provides a simplified API for building RAG (Retrieval-Augmented Generation) pipelines. It reduces the boilerplate typically required to set up document loading, embedding, storage, and retrieval.

> **Note**: This package is a work in progress. For production use, please use the individual packages directly (embeddings, vectorstores, llms, documentloaders).

## Features

- **Fluent Builder Pattern**: Chain configuration methods for easy pipeline setup
- **Document Source Management**: Configure multiple document sources with extensions
- **Chunking Configuration**: Set chunk size and overlap for text splitting
- **Retrieval Settings**: Configure top-K retrieval parameters

## Usage

### Basic Builder Usage

```go
import "github.com/lookatitude/beluga-ai/pkg/convenience/rag"

// Create a new RAG pipeline builder
builder := rag.NewBuilder().
    WithDocumentSource("./docs", "md", "txt").
    WithDocumentSource("./data", "json").
    WithTopK(10).
    WithChunkSize(500).
    WithOverlap(100)

// Access configuration
fmt.Println(builder.GetDocPaths())    // ["./docs", "./data"]
fmt.Println(builder.GetExtensions())  // ["md", "txt", "json"]
fmt.Println(builder.GetTopK())        // 10
fmt.Println(builder.GetChunkSize())   // 500
fmt.Println(builder.GetOverlap())     // 100
```

## Configuration Options

### Document Sources
```go
builder.WithDocumentSource("./docs", "md", "txt", "pdf")
```

### Top-K Retrieval
```go
builder.WithTopK(10)  // Number of documents to retrieve (default: 5)
```

### Chunk Size
```go
builder.WithChunkSize(500)  // Characters per chunk (default: 1000)
```

### Chunk Overlap
```go
builder.WithOverlap(100)  // Overlap between chunks (default: 200)
```

## Default Values

- **Top-K**: 5
- **Chunk Size**: 1000
- **Overlap**: 200

## Future API (Planned)

The intended future API will look like:

```go
pipeline, err := rag.NewBuilder().
    WithDocumentSource("./docs", "md", "txt").
    WithEmbedder("openai").
    WithVectorStore("pgvector").
    WithLLM("openai").
    Build(ctx)

answer, sources, err := pipeline.Query(ctx, "What is X?")
```

## Production Usage

For production use, compose the individual packages:

```go
import (
    "github.com/lookatitude/beluga-ai/pkg/embeddings"
    "github.com/lookatitude/beluga-ai/pkg/vectorstores"
    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/documentloaders"
    "github.com/lookatitude/beluga-ai/pkg/textsplitters"
)

// Load documents
loader := documentloaders.NewTextLoader("./docs")
docs, err := loader.Load(ctx)

// Split into chunks
splitter := textsplitters.NewRecursiveCharacterSplitter(
    textsplitters.WithChunkSize(1000),
    textsplitters.WithChunkOverlap(200),
)
chunks := splitter.SplitDocuments(docs)

// Create embedder and vector store
embedder, err := embeddings.NewProvider(ctx, "openai", config)
store, err := vectorstores.NewProvider(ctx, "memory", storeConfig)

// Add documents to store
store.AddDocuments(ctx, chunks, vectorstores.WithEmbedder(embedder))

// Query
results, scores, err := store.SimilaritySearchByQuery(ctx, "query", 5, embedder)
```
