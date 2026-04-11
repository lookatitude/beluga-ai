// Package blackboard provides an enhanced shared state layer for multi-agent
// coordination. It wraps a state.VersionedStore with ownership enforcement,
// per-key reducers, and iter.Seq2 watch streams.
//
// # Usage
//
//	store := inmemory.New()
//	bb := blackboard.New(store,
//	    blackboard.WithReducers(
//	        state.WithReducer("counter", func(old, new any) any {
//	            o, _ := old.(int)
//	            n, _ := new.(int)
//	            return o + n
//	        }),
//	    ),
//	)
//	defer bb.Close()
//
//	bb.Set(ctx, "agent-1", "counter", 5)
//	bb.Set(ctx, "agent-2", "counter", 3) // merged to 8
package blackboard

import (
	"context"
	"iter"
	"sync"

	"github.com/lookatitude/beluga-ai/state"
)

// Option configures a Blackboard.
type Option func(*options)

type options struct {
	reducerOpts      []state.ReducerOption
	enforceOwnership bool
	hooks            state.Hooks
	hasHooks         bool
}

// WithReducers adds reducer options to the blackboard. Reducers are applied
// when Set is called, merging old and new values instead of overwriting.
func WithReducers(opts ...state.ReducerOption) Option {
	return func(o *options) {
		o.reducerOpts = append(o.reducerOpts, opts...)
	}
}

// WithEnforceOwnership enables ownership enforcement. When enabled, only the
// agent that claimed a key can write to it.
func WithEnforceOwnership() Option {
	return func(o *options) {
		o.enforceOwnership = true
	}
}

// WithBlackboardHooks attaches hooks to the underlying store operations.
func WithBlackboardHooks(hooks state.Hooks) Option {
	return func(o *options) {
		o.hooks = hooks
		o.hasHooks = true
	}
}

// Blackboard is a shared state layer for multi-agent coordination. It
// combines versioned state storage with ownership tracking and per-key
// reducer merging.
type Blackboard struct {
	store     state.VersionedStore
	reducer   *state.ReducerStore
	ownership *state.OwnershipManager
	opts      options
	mu        sync.RWMutex
	closed    bool
}

// New creates a Blackboard backed by the given VersionedStore. When hooks
// are provided via WithBlackboardHooks they are wrapped around the
// underlying VersionedStore so that Get/Set/Delete/Watch fire the callbacks.
func New(store state.VersionedStore, opts ...Option) *Blackboard {
	o := options{}
	for _, opt := range opts {
		opt(&o)
	}

	// Apply hook middleware to the underlying versioned store when configured.
	versioned := store
	if o.hasHooks {
		versioned = state.WrapVersionedWithHooks(store, o.hooks)
	}

	// Build the reducer store on top of the (possibly hooked) versioned store.
	reducer := state.NewReducerStore(versioned, o.reducerOpts...)

	return &Blackboard{
		store:     versioned,
		reducer:   reducer,
		ownership: state.NewOwnershipManager(),
		opts:      o,
	}
}

// Set stores a value under the given key, attributed to agentID. If a reducer
// is configured for the key, the old and new values are merged. If ownership
// enforcement is enabled, only the owner may write to claimed keys.
func (b *Blackboard) Set(ctx context.Context, agentID, key string, value any) error {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return errClosed
	}
	b.mu.RUnlock()

	if b.opts.enforceOwnership {
		if err := b.ownership.CheckWrite(key, agentID); err != nil {
			return err
		}
	}

	ctx = state.WithOwnerID(ctx, agentID)
	return b.reducer.Set(ctx, key, value)
}

// Get retrieves the value for the given key.
func (b *Blackboard) Get(ctx context.Context, key string) (any, error) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return nil, errClosed
	}
	b.mu.RUnlock()

	return b.store.Get(ctx, key)
}

// GetVersioned retrieves the value and version for the given key.
func (b *Blackboard) GetVersioned(ctx context.Context, key string) (any, uint64, error) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return nil, 0, errClosed
	}
	b.mu.RUnlock()

	return b.store.GetVersioned(ctx, key)
}

// Delete removes the given key, attributed to agentID. If ownership
// enforcement is enabled, only the owner may delete claimed keys.
func (b *Blackboard) Delete(ctx context.Context, agentID, key string) error {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return errClosed
	}
	b.mu.RUnlock()

	if b.opts.enforceOwnership {
		if err := b.ownership.CheckWrite(key, agentID); err != nil {
			return err
		}
	}

	ctx = state.WithOwnerID(ctx, agentID)
	return b.store.Delete(ctx, key)
}

// Watch returns an iter.Seq2 stream of state changes for the specified keys.
// If no keys are provided, this returns immediately with no events.
// The stream ends when the context is cancelled or the blackboard is closed.
// If the blackboard is already closed, the stream yields a single errClosed
// and returns, matching the behavior of Set/Get/Delete.
func (b *Blackboard) Watch(ctx context.Context, keys ...string) iter.Seq2[state.StateChange, error] {
	b.mu.RLock()
	closed := b.closed
	b.mu.RUnlock()
	if closed {
		return func(yield func(state.StateChange, error) bool) {
			yield(state.StateChange{}, errClosed)
		}
	}

	if len(keys) == 0 {
		return func(yield func(state.StateChange, error) bool) {}
	}

	if len(keys) == 1 {
		return state.WatchSeq(ctx, b.store, keys[0])
	}

	// For multiple keys, merge watch channels into a single stream.
	return func(yield func(state.StateChange, error) bool) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		merged := make(chan state.StateChange, 16*len(keys))
		var wg sync.WaitGroup

		for _, key := range keys {
			ch, err := b.store.Watch(ctx, key)
			if err != nil {
				yield(state.StateChange{}, err)
				return
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					case change, ok := <-ch:
						if !ok {
							return
						}
						select {
						case merged <- change:
						case <-ctx.Done():
							return
						}
					}
				}
			}()
		}

		// Close merged when all watchers complete.
		go func() {
			wg.Wait()
			close(merged)
		}()

		for {
			select {
			case <-ctx.Done():
				yield(state.StateChange{}, ctx.Err())
				return
			case change, ok := <-merged:
				if !ok {
					return
				}
				if !yield(change, nil) {
					return
				}
			}
		}
	}
}

// ClaimOwnership grants ownerID exclusive write access to key. Returns an
// error if the key is already claimed by a different owner.
func (b *Blackboard) ClaimOwnership(ctx context.Context, agentID, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return b.ownership.Claim(key, agentID)
}

// ReleaseOwnership removes the ownership claim on key. Only the current owner
// can release.
func (b *Blackboard) ReleaseOwnership(ctx context.Context, agentID, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return b.ownership.Release(key, agentID)
}

// CompareAndSwap atomically sets the value for key only if the current version
// matches expectedVersion.
func (b *Blackboard) CompareAndSwap(ctx context.Context, agentID, key string, expectedVersion uint64, value any) (uint64, error) {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return 0, errClosed
	}
	b.mu.RUnlock()

	if b.opts.enforceOwnership {
		if err := b.ownership.CheckWrite(key, agentID); err != nil {
			return 0, err
		}
	}

	return b.store.CompareAndSwap(ctx, key, expectedVersion, value)
}

// Close releases resources held by the blackboard.
func (b *Blackboard) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}
	b.closed = true
	return b.store.Close()
}

var errClosed = state.ErrStoreClosed
