package state

import (
	"context"
	"errors"
	"iter"
	"sort"
	"sync"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// KV is the key/value part of a [Store]: read, write and delete.
// Code that only reads or writes state should accept a KV rather than a full
// Store.
type KV interface {
	// Get retrieves the value for the given key.
	// Returns nil, nil if the key does not exist.
	Get(ctx context.Context, key string) (any, error)

	// Set stores a value under the given key.
	Set(ctx context.Context, key string, value any) error

	// Delete removes the given key. Deleting a non-existent key is a no-op.
	Delete(ctx context.Context, key string) error
}

// Watcher is the change-notification part of a [Store].
type Watcher interface {
	// Watch returns an iter.Seq2 stream of StateChange notifications for
	// the given key. The subscription is established eagerly before Watch
	// returns, so events produced after this call but before the caller
	// starts iterating are buffered (implementation-defined capacity) and
	// will be delivered on the first iteration.
	//
	// The iterator ends when ctx is cancelled, the store is closed, or
	// the caller breaks out of the loop. Errors (including initial
	// subscription errors such as a closed store) are reported by yielding
	// a zero-value StateChange together with a non-nil error.
	Watch(ctx context.Context, key string) iter.Seq2[StateChange, error]
}

// Store is the interface for shared agent state storage.
// Implementations must be safe for concurrent use.
//
// Store is a documented exception to the framework's "interfaces have at
// most 4 methods" rule: it predates the rule and changing its method set
// would break every implementation. It is composed of the smaller [KV] (3
// methods) and [Watcher] (1 method) interfaces plus io.Closer; new APIs
// should accept the narrowest of these that they need.
type Store interface {
	KV
	Watcher

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
type StateChange struct { //nolint:revive // exported name is public API; renaming to remove the stutter would break callers
	// Key is the affected key.
	Key string
	// OldValue is the previous value (nil if the key was new).
	OldValue any
	// Value is the new value (nil for deletes).
	Value any
	// Op is the type of change.
	Op ChangeOp
	// Version is the monotonic revision counter for the key after the change.
	// Zero means the store does not support versioning.
	Version uint64
}

// VersionedStore extends Store with compare-and-swap semantics and per-key
// version tracking. Implementations maintain a monotonic revision counter
// per key, incremented on every mutation.
type VersionedStore interface {
	Store

	// CompareAndSwap atomically sets the value for key only if the current
	// version matches expectedVersion. It returns the new version on success.
	// If the versions do not match, it returns ErrVersionMismatch.
	CompareAndSwap(ctx context.Context, key string, expectedVersion uint64, value any) (newVersion uint64, err error)

	// GetVersioned retrieves the value and current version for the given key.
	// Returns (nil, 0, nil) if the key does not exist.
	GetVersioned(ctx context.Context, key string) (value any, version uint64, err error)
}

// ErrVersionMismatch is returned by CompareAndSwap when the expected version
// does not match the current version of the key.
var ErrVersionMismatch = errors.New("state: version mismatch")

// ErrStoreClosed is returned when an operation is attempted on a closed store.
var ErrStoreClosed = errors.New("state: store is closed")

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
		return nil, core.Errorf(core.ErrNotFound, "state: store %q not registered", name)
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
