# MultiQueryRetriever

**Role:** Wraps a `core.Retriever` and an LLM. Generates multiple query variants, calls `GetRelevantDocuments` for each, then merges and deduplicates.

**Constructor:** Both Retriever and LLM are required. If either is nil, return the package error with `ErrCodeInvalidConfig` (or equivalent).

**NumQueries:** Options must support a `NumQueries`-style setting with a default and a maximum cap. Exact default and cap are defined by the implementation.
