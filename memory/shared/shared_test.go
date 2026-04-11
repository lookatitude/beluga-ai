package shared

import (
	"context"
	"crypto/sha256"
	"iter"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestMemory(opts ...Option) (*SharedMemory, *InMemorySharedStore) {
	store := NewInMemorySharedStore()
	sm := NewSharedMemory(store, opts...)
	return sm, store
}

// seedFragment writes a fragment directly via the store, bypassing ACL checks.
func seedFragment(t *testing.T, store *InMemorySharedStore, frag *Fragment) {
	t.Helper()
	if frag.CreatedAt.IsZero() {
		frag.CreatedAt = time.Now()
	}
	frag.UpdatedAt = frag.CreatedAt
	err := store.WriteFragment(context.Background(), frag)
	require.NoError(t, err)
}

// --- ACL enforcement tests ---

func TestACL_ReadDenied(t *testing.T) {
	sm, store := newTestMemory()
	seedFragment(t, store, &Fragment{
		Key:      "secret",
		Content:  "classified",
		AuthorID: "agent-1",
		Readers:  []string{"agent-1"},
		Writers:  []string{"agent-1"},
	})

	_, err := sm.Read(context.Background(), "secret", "agent-2")
	require.Error(t, err)
	var coreErr *core.Error
	require.ErrorAs(t, err, &coreErr)
	assert.Equal(t, core.ErrAuth, coreErr.Code)
}

func TestACL_ReadAllowed(t *testing.T) {
	sm, store := newTestMemory()
	seedFragment(t, store, &Fragment{
		Key:      "shared-doc",
		Content:  "hello",
		AuthorID: "agent-1",
		Readers:  []string{"agent-1", "agent-2"},
		Writers:  []string{"agent-1"},
	})

	frag, err := sm.Read(context.Background(), "shared-doc", "agent-2")
	require.NoError(t, err)
	assert.Equal(t, "hello", frag.Content)
}

func TestACL_ReadUnrestricted(t *testing.T) {
	sm, store := newTestMemory()
	seedFragment(t, store, &Fragment{
		Key:      "open",
		Content:  "public",
		AuthorID: "agent-1",
		Readers:  nil, // unrestricted
	})

	frag, err := sm.Read(context.Background(), "open", "anyone")
	require.NoError(t, err)
	assert.Equal(t, "public", frag.Content)
}

func TestACL_WriteDenied(t *testing.T) {
	sm, store := newTestMemory()
	seedFragment(t, store, &Fragment{
		Key:      "locked",
		Content:  "original",
		AuthorID: "agent-1",
		Writers:  []string{"agent-1"},
	})

	err := sm.Write(context.Background(), &Fragment{
		Key:      "locked",
		Content:  "overwrite",
		AuthorID: "agent-2",
	})
	require.Error(t, err)
	var coreErr *core.Error
	require.ErrorAs(t, err, &coreErr)
	assert.Equal(t, core.ErrAuth, coreErr.Code)
}

func TestACL_WriteAllowed(t *testing.T) {
	sm, store := newTestMemory()
	seedFragment(t, store, &Fragment{
		Key:      "collab",
		Content:  "v1",
		AuthorID: "agent-1",
		Writers:  []string{"agent-1", "agent-2"},
	})

	err := sm.Write(context.Background(), &Fragment{
		Key:      "collab",
		Content:  "v2",
		AuthorID: "agent-2",
	})
	require.NoError(t, err)

	frag, err := sm.Read(context.Background(), "collab", "agent-2")
	require.NoError(t, err)
	assert.Equal(t, "v2", frag.Content)
}

func TestACL_DeleteDenied(t *testing.T) {
	sm, store := newTestMemory()
	seedFragment(t, store, &Fragment{
		Key:      "protected",
		Content:  "important",
		AuthorID: "agent-1",
		Writers:  []string{"agent-1"},
	})

	err := sm.Delete(context.Background(), "protected", "agent-2")
	require.Error(t, err)
	var coreErr *core.Error
	require.ErrorAs(t, err, &coreErr)
	assert.Equal(t, core.ErrAuth, coreErr.Code)
}

// --- Conflict policy tests ---

func TestConflict_AppendOnly(t *testing.T) {
	sm, _ := newTestMemory(WithConflictPolicy(AppendOnly))
	ctx := context.Background()

	err := sm.Write(ctx, &Fragment{
		Key:      "log",
		Content:  "line1\n",
		AuthorID: "agent-1",
	})
	require.NoError(t, err)

	err = sm.Write(ctx, &Fragment{
		Key:      "log",
		Content:  "line2\n",
		AuthorID: "agent-1",
	})
	require.NoError(t, err)

	frag, err := sm.Read(ctx, "log", "agent-1")
	require.NoError(t, err)
	assert.Equal(t, "line1\nline2\n", frag.Content)
	assert.Equal(t, int64(2), frag.Version)
}

func TestConflict_LastWriteWins(t *testing.T) {
	sm, _ := newTestMemory(WithConflictPolicy(LastWriteWins))
	ctx := context.Background()

	err := sm.Write(ctx, &Fragment{
		Key:      "data",
		Content:  "old",
		AuthorID: "agent-1",
	})
	require.NoError(t, err)

	err = sm.Write(ctx, &Fragment{
		Key:      "data",
		Content:  "new",
		AuthorID: "agent-1",
	})
	require.NoError(t, err)

	frag, err := sm.Read(ctx, "data", "agent-1")
	require.NoError(t, err)
	assert.Equal(t, "new", frag.Content)
	assert.Equal(t, int64(2), frag.Version)
}

func TestConflict_RejectOnConflict(t *testing.T) {
	sm, _ := newTestMemory(WithConflictPolicy(RejectOnConflict))
	ctx := context.Background()

	err := sm.Write(ctx, &Fragment{
		Key:      "versioned",
		Content:  "v1",
		AuthorID: "agent-1",
	})
	require.NoError(t, err)

	// Write with correct version succeeds.
	err = sm.Write(ctx, &Fragment{
		Key:      "versioned",
		Content:  "v2",
		AuthorID: "agent-1",
		Version:  1,
	})
	require.NoError(t, err)

	// Write with stale version fails.
	err = sm.Write(ctx, &Fragment{
		Key:      "versioned",
		Content:  "v3",
		AuthorID: "agent-1",
		Version:  1, // stale
	})
	require.Error(t, err)
	var coreErr *core.Error
	require.ErrorAs(t, err, &coreErr)
	assert.Equal(t, core.ErrInvalidInput, coreErr.Code)
}

// --- Provenance tests ---

func TestProvenance_Chain(t *testing.T) {
	sm, _ := newTestMemory(WithProvenanceEnabled(true))
	ctx := context.Background()

	err := sm.Write(ctx, &Fragment{
		Key:      "doc",
		Content:  "first",
		AuthorID: "agent-1",
	})
	require.NoError(t, err)

	frag1, err := sm.Read(ctx, "doc", "agent-1")
	require.NoError(t, err)
	require.NotNil(t, frag1.Provenance)
	assert.Equal(t, "agent-1", frag1.Provenance.AuthorID)
	assert.True(t, frag1.Provenance.Verify("first"))
	assert.Equal(t, [32]byte{}, frag1.Provenance.ParentHash) // first version has zero parent

	// Second write should chain.
	err = sm.Write(ctx, &Fragment{
		Key:      "doc",
		Content:  "second",
		AuthorID: "agent-2",
	})
	require.NoError(t, err)

	frag2, err := sm.Read(ctx, "doc", "agent-2")
	require.NoError(t, err)
	require.NotNil(t, frag2.Provenance)
	assert.True(t, frag2.Provenance.Verify("second"))
	// Parent hash should be the content hash of "first".
	assert.Equal(t, sha256.Sum256([]byte("first")), frag2.Provenance.ParentHash)
}

func TestProvenance_Disabled(t *testing.T) {
	sm, _ := newTestMemory(WithProvenanceEnabled(false))
	ctx := context.Background()

	err := sm.Write(ctx, &Fragment{
		Key:      "doc",
		Content:  "data",
		AuthorID: "agent-1",
	})
	require.NoError(t, err)

	frag, err := sm.Read(ctx, "doc", "agent-1")
	require.NoError(t, err)
	assert.Nil(t, frag.Provenance)
}

func TestProvenance_Verify(t *testing.T) {
	p := ComputeProvenance("hello world", "agent-1", [32]byte{})
	assert.True(t, p.Verify("hello world"))
	assert.False(t, p.Verify("hello worl"))
	assert.False(t, p.Verify(""))
}

// --- Grant / Revoke tests ---

func TestGrant_ReadAccess(t *testing.T) {
	sm, _ := newTestMemory()
	ctx := context.Background()

	// Create with restricted readers.
	err := sm.Write(ctx, &Fragment{
		Key:      "restricted",
		Content:  "data",
		AuthorID: "agent-1",
		Readers:  []string{"agent-1"},
		Writers:  []string{"agent-1"},
	})
	require.NoError(t, err)

	// agent-2 cannot read.
	_, err = sm.Read(ctx, "restricted", "agent-2")
	require.Error(t, err)

	// Grant read to agent-2.
	err = sm.Grant(ctx, "restricted", "agent-1", "agent-2", PermRead)
	require.NoError(t, err)

	// Now agent-2 can read.
	frag, err := sm.Read(ctx, "restricted", "agent-2")
	require.NoError(t, err)
	assert.Equal(t, "data", frag.Content)
}

func TestGrant_WriteAccess(t *testing.T) {
	sm, _ := newTestMemory()
	ctx := context.Background()

	err := sm.Write(ctx, &Fragment{
		Key:      "team-doc",
		Content:  "initial",
		AuthorID: "agent-1",
		Writers:  []string{"agent-1"},
	})
	require.NoError(t, err)

	// agent-2 cannot write.
	err = sm.Write(ctx, &Fragment{
		Key:      "team-doc",
		Content:  "update",
		AuthorID: "agent-2",
	})
	require.Error(t, err)

	// Grant write to agent-2.
	err = sm.Grant(ctx, "team-doc", "agent-1", "agent-2", PermWrite)
	require.NoError(t, err)

	// Now agent-2 can write.
	err = sm.Write(ctx, &Fragment{
		Key:      "team-doc",
		Content:  "update",
		AuthorID: "agent-2",
	})
	require.NoError(t, err)
}

func TestGrant_DeniedForNonWriter(t *testing.T) {
	sm, _ := newTestMemory()
	ctx := context.Background()

	err := sm.Write(ctx, &Fragment{
		Key:      "locked",
		Content:  "data",
		AuthorID: "agent-1",
		Writers:  []string{"agent-1"},
	})
	require.NoError(t, err)

	// agent-2 (non-writer) tries to grant.
	err = sm.Grant(ctx, "locked", "agent-2", "agent-3", PermRead)
	require.Error(t, err)
	var coreErr *core.Error
	require.ErrorAs(t, err, &coreErr)
	assert.Equal(t, core.ErrAuth, coreErr.Code)
}

func TestRevoke_ReadAccess(t *testing.T) {
	sm, _ := newTestMemory()
	ctx := context.Background()

	err := sm.Write(ctx, &Fragment{
		Key:      "doc",
		Content:  "data",
		AuthorID: "agent-1",
		Readers:  []string{"agent-1", "agent-2"},
		Writers:  []string{"agent-1"},
	})
	require.NoError(t, err)

	// agent-2 can read.
	_, err = sm.Read(ctx, "doc", "agent-2")
	require.NoError(t, err)

	// Revoke read from agent-2.
	err = sm.Revoke(ctx, "doc", "agent-1", "agent-2", PermRead)
	require.NoError(t, err)

	// agent-2 can no longer read.
	_, err = sm.Read(ctx, "doc", "agent-2")
	require.Error(t, err)
}

// --- Watch tests ---

func TestWatch_ReceivesWriteNotification(t *testing.T) {
	sm, _ := newTestMemory()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Watch eagerly subscribes, so events produced after this call are
	// buffered even before we start pulling.
	next, stop := iter.Pull2(sm.Watch(ctx, "watched-key"))
	defer stop()

	err := sm.Write(context.Background(), &Fragment{
		Key:      "watched-key",
		Content:  "hello",
		AuthorID: "agent-1",
	})
	require.NoError(t, err)

	change, pullErr, ok := next()
	require.True(t, ok, "expected a notification before ctx timeout")
	require.NoError(t, pullErr)
	assert.Equal(t, OpWrite, change.Op)
	assert.Equal(t, "watched-key", change.Key)
	require.NotNil(t, change.Fragment)
	assert.Equal(t, "hello", change.Fragment.Content)
}

func TestWatch_CancelUnsubscribes(t *testing.T) {
	sm, _ := newTestMemory()
	ctx, cancel := context.WithCancel(context.Background())

	next, stop := iter.Pull2(sm.Watch(ctx, "key"))
	defer stop()

	cancel()

	// The iterator should end after ctx cancellation.
	_, _, ok := next()
	assert.False(t, ok, "iterator should end after context cancel")

	// Wait briefly for the background unsubscribe goroutine.
	time.Sleep(50 * time.Millisecond)

	// No watchers should remain.
	sm.watchMu.RLock()
	assert.Empty(t, sm.watchers["key"])
	sm.watchMu.RUnlock()
}

// --- Thread safety tests ---

func TestConcurrentWriteRead(t *testing.T) {
	sm, _ := newTestMemory(WithConflictPolicy(LastWriteWins))
	ctx := context.Background()

	// Seed the fragment.
	err := sm.Write(ctx, &Fragment{
		Key:      "concurrent",
		Content:  "init",
		AuthorID: "agent-0",
	})
	require.NoError(t, err)

	var wg sync.WaitGroup
	const writers = 10
	const readers = 10

	for i := range writers {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			writeErr := sm.Write(ctx, &Fragment{
				Key:      "concurrent",
				Content:  "data",
				AuthorID: "agent-0",
			})
			assert.NoError(t, writeErr, "writer %d failed", id)
		}(i)
	}

	for i := range readers {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, readErr := sm.Read(ctx, "concurrent", "agent-0")
			assert.NoError(t, readErr, "reader %d failed", id)
		}(i)
	}

	wg.Wait()
}

// --- Hooks tests ---

func TestHooks_OnDeniedCalled(t *testing.T) {
	var deniedKey, deniedAgent string
	var deniedPerm Permission

	sm, store := newTestMemory(WithHooks(Hooks{
		OnDenied: func(_ context.Context, key, agentID string, perm Permission) {
			deniedKey = key
			deniedAgent = agentID
			deniedPerm = perm
		},
	}))

	seedFragment(t, store, &Fragment{
		Key:      "secret",
		Content:  "data",
		AuthorID: "agent-1",
		Readers:  []string{"agent-1"},
	})

	_, _ = sm.Read(context.Background(), "secret", "agent-2")

	assert.Equal(t, "secret", deniedKey)
	assert.Equal(t, "agent-2", deniedAgent)
	assert.Equal(t, PermRead, deniedPerm)
}

func TestHooks_OnWriteCalled(t *testing.T) {
	var writtenKey string
	sm, _ := newTestMemory(WithHooks(Hooks{
		OnWrite: func(_ context.Context, frag *Fragment) {
			writtenKey = frag.Key
		},
	}))

	err := sm.Write(context.Background(), &Fragment{
		Key:      "noted",
		Content:  "data",
		AuthorID: "agent-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "noted", writtenKey)
}

func TestHooks_OnGrantCalled(t *testing.T) {
	var grantedAgent string
	var grantedPerm Permission
	sm, _ := newTestMemory(WithHooks(Hooks{
		OnGrant: func(_ context.Context, _ string, agentID string, perm Permission) {
			grantedAgent = agentID
			grantedPerm = perm
		},
	}))

	ctx := context.Background()
	err := sm.Write(ctx, &Fragment{
		Key:      "doc",
		Content:  "data",
		AuthorID: "agent-1",
	})
	require.NoError(t, err)

	err = sm.Grant(ctx, "doc", "agent-1", "agent-2", PermRead)
	require.NoError(t, err)

	assert.Equal(t, "agent-2", grantedAgent)
	assert.Equal(t, PermRead, grantedPerm)
}

func TestComposeHooks(t *testing.T) {
	var calls []string
	h1 := Hooks{
		OnWrite: func(_ context.Context, _ *Fragment) { calls = append(calls, "h1") },
	}
	h2 := Hooks{
		OnWrite: func(_ context.Context, _ *Fragment) { calls = append(calls, "h2") },
	}

	composed := ComposeHooks(h1, h2)
	composed.OnWrite(context.Background(), &Fragment{})

	assert.Equal(t, []string{"h1", "h2"}, calls)
}

// --- Input validation tests ---

func TestWrite_EmptyKey(t *testing.T) {
	sm, _ := newTestMemory()
	err := sm.Write(context.Background(), &Fragment{
		Content:  "data",
		AuthorID: "agent-1",
	})
	require.Error(t, err)
	var coreErr *core.Error
	require.ErrorAs(t, err, &coreErr)
	assert.Equal(t, core.ErrInvalidInput, coreErr.Code)
}

func TestWrite_EmptyAuthorID(t *testing.T) {
	sm, _ := newTestMemory()
	err := sm.Write(context.Background(), &Fragment{
		Key:     "key",
		Content: "data",
	})
	require.Error(t, err)
	var coreErr *core.Error
	require.ErrorAs(t, err, &coreErr)
	assert.Equal(t, core.ErrInvalidInput, coreErr.Code)
}

func TestRead_EmptyKey(t *testing.T) {
	sm, _ := newTestMemory()
	_, err := sm.Read(context.Background(), "", "agent-1")
	require.Error(t, err)
}

func TestRead_NotFound(t *testing.T) {
	sm, _ := newTestMemory()
	_, err := sm.Read(context.Background(), "nonexistent", "agent-1")
	require.Error(t, err)
	var coreErr *core.Error
	require.ErrorAs(t, err, &coreErr)
	assert.Equal(t, core.ErrNotFound, coreErr.Code)
}

// --- Store tests ---

func TestInMemoryStore_ListByScope(t *testing.T) {
	store := NewInMemorySharedStore()
	ctx := context.Background()

	_ = store.WriteFragment(ctx, &Fragment{Key: "a", Scope: ScopeTeam})
	_ = store.WriteFragment(ctx, &Fragment{Key: "b", Scope: ScopeGlobal})
	_ = store.WriteFragment(ctx, &Fragment{Key: "c", Scope: ScopeTeam})

	frags, err := store.ListFragments(ctx, ScopeTeam)
	require.NoError(t, err)
	assert.Len(t, frags, 2)

	all, err := store.ListFragments(ctx, "")
	require.NoError(t, err)
	assert.Len(t, all, 3)
}

func TestInMemoryStore_DeleteNonExistent(t *testing.T) {
	store := NewInMemorySharedStore()
	err := store.DeleteFragment(context.Background(), "nope")
	assert.NoError(t, err)
}

func TestInMemoryStore_ContextCancelled(t *testing.T) {
	store := NewInMemorySharedStore()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := store.WriteFragment(ctx, &Fragment{Key: "k"})
	assert.Error(t, err)

	_, err = store.ReadFragment(ctx, "k")
	assert.Error(t, err)

	_, err = store.ListFragments(ctx, "")
	assert.Error(t, err)

	err = store.DeleteFragment(ctx, "k")
	assert.Error(t, err)
}

// --- Default scope tests ---

func TestDefaultScope_Applied(t *testing.T) {
	sm, _ := newTestMemory(WithDefaultScope(ScopeGlobal))
	ctx := context.Background()

	err := sm.Write(ctx, &Fragment{
		Key:      "scoped",
		Content:  "data",
		AuthorID: "agent-1",
	})
	require.NoError(t, err)

	frag, err := sm.Read(ctx, "scoped", "agent-1")
	require.NoError(t, err)
	assert.Equal(t, ScopeGlobal, frag.Scope)
}

func TestDefaultScope_NotOverridden(t *testing.T) {
	sm, _ := newTestMemory(WithDefaultScope(ScopeGlobal))
	ctx := context.Background()

	err := sm.Write(ctx, &Fragment{
		Key:      "private",
		Content:  "data",
		AuthorID: "agent-1",
		Scope:    ScopePrivate,
	})
	require.NoError(t, err)

	frag, err := sm.Read(ctx, "private", "agent-1")
	require.NoError(t, err)
	assert.Equal(t, ScopePrivate, frag.Scope)
}
