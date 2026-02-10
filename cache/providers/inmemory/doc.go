// Package inmemory provides an in-memory LRU cache implementation for the
// Beluga AI framework. It registers itself under the name "inmemory" in the
// cache registry.
//
// The cache uses a doubly-linked list combined with a hash map for O(1) get,
// set, and eviction. Entries expire lazily on access based on their TTL.
// When MaxSize is reached, the least-recently-used entry is evicted.
//
// # Key Types
//
//   - InMemoryCache implements the cache.Cache interface with thread-safe
//     LRU eviction and lazy TTL expiration.
//
// # Usage
//
// Import for side-effect registration, then create via the cache registry:
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
//
// Or create directly:
//
//	c := inmemory.New(cache.Config{
//	    TTL:     5 * time.Minute,
//	    MaxSize: 1000,
//	})
package inmemory
