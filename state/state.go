package state

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Store is the interface for shared agent state storage.
// Implementations must be safe for concurrent use.
type Store interface {
	// Get retrieves the value for the given key.
	// Returns nil, nil if the key does not exist.
	Get(ctx context.Context, key string) (any, error)

	// Set stores a value under the given key.
	Set(ctx context.Context, key string, value any) error

	// Delete removes the given key. Deleting a non-existent key is a no-op.
	Delete(ctx context.Context, key string) error

	// Watch returns a channel that receives StateChange notifications
	// whenever the given key is modified or deleted. The caller must read
	// from the channel to avoid blocking writers. The channel is closed
	// when the store is closed.
	Watch(ctx context.Context, key string) (<-chan StateChange, error)

	// Close releases resources held by the store and closes all watcher
	// channels.
	Close() error
}

// ChangeOp describes the type of state change.
type ChangeOp string

const (
	// OpSet indicates a key was created or updated.
	OpSet ChangeOp = "set"
	// OpDelete indicates a key was removed.
	OpDelete ChangeOp = "delete"
)

// StateChange describes a mutation to a key in the store.
type StateChange struct {
	// Key is the affected key.
	Key string
	// OldValue is the previous value (nil if the key was new).
	OldValue any
	// Value is the new value (nil for deletes).
	Value any
	// Op is the type of change.
	Op ChangeOp
}

// Scope defines the visibility level for state keys.
type Scope string

const (
	// ScopeAgent limits the key to a single agent instance.
	ScopeAgent Scope = "agent"
	// ScopeSession limits the key to the current session.
	ScopeSession Scope = "session"
	// ScopeGlobal makes the key visible across all agents and sessions.
	ScopeGlobal Scope = "global"
)

// ScopedKey returns a key prefixed with the scope, formatted as "scope:key".
func ScopedKey(scope Scope, key string) string {
	return string(scope) + ":" + key
}

// Config holds configuration for creating a Store via the registry.
type Config struct {
	// Extra holds provider-specific configuration.
	Extra map[string]any
}

// Factory is a constructor function for creating a Store from config.
type Factory func(cfg Config) (Store, error)

var (
	mu       sync.RWMutex
	registry = make(map[string]Factory)
)

// Register registers a Store factory under the given name.
// This is typically called from init() in provider packages.
func Register(name string, f Factory) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = f
}

// New creates a new Store by looking up the registered factory.
func New(name string, cfg Config) (Store, error) {
	mu.RLock()
	f, ok := registry[name]
	mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("state: store %q not registered", name)
	}
	return f(cfg)
}

// List returns the sorted names of all registered stores.
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
