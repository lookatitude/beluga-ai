package blackboard

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/v2/state"
	"github.com/lookatitude/beluga-ai/v2/state/providers/inmemory"
)

func newTestBlackboard(opts ...Option) (*Blackboard, func()) {
	store := inmemory.New()
	bb := New(store, opts...)
	return bb, func() { bb.Close() }
}

func TestBlackboard_SetAndGet(t *testing.T) {
	bb, cleanup := newTestBlackboard()
	defer cleanup()
	ctx := context.Background()

	require.NoError(t, bb.Set(ctx, "agent-1", "key", "value"))

	val, err := bb.Get(ctx, "key")
	require.NoError(t, err)
	assert.Equal(t, "value", val)
}

func TestBlackboard_GetVersioned(t *testing.T) {
	bb, cleanup := newTestBlackboard()
	defer cleanup()
	ctx := context.Background()

	require.NoError(t, bb.Set(ctx, "agent-1", "key", "v1"))
	val, ver, err := bb.GetVersioned(ctx, "key")
	require.NoError(t, err)
	assert.Equal(t, "v1", val)
	assert.Equal(t, uint64(1), ver)

	require.NoError(t, bb.Set(ctx, "agent-1", "key", "v2"))
	val, ver, err = bb.GetVersioned(ctx, "key")
	require.NoError(t, err)
	assert.Equal(t, "v2", val)
	assert.Equal(t, uint64(2), ver)
}

func TestBlackboard_Delete(t *testing.T) {
	bb, cleanup := newTestBlackboard()
	defer cleanup()
	ctx := context.Background()

	require.NoError(t, bb.Set(ctx, "agent-1", "key", "value"))
	require.NoError(t, bb.Delete(ctx, "agent-1", "key"))

	val, err := bb.Get(ctx, "key")
	require.NoError(t, err)
	assert.Nil(t, val)
}

func TestBlackboard_WatchSingleKey(t *testing.T) {
	bb, cleanup := newTestBlackboard()
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start watching in a goroutine.
	var received []state.StateChange
	var watchErr error
	done := make(chan struct{})

	go func() {
		defer close(done)
		for change, err := range bb.Watch(ctx, "key") {
			if err != nil {
				watchErr = err
				return
			}
			received = append(received, change)
			if len(received) >= 2 {
				return
			}
		}
	}()

	// Give the watch time to set up.
	time.Sleep(20 * time.Millisecond)

	require.NoError(t, bb.Set(context.Background(), "agent-1", "key", "v1"))
	require.NoError(t, bb.Set(context.Background(), "agent-1", "key", "v2"))

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for watch events")
	}

	assert.NoError(t, watchErr)
	assert.Len(t, received, 2)
	assert.Equal(t, "v1", received[0].Value)
	assert.Equal(t, "v2", received[1].Value)
}

func TestBlackboard_WatchMultipleKeys(t *testing.T) {
	bb, cleanup := newTestBlackboard()
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var received []state.StateChange
	var mu sync.Mutex
	done := make(chan struct{})

	go func() {
		defer close(done)
		for change, err := range bb.Watch(ctx, "k1", "k2") {
			if err != nil {
				return
			}
			mu.Lock()
			received = append(received, change)
			count := len(received)
			mu.Unlock()
			if count >= 2 {
				return
			}
		}
	}()

	time.Sleep(20 * time.Millisecond)

	require.NoError(t, bb.Set(context.Background(), "agent-1", "k1", "a"))
	require.NoError(t, bb.Set(context.Background(), "agent-2", "k2", "b"))

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for multi-key watch")
	}

	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, received, 2)

	keys := map[string]bool{}
	for _, c := range received {
		keys[c.Key] = true
	}
	assert.True(t, keys["k1"])
	assert.True(t, keys["k2"])
}

func TestBlackboard_WatchNoKeys(t *testing.T) {
	bb, cleanup := newTestBlackboard()
	defer cleanup()

	ctx := context.Background()
	count := 0
	for range bb.Watch(ctx) {
		count++
	}
	assert.Equal(t, 0, count)
}

func TestBlackboard_OwnershipEnforcement(t *testing.T) {
	bb, cleanup := newTestBlackboard(WithEnforceOwnership())
	defer cleanup()
	ctx := context.Background()

	// Claim key.
	require.NoError(t, bb.ClaimOwnership(ctx, "agent-a", "protected"))

	// Owner can write.
	require.NoError(t, bb.Set(ctx, "agent-a", "protected", "data"))

	// Non-owner cannot write.
	err := bb.Set(ctx, "agent-b", "protected", "attack")
	assert.True(t, errors.Is(err, state.ErrOwnershipDenied))

	// Value unchanged.
	val, err := bb.Get(ctx, "protected")
	require.NoError(t, err)
	assert.Equal(t, "data", val)

	// Non-owner cannot delete.
	err = bb.Delete(ctx, "agent-b", "protected")
	assert.True(t, errors.Is(err, state.ErrOwnershipDenied))

	// Release and then another agent can claim.
	require.NoError(t, bb.ReleaseOwnership(ctx, "agent-a", "protected"))
	require.NoError(t, bb.ClaimOwnership(ctx, "agent-b", "protected"))
	require.NoError(t, bb.Set(ctx, "agent-b", "protected", "new-data"))
}

func TestBlackboard_OwnershipNotEnforced(t *testing.T) {
	bb, cleanup := newTestBlackboard() // No WithEnforceOwnership
	defer cleanup()
	ctx := context.Background()

	// Claim key.
	require.NoError(t, bb.ClaimOwnership(ctx, "agent-a", "key"))

	// Without enforcement, any agent can write to a claimed key.
	require.NoError(t, bb.Set(ctx, "agent-b", "key", "data"))
}

func TestBlackboard_Reducers(t *testing.T) {
	bb, cleanup := newTestBlackboard(
		WithReducers(
			state.WithReducer("counter", func(old, new any) any {
				o, _ := old.(int)
				n, _ := new.(int)
				return o + n
			}),
		),
	)
	defer cleanup()
	ctx := context.Background()

	require.NoError(t, bb.Set(ctx, "agent-1", "counter", 5))
	require.NoError(t, bb.Set(ctx, "agent-2", "counter", 3))

	val, err := bb.Get(ctx, "counter")
	require.NoError(t, err)
	assert.Equal(t, 8, val)
}

func TestBlackboard_CompareAndSwap(t *testing.T) {
	bb, cleanup := newTestBlackboard()
	defer cleanup()
	ctx := context.Background()

	// CAS on new key.
	newVer, err := bb.CompareAndSwap(ctx, "agent-1", "key", 0, "first")
	require.NoError(t, err)
	assert.Equal(t, uint64(1), newVer)

	// CAS with correct version.
	newVer, err = bb.CompareAndSwap(ctx, "agent-1", "key", 1, "second")
	require.NoError(t, err)
	assert.Equal(t, uint64(2), newVer)

	// CAS with wrong version.
	_, err = bb.CompareAndSwap(ctx, "agent-1", "key", 1, "conflict")
	assert.ErrorIs(t, err, state.ErrVersionMismatch)
}

func TestBlackboard_CompareAndSwap_OwnershipEnforced(t *testing.T) {
	bb, cleanup := newTestBlackboard(WithEnforceOwnership())
	defer cleanup()
	ctx := context.Background()

	require.NoError(t, bb.ClaimOwnership(ctx, "agent-a", "key"))

	// Owner CAS works.
	_, err := bb.CompareAndSwap(ctx, "agent-a", "key", 0, "v1")
	require.NoError(t, err)

	// Non-owner CAS denied.
	_, err = bb.CompareAndSwap(ctx, "agent-b", "key", 1, "v2")
	assert.True(t, errors.Is(err, state.ErrOwnershipDenied))
}

func TestBlackboard_ClosedOperations(t *testing.T) {
	bb, _ := newTestBlackboard()
	require.NoError(t, bb.Close())

	ctx := context.Background()

	err := bb.Set(ctx, "a", "k", "v")
	assert.ErrorIs(t, err, state.ErrStoreClosed)

	_, err = bb.Get(ctx, "k")
	assert.ErrorIs(t, err, state.ErrStoreClosed)

	_, _, err = bb.GetVersioned(ctx, "k")
	assert.ErrorIs(t, err, state.ErrStoreClosed)

	err = bb.Delete(ctx, "a", "k")
	assert.ErrorIs(t, err, state.ErrStoreClosed)

	_, err = bb.CompareAndSwap(ctx, "a", "k", 0, "v")
	assert.ErrorIs(t, err, state.ErrStoreClosed)
}

func TestBlackboard_CloseIdempotent(t *testing.T) {
	bb, _ := newTestBlackboard()
	require.NoError(t, bb.Close())
	require.NoError(t, bb.Close())
}

func TestBlackboard_ConcurrentMultiAgent(t *testing.T) {
	bb, cleanup := newTestBlackboard(
		WithReducers(
			state.WithReducer("shared", func(old, new any) any {
				o, _ := old.(int)
				n, _ := new.(int)
				return o + n
			}),
		),
	)
	defer cleanup()
	ctx := context.Background()

	const agents = 10
	const opsPerAgent = 50

	var wg sync.WaitGroup
	wg.Add(agents)

	for i := 0; i < agents; i++ {
		go func(agentID string) {
			defer wg.Done()
			for j := 0; j < opsPerAgent; j++ {
				_ = bb.Set(ctx, agentID, "shared", 1)
			}
		}(string(rune('A' + i)))
	}

	wg.Wait()

	val, err := bb.Get(ctx, "shared")
	require.NoError(t, err)
	assert.Equal(t, agents*opsPerAgent, val.(int))
}

func TestBlackboard_ClaimOwnership_ContextCancelled(t *testing.T) {
	bb, cleanup := newTestBlackboard()
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := bb.ClaimOwnership(ctx, "agent", "key")
	assert.ErrorIs(t, err, context.Canceled)

	err = bb.ReleaseOwnership(ctx, "agent", "key")
	assert.ErrorIs(t, err, context.Canceled)
}
