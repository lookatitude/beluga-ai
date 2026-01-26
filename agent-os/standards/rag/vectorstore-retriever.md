# VectorStoreRetriever

**Role:** Wraps a `VectorStore`; uses `SimilaritySearchByQuery`; implements `core.Retriever`, `core.Runnable`, `HealthChecker`.

**Tracer and logger:** Use the global tracer and logger. Do not add `WithTracer`, `WithLogger`, or `WithTracing` as options.

**Options:** `WithDefaultK`, `WithScoreThreshold`, `WithTimeout`, and `WithMetrics` / `WithMeter` when metrics are not global.

**Validation:** Constructor must validate (e.g. DefaultK 1–100, ScoreThreshold 0–1). On invalid config, return the package error with `ErrCodeInvalidConfig`.
