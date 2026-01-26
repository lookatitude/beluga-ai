# Loader and Splitter

**Loader (core.Loader / DocumentLoader):** `Load(ctx) ([]schema.Document, error)`, `LazyLoad(ctx) (<-chan any, error)`. Set `Metadata["source"]` on each document only when the loader has a meaningful source (e.g. file path, URL); omit when there is none.

**Splitter:** `SplitDocuments(ctx, docs) ([]schema.Document, error)`, `CreateDocuments(ctx, texts, metadatas) ([]schema.Document, error)`, `SplitText(ctx, text) ([]string, error)`.

**Splitter factory:** `NewXxxSplitter(opts ...XxxOption)` using functional options (e.g. `WithChunkSize`, `WithChunkOverlap`, `WithSeparators`). Options are applied to a config struct, then passed to the provider. Other splitters follow the same pattern.
