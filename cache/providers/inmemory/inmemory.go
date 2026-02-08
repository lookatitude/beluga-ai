// Package inmemory provides an in-memory LRU cache implementation for the
// Beluga AI framework. It registers itself under the name "inmemory" in the
// cache registry.
//
// The cache uses a doubly-linked list combined with a hash map for O(1) get,
// set, and eviction. Entries expire lazily on access based on their TTL.
// When MaxSize is reached, the least-recently-used entry is evicted.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/cache/providers/inmemory"
//
//	c, _ := cache.New("inmemory", cache.Config{
//	    TTL:     5 * time.Minute,
//	    MaxSize: 1000,
//	})
package inmemory

import (
	"container/list"
	"context"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/cache"
)

func init() {
	cache.Register("inmemory", func(cfg cache.Config) (cache.Cache, error) {
		return New(cfg), nil
	})
}

// entry is a single cache entry stored in the LRU list.
type entry struct {
	key       string
	value     any
	expiresAt time.Time // zero value means no expiration
}

// InMemoryCache is a thread-safe, in-memory LRU cache with TTL-based
// expiration. It implements the cache.Cache interface.
type InMemoryCache struct {
	mu         sync.Mutex
	items      map[string]*list.Element
	order      *list.List // front = most recent, back = least recent
	defaultTTL time.Duration
	maxSize    int
	now        func() time.Time // injectable for testing
}

// New creates a new InMemoryCache with the given configuration.
// If MaxSize is 0, the cache grows without bound.
func New(cfg cache.Config) *InMemoryCache {
	return &InMemoryCache{
		items:      make(map[string]*list.Element),
		order:      list.New(),
		defaultTTL: cfg.TTL,
		maxSize:    cfg.MaxSize,
		now:        time.Now,
	}
}

// Get retrieves a value by key. If the entry exists but has expired, it is
// removed and (nil, false, nil) is returned. Found entries are promoted to
// the front of the LRU list.
func (c *InMemoryCache) Get(_ context.Context, key string) (any, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.items[key]
	if !ok {
		return nil, false, nil
	}

	e := elem.Value.(*entry)

	// Lazy expiration check.
	if !e.expiresAt.IsZero() && c.now().After(e.expiresAt) {
		c.removeLocked(elem)
		return nil, false, nil
	}

	// Promote to most-recently-used.
	c.order.MoveToFront(elem)
	return e.value, true, nil
}

// Set stores a value with the given key and TTL. If the key already exists,
// its value and TTL are updated and it is promoted to the front of the LRU
// list. When the cache exceeds MaxSize, the least-recently-used entry is
// evicted.
//
// A zero TTL uses the cache's default TTL. A negative TTL means the entry
// never expires.
func (c *InMemoryCache) Set(_ context.Context, key string, value any, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiresAt := c.computeExpiry(ttl)

	// Update existing entry.
	if elem, ok := c.items[key]; ok {
		e := elem.Value.(*entry)
		e.value = value
		e.expiresAt = expiresAt
		c.order.MoveToFront(elem)
		return nil
	}

	// Add new entry.
	e := &entry{
		key:       key,
		value:     value,
		expiresAt: expiresAt,
	}
	elem := c.order.PushFront(e)
	c.items[key] = elem

	// Evict LRU entry if over capacity.
	if c.maxSize > 0 && c.order.Len() > c.maxSize {
		c.evictLocked()
	}

	return nil
}

// Delete removes a key from the cache. Deleting a non-existent key is a no-op.
func (c *InMemoryCache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.removeLocked(elem)
	}
	return nil
}

// Clear removes all entries from the cache.
func (c *InMemoryCache) Clear(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.order.Init()
	return nil
}

// Len returns the current number of entries in the cache. This includes
// entries that may have expired but have not yet been lazily removed.
func (c *InMemoryCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.order.Len()
}

// computeExpiry calculates the expiration time for an entry.
func (c *InMemoryCache) computeExpiry(ttl time.Duration) time.Time {
	if ttl < 0 {
		return time.Time{} // no expiration
	}
	if ttl == 0 {
		ttl = c.defaultTTL
	}
	if ttl <= 0 {
		return time.Time{} // default TTL is also zero or negative
	}
	return c.now().Add(ttl)
}

// evictLocked removes the least-recently-used entry. Must be called with mu held.
func (c *InMemoryCache) evictLocked() {
	back := c.order.Back()
	if back != nil {
		c.removeLocked(back)
	}
}

// removeLocked removes the given list element from both the list and map.
// Must be called with mu held.
func (c *InMemoryCache) removeLocked(elem *list.Element) {
	e := elem.Value.(*entry)
	delete(c.items, e.key)
	c.order.Remove(elem)
}
