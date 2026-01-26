# Graceful Degradation in Multi-Query

Continue on partial failures when partial results are useful.

```go
for _, q := range allQueries {
    docs, err := m.retriever.GetRelevantDocuments(ctx, q)
    if err != nil {
        m.logger.Warn("retrieval failed for query variation",
            "error", err,
            "query", q,
        )
        continue  // NOT break - partial results are useful
    }
    // merge docs into result set...
}
```

## Why Continue, Not Break
- Search benefits from partial results (3/5 queries returning is useful)
- Individual query failures are often transient
- User gets best-effort results rather than complete failure

## When to Use
- Multi-query retrievers
- Batch operations where partial success is acceptable
- Parallel API calls with independent results

## When NOT to Use
- Sequential operations where each step depends on previous
- Transactions requiring atomicity
- Security-sensitive operations (fail fast)

## Deduplication Pattern
```go
allDocuments := make(map[string]schema.Document)  // Dedupe by ID

for _, doc := range docs {
    docID := doc.ID
    if docID == "" {
        docID = fmt.Sprintf("doc-%d", len(doc.GetContent()))  // Fallback ID
    }
    if _, exists := allDocuments[docID]; !exists {
        allDocuments[docID] = doc
    }
}
```
