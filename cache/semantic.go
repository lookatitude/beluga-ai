package cache

import (
	"context"
	"crypto/sha256"
	"fmt"
)

// SemanticCache wraps a Cache to provide similarity-based lookups using
// embedding vectors. When an exact key match is not found, it falls back
// to comparing embedding vectors against stored entries using cosine
// similarity.
//
// Currently this is a placeholder implementation that stores embeddings
// keyed by their hash and falls back to exact key matching. A full
// implementation requires an Embedder to generate embeddings from text
// queries.
type SemanticCache struct {
	cache     Cache
	threshold float64
}

// NewSemanticCache creates a SemanticCache wrapping the given Cache.
// The threshold (0–1) controls the minimum cosine similarity required
// for a semantic match. A threshold of 0.95 requires very high similarity;
// 0.8 is more permissive.
func NewSemanticCache(cache Cache, threshold float64) *SemanticCache {
	if threshold < 0 {
		threshold = 0
	}
	if threshold > 1 {
		threshold = 1
	}
	return &SemanticCache{
		cache:     cache,
		threshold: threshold,
	}
}

// GetSemantic searches the cache for an entry whose embedding is similar
// to the provided embedding within the given threshold.
//
// This is a placeholder implementation: it hashes the embedding vector to
// produce a cache key and performs an exact lookup. A full implementation
// would iterate stored embeddings and compute cosine similarity.
//
// The threshold parameter overrides the SemanticCache's default threshold
// for this single lookup. Pass 0 to use the default.
func (sc *SemanticCache) GetSemantic(ctx context.Context, embedding []float32, threshold float64) (any, bool, error) {
	if threshold <= 0 {
		threshold = sc.threshold
	}
	_ = threshold // will be used when full similarity search is implemented

	key := embeddingKey(embedding)
	return sc.cache.Get(ctx, key)
}

// SetSemantic stores a value keyed by the hash of its embedding vector.
// The embedding can later be looked up via GetSemantic.
func (sc *SemanticCache) SetSemantic(ctx context.Context, embedding []float32, value any) error {
	key := embeddingKey(embedding)
	return sc.cache.Set(ctx, key, value, 0)
}

// Cache returns the underlying Cache instance.
func (sc *SemanticCache) Cache() Cache {
	return sc.cache
}

// embeddingKey produces a deterministic cache key from an embedding vector
// by hashing the float32 values.
func embeddingKey(embedding []float32) string {
	h := sha256.New()
	for _, v := range embedding {
		// Use fmt to produce a deterministic string representation.
		fmt.Fprintf(h, "%v,", v)
	}
	return fmt.Sprintf("sem:%x", h.Sum(nil))
}
