---
title: RAG Package API
description: API documentation for embedding, vector storage, and retrieval.
---

```go
import (
    "github.com/lookatitude/beluga-ai/rag/embedding"
    "github.com/lookatitude/beluga-ai/rag/vectorstore"
    "github.com/lookatitude/beluga-ai/rag/retriever"
)
```

Package rag provides the complete RAG pipeline: embedders convert text to vectors, vector stores persist and search embeddings, and retrievers implement advanced retrieval strategies.

## Quick Start

```go
// Embedder
emb, _ := embedding.New("openai", embedCfg)
vecs, _ := emb.Embed(ctx, []string{"hello", "world"})

// Vector Store
store, _ := vectorstore.New("inmemory", storeCfg)
store.Add(ctx, docs, embeddings)
results, _ := store.Search(ctx, queryVec, 10)

// Retriever
ret := retriever.NewHybridRetriever(store, emb, bm25Searcher)
docs, _ := ret.Retrieve(ctx, "Go concurrency", retriever.WithTopK(10))
```

## Embedding

### Embedder Interface

```go
type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    EmbedSingle(ctx context.Context, text string) ([]float32, error)
    Dimensions() int
}
```

### Usage

```go
embedder, err := embedding.New("openai", config.ProviderConfig{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "text-embedding-3-small",
})

// Batch embedding
vecs, err := embedder.Embed(ctx, []string{
    "Go is a programming language",
    "Python is also a programming language",
})

// Single embedding
vec, err := embedder.EmbedSingle(ctx, "Hello world")

// Dimensions
dims := embedder.Dimensions() // 1536 for OpenAI text-embedding-3-small
```

## Vector Store

### VectorStore Interface

```go
type VectorStore interface {
    Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error
    Search(ctx context.Context, query []float32, k int, opts ...SearchOption) ([]schema.Document, error)
    Delete(ctx context.Context, ids []string) error
}
```

### Usage

```go
store, err := vectorstore.New("pgvector", config.ProviderConfig{
    Options: map[string]any{
        "connection_string": os.Getenv("DATABASE_URL"),
        "table_name":        "documents",
    },
})

// Add documents
docs := []schema.Document{
    {ID: "1", Content: "Go concurrency with goroutines"},
    {ID: "2", Content: "Python async/await patterns"},
}
embeddings, _ := embedder.Embed(ctx, []string{docs[0].Content, docs[1].Content})
store.Add(ctx, docs, embeddings)

// Search
queryVec, _ := embedder.EmbedSingle(ctx, "concurrency in Go")
results, err := store.Search(ctx, queryVec, 5,
    vectorstore.WithThreshold(0.7),
    vectorstore.WithFilter(map[string]any{"language": "go"}),
    vectorstore.WithStrategy(vectorstore.Cosine),
)
```

## Retriever

### Retriever Interface

```go
type Retriever interface {
    Retrieve(ctx context.Context, query string, opts ...Option) ([]schema.Document, error)
}
```

### Vector Retriever

Basic semantic search:

```go
ret := retriever.NewVectorStoreRetriever(store, embedder)
docs, err := ret.Retrieve(ctx, "Go concurrency patterns",
    retriever.WithTopK(10),
    retriever.WithThreshold(0.7),
)
```

### Hybrid Retriever (Default)

Combines vector + BM25 with RRF fusion:

```go
ret := retriever.NewHybridRetriever(store, embedder, bm25Searcher,
    retriever.WithHybridRRFK(60),
)
docs, err := ret.Retrieve(ctx, "Go concurrency")
```

### HyDE Retriever

Hypothetical Document Embeddings:

```go
ret := retriever.NewHyDERetriever(model, embedder, store,
    retriever.WithHyDEPrompt("Write a detailed answer to: %s"),
)
docs, err := ret.Retrieve(ctx, "What is Go's concurrency model?")
```

### CRAG Retriever

Corrective RAG with relevance evaluation:

```go
ret := retriever.NewCRAGRetriever(innerRetriever, model, webSearcher,
    retriever.WithCRAGThreshold(0.5),
)
docs, err := ret.Retrieve(ctx, "latest Go 1.23 features")
```

### Multi-Query Retriever

Generate query variations:

```go
ret := retriever.NewMultiQueryRetriever(innerRetriever, model,
    retriever.WithMultiQueryCount(3),
)
docs, err := ret.Retrieve(ctx, "Go error handling")
```

### Ensemble Retriever

Combine multiple retrievers:

```go
ret := retriever.NewEnsembleRetriever(
    []retriever.Retriever{vectorRet, bm25Ret, hydeRet},
    retriever.NewRRFStrategy(60),
)
docs, err := ret.Retrieve(ctx, "Go best practices")
```

### Rerank Retriever

Two-stage retrieve-then-rerank:

```go
ret := retriever.NewRerankRetriever(innerRetriever, reranker,
    retriever.WithRerankTopN(5),
)
docs, err := ret.Retrieve(ctx, "Go concurrency")
```

### Adaptive Retriever

Route by query complexity:

```go
ret := retriever.NewAdaptiveRetriever(
    model,
    simpleRetriever,  // For simple queries
    complexRetriever, // For complex queries
)
docs, err := ret.Retrieve(ctx, "What is Go?") // Uses simpleRetriever
```

## Fusion Strategies

### RRF (Reciprocal Rank Fusion)

```go
strategy := retriever.NewRRFStrategy(60) // k=60 (default)
```

### Weighted

```go
strategy := retriever.NewWeightedStrategy([]float64{0.6, 0.4})
```

## Middleware & Hooks

```go
ret = retriever.ApplyMiddleware(ret,
    retriever.WithHooks(retriever.Hooks{
        BeforeRetrieve: func(ctx context.Context, query string) error {
            log.Printf("Retrieving: %s", query)
            return nil
        },
        AfterRetrieve: func(ctx context.Context, docs []schema.Document, err error) {
            log.Printf("Found %d docs", len(docs))
        },
    }),
)
```

## Full RAG Pipeline Example

```go
// Setup
embedder, _ := embedding.New("openai", embedCfg)
store, _ := vectorstore.New("pgvector", storeCfg)

// Index documents
docs := loadDocuments()
embeddings, _ := embedder.Embed(ctx, extractTexts(docs))
store.Add(ctx, docs, embeddings)

// Retriever
retriever := retriever.NewHybridRetriever(store, embedder, bm25)

// RAG in agent
agent := agent.New("rag-assistant",
    agent.WithLLM(model),
    agent.WithTools(ragTool(retriever)),
)

result, _ := agent.Invoke(ctx, "Explain Go concurrency")
```

## See Also

- [Memory Package](./memory.md) for archival memory integration
- [Agent Package](./agent.md) for RAG tool usage
- [Schema Package](./schema.md) for document types
