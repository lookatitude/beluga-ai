# RAG Pipeline Flow

**Typical flow:** Load (Loader) → Split (Splitter) → Embed (Embedder) → AddDocuments (VectorStore). Retrieve via `VectorStore.SimilaritySearchByQuery` or `Retriever.GetRelevantDocuments`. Variations are allowed (e.g. skip Split for pre-chunked docs, or embed only at query time).

**Options:** `AddDocuments`, `AsRetriever`, and per-call overrides use `core.Option` or a package-defined `Option` that maps to the same concepts (k, score threshold, filters). Be consistent within the package.
