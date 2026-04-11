package state

import (
	"context"
	"iter"
)

// Middleware wraps a Store to add cross-cutting behavior.
// Middlewares are composed via ApplyMiddleware and applied outside-in
// (the last middleware in the list is the outermost wrapper).
type Middleware func(Store) Store

// ApplyMiddleware wraps s with the given middlewares in reverse order so
// that the first middleware in the list is the outermost (first to execute).
func ApplyMiddleware(s Store, mws ...Middleware) Store {
	for i := len(mws) - 1; i >= 0; i-- {
		s = mws[i](s)
	}
	return s
}

// WithHooks returns middleware that invokes the given Hooks around each
// Store operation.
func WithHooks(hooks Hooks) Middleware {
	return func(next Store) Store {
		return &hookedStore{next: next, hooks: hooks}
	}
}

// hookedStore wraps a Store with hook callbacks.
type hookedStore struct {
	next  Store
	hooks Hooks
}

func (s *hookedStore) Get(ctx context.Context, key string) (any, error) {
	if s.hooks.BeforeGet != nil {
		if err := s.hooks.BeforeGet(ctx, key); err != nil {
			return nil, err
		}
	}

	val, err := s.next.Get(ctx, key)

	if err != nil && s.hooks.OnError != nil {
		err = s.hooks.OnError(ctx, err)
	}

	if s.hooks.AfterGet != nil {
		s.hooks.AfterGet(ctx, key, val, err)
	}

	return val, err
}

func (s *hookedStore) Set(ctx context.Context, key string, value any) error {
	if s.hooks.BeforeSet != nil {
		if err := s.hooks.BeforeSet(ctx, key, value); err != nil {
			return err
		}
	}

	err := s.next.Set(ctx, key, value)

	if err != nil && s.hooks.OnError != nil {
		err = s.hooks.OnError(ctx, err)
	}

	if s.hooks.AfterSet != nil {
		s.hooks.AfterSet(ctx, key, value, err)
	}

	return err
}

func (s *hookedStore) Delete(ctx context.Context, key string) error {
	if s.hooks.OnDelete != nil {
		if err := s.hooks.OnDelete(ctx, key); err != nil {
			return err
		}
	}

	err := s.next.Delete(ctx, key)

	if err != nil && s.hooks.OnError != nil {
		err = s.hooks.OnError(ctx, err)
	}

	return err
}

func (s *hookedStore) Watch(ctx context.Context, key string) iter.Seq2[StateChange, error] {
	if s.hooks.OnWatch != nil {
		if err := s.hooks.OnWatch(ctx, key); err != nil {
			return func(yield func(StateChange, error) bool) {
				yield(StateChange{}, err)
			}
		}
	}

	inner := s.next.Watch(ctx, key)

	// If OnError is configured, intercept errors yielded by the inner stream.
	if s.hooks.OnError == nil {
		return inner
	}

	return func(yield func(StateChange, error) bool) {
		for change, err := range inner {
			if err != nil {
				err = s.hooks.OnError(ctx, err)
				if err == nil {
					continue
				}
			}
			if !yield(change, err) {
				return
			}
		}
	}
}

func (s *hookedStore) Close() error {
	return s.next.Close()
}

// Ensure hookedStore implements Store at compile time.
var _ Store = (*hookedStore)(nil)

// WrapVersionedWithHooks returns a VersionedStore that invokes the given
// Hooks around Get/Set/Delete/Watch and delegates the versioned operations
// (GetVersioned, CompareAndSwap) to the underlying store without hooks.
// This is the VersionedStore counterpart of WithHooks.
func WrapVersionedWithHooks(inner VersionedStore, hooks Hooks) VersionedStore {
	return &hookedVersionedStore{
		hookedStore: hookedStore{next: inner, hooks: hooks},
		inner:       inner,
	}
}

// hookedVersionedStore extends hookedStore with the VersionedStore methods,
// delegating them directly to the inner versioned store.
type hookedVersionedStore struct {
	hookedStore
	inner VersionedStore
}

// GetVersioned delegates to the inner versioned store. Hooks do not fire
// on versioned reads because they are conceptually a read variant.
func (s *hookedVersionedStore) GetVersioned(ctx context.Context, key string) (any, uint64, error) {
	return s.inner.GetVersioned(ctx, key)
}

// CompareAndSwap delegates to the inner versioned store. Hooks do not fire
// on CAS because callers typically use CAS as an explicit conflict path.
func (s *hookedVersionedStore) CompareAndSwap(ctx context.Context, key string, expectedVersion uint64, value any) (uint64, error) {
	return s.inner.CompareAndSwap(ctx, key, expectedVersion, value)
}

var _ VersionedStore = (*hookedVersionedStore)(nil)
