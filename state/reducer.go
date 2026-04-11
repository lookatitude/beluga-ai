package state

import (
	"context"
	"iter"
)

// ReducerFunc merges an old value with a new value. The old value may be nil
// if the key does not exist yet. The returned value replaces the stored value.
type ReducerFunc func(old, new any) any

// ReducerOption configures a ReducerStore.
type ReducerOption func(*reducerOptions)

type reducerOptions struct {
	reducers       map[string]ReducerFunc
	defaultReducer ReducerFunc
}

// WithReducer registers a per-key reducer function. When Set is called for
// this key, the reducer merges the old and new values instead of overwriting.
func WithReducer(key string, fn ReducerFunc) ReducerOption {
	return func(o *reducerOptions) {
		o.reducers[key] = fn
	}
}

// WithDefaultReducer sets a default reducer applied to any key that does not
// have a specific reducer registered. If nil (the default), keys without a
// specific reducer use plain overwrite semantics.
func WithDefaultReducer(fn ReducerFunc) ReducerOption {
	return func(o *reducerOptions) {
		o.defaultReducer = fn
	}
}

// ReducerStore wraps a VersionedStore and applies per-key reducer functions
// on Set. Reducers merge old and new values atomically using CompareAndSwap.
// opts is set once at construction and treated as immutable thereafter, so
// no lock is required around reducer lookups.
type ReducerStore struct {
	inner VersionedStore
	opts  reducerOptions
}

// NewReducerStore creates a ReducerStore wrapping inner with the given options.
func NewReducerStore(inner VersionedStore, opts ...ReducerOption) *ReducerStore {
	o := reducerOptions{
		reducers: make(map[string]ReducerFunc),
	}
	for _, opt := range opts {
		opt(&o)
	}
	return &ReducerStore{inner: inner, opts: o}
}

// Get delegates to the inner store.
func (rs *ReducerStore) Get(ctx context.Context, key string) (any, error) {
	return rs.inner.Get(ctx, key)
}

// GetVersioned delegates to the inner store.
func (rs *ReducerStore) GetVersioned(ctx context.Context, key string) (any, uint64, error) {
	return rs.inner.GetVersioned(ctx, key)
}

// Set applies the reducer for the key (if any) to merge old and new values,
// then writes the result atomically via CompareAndSwap. If no reducer is
// registered for the key, a plain overwrite is performed.
func (rs *ReducerStore) Set(ctx context.Context, key string, value any) error {
	reducer := rs.reducerFor(key)
	if reducer == nil {
		return rs.inner.Set(ctx, key, value)
	}

	// Retry loop for CAS contention. Bounded but generous: under heavy
	// multi-writer contention we need enough headroom to avoid spurious
	// ErrVersionMismatch failures on well-behaved workloads.
	const maxRetries = 100
	for i := 0; i < maxRetries; i++ {
		old, version, err := rs.inner.GetVersioned(ctx, key)
		if err != nil {
			return err
		}

		merged := reducer(old, value)
		_, casErr := rs.inner.CompareAndSwap(ctx, key, version, merged)
		if casErr == nil {
			return nil
		}
		if casErr != ErrVersionMismatch {
			return casErr
		}
		// Version mismatch — retry.
	}
	return ErrVersionMismatch
}

// CompareAndSwap delegates to the inner store. Reducers are not applied
// because CAS implies the caller is managing conflict resolution explicitly.
func (rs *ReducerStore) CompareAndSwap(ctx context.Context, key string, expectedVersion uint64, value any) (uint64, error) {
	return rs.inner.CompareAndSwap(ctx, key, expectedVersion, value)
}

// Delete delegates to the inner store.
func (rs *ReducerStore) Delete(ctx context.Context, key string) error {
	return rs.inner.Delete(ctx, key)
}

// Watch delegates to the inner store.
func (rs *ReducerStore) Watch(ctx context.Context, key string) iter.Seq2[StateChange, error] {
	return rs.inner.Watch(ctx, key)
}

// Close delegates to the inner store.
func (rs *ReducerStore) Close() error {
	return rs.inner.Close()
}

// reducerFor returns the reducer for key, falling back to the default reducer.
// opts is constructed once in NewReducerStore and never mutated afterwards,
// so no synchronization is needed.
func (rs *ReducerStore) reducerFor(key string) ReducerFunc {
	if fn, ok := rs.opts.reducers[key]; ok {
		return fn
	}
	return rs.opts.defaultReducer
}

// Compile-time checks.
var _ Store = (*ReducerStore)(nil)
var _ VersionedStore = (*ReducerStore)(nil)
