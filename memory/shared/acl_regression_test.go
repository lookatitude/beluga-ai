package shared

import (
	"context"
	"errors"
	"iter"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireCode(t *testing.T, err error, code core.ErrorCode) {
	t.Helper()
	require.Error(t, err)
	var ce *core.Error
	require.ErrorAs(t, err, &ce)
	assert.Equal(t, code, ce.Code)
}

// FI-06(a): a content-only update must not reset the ACL.
func TestWrite_ContentOnlyUpdatePreservesACL(t *testing.T) {
	sm, _ := newTestMemory()
	ctx := context.Background()
	require.NoError(t, sm.Write(ctx, &Fragment{
		Key: "k", Content: "secret", AuthorID: "a",
		Readers: []string{"a"}, Writers: []string{"a"},
	}))
	require.NoError(t, sm.Write(ctx, &Fragment{Key: "k", Content: "secret2", AuthorID: "a"}))

	_, err := sm.Read(ctx, "k", "mallory")
	requireCode(t, err, core.ErrAuth)
	err = sm.Write(ctx, &Fragment{Key: "k", Content: "pwned", AuthorID: "mallory"})
	requireCode(t, err, core.ErrAuth)

	frag, err := sm.Read(ctx, "k", "a")
	require.NoError(t, err)
	assert.Equal(t, "secret2", frag.Content)
	assert.Equal(t, []string{"a"}, frag.Readers)
	assert.Equal(t, []string{"a"}, frag.Writers)
}

func TestWrite_ACLChangeViaWriteRejected(t *testing.T) {
	sm, _ := newTestMemory()
	ctx := context.Background()
	require.NoError(t, sm.Write(ctx, &Fragment{Key: "k", Content: "x", AuthorID: "a", Readers: []string{"a"}}))

	tests := []struct {
		name string
		frag *Fragment
	}{
		{"widen readers", &Fragment{Key: "k", AuthorID: "a", Readers: []string{"a", "b"}}},
		{"change writers", &Fragment{Key: "k", AuthorID: "a", Writers: []string{"b"}}},
		{"open read", &Fragment{Key: "k", AuthorID: "a", UnrestrictedRead: true}},
		{"open write", &Fragment{Key: "k", AuthorID: "a", UnrestrictedWrite: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireCode(t, sm.Write(ctx, tt.frag), core.ErrInvalidInput)
		})
	}
	// Resending the identical ACL is allowed.
	require.NoError(t, sm.Write(ctx, &Fragment{Key: "k", Content: "y", AuthorID: "a", Readers: []string{"a"}, Writers: []string{"a"}}))
}

func TestWrite_NewFragmentDefaultsToAuthorOnly(t *testing.T) {
	sm, _ := newTestMemory()
	ctx := context.Background()
	require.NoError(t, sm.Write(ctx, &Fragment{Key: "k", Content: "x", AuthorID: "a"}))

	_, err := sm.Read(ctx, "k", "b")
	requireCode(t, err, core.ErrAuth)
	_, err = sm.Read(ctx, "k", "a")
	require.NoError(t, err)
	requireCode(t, sm.Write(ctx, &Fragment{Key: "k", Content: "y", AuthorID: "b"}), core.ErrAuth)
}

func TestWrite_UnrestrictedFlags(t *testing.T) {
	sm, _ := newTestMemory()
	ctx := context.Background()
	require.NoError(t, sm.Write(ctx, &Fragment{
		Key: "k", Content: "x", AuthorID: "a", UnrestrictedRead: true, UnrestrictedWrite: true,
	}))
	_, err := sm.Read(ctx, "k", "anyone")
	require.NoError(t, err)
	require.NoError(t, sm.Write(ctx, &Fragment{Key: "k", Content: "y", AuthorID: "anyone"}))
	list, err := sm.List(ctx, "", "anyone")
	require.NoError(t, err)
	assert.Len(t, list, 1)

	requireCode(t, sm.Revoke(ctx, "k", "a", "anyone", PermRead), core.ErrInvalidInput)
	requireCode(t, sm.Revoke(ctx, "k", "a", "anyone", PermWrite), core.ErrInvalidInput)
}

// FI-06(b): revoking the last reader/writer must deny everyone.
func TestRevoke_LastEntryDeniesAll(t *testing.T) {
	sm, _ := newTestMemory()
	ctx := context.Background()
	require.NoError(t, sm.Write(ctx, &Fragment{
		Key: "k", Content: "secret", AuthorID: "a",
		Readers: []string{"a"}, Writers: []string{"a"},
	}))
	require.NoError(t, sm.Revoke(ctx, "k", "a", "a", PermRead))

	_, err := sm.Read(ctx, "k", "mallory")
	requireCode(t, err, core.ErrAuth)
	_, err = sm.Read(ctx, "k", "a")
	requireCode(t, err, core.ErrAuth)

	require.NoError(t, sm.Revoke(ctx, "k", "a", "a", PermWrite))
	requireCode(t, sm.Write(ctx, &Fragment{Key: "k", Content: "x", AuthorID: "mallory"}), core.ErrAuth)
	requireCode(t, sm.Delete(ctx, "k", "mallory"), core.ErrAuth)
}

// A store whose reads fail with a non-not-found error.
type failingReadStore struct {
	*InMemorySharedStore
	readErr error
}

func (s *failingReadStore) ReadFragment(context.Context, string) (*Fragment, error) {
	return nil, s.readErr
}

// FI-06(c): a read failure must not skip the writer-ACL check.
func TestWrite_ReadFailureFailsClosed(t *testing.T) {
	inner := NewInMemorySharedStore()
	backendErr := core.NewError("backend", core.ErrProviderDown, "down", nil)
	store := &failingReadStore{InMemorySharedStore: inner, readErr: backendErr}
	sm := NewSharedMemory(store)

	err := sm.Write(context.Background(), &Fragment{Key: "k", Content: "x", AuthorID: "mallory"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, backendErr))
	_, rerr := inner.ReadFragment(context.Background(), "k")
	requireCode(t, rerr, core.ErrNotFound)
}

// FI-06(d): the conflict policy is fixed at creation.
func TestWrite_ConflictPolicyNotChosenByIncoming(t *testing.T) {
	sm, _ := newTestMemory()
	ctx := context.Background()
	require.NoError(t, sm.Write(ctx, &Fragment{
		Key: "k", Content: "v1", AuthorID: "a", ConflictPolicy: RejectOnConflict,
	}))
	// Stale version with a downgraded policy must still be rejected.
	err := sm.Write(ctx, &Fragment{
		Key: "k", Content: "v2", AuthorID: "a", ConflictPolicy: LastWriteWins, Version: 0,
	})
	requireCode(t, err, core.ErrInvalidInput)

	frag, err := sm.Read(ctx, "k", "a")
	require.NoError(t, err)
	assert.Equal(t, "v1", frag.Content)
	assert.Equal(t, RejectOnConflict, frag.ConflictPolicy)
}

func TestGrant_AppendOnlyDoesNotDuplicateContent(t *testing.T) {
	sm, _ := newTestMemory()
	ctx := context.Background()
	require.NoError(t, sm.Write(ctx, &Fragment{
		Key: "k", Content: "abc", AuthorID: "a", ConflictPolicy: AppendOnly,
	}))
	require.NoError(t, sm.Grant(ctx, "k", "a", "b", PermRead))
	frag, err := sm.Read(ctx, "k", "b")
	require.NoError(t, err)
	assert.Equal(t, "abc", frag.Content)
	assert.Equal(t, int64(1), frag.Version)
}

// A store without the ACLUpdater extension.
type plainStore struct{ s *InMemorySharedStore }

func (p plainStore) WriteFragment(ctx context.Context, f *Fragment) error {
	return p.s.WriteFragment(ctx, f)
}
func (p plainStore) ReadFragment(ctx context.Context, k string) (*Fragment, error) {
	return p.s.ReadFragment(ctx, k)
}
func (p plainStore) ListFragments(ctx context.Context, sc Scope) ([]*Fragment, error) {
	return p.s.ListFragments(ctx, sc)
}
func (p plainStore) DeleteFragment(ctx context.Context, k string) error {
	return p.s.DeleteFragment(ctx, k)
}
func (p plainStore) UpdateProvenance(ctx context.Context, k string, pr *Provenance) error {
	return p.s.UpdateProvenance(ctx, k, pr)
}

func TestGrant_FallbackWithoutACLUpdater(t *testing.T) {
	sm := NewSharedMemory(plainStore{s: NewInMemorySharedStore()})
	ctx := context.Background()
	require.NoError(t, sm.Write(ctx, &Fragment{Key: "k", Content: "x", AuthorID: "a"}))
	require.NoError(t, sm.Grant(ctx, "k", "a", "b", PermRead))
	_, err := sm.Read(ctx, "k", "b")
	require.NoError(t, err)
}

func TestACLChange_InputValidation(t *testing.T) {
	sm, _ := newTestMemory()
	ctx := context.Background()
	require.NoError(t, sm.Write(ctx, &Fragment{Key: "k", Content: "x", AuthorID: "a"}))
	requireCode(t, sm.Grant(ctx, "k", "", "b", PermRead), core.ErrInvalidInput)
	requireCode(t, sm.Revoke(ctx, "k", "", "b", PermRead), core.ErrInvalidInput)
	requireCode(t, sm.Delete(ctx, "k", ""), core.ErrInvalidInput)
	requireCode(t, sm.Revoke(ctx, "k", "a", "b", Permission("x")), core.ErrInvalidInput)
	requireCode(t, sm.Revoke(ctx, "k", "b", "a", PermRead), core.ErrAuth)
	requireCode(t, sm.Revoke(ctx, "missing", "a", "b", PermRead), core.ErrNotFound)
	require.NoError(t, sm.Delete(ctx, "k", "a"))
}

// Codex finding: Watch must enforce read authorization and require identity.
func TestWatch_DeprecatedFormRequiresIdentity(t *testing.T) {
	sm, _ := newTestMemory()
	for _, err := range sm.Watch(context.Background(), "k") {
		requireCode(t, err, core.ErrInvalidInput)
	}
}

func TestWatchAs_DeniedForNonReader(t *testing.T) {
	sm, _ := newTestMemory()
	ctx := context.Background()
	require.NoError(t, sm.Write(ctx, &Fragment{Key: "k", Content: "secret", AuthorID: "a"}))

	var got []error
	for _, err := range sm.WatchAs(ctx, "k", "mallory") {
		got = append(got, err)
	}
	require.Len(t, got, 1)
	requireCode(t, got[0], core.ErrAuth)
}

func TestWatchAs_ReadErrorPropagates(t *testing.T) {
	backendErr := core.NewError("backend", core.ErrProviderDown, "down", nil)
	sm := NewSharedMemory(&failingReadStore{InMemorySharedStore: NewInMemorySharedStore(), readErr: backendErr})
	for _, err := range sm.WatchAs(context.Background(), "k", "a") {
		assert.True(t, errors.Is(err, backendErr))
	}
}

// A watcher subscribed before creation only sees changes it may read.
func TestWatchAs_FiltersEventsByACL(t *testing.T) {
	sm, _ := newTestMemory()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	nextM, stopM := iter.Pull2(sm.WatchAs(ctx, "k", "mallory"))
	defer stopM()
	nextB, stopB := iter.Pull2(sm.WatchAs(ctx, "k", "b"))
	defer stopB()

	require.NoError(t, sm.Write(context.Background(), &Fragment{
		Key: "k", Content: "secret", AuthorID: "a", Readers: []string{"a", "b"},
	}))
	require.NoError(t, sm.Revoke(context.Background(), "k", "a", "b", PermRead))
	require.NoError(t, sm.Write(context.Background(), &Fragment{Key: "k", Content: "secret2", AuthorID: "a"}))

	change, err, ok := nextB()
	require.True(t, ok)
	require.NoError(t, err)
	assert.Equal(t, OpWrite, change.Op)
	assert.Equal(t, "secret", change.Fragment.Content)

	// b sees neither its own revoke nor the later write; mallory sees nothing.
	sm.watchMu.RLock()
	var bQueued, mQueued int
	for _, w := range sm.watchers["k"] {
		switch w.agentID {
		case "b":
			bQueued = len(w.ch)
		case "mallory":
			mQueued = len(w.ch)
		}
	}
	sm.watchMu.RUnlock()
	assert.Equal(t, 0, bQueued)
	assert.Equal(t, 0, mQueued)

	cancel()
	_, _, ok = nextM()
	assert.False(t, ok)
}

func TestWatchAs_DeleteAuthorizedAgainstPreDeleteACL(t *testing.T) {
	sm, _ := newTestMemory()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, sm.Write(context.Background(), &Fragment{Key: "k", Content: "x", AuthorID: "a"}))

	next, stop := iter.Pull2(sm.WatchAs(ctx, "k", "a"))
	defer stop()
	require.NoError(t, sm.Delete(context.Background(), "k", "a"))
	change, err, ok := next()
	require.True(t, ok)
	require.NoError(t, err)
	assert.Equal(t, OpDelete, change.Op)
	assert.Nil(t, change.Fragment)
}

// Breaking out of the loop must stop the background goroutine and remove
// the watcher even while ctx stays alive.
func TestWatchAs_BreakUnsubscribes(t *testing.T) {
	sm, _ := newTestMemory()
	ctx := context.Background()
	require.NoError(t, sm.Write(ctx, &Fragment{Key: "k", Content: "x", AuthorID: "a"}))

	seq := sm.WatchAs(ctx, "k", "a")
	go func() {
		time.Sleep(10 * time.Millisecond)
		_ = sm.Write(ctx, &Fragment{Key: "k", Content: "y", AuthorID: "a"})
	}()
	for range seq {
		break
	}
	sm.watchMu.RLock()
	defer sm.watchMu.RUnlock()
	assert.Empty(t, sm.watchers["k"])
}

func TestInMemoryStore_UpdateACL(t *testing.T) {
	s := NewInMemorySharedStore()
	ctx := context.Background()
	requireCode(t, s.UpdateACL(ctx, &Fragment{Key: "nope"}), core.ErrNotFound)
	cctx, cancel := context.WithCancel(ctx)
	cancel()
	require.Error(t, s.UpdateACL(cctx, &Fragment{Key: "nope"}))
}
