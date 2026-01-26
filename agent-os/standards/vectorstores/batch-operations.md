# Batch Operations

Convenience functions for large operations.

```go
BatchAddDocuments(ctx, store, documents, batchSize, embedder, opts...)
BatchSearch(ctx, store, queries, k, batchSize, opts...)
```

## Batch Size Default
```go
if batchSize <= 0 {
    batchSize = 100  // Silent default
}
```

## Partial Success Semantics
```go
for i := 0; i < totalBatches; i++ {
    ids, err := AddDocuments(ctx, store, batch, embedder, opts...)
    if err != nil {
        return allIDs, err  // Returns partial results!
    }
    allIDs = append(allIDs, ids...)
}
```
- On error, returns IDs of successfully processed batches
- Caller responsible for handling partial success
- No rollback of previous batches

## Usage
```go
ids, err := BatchAddDocuments(ctx, store, docs, 100, embedder)
if err != nil {
    // ids contains successfully added docs before failure
    log.Printf("Partial success: %d/%d docs added", len(ids), len(docs))
}
```

## When to Use
- Adding >100 documents
- Large similarity search queries
- Memory-constrained environments
