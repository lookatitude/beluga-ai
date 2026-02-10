package cache

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Cache is the interface that all cache backends implement. It provides
// basic key-value operations with TTL support.
type Cache interface {
	// Get retrieves a value by key. Returns the value, whether the key was
	// found, and any error. A missing key returns (nil, false, nil).
	Get(ctx context.Context, key string) (any, bool, error)

	// Set stores a value with the given key and TTL. A zero TTL means the
	// entry uses the cache's default TTL. A negative TTL means no expiration.
	Set(ctx context.Context, key string, value any, ttl time.Duration) error

	// Delete removes a key from the cache. Deleting a non-existent key is a no-op.
	Delete(ctx context.Context, key string) error

	// Clear removes all entries from the cache.
	Clear(ctx context.Context) error
}

// Config holds configuration for creating a cache instance via the registry.
type Config struct {
	// TTL is the default time-to-live for cache entries.
	TTL time.Duration

	// MaxSize is the maximum number of entries the cache can hold.
	// Zero means unlimited.
	MaxSize int

	// Options holds provider-specific configuration key-value pairs.
	Options map[string]any
}

// Factory is a constructor function that creates a Cache from a Config.
type Factory func(cfg Config) (Cache, error)

var (
	mu       sync.RWMutex
	registry = make(map[string]Factory)
)

// Register adds a named cache factory to the global registry. It is intended
// to be called from provider init() functions. Registering a duplicate name
// overwrites the previous factory.
func Register(name string, f Factory) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = f
}

// New creates a Cache by looking up the named factory in the registry and
// calling it with the provided Config. Returns an error if no factory is
// registered under the given name.
func New(name string, cfg Config) (Cache, error) {
	mu.RLock()
	f, ok := registry[name]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("cache: unknown provider %q (registered: %v)", name, List())
	}
	return f(cfg)
}

// List returns the sorted names of all registered cache providers.
func List() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
