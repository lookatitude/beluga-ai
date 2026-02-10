// Package cache provides exact and semantic caching for the Beluga AI framework.
// It defines the Cache interface for key-value storage with TTL support, a registry
// for pluggable cache backends, and a SemanticCache wrapper for embedding-based
// similarity lookups.
//
// # Cache Interface
//
// The Cache interface provides four operations:
//
//   - Get retrieves a value by key, returning (value, found, error).
//   - Set stores a value with a key and TTL.
//   - Delete removes a key from the cache.
//   - Clear removes all entries.
//
// # Registry
//
// Cache backends register via the standard Beluga registry pattern. Import a
// provider package for side-effect registration, then create instances via New.
//
// # SemanticCache
//
// SemanticCache wraps any Cache to provide similarity-based lookups using
// embedding vectors. When an exact key match is not found, it falls back to
// comparing embedding vectors using cosine similarity. Configure the minimum
// similarity threshold when creating the wrapper.
//
// # Usage
//
// Exact caching with the in-memory provider:
//
//	import _ "github.com/lookatitude/beluga-ai/cache/providers/inmemory"
//
//	c, err := cache.New("inmemory", cache.Config{
//	    TTL:     5 * time.Minute,
//	    MaxSize: 1000,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	err = c.Set(ctx, "key", "value", 10*time.Minute)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	val, ok, err := c.Get(ctx, "key")
//
// Semantic caching:
//
//	sc := cache.NewSemanticCache(c, 0.95) // 95% similarity threshold
//	err = sc.SetSemantic(ctx, embedding, cachedResponse)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	val, ok, err = sc.GetSemantic(ctx, queryEmbedding, 0)
package cache
