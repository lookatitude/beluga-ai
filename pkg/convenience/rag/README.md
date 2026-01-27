# RAG Package

The rag package provides a simplified API for building RAG (Retrieval-Augmented Generation) pipelines. It reduces the boilerplate typically required to set up document loading, embedding, storage, and retrieval.

## Features

- **Fluent Builder Pattern**: Chain configuration methods for easy pipeline setup
- **Document Management**: Load, chunk, and index documents from multiple sources
- **Similarity Search**: Retrieve relevant documents with configurable top-K and score threshold
- **LLM Integration**: Optional query answering with source attribution
- **OpenTelemetry Integration**: Full observability with metrics and tracing
- **Structured Errors**: Op/Err/Code error pattern for clear error handling

## Installation

```go
import "github.com/lookatitude/beluga-ai/pkg/convenience/rag"
```

## Quick Start

```go
import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/convenience/rag"
    "github.com/lookatitude/beluga-ai/pkg/embeddings/providers/openai"
)

// Create an embedder
embedder, _ := openai.NewOpenAIEmbedder(ctx, openai.WithAPIKey("your-key"))

// Build the pipeline
pipeline, err := rag.NewBuilder().
    WithDocumentSource("./docs", "md", "txt").
    WithEmbedder(embedder).
    WithInMemoryVectorStore().
    WithTopK(5).
    Build(ctx)
if err != nil {
    log.Fatal(err)
}

// Ingest documents
err = pipeline.IngestDocuments(ctx)

// Search for relevant documents
docs, scores, err := pipeline.Search(ctx, "How do I configure X?", 5)
```

## Builder API

### Creating a Builder

```go
builder := rag.NewBuilder()
```

### Configuration Methods

#### Document Sources

```go
// Add document source with file extensions
builder.WithDocumentSource("./docs", "md", "txt")
builder.WithDocumentSource("./data", "json", "yaml")
```

#### Chunking Configuration

```go
builder.WithChunkSize(500)    // Characters per chunk (default: 1000)
builder.WithOverlap(100)      // Overlap between chunks (default: 200)
```

#### Embedder Configuration

```go
// Use an embedder instance
builder.WithEmbedder(embedder)

// Use provider-based resolution (not yet implemented)
builder.WithEmbedderProvider("openai", "api-key")
```

#### Vector Store Configuration

```go
// Use in-memory vector store (default)
builder.WithInMemoryVectorStore()

// Use a pre-configured vector store
builder.WithVectorStore(customStore)
```

#### LLM Configuration (Optional)

```go
// Use an LLM for query answering
builder.WithLLM(chatModel)

// Use provider-based resolution (not yet implemented)
builder.WithLLMProvider("openai", "api-key", "gpt-4")
```

#### Retrieval Configuration

```go
builder.WithTopK(10)              // Number of documents to retrieve (default: 5)
builder.WithScoreThreshold(0.7)   // Minimum similarity score (default: 0.0)
builder.WithReturnSources(true)   // Include source documents in responses (default: true)
```

#### System Prompt

```go
builder.WithSystemPrompt("Answer questions using the provided context...")
```

#### Metrics Configuration

```go
builder.WithMetrics(customMetrics)
```

### Building the Pipeline

```go
pipeline, err := builder.Build(ctx)
if err != nil {
    // Handle error
}
```

## Pipeline Interface

The built pipeline implements the `Pipeline` interface:

```go
type Pipeline interface {
    // Document operations
    IngestDocuments(ctx context.Context) error
    IngestFromPaths(ctx context.Context, paths []string) error
    AddDocuments(ctx context.Context, docs []schema.Document) error

    // Search operations
    Search(ctx context.Context, query string, k int) ([]schema.Document, []float32, error)

    // Query operations (requires LLM)
    Query(ctx context.Context, query string) (string, error)
    QueryWithSources(ctx context.Context, query string) (string, []schema.Document, error)

    // Management
    GetDocumentCount() int
    Clear(ctx context.Context) error
}
```

### Document Ingestion

```go
// Ingest from configured document sources
err := pipeline.IngestDocuments(ctx)

// Ingest from specific paths
err := pipeline.IngestFromPaths(ctx, []string{"./additional/docs"})

// Add documents directly
docs := []schema.Document{
    rag.NewDocument("Content here", map[string]string{"source": "manual"}),
}
err := pipeline.AddDocuments(ctx, docs)
```

### Searching Documents

```go
// Search for relevant documents
docs, scores, err := pipeline.Search(ctx, "How do I configure X?", 5)
for i, doc := range docs {
    fmt.Printf("Score: %.2f - %s\n", scores[i], doc.PageContent)
}
```

### Querying with LLM

```go
// Simple query (requires LLM to be configured)
answer, err := pipeline.Query(ctx, "What is the purpose of X?")

// Query with source attribution
answer, sources, err := pipeline.QueryWithSources(ctx, "What is the purpose of X?")
fmt.Println("Answer:", answer)
fmt.Println("Sources:")
for _, src := range sources {
    fmt.Printf("- %s\n", src.Metadata["source"])
}
```

### Management

```go
// Get document count
count := pipeline.GetDocumentCount()

// Clear all documents
err := pipeline.Clear(ctx)
```

## Convenience Types

```go
// Document alias for schema.Document
type Document = schema.Document

// Create a new document
doc := rag.NewDocument("Content", map[string]string{"key": "value"})
```

## Error Handling

The package uses structured errors with Op/Err/Code pattern:

```go
pipeline, err := builder.Build(ctx)
if err != nil {
    var ragErr *rag.Error
    if errors.As(err, &ragErr) {
        switch ragErr.Code {
        case rag.ErrCodeMissingEmbedder:
            // No embedder configured
        case rag.ErrCodeEmbedderCreation:
            // Failed to create embedder from provider
        case rag.ErrCodeVectorStoreCreation:
            // Failed to create vector store
        case rag.ErrCodeSplitterCreation:
            // Failed to create text splitter
        }
    }
}
```

### Error Codes

| Code | Description |
|------|-------------|
| `missing_embedder` | No embedder configured |
| `embedder_creation_failed` | Failed to create embedder from provider name |
| `vectorstore_creation_failed` | Failed to create vector store |
| `llm_creation_failed` | Failed to create LLM from provider name |
| `splitter_creation_failed` | Failed to create text splitter |
| `retriever_creation_failed` | Failed to create retriever |
| `retrieval_failed` | Document retrieval failed |
| `generation_failed` | LLM generation failed |
| `no_llm_configured` | Query attempted without LLM |

## Observability

The package includes OpenTelemetry metrics and tracing:

```go
// Get global metrics instance
metrics := rag.GetMetrics()

// Create custom metrics
metrics, err := rag.NewMetrics("custom-prefix")

// Use no-op metrics (for testing)
metrics := rag.NoOpMetrics()
```

### Metrics Recorded

- `rag_builds_total` - Counter for build operations
- `rag_build_duration_seconds` - Histogram for build duration
- `rag_ingestions_total` - Counter for document ingestions
- `rag_ingestion_duration_seconds` - Histogram for ingestion duration
- `rag_searches_total` - Counter for search operations
- `rag_search_duration_seconds` - Histogram for search duration
- `rag_queries_total` - Counter for query operations
- `rag_query_duration_seconds` - Histogram for query duration
- `rag_errors_total` - Counter for errors by type

## Examples

### Basic Retrieval Pipeline

```go
embedder, _ := openai.NewOpenAIEmbedder(ctx, openai.WithAPIKey(apiKey))

pipeline, err := rag.NewBuilder().
    WithDocumentSource("./knowledge-base", "md", "txt").
    WithEmbedder(embedder).
    WithChunkSize(500).
    WithOverlap(50).
    WithTopK(3).
    Build(ctx)

// Index documents
pipeline.IngestDocuments(ctx)

// Search
docs, scores, _ := pipeline.Search(ctx, "deployment instructions", 3)
```

### Full RAG Pipeline with LLM

```go
embedder, _ := openai.NewOpenAIEmbedder(ctx, openai.WithAPIKey(apiKey))
llm, _ := openai.NewOpenAIChatModel(ctx, openai.WithAPIKey(apiKey))

pipeline, err := rag.NewBuilder().
    WithDocumentSource("./docs", "md").
    WithEmbedder(embedder).
    WithLLM(llm).
    WithSystemPrompt("Answer questions based on the provided context. If the answer is not in the context, say so.").
    WithTopK(5).
    WithScoreThreshold(0.5).
    Build(ctx)

// Ingest and query
pipeline.IngestDocuments(ctx)
answer, sources, _ := pipeline.QueryWithSources(ctx, "How do I deploy to production?")
```

### Adding Documents Programmatically

```go
pipeline, _ := rag.NewBuilder().
    WithEmbedder(embedder).
    Build(ctx)

// Add documents from various sources
docs := []schema.Document{
    rag.NewDocument("API documentation content...", map[string]string{
        "source": "api-docs",
        "version": "v2",
    }),
    rag.NewDocument("Tutorial content...", map[string]string{
        "source": "tutorials",
        "level": "beginner",
    }),
}

pipeline.AddDocuments(ctx, docs)
```

## Default Values

| Option | Default |
|--------|---------|
| Top-K | 5 |
| Chunk Size | 1000 |
| Overlap | 200 |
| Score Threshold | 0.0 |
| Return Sources | true |
| Vector Store | In-memory |

## Thread Safety

The built pipeline is safe for concurrent use. Multiple goroutines can perform searches and queries simultaneously.

## See Also

- [pkg/embeddings](../../embeddings/) - Embedding providers
- [pkg/vectorstores](../../vectorstores/) - Vector store providers
- [pkg/llms](../../llms/) - LLM providers
- [pkg/textsplitters](../../textsplitters/) - Text chunking
- [pkg/documentloaders](../../documentloaders/) - Document loading
- [pkg/retrievers](../../retrievers/) - Retriever implementations
