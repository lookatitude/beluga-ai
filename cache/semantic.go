package cache

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/rag/embedding"
)

// Compile-time interface check.
var _ Cache = (*SemanticCache)(nil)

// cosineSimilarity computes the cosine similarity between two float32 vectors.
// Returns 0 for zero-magnitude vectors or mismatched dimensions.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		ai, bi := float64(a[i]), float64(b[i])
		dot += ai * bi
		normA += ai * ai
		normB += bi * bi
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// semanticEntry stores a single cached entry with its embedding vector.
type semanticEntry struct {
	key       string
	embedding []float32
	value     any
	expiresAt time.Time
}

// SemanticCache provides similarity-based cache lookups using embedding vectors.
// When Get is called with a text key, the key is embedded and compared against
// stored entries using cosine similarity. If a match exceeds the threshold, the
// cached value is returned.
type SemanticCache struct {
	embedder      embedding.Embedder
	threshold     float64
	defaultTTL    time.Duration
	maxEntries    int
	maxDimensions int
	entries       []semanticEntry
	mu            sync.RWMutex
	now           func() time.Time
}

// Option configures a SemanticCache.
type Option func(*SemanticCache)

// WithThreshold sets the minimum cosine similarity required for a cache hit.
// Values are clamped to [0.01, 1.0]. Default is 0.85.
func WithThreshold(t float64) Option {
	return func(sc *SemanticCache) {
		if t < 0.01 {
			t = 0.01
		}
		if t > 1.0 {
			t = 1.0
		}
		sc.threshold = t
	}
}

// WithDefaultTTL sets the default time-to-live for cache entries when a zero
// TTL is passed to Set.
func WithDefaultTTL(d time.Duration) Option {
	return func(sc *SemanticCache) {
		sc.defaultTTL = d
	}
}

// WithMaxEntries sets the maximum number of entries the cache can hold.
// When exceeded, the oldest entry is evicted. Zero means unlimited.
// Negative values are clamped to 0 (unlimited).
func WithMaxEntries(n int) Option {
	return func(sc *SemanticCache) {
		if n < 0 {
			n = 0
		}
		sc.maxEntries = n
	}
}

// WithMaxDimensions sets the maximum allowed embedding vector length.
// Vectors exceeding this length are rejected. Default is 8192.
func WithMaxDimensions(n int) Option {
	return func(sc *SemanticCache) {
		if n > 0 {
			sc.maxDimensions = n
		}
	}
}

// NewSemanticCache creates a SemanticCache that uses the given Embedder to
// convert text keys into vectors for similarity comparison.
func NewSemanticCache(embedder embedding.Embedder, opts ...Option) *SemanticCache {
	sc := &SemanticCache{
		embedder:      embedder,
		threshold:     0.85,
		maxDimensions: 8192,
		now:           time.Now,
	}
	for _, o := range opts {
		o(sc)
	}
	return sc
}

// Get retrieves a value by embedding the key text and scanning entries for the
// best cosine similarity match above the threshold. Expired entries are skipped.
func (sc *SemanticCache) Get(ctx context.Context, key string) (any, bool, error) {
	emb, err := sc.embedder.EmbedSingle(ctx, key)
	if err != nil {
		return nil, false, fmt.Errorf("cache: semantic embed: %w", err)
	}
	return sc.GetByEmbedding(ctx, emb)
}

// Set stores a value by embedding the key text. If an entry with the same exact
// key already exists, it is updated in place. If maxEntries is exceeded, the
// oldest entry is evicted.
//
// The value is stored by reference. Callers should treat stored values as
// immutable once passed to Set.
func (sc *SemanticCache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	emb, err := sc.embedder.EmbedSingle(ctx, key)
	if err != nil {
		return fmt.Errorf("cache: semantic embed: %w", err)
	}
	return sc.SetByEmbedding(ctx, key, emb, value, ttl)
}

// Delete removes an entry by exact key string match.
func (sc *SemanticCache) Delete(_ context.Context, key string) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	for i, e := range sc.entries {
		if e.key == key {
			sc.entries = append(sc.entries[:i], sc.entries[i+1:]...)
			return nil
		}
	}
	return nil
}

// Clear removes all entries from the cache.
func (sc *SemanticCache) Clear(_ context.Context) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.entries = nil
	return nil
}

// GetByEmbedding searches for the best matching entry using a pre-computed
// embedding vector. Returns the value, whether a match was found, and any error.
func (sc *SemanticCache) GetByEmbedding(_ context.Context, emb []float32) (any, bool, error) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	now := sc.now()
	var bestVal any
	bestSim := -1.0
	for _, e := range sc.entries {
		// Skip expired entries. Zero expiresAt means no expiration.
		if !e.expiresAt.IsZero() && now.After(e.expiresAt) {
			continue
		}
		sim := cosineSimilarity(emb, e.embedding)
		if sim >= sc.threshold && sim > bestSim {
			bestSim = sim
			bestVal = e.value
		}
	}
	if bestSim < sc.threshold {
		return nil, false, nil
	}
	return bestVal, true, nil
}

// SetByEmbedding stores an entry with a pre-computed embedding vector. If an
// entry with the same exact key exists, it is updated. If maxEntries is
// exceeded, the oldest entry is evicted.
//
// The embedding slice is copied defensively; the caller may safely mutate it
// after this call returns. The value is stored by reference and should be
// treated as immutable once passed.
func (sc *SemanticCache) SetByEmbedding(_ context.Context, key string, emb []float32, value any, ttl time.Duration) error {
	if sc.maxDimensions > 0 && len(emb) > sc.maxDimensions {
		return fmt.Errorf("cache: embedding dimension %d exceeds maximum %d", len(emb), sc.maxDimensions)
	}

	// Defensive copy to prevent caller mutations from corrupting cache.
	embCopy := make([]float32, len(emb))
	copy(embCopy, emb)

	sc.mu.Lock()
	defer sc.mu.Unlock()

	exp := sc.computeExpiry(ttl)

	// Update in place if exact key exists.
	for i, e := range sc.entries {
		if e.key == key {
			sc.entries[i].embedding = embCopy
			sc.entries[i].value = value
			sc.entries[i].expiresAt = exp
			return nil
		}
	}

	// Sweep expired entries before appending to bound memory usage.
	sc.sweepExpiredLocked()

	// Evict oldest if at capacity.
	if sc.maxEntries > 0 && len(sc.entries) >= sc.maxEntries {
		sc.entries = sc.entries[1:]
	}

	sc.entries = append(sc.entries, semanticEntry{
		key:       key,
		embedding: embCopy,
		value:     value,
		expiresAt: exp,
	})
	return nil
}

// Prune removes all expired entries from the cache. It is safe for concurrent
// use and can be called by callers who want explicit control over eviction.
func (sc *SemanticCache) Prune(_ context.Context) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.sweepExpiredLocked()
	return nil
}

// sweepExpiredLocked removes expired entries from the slice.
// The caller MUST hold sc.mu in write mode.
func (sc *SemanticCache) sweepExpiredLocked() {
	now := sc.now()
	n := 0
	for _, e := range sc.entries {
		if !e.expiresAt.IsZero() && now.After(e.expiresAt) {
			continue
		}
		sc.entries[n] = e
		n++
	}
	// Clear trailing references for GC.
	for i := n; i < len(sc.entries); i++ {
		sc.entries[i] = semanticEntry{}
	}
	sc.entries = sc.entries[:n]
}

// computeExpiry returns the expiration time for a given TTL. A zero TTL uses
// the default TTL. A negative TTL means no expiration (zero time).
func (sc *SemanticCache) computeExpiry(ttl time.Duration) time.Time {
	if ttl == 0 {
		ttl = sc.defaultTTL
	}
	if ttl <= 0 {
		return time.Time{}
	}
	return sc.now().Add(ttl)
}

func init() {
	Register("semantic", func(cfg Config) (Cache, error) {
		emb, ok := cfg.Options["embedder"]
		if !ok {
			return nil, fmt.Errorf("cache: semantic factory requires Options[\"embedder\"] as embedding.Embedder")
		}
		embedder, ok := emb.(embedding.Embedder)
		if !ok {
			return nil, fmt.Errorf("cache: semantic factory: Options[\"embedder\"] is not an embedding.Embedder")
		}
		opts := []Option{}
		if cfg.TTL > 0 {
			opts = append(opts, WithDefaultTTL(cfg.TTL))
		}
		if cfg.MaxSize > 0 {
			opts = append(opts, WithMaxEntries(cfg.MaxSize))
		}
		if t, ok := cfg.Options["threshold"]; ok {
			if tv, ok := t.(float64); ok {
				opts = append(opts, WithThreshold(tv))
			}
		}
		return NewSemanticCache(embedder, opts...), nil
	})
}
