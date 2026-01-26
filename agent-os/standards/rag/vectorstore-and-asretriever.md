# VectorStore and AsRetriever

**Core methods:** `AddDocuments(ctx, docs, opts)`, `DeleteDocuments(ctx, ids, opts)`, `SimilaritySearch(ctx, queryVector, k, opts)`, `SimilaritySearchByQuery(ctx, query, k, embedder, opts)`. Ideally implement both search methods; exceptions allowed (e.g. only `SimilaritySearchByQuery` when the store always uses an embedder).

**AsRetriever(opts) Retriever** — Returns a Retriever so the store can be used in chains. `opts` configure the retriever (e.g. k, score threshold, filters). The returned Retriever uses these as defaults for `GetRelevantDocuments`.

**GetName() string** — For logging and metrics.
