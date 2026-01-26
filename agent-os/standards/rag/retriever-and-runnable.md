# Retriever and Runnable

**core.Retriever** embeds **Runnable**: input `string` (query), output `[]schema.Document`. `GetRelevantDocuments(ctx, query string) ([]schema.Document, error)`.

**Why:** So retrievers fit `Invoke`, `Batch`, `Stream` and composition in chains and graphs.

**HealthChecker:** Every Retriever implementation with non-trivial config or dependencies (e.g. VectorStore, embedder) must also implement `HealthChecker` and provide `CheckHealth(ctx) error`.
