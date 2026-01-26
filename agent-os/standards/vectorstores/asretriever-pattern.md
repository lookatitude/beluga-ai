# AsRetriever Pattern

VectorStore provides lightweight Retriever wrapper.

```go
type VectorStore interface {
    // ... store methods ...
    AsRetriever(opts ...Option) Retriever
}
```

## Usage
```go
store := inmemory.NewInMemoryVectorStore(...)
retriever := store.AsRetriever(WithSearchK(10))

// Use in RAG pipeline
docs, _ := retriever.GetRelevantDocuments(ctx, "query")
```

## Implementation Detail
```go
type InMemoryRetriever struct {
    store *InMemoryVectorStore
    opts  []Option  // Options captured at creation
}

func (r *InMemoryRetriever) GetRelevantDocuments(ctx, query) {
    opts := append(r.opts, WithSearchK(5))  // Note: hardcoded override!
    return r.store.SimilaritySearchByQuery(ctx, query, 5, nil, opts...)
}
```

## Known Issue
SearchK is hardcoded to 5 at retrieval time, overriding options passed to `AsRetriever()`. Options are captured but partially ignored.

## Purpose
- Enables composition in RAG pipelines
- Retriever implements `Runnable` interface
- Pattern borrowed from LangChain
