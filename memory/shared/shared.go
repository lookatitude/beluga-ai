package shared

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// Operation names attached to errors returned by SharedMemory methods.
const (
	opNameWrite  = "shared.write"
	opNameRead   = "shared.read"
	opNameGrant  = "shared.grant"
	opNameRevoke = "shared.revoke"
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
type SharedMemory struct { //nolint:revive // stutter accepted: renaming this exported type is a breaking API change
	store SharedStore
	opts  options

	watchMu  sync.RWMutex
	watchers map[string][]*watcher
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
		watchers: make(map[string][]*watcher),
	}
}

// Write stores a fragment after checking that the caller (identified by
// frag.AuthorID) has write access. If provenance tracking is enabled, a
// provenance record is computed and attached to the fragment.
//
// When the fragment is new, its Readers/Writers lists, Unrestricted* flags
// and ConflictPolicy are taken from frag. An empty Readers (or Writers) list
// without the matching Unrestricted flag defaults to the author only, so an
// unset ACL never means "everyone".
//
// When the fragment already exists, its access-control fields and conflict
// policy are preserved: a content-only update (nil ACL fields) keeps the
// stored ACL, and frag.ConflictPolicy is ignored. Sending a different ACL in
// an update is rejected with core.ErrInvalidInput; use [SharedMemory.Grant]
// and [SharedMemory.Revoke] to change access.
func (sm *SharedMemory) Write(ctx context.Context, frag *Fragment) error {
	if frag.Key == "" {
		return core.NewError(opNameWrite, core.ErrInvalidInput, "fragment key must not be empty", nil)
	}
	if frag.AuthorID == "" {
		return core.NewError(opNameWrite, core.ErrInvalidInput, "fragment author ID must not be empty", nil)
	}

	if frag.Scope == "" {
		frag.Scope = sm.opts.defaultScope
	}

	existing, err := sm.store.ReadFragment(ctx, frag.Key)
	switch {
	case err == nil:
		if err := sm.checkUpdate(ctx, frag, existing); err != nil {
			return err
		}
	case isNotFound(err):
		existing = nil
		applyCreateDefaults(frag, sm.opts.conflictPolicy)
	default:
		// Any other read failure (backend outage, cancelled ctx) must not
		// skip the writer-ACL check: fail closed.
		return err
	}

	stampTimes(frag, existing)

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
		if err := sm.attachProvenance(ctx, frag, existing); err != nil {
			return err
		}
	}

	if sm.opts.hooks.OnWrite != nil {
		sm.opts.hooks.OnWrite(ctx, frag)
	}

	sm.notify(frag, FragmentChange{
		Key:      frag.Key,
		Fragment: frag,
		Op:       OpWrite,
	})

	return nil
}

// checkUpdate enforces the writer ACL of an existing fragment and carries its
// access-control fields and conflict policy over to frag.
func (sm *SharedMemory) checkUpdate(ctx context.Context, frag, existing *Fragment) error {
	if !canWrite(existing, frag.AuthorID) {
		sm.denied(ctx, frag.Key, frag.AuthorID, PermWrite)
		return core.NewError(
			opNameWrite,
			core.ErrAuth,
			fmt.Sprintf("agent %q is not authorized to write fragment %q", frag.AuthorID, frag.Key),
			nil,
		)
	}
	if aclChanged(frag, existing) {
		return core.NewError(
			opNameWrite,
			core.ErrInvalidInput,
			fmt.Sprintf("fragment %q: access lists cannot be changed by Write; use Grant or Revoke", frag.Key),
			nil,
		)
	}
	frag.Readers = cloneStrings(existing.Readers)
	frag.Writers = cloneStrings(existing.Writers)
	frag.UnrestrictedRead = existing.UnrestrictedRead
	frag.UnrestrictedWrite = existing.UnrestrictedWrite
	frag.ConflictPolicy = existing.ConflictPolicy
	if frag.ConflictPolicy == "" {
		frag.ConflictPolicy = sm.opts.conflictPolicy
	}
	return nil
}

// applyCreateDefaults fills in the conflict policy and the owner-only default
// ACL of a fragment that is being created.
func applyCreateDefaults(frag *Fragment, defaultPolicy ConflictPolicy) {
	if frag.ConflictPolicy == "" {
		frag.ConflictPolicy = defaultPolicy
	}
	if len(frag.Readers) == 0 && !frag.UnrestrictedRead {
		frag.Readers = []string{frag.AuthorID}
	}
	if len(frag.Writers) == 0 && !frag.UnrestrictedWrite {
		frag.Writers = []string{frag.AuthorID}
	}
}

// aclChanged reports whether an update carries access-control fields that
// differ from the stored fragment. Nil/empty lists and unset flags mean
// "keep the stored ACL" and never count as a change.
func aclChanged(frag, existing *Fragment) bool {
	if len(frag.Readers) > 0 && !sameSet(frag.Readers, existing.Readers) {
		return true
	}
	if len(frag.Writers) > 0 && !sameSet(frag.Writers, existing.Writers) {
		return true
	}
	if frag.UnrestrictedRead && !existing.UnrestrictedRead {
		return true
	}
	return frag.UnrestrictedWrite && !existing.UnrestrictedWrite
}

// sameSet reports whether a and b contain the same distinct elements.
func sameSet(a, b []string) bool {
	for _, v := range a {
		if !contains(b, v) {
			return false
		}
	}
	for _, v := range b {
		if !contains(a, v) {
			return false
		}
	}
	return true
}

// isNotFound reports whether err carries core.ErrNotFound.
func isNotFound(err error) bool {
	var ce *core.Error
	return errors.As(err, &ce) && ce.Code == core.ErrNotFound
}

// denied invokes the OnDenied hook, if set.
func (sm *SharedMemory) denied(ctx context.Context, key, agentID string, perm Permission) {
	if sm.opts.hooks.OnDenied != nil {
		sm.opts.hooks.OnDenied(ctx, key, agentID, perm)
	}
}

// stampTimes sets frag.UpdatedAt to now and, when frag.CreatedAt is unset,
// inherits it from the existing fragment (if any) or sets it to now.
func stampTimes(frag, existing *Fragment) {
	now := time.Now()
	if frag.CreatedAt.IsZero() {
		if existing != nil {
			frag.CreatedAt = existing.CreatedAt
		} else {
			frag.CreatedAt = now
		}
	}
	frag.UpdatedAt = now
}

// attachProvenance computes the provenance of the stored fragment, chaining it
// to the existing fragment's content hash, and persists it without bumping the
// fragment version.
func (sm *SharedMemory) attachProvenance(ctx context.Context, frag, existing *Fragment) error {
	var parentHash [32]byte
	if existing != nil && existing.Provenance != nil {
		parentHash = existing.Provenance.ContentHash
	}
	frag.Provenance = ComputeProvenance(frag.Content, frag.AuthorID, parentHash)
	return sm.store.UpdateProvenance(ctx, frag.Key, frag.Provenance)
}

// Read retrieves a fragment by key after checking that the caller has read
// access.
func (sm *SharedMemory) Read(ctx context.Context, key string, agentID string) (*Fragment, error) {
	if key == "" {
		return nil, core.NewError(opNameRead, core.ErrInvalidInput, "key must not be empty", nil)
	}
	if agentID == "" {
		return nil, core.NewError(opNameRead, core.ErrInvalidInput, "agent ID must not be empty", nil)
	}

	frag, err := sm.store.ReadFragment(ctx, key)
	if err != nil {
		return nil, err
	}

	if !canRead(frag, agentID) {
		sm.denied(ctx, key, agentID, PermRead)
		return nil, core.NewError(
			opNameRead,
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
// authorised to read. Fragments agentID may not read are filtered out. An
// empty agentID is rejected.
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
		if canRead(frag, agentID) {
			filtered = append(filtered, frag)
		}
	}
	return filtered, nil
}

// Delete removes a fragment by key after checking that the caller has write
// access.
func (sm *SharedMemory) Delete(ctx context.Context, key string, agentID string) error {
	if key == "" || agentID == "" {
		return core.NewError("shared.delete", core.ErrInvalidInput, "key and agent ID must not be empty", nil)
	}

	frag, err := sm.store.ReadFragment(ctx, key)
	if err != nil {
		return err
	}

	if !canWrite(frag, agentID) {
		sm.denied(ctx, key, agentID, PermWrite)
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

	// Authorise delete notifications against the pre-delete ACL; the
	// delivered change carries no fragment.
	sm.notify(frag, FragmentChange{
		Key: key,
		Op:  OpDelete,
	})

	return nil
}

// Watch is the identity-less form of [SharedMemory.WatchAs]. Because it
// cannot enforce read authorization, it yields a single core.ErrInvalidInput
// error and ends.
//
// Deprecated: Watch delivered fragment contents to any caller that knew the
// key, bypassing the Readers ACL. Use [SharedMemory.WatchAs].
func (sm *SharedMemory) Watch(ctx context.Context, key string) iter.Seq2[FragmentChange, error] {
	return sm.WatchAs(ctx, key, "")
}

// WatchAs returns an iterator over FragmentChange notifications for the
// given key, delivered only while agentID is authorized to read the
// fragment. Each change is authorized against the fragment's ACL at the time
// of the change (the pre-delete ACL for deletes), so an agent whose read
// access is revoked stops receiving changes, including the revoke itself.
// If the fragment already exists and agentID may not read it, the iterator
// yields a core.ErrAuth error and ends. An empty agentID yields
// core.ErrInvalidInput.
//
// The subscription is established eagerly before WatchAs returns, so events
// produced after this call but before the caller starts iterating are
// buffered (capacity 16) and will be delivered on the first iteration.
// Events that arrive while the buffer is full are dropped.
//
// The iterator ends when ctx is cancelled or when the caller breaks out of
// the loop; either unsubscribes the watcher.
func (sm *SharedMemory) WatchAs(ctx context.Context, key, agentID string) iter.Seq2[FragmentChange, error] {
	if err := sm.authorizeWatch(ctx, key, agentID); err != nil {
		return func(yield func(FragmentChange, error) bool) {
			yield(FragmentChange{}, err)
		}
	}

	w := &watcher{agentID: agentID, ch: make(chan FragmentChange, 16), stop: make(chan struct{})}

	sm.watchMu.Lock()
	sm.watchers[key] = append(sm.watchers[key], w)
	sm.watchMu.Unlock()

	var unsubOnce sync.Once
	unsub := func() {
		unsubOnce.Do(func() {
			close(w.stop)
			sm.watchMu.Lock()
			defer sm.watchMu.Unlock()
			watchers := sm.watchers[key]
			for i, x := range watchers {
				if x == w {
					sm.watchers[key] = append(watchers[:i], watchers[i+1:]...)
					break
				}
			}
			if len(sm.watchers[key]) == 0 {
				delete(sm.watchers, key)
			}
		})
	}

	// Unsubscribe eagerly on ctx cancellation so callers who never iterate
	// the returned sequence don't leak a watcher slot. The goroutine also
	// exits when the iterator unsubscribes, so it never outlives the watch.
	go func() {
		select {
		case <-ctx.Done():
			unsub()
		case <-w.stop:
		}
	}()

	return func(yield func(FragmentChange, error) bool) {
		defer unsub()
		for {
			select {
			case change := <-w.ch:
				if !yield(change, nil) {
					return
				}
			case <-ctx.Done():
				return
			case <-w.stop:
				return
			}
		}
	}
}

// authorizeWatch validates the watch request and, when the fragment already
// exists, checks that agentID may read it. A missing fragment is allowed:
// later changes are authorized one by one.
func (sm *SharedMemory) authorizeWatch(ctx context.Context, key, agentID string) error {
	if key == "" || agentID == "" {
		return core.NewError("shared.watch", core.ErrInvalidInput, "key and agent ID must not be empty", nil)
	}
	frag, err := sm.store.ReadFragment(ctx, key)
	if err != nil {
		if isNotFound(err) {
			return nil
		}
		return err
	}
	if !canRead(frag, agentID) {
		sm.denied(ctx, key, agentID, PermRead)
		return core.NewError(
			"shared.watch",
			core.ErrAuth,
			fmt.Sprintf("agent %q is not authorized to watch fragment %q", agentID, key),
			nil,
		)
	}
	return nil
}

// watcher is one WatchAs subscription.
type watcher struct {
	agentID string
	ch      chan FragmentChange
	stop    chan struct{}
}

// Grant adds agentID to the readers or writers list of the fragment
// identified by key. The grantor must have write access to the fragment.
func (sm *SharedMemory) Grant(ctx context.Context, key string, grantor string, agentID string, perm Permission) error {
	if key == "" || agentID == "" || grantor == "" {
		return core.NewError(opNameGrant, core.ErrInvalidInput, "key, grantor and agent ID must not be empty", nil)
	}

	frag, err := sm.authorizeACLChange(ctx, opNameGrant, key, grantor)
	if err != nil {
		return err
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
		return core.NewError(opNameGrant, core.ErrInvalidInput, fmt.Sprintf("unknown permission %q", perm), nil)
	}

	if err := sm.storeACL(ctx, frag); err != nil {
		return err
	}

	if sm.opts.hooks.OnGrant != nil {
		sm.opts.hooks.OnGrant(ctx, key, agentID, perm)
	}

	sm.notify(frag, FragmentChange{
		Key:      key,
		Fragment: frag,
		Op:       OpGrant,
	})

	return nil
}

// Revoke removes agentID from the readers or writers list of the fragment
// identified by key. The revoker must have write access to the fragment.
// Revoking the last entry leaves an empty list, which denies everyone.
// Revoking from an unrestricted permission is rejected with
// core.ErrInvalidInput, because "everyone except agentID" cannot be
// expressed.
func (sm *SharedMemory) Revoke(ctx context.Context, key string, revoker string, agentID string, perm Permission) error {
	if key == "" || agentID == "" || revoker == "" {
		return core.NewError(opNameRevoke, core.ErrInvalidInput, "key, revoker and agent ID must not be empty", nil)
	}

	frag, err := sm.authorizeACLChange(ctx, opNameRevoke, key, revoker)
	if err != nil {
		return err
	}

	switch perm {
	case PermRead:
		if frag.UnrestrictedRead {
			return unrestrictedRevokeErr(key, perm)
		}
		frag.Readers = removeFromSlice(frag.Readers, agentID)
	case PermWrite:
		if frag.UnrestrictedWrite {
			return unrestrictedRevokeErr(key, perm)
		}
		frag.Writers = removeFromSlice(frag.Writers, agentID)
	default:
		return core.NewError(opNameRevoke, core.ErrInvalidInput, fmt.Sprintf("unknown permission %q", perm), nil)
	}

	if err := sm.storeACL(ctx, frag); err != nil {
		return err
	}

	sm.notify(frag, FragmentChange{
		Key:      key,
		Fragment: frag,
		Op:       OpRevoke,
	})

	return nil
}

func unrestrictedRevokeErr(key string, perm Permission) error {
	return core.NewError(
		opNameRevoke,
		core.ErrInvalidInput,
		fmt.Sprintf("fragment %q has unrestricted %s access; individual agents cannot be revoked", key, perm),
		nil,
	)
}

// authorizeACLChange reads the fragment and checks that actor may change its
// access lists.
func (sm *SharedMemory) authorizeACLChange(ctx context.Context, op, key, actor string) (*Fragment, error) {
	frag, err := sm.store.ReadFragment(ctx, key)
	if err != nil {
		return nil, err
	}
	if !canWrite(frag, actor) {
		sm.denied(ctx, key, actor, PermWrite)
		return nil, core.NewError(
			op,
			core.ErrAuth,
			fmt.Sprintf("agent %q is not authorized to change permissions on fragment %q", actor, key),
			nil,
		)
	}
	return frag, nil
}

// storeACL persists only the access-control fields when the store supports
// it, so an ACL change neither bumps the version nor re-applies the content
// conflict policy.
func (sm *SharedMemory) storeACL(ctx context.Context, frag *Fragment) error {
	if u, ok := sm.store.(ACLUpdater); ok {
		return u.UpdateACL(ctx, frag)
	}
	return sm.store.WriteFragment(ctx, frag)
}

// notify sends change to every watcher of authz.Key that may read authz.
// Each watcher receives its own copy of the fragment.
func (sm *SharedMemory) notify(authz *Fragment, change FragmentChange) {
	sm.watchMu.RLock()
	defer sm.watchMu.RUnlock()

	for _, w := range sm.watchers[authz.Key] {
		if !canRead(authz, w.agentID) {
			continue
		}
		c := change
		if change.Fragment != nil {
			c.Fragment = cloneFragment(change.Fragment)
		}
		select {
		case w.ch <- c:
		default:
			// Drop if the watcher is not keeping up.
		}
	}
}

// canRead reports whether agentID may read frag.
func canRead(frag *Fragment, agentID string) bool {
	return agentID != "" && (frag.UnrestrictedRead || contains(frag.Readers, agentID))
}

// canWrite reports whether agentID may write frag.
func canWrite(frag *Fragment, agentID string) bool {
	return agentID != "" && (frag.UnrestrictedWrite || contains(frag.Writers, agentID))
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
