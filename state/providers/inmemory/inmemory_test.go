package inmemory

import (
	"context"
	"iter"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/v2/state"
)

// pullWatch is a helper that subscribes via Watch and returns a pull-based
// iterator plus a stop function. Subscribe is eager (the goroutine started
// by iter.Pull2 runs before Pull2 returns for the first next() call, but the
// Watch implementation establishes the watcher slot at call time so the test
// can safely write to the store after this helper returns and still observe
// the change).
func pullWatch(t *testing.T, s *Store, ctx context.Context, key string) (func() (state.StateChange, error, bool), func()) {
	t.Helper()
	seq := s.Watch(ctx, key)
	next, stop := iter.Pull2(seq)
	return next, stop
}

// recvOne pulls one event. It blocks until next() returns. Because the
// inmemory Watch eager-subscribes, events produced before this call are
// buffered on the store's watcher channel and returned on the next pull.
// This function is intended for use only when the test has already caused
// an event to be produced (so next() is guaranteed to return quickly).
func recvOne(t *testing.T, next func() (state.StateChange, error, bool)) state.StateChange {
	t.Helper()
	change, err, ok := next()
	if !ok {
		t.Fatal("iterator ended before yielding")
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return change
}

func TestGetSetDelete(t *testing.T) {
	s := New()
	defer s.Close()
	ctx := context.Background()

	// Get non-existent key returns nil, nil.
	val, err := s.Get(ctx, "missing")
	require.NoError(t, err)
	assert.Nil(t, val)

	// Set and Get.
	require.NoError(t, s.Set(ctx, "key1", "hello"))
	val, err = s.Get(ctx, "key1")
	require.NoError(t, err)
	assert.Equal(t, "hello", val)

	// Overwrite.
	require.NoError(t, s.Set(ctx, "key1", 42))
	val, err = s.Get(ctx, "key1")
	require.NoError(t, err)
	assert.Equal(t, 42, val)

	// Delete.
	require.NoError(t, s.Delete(ctx, "key1"))
	val, err = s.Get(ctx, "key1")
	require.NoError(t, err)
	assert.Nil(t, val)

	// Delete non-existent is a no-op.
	require.NoError(t, s.Delete(ctx, "missing"))
}

func TestWatch_SetNotification(t *testing.T) {
	s := New()
	defer s.Close()
	ctx := context.Background()

	next, stop := pullWatch(t, s, ctx, "watched")
	defer stop()

	// Set triggers a notification.
	require.NoError(t, s.Set(ctx, "watched", "v1"))

	change := recvOne(t, next)
	assert.Equal(t, "watched", change.Key)
	assert.Nil(t, change.OldValue)
	assert.Equal(t, "v1", change.Value)
	assert.Equal(t, state.OpSet, change.Op)
}

func TestWatch_DeleteNotification(t *testing.T) {
	s := New()
	defer s.Close()
	ctx := context.Background()

	require.NoError(t, s.Set(ctx, "key", "old"))

	next, stop := pullWatch(t, s, ctx, "key")
	defer stop()

	require.NoError(t, s.Delete(ctx, "key"))

	change := recvOne(t, next)
	assert.Equal(t, "key", change.Key)
	assert.Equal(t, "old", change.OldValue)
	assert.Nil(t, change.Value)
	assert.Equal(t, state.OpDelete, change.Op)
}

func TestWatch_UpdateOldValue(t *testing.T) {
	s := New()
	defer s.Close()
	ctx := context.Background()

	require.NoError(t, s.Set(ctx, "key", "v1"))

	next, stop := pullWatch(t, s, ctx, "key")
	defer stop()

	require.NoError(t, s.Set(ctx, "key", "v2"))

	change := recvOne(t, next)
	assert.Equal(t, "v1", change.OldValue)
	assert.Equal(t, "v2", change.Value)
	assert.Equal(t, state.OpSet, change.Op)
}

func TestWatch_MultipleWatchers(t *testing.T) {
	s := New()
	defer s.Close()
	ctx := context.Background()

	next1, stop1 := pullWatch(t, s, ctx, "k")
	defer stop1()
	next2, stop2 := pullWatch(t, s, ctx, "k")
	defer stop2()

	require.NoError(t, s.Set(ctx, "k", "val"))

	for _, next := range []func() (state.StateChange, error, bool){next1, next2} {
		change := recvOne(t, next)
		assert.Equal(t, "val", change.Value)
	}
}

func TestWatch_UnrelatedKeyNoNotification(t *testing.T) {
	s := New()
	defer s.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	next, stop := pullWatch(t, s, ctx, "watched")
	defer stop()

	// Set a different key — should NOT trigger the watcher.
	require.NoError(t, s.Set(ctx, "other", "v"))

	// Let the goroutine pipeline settle, then cancel ctx. After ctx is
	// cancelled the iterator must end without yielding any event for the
	// unrelated write.
	time.Sleep(50 * time.Millisecond)
	cancel()

	_, _, ok := next()
	assert.False(t, ok, "iterator should end without yielding for unrelated key")
}

func TestWatch_ContextCancellation(t *testing.T) {
	s := New()
	defer s.Close()

	ctx, cancel := context.WithCancel(context.Background())
	next, stop := pullWatch(t, s, ctx, "k")
	defer stop()

	cancel()

	_, _, ok := next()
	assert.False(t, ok, "iterator should end after context cancellation")
}

func TestClose_ClosesWatchers(t *testing.T) {
	s := New()
	ctx := context.Background()

	next, stop := pullWatch(t, s, ctx, "k")
	defer stop()

	require.NoError(t, s.Close())

	_, _, ok := next()
	assert.False(t, ok, "iterator should end after store.Close()")
}

func TestClose_OperationsAfterClose(t *testing.T) {
	s := New()
	require.NoError(t, s.Close())
	ctx := context.Background()

	_, err := s.Get(ctx, "k")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")

	err = s.Set(ctx, "k", "v")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")

	err = s.Delete(ctx, "k")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")

	// Watch on closed store yields an error via the iterator.
	seq := s.Watch(ctx, "k")
	next, stop := iter.Pull2(seq)
	defer stop()
	_, werr, ok := next()
	assert.True(t, ok, "expected an error yield")
	assert.Error(t, werr)
	assert.Contains(t, werr.Error(), "closed")
}

func TestClose_Idempotent(t *testing.T) {
	s := New()
	require.NoError(t, s.Close())
	require.NoError(t, s.Close()) // second close should not panic or error
}

func TestCancelledContext(t *testing.T) {
	s := New()
	defer s.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := s.Get(ctx, "k")
	assert.Error(t, err)

	err = s.Set(ctx, "k", "v")
	assert.Error(t, err)

	err = s.Delete(ctx, "k")
	assert.Error(t, err)

	// Cancelled ctx: Watch yields an error via the iterator.
	seq := s.Watch(ctx, "k")
	next, stop := iter.Pull2(seq)
	defer stop()
	_, werr, ok := next()
	assert.True(t, ok, "expected an error yield")
	assert.Error(t, werr)
}

func TestConcurrentAccess(t *testing.T) {
	s := New()
	defer s.Close()
	ctx := context.Background()

	const goroutines = 50
	const ops = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			key := "key"
			for i := 0; i < ops; i++ {
				_ = s.Set(ctx, key, i)
				_, _ = s.Get(ctx, key)
				if i%10 == 0 {
					_ = s.Delete(ctx, key)
				}
			}
		}(g)
	}

	wg.Wait()
}

func TestConcurrentWatchAndSet(t *testing.T) {
	s := New()
	ctx := context.Background()

	seq := s.Watch(ctx, "counter")

	const n = 10
	var wg sync.WaitGroup
	wg.Add(1)

	received := make([]state.StateChange, 0, n)
	go func() {
		defer wg.Done()
		for change, err := range seq {
			if err != nil {
				return
			}
			received = append(received, change)
		}
	}()

	for i := 0; i < n; i++ {
		require.NoError(t, s.Set(ctx, "counter", i))
	}

	// Give the reader a moment to drain the buffer before closing.
	time.Sleep(50 * time.Millisecond)

	// Close the store to signal the reader goroutine to stop.
	require.NoError(t, s.Close())
	wg.Wait()

	// We should have received all notifications (all fit in buffer of 16).
	assert.Equal(t, n, len(received), "expected all notifications to be received")
	for _, change := range received {
		assert.Equal(t, "counter", change.Key)
		assert.Equal(t, state.OpSet, change.Op)
	}
}

func TestRegistryIntegration(t *testing.T) {
	// The inmemory package registers itself via init().
	names := state.List()
	assert.Contains(t, names, "inmemory")

	s, err := state.New("inmemory", state.Config{})
	require.NoError(t, err)
	defer s.Close()

	ctx := context.Background()
	require.NoError(t, s.Set(ctx, "hello", "world"))
	val, err := s.Get(ctx, "hello")
	require.NoError(t, err)
	assert.Equal(t, "world", val)
}

func TestScopedKeyIntegration(t *testing.T) {
	s := New()
	defer s.Close()
	ctx := context.Background()

	agentKey := state.ScopedKey(state.ScopeAgent, "counter")
	sessionKey := state.ScopedKey(state.ScopeSession, "counter")

	require.NoError(t, s.Set(ctx, agentKey, 1))
	require.NoError(t, s.Set(ctx, sessionKey, 2))

	v1, err := s.Get(ctx, agentKey)
	require.NoError(t, err)
	assert.Equal(t, 1, v1)

	v2, err := s.Get(ctx, sessionKey)
	require.NoError(t, err)
	assert.Equal(t, 2, v2)
}

func TestVariousValueTypes(t *testing.T) {
	s := New()
	defer s.Close()
	ctx := context.Background()

	tests := []struct {
		name  string
		key   string
		value any
	}{
		{"string", "s", "hello"},
		{"int", "i", 42},
		{"float", "f", 3.14},
		{"bool", "b", true},
		{"nil", "n", nil},
		{"slice", "sl", []int{1, 2, 3}},
		{"map", "m", map[string]int{"a": 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, s.Set(ctx, tt.key, tt.value))
			got, err := s.Get(ctx, tt.key)
			require.NoError(t, err)
			assert.Equal(t, tt.value, got)
		})
	}
}

// --- Versioned Store Tests ---

func TestGetVersioned(t *testing.T) {
	s := New()
	defer s.Close()
	ctx := context.Background()

	// Non-existent key returns zero version.
	val, ver, err := s.GetVersioned(ctx, "missing")
	require.NoError(t, err)
	assert.Nil(t, val)
	assert.Equal(t, uint64(0), ver)

	// Set increments version.
	require.NoError(t, s.Set(ctx, "k", "v1"))
	val, ver, err = s.GetVersioned(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, "v1", val)
	assert.Equal(t, uint64(1), ver)

	// Second set increments again.
	require.NoError(t, s.Set(ctx, "k", "v2"))
	val, ver, err = s.GetVersioned(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, "v2", val)
	assert.Equal(t, uint64(2), ver)
}

func TestGetVersioned_ContextCancelled(t *testing.T) {
	s := New()
	defer s.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := s.GetVersioned(ctx, "k")
	assert.Error(t, err)
}

func TestGetVersioned_StoreClosed(t *testing.T) {
	s := New()
	require.NoError(t, s.Close())

	_, _, err := s.GetVersioned(context.Background(), "k")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

func TestCompareAndSwap_Success(t *testing.T) {
	s := New()
	defer s.Close()
	ctx := context.Background()

	// CAS on new key with expectedVersion=0.
	newVer, err := s.CompareAndSwap(ctx, "k", 0, "first")
	require.NoError(t, err)
	assert.Equal(t, uint64(1), newVer)

	val, ver, err := s.GetVersioned(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, "first", val)
	assert.Equal(t, uint64(1), ver)

	// CAS with correct version.
	newVer, err = s.CompareAndSwap(ctx, "k", 1, "second")
	require.NoError(t, err)
	assert.Equal(t, uint64(2), newVer)

	val, _, err = s.GetVersioned(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, "second", val)
}

func TestCompareAndSwap_VersionMismatch(t *testing.T) {
	s := New()
	defer s.Close()
	ctx := context.Background()

	require.NoError(t, s.Set(ctx, "k", "v1"))

	// CAS with wrong version.
	_, err := s.CompareAndSwap(ctx, "k", 0, "nope")
	assert.ErrorIs(t, err, state.ErrVersionMismatch)

	// Value should be unchanged.
	val, _, err := s.GetVersioned(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, "v1", val)
}

func TestCompareAndSwap_ContextCancelled(t *testing.T) {
	s := New()
	defer s.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := s.CompareAndSwap(ctx, "k", 0, "v")
	assert.Error(t, err)
}

func TestCompareAndSwap_StoreClosed(t *testing.T) {
	s := New()
	require.NoError(t, s.Close())

	_, err := s.CompareAndSwap(context.Background(), "k", 0, "v")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

func TestVersionInWatchNotification(t *testing.T) {
	s := New()
	defer s.Close()
	ctx := context.Background()

	next, stop := pullWatch(t, s, ctx, "k")
	defer stop()

	require.NoError(t, s.Set(ctx, "k", "v1"))
	change := recvOne(t, next)
	assert.Equal(t, uint64(1), change.Version)

	require.NoError(t, s.Set(ctx, "k", "v2"))
	change = recvOne(t, next)
	assert.Equal(t, uint64(2), change.Version)
}

func TestCompareAndSwap_ConcurrentContention(t *testing.T) {
	s := New()
	defer s.Close()
	ctx := context.Background()

	// Initialize key.
	require.NoError(t, s.Set(ctx, "counter", 0))

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	successes := make(chan bool, goroutines*100)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				val, ver, err := s.GetVersioned(ctx, "counter")
				if err != nil {
					continue
				}
				n, _ := val.(int)
				_, err = s.CompareAndSwap(ctx, "counter", ver, n+1)
				if err == nil {
					successes <- true
				}
			}
		}()
	}

	wg.Wait()
	close(successes)

	successCount := 0
	for range successes {
		successCount++
	}

	// Final value should equal number of successful CAS operations.
	val, _, err := s.GetVersioned(ctx, "counter")
	require.NoError(t, err)
	assert.Equal(t, successCount, val.(int))
}

func TestDeleteIncrementsVersion(t *testing.T) {
	s := New()
	defer s.Close()
	ctx := context.Background()

	next, stop := pullWatch(t, s, ctx, "k")
	defer stop()

	require.NoError(t, s.Set(ctx, "k", "v1"))
	_ = recvOne(t, next) // consume set notification

	require.NoError(t, s.Delete(ctx, "k"))

	change := recvOne(t, next)
	assert.Equal(t, state.OpDelete, change.Op)
	assert.Equal(t, uint64(2), change.Version) // version 1 from set + 1
}

// Compile-time check for VersionedStore.
var _ state.VersionedStore = (*Store)(nil)
