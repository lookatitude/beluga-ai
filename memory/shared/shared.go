package shared

import (
	"context"
	"fmt"
	"iter"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/core"
)

// options holds configuration for SharedMemory.
type options struct {
	defaultScope      Scope
	conflictPolicy    ConflictPolicy
	hooks             Hooks
	provenanceEnabled bool
}

// Option configures a SharedMemory instance.
type Option func(*options)

// WithDefaultScope sets the default scope for fragments that do not specify one.
func WithDefaultScope(scope Scope) Option {
	return func(o *options) { o.defaultScope = scope }
}

// WithConflictPolicy sets the default conflict resolution policy.
func WithConflictPolicy(policy ConflictPolicy) Option {
	return func(o *options) { o.conflictPolicy = policy }
}

// WithHooks sets lifecycle hooks on the SharedMemory.
func WithHooks(h Hooks) Option {
	return func(o *options) { o.hooks = h }
}

// WithProvenanceEnabled enables or disables provenance tracking.
func WithProvenanceEnabled(enabled bool) Option {
	return func(o *options) { o.provenanceEnabled = enabled }
}

// SharedMemory orchestrates access-controlled, provenance-tracked shared
// memory on top of a SharedStore backend.
type SharedMemory struct {
	store SharedStore
	opts  options

	watchMu  sync.RWMutex
	watchers map[string][]chan FragmentChange
}

// NewSharedMemory creates a new SharedMemory with the given store and options.
func NewSharedMemory(store SharedStore, opts ...Option) *SharedMemory {
	o := options{
		defaultScope:      ScopeTeam,
		conflictPolicy:    LastWriteWins,
		provenanceEnabled: true,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return &SharedMemory{
		store:    store,
		opts:     o,
		watchers: make(map[string][]chan FragmentChange),
	}
}

// Write stores a fragment after checking that the caller (identified by
// frag.AuthorID) has write access. If provenance tracking is enabled, a
// provenance record is computed and attached to the fragment.
func (sm *SharedMemory) Write(ctx context.Context, frag *Fragment) error {
	if frag.Key == "" {
		return core.NewError("shared.write", core.ErrInvalidInput, "fragment key must not be empty", nil)
	}
	if frag.AuthorID == "" {
		return core.NewError("shared.write", core.ErrInvalidInput, "fragment author ID must not be empty", nil)
	}

	// Apply defaults.
	if frag.Scope == "" {
		frag.Scope = sm.opts.defaultScope
	}
	if frag.ConflictPolicy == "" {
		frag.ConflictPolicy = sm.opts.conflictPolicy
	}

	// Check write access on existing fragment.
	existing, err := sm.store.ReadFragment(ctx, frag.Key)
	if err == nil {
		// Fragment exists — check writer ACL from the existing fragment.
		if !isAllowed(existing.Writers, frag.AuthorID) {
			if sm.opts.hooks.OnDenied != nil {
				sm.opts.hooks.OnDenied(ctx, frag.Key, frag.AuthorID, PermWrite)
			}
			return core.NewError(
				"shared.write",
				core.ErrAuth,
				fmt.Sprintf("agent %q is not authorized to write fragment %q", frag.AuthorID, frag.Key),
				nil,
			)
		}
	}

	now := time.Now()
	if frag.CreatedAt.IsZero() {
		if existing != nil {
			frag.CreatedAt = existing.CreatedAt
		} else {
			frag.CreatedAt = now
		}
	}
	frag.UpdatedAt = now

	if err := sm.store.WriteFragment(ctx, frag); err != nil {
		return err
	}

	// Compute provenance AFTER WriteFragment so that frag.Content
	// reflects the final stored value (the AppendOnly conflict policy
	// mutates Content to the accumulated string). Without this ordering,
	// Provenance.Verify(frag.Content) would fail on every subsequent
	// version. We then patch only the Provenance field on the stored
	// fragment via UpdateProvenance so the version is not bumped again.
	if sm.opts.provenanceEnabled {
		var parentHash [32]byte
		if existing != nil && existing.Provenance != nil {
			parentHash = existing.Provenance.ContentHash
		}
		frag.Provenance = ComputeProvenance(frag.Content, frag.AuthorID, parentHash)
		if err := sm.store.UpdateProvenance(ctx, frag.Key, frag.Provenance); err != nil {
			return err
		}
	}

	if sm.opts.hooks.OnWrite != nil {
		sm.opts.hooks.OnWrite(ctx, frag)
	}

	sm.notify(frag.Key, FragmentChange{
		Key:      frag.Key,
		Fragment: frag,
		Op:       OpWrite,
	})

	return nil
}

// Read retrieves a fragment by key after checking that the caller has read
// access.
func (sm *SharedMemory) Read(ctx context.Context, key string, agentID string) (*Fragment, error) {
	if key == "" {
		return nil, core.NewError("shared.read", core.ErrInvalidInput, "key must not be empty", nil)
	}
	if agentID == "" {
		return nil, core.NewError("shared.read", core.ErrInvalidInput, "agent ID must not be empty", nil)
	}

	frag, err := sm.store.ReadFragment(ctx, key)
	if err != nil {
		return nil, err
	}

	if !isAllowed(frag.Readers, agentID) {
		if sm.opts.hooks.OnDenied != nil {
			sm.opts.hooks.OnDenied(ctx, key, agentID, PermRead)
		}
		return nil, core.NewError(
			"shared.read",
			core.ErrAuth,
			fmt.Sprintf("agent %q is not authorized to read fragment %q", agentID, key),
			nil,
		)
	}

	if sm.opts.hooks.OnRead != nil {
		sm.opts.hooks.OnRead(ctx, frag)
	}

	return frag, nil
}

// List returns fragments matching the given scope that agentID is
// authorised to read. Fragments whose Readers ACL does not include
// agentID are filtered out. An empty agentID is rejected.
func (sm *SharedMemory) List(ctx context.Context, scope Scope, agentID string) ([]*Fragment, error) {
	if agentID == "" {
		return nil, core.NewError("shared.list", core.ErrInvalidInput, "agent ID must not be empty", nil)
	}
	all, err := sm.store.ListFragments(ctx, scope)
	if err != nil {
		return nil, err
	}
	filtered := make([]*Fragment, 0, len(all))
	for _, frag := range all {
		if isAllowed(frag.Readers, agentID) {
			filtered = append(filtered, frag)
		}
	}
	return filtered, nil
}

// Delete removes a fragment by key after checking that the caller has write
// access.
func (sm *SharedMemory) Delete(ctx context.Context, key string, agentID string) error {
	if key == "" {
		return core.NewError("shared.delete", core.ErrInvalidInput, "key must not be empty", nil)
	}

	frag, err := sm.store.ReadFragment(ctx, key)
	if err != nil {
		return err
	}

	if !isAllowed(frag.Writers, agentID) {
		if sm.opts.hooks.OnDenied != nil {
			sm.opts.hooks.OnDenied(ctx, key, agentID, PermWrite)
		}
		return core.NewError(
			"shared.delete",
			core.ErrAuth,
			fmt.Sprintf("agent %q is not authorized to delete fragment %q", agentID, key),
			nil,
		)
	}

	if err := sm.store.DeleteFragment(ctx, key); err != nil {
		return err
	}

	sm.notify(key, FragmentChange{
		Key: key,
		Op:  OpDelete,
	})

	return nil
}

// Watch returns an iterator over FragmentChange notifications for the given
// key. The subscription is established eagerly before Watch returns, so
// events produced after this call but before the caller starts iterating
// are buffered (capacity 16) and will be delivered on the first iteration.
// Events that arrive while the buffer is full are dropped.
//
// The iterator ends when ctx is cancelled or when the caller breaks out of
// the loop. A background goroutine unsubscribes the watcher when ctx is
// cancelled, so callers who never iterate the returned sequence must still
// cancel ctx to avoid leaking a watcher slot.
func (sm *SharedMemory) Watch(ctx context.Context, key string) iter.Seq2[FragmentChange, error] {
	ch := make(chan FragmentChange, 16)

	sm.watchMu.Lock()
	sm.watchers[key] = append(sm.watchers[key], ch)
	sm.watchMu.Unlock()

	var unsubOnce sync.Once
	unsub := func() {
		unsubOnce.Do(func() {
			sm.watchMu.Lock()
			defer sm.watchMu.Unlock()
			watchers := sm.watchers[key]
			for i, w := range watchers {
				if w == ch {
					sm.watchers[key] = append(watchers[:i], watchers[i+1:]...)
					return
				}
			}
		})
	}

	// Unsubscribe eagerly on ctx cancellation so callers who never iterate
	// the returned sequence don't leak a watcher slot.
	go func() {
		<-ctx.Done()
		unsub()
	}()

	return func(yield func(FragmentChange, error) bool) {
		defer unsub()
		for {
			select {
			case change := <-ch:
				if !yield(change, nil) {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}
}

// Grant adds agentID to the readers or writers list of the fragment
// identified by key. The grantor must have write access to the fragment.
func (sm *SharedMemory) Grant(ctx context.Context, key string, grantor string, agentID string, perm Permission) error {
	if key == "" || agentID == "" {
		return core.NewError("shared.grant", core.ErrInvalidInput, "key and agent ID must not be empty", nil)
	}

	frag, err := sm.store.ReadFragment(ctx, key)
	if err != nil {
		return err
	}

	if !isAllowed(frag.Writers, grantor) {
		if sm.opts.hooks.OnDenied != nil {
			sm.opts.hooks.OnDenied(ctx, key, grantor, PermWrite)
		}
		return core.NewError(
			"shared.grant",
			core.ErrAuth,
			fmt.Sprintf("agent %q is not authorized to grant permissions on fragment %q", grantor, key),
			nil,
		)
	}

	switch perm {
	case PermRead:
		if !contains(frag.Readers, agentID) {
			frag.Readers = append(frag.Readers, agentID)
		}
	case PermWrite:
		if !contains(frag.Writers, agentID) {
			frag.Writers = append(frag.Writers, agentID)
		}
	default:
		return core.NewError("shared.grant", core.ErrInvalidInput, fmt.Sprintf("unknown permission %q", perm), nil)
	}

	if err := sm.store.WriteFragment(ctx, frag); err != nil {
		return err
	}

	if sm.opts.hooks.OnGrant != nil {
		sm.opts.hooks.OnGrant(ctx, key, agentID, perm)
	}

	sm.notify(key, FragmentChange{
		Key:      key,
		Fragment: frag,
		Op:       OpGrant,
	})

	return nil
}

// Revoke removes agentID from the readers or writers list of the fragment
// identified by key. The revoker must have write access to the fragment.
func (sm *SharedMemory) Revoke(ctx context.Context, key string, revoker string, agentID string, perm Permission) error {
	if key == "" || agentID == "" {
		return core.NewError("shared.revoke", core.ErrInvalidInput, "key and agent ID must not be empty", nil)
	}

	frag, err := sm.store.ReadFragment(ctx, key)
	if err != nil {
		return err
	}

	if !isAllowed(frag.Writers, revoker) {
		if sm.opts.hooks.OnDenied != nil {
			sm.opts.hooks.OnDenied(ctx, key, revoker, PermWrite)
		}
		return core.NewError(
			"shared.revoke",
			core.ErrAuth,
			fmt.Sprintf("agent %q is not authorized to revoke permissions on fragment %q", revoker, key),
			nil,
		)
	}

	switch perm {
	case PermRead:
		frag.Readers = removeFromSlice(frag.Readers, agentID)
	case PermWrite:
		frag.Writers = removeFromSlice(frag.Writers, agentID)
	default:
		return core.NewError("shared.revoke", core.ErrInvalidInput, fmt.Sprintf("unknown permission %q", perm), nil)
	}

	if err := sm.store.WriteFragment(ctx, frag); err != nil {
		return err
	}

	sm.notify(key, FragmentChange{
		Key:      key,
		Fragment: frag,
		Op:       OpRevoke,
	})

	return nil
}

// notify sends a change to all watchers for the given key.
func (sm *SharedMemory) notify(key string, change FragmentChange) {
	sm.watchMu.RLock()
	defer sm.watchMu.RUnlock()

	for _, ch := range sm.watchers[key] {
		select {
		case ch <- change:
		default:
			// Drop if the watcher is not keeping up.
		}
	}
}

// isAllowed checks whether agentID appears in the access list. An empty
// list means unrestricted access.
func isAllowed(acl []string, agentID string) bool {
	if len(acl) == 0 {
		return true
	}
	return contains(acl, agentID)
}

// contains reports whether s contains v.
func contains(s []string, v string) bool {
	for _, item := range s {
		if item == v {
			return true
		}
	}
	return false
}

// removeFromSlice returns a new slice with all occurrences of v removed.
func removeFromSlice(s []string, v string) []string {
	result := make([]string, 0, len(s))
	for _, item := range s {
		if item != v {
			result = append(result, item)
		}
	}
	return result
}
