package inmemory

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/state"
)

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

	ch, err := s.Watch(ctx, "watched")
	require.NoError(t, err)

	// Set triggers a notification.
	require.NoError(t, s.Set(ctx, "watched", "v1"))

	select {
	case change := <-ch:
		assert.Equal(t, "watched", change.Key)
		assert.Nil(t, change.OldValue)
		assert.Equal(t, "v1", change.Value)
		assert.Equal(t, state.OpSet, change.Op)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for watch notification")
	}
}

func TestWatch_DeleteNotification(t *testing.T) {
	s := New()
	defer s.Close()
	ctx := context.Background()

	require.NoError(t, s.Set(ctx, "key", "old"))

	ch, err := s.Watch(ctx, "key")
	require.NoError(t, err)

	require.NoError(t, s.Delete(ctx, "key"))

	select {
	case change := <-ch:
		assert.Equal(t, "key", change.Key)
		assert.Equal(t, "old", change.OldValue)
		assert.Nil(t, change.Value)
		assert.Equal(t, state.OpDelete, change.Op)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for delete notification")
	}
}

func TestWatch_UpdateOldValue(t *testing.T) {
	s := New()
	defer s.Close()
	ctx := context.Background()

	require.NoError(t, s.Set(ctx, "key", "v1"))

	ch, err := s.Watch(ctx, "key")
	require.NoError(t, err)

	require.NoError(t, s.Set(ctx, "key", "v2"))

	select {
	case change := <-ch:
		assert.Equal(t, "v1", change.OldValue)
		assert.Equal(t, "v2", change.Value)
		assert.Equal(t, state.OpSet, change.Op)
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestWatch_MultipleWatchers(t *testing.T) {
	s := New()
	defer s.Close()
	ctx := context.Background()

	ch1, err := s.Watch(ctx, "k")
	require.NoError(t, err)
	ch2, err := s.Watch(ctx, "k")
	require.NoError(t, err)

	require.NoError(t, s.Set(ctx, "k", "val"))

	for _, ch := range []<-chan state.StateChange{ch1, ch2} {
		select {
		case change := <-ch:
			assert.Equal(t, "val", change.Value)
		case <-time.After(time.Second):
			t.Fatal("timeout")
		}
	}
}

func TestWatch_UnrelatedKeyNoNotification(t *testing.T) {
	s := New()
	defer s.Close()
	ctx := context.Background()

	ch, err := s.Watch(ctx, "watched")
	require.NoError(t, err)

	// Set a different key — should NOT trigger the watcher.
	require.NoError(t, s.Set(ctx, "other", "v"))

	select {
	case <-ch:
		t.Fatal("should not receive notification for unrelated key")
	case <-time.After(50 * time.Millisecond):
		// Expected.
	}
}

func TestWatch_ContextCancellation(t *testing.T) {
	s := New()
	defer s.Close()

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := s.Watch(ctx, "k")
	require.NoError(t, err)

	cancel()

	// Give time for goroutine to process cancellation.
	time.Sleep(50 * time.Millisecond)

	// Channel should be closed.
	select {
	case _, ok := <-ch:
		assert.False(t, ok, "channel should be closed after context cancellation")
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for channel close")
	}
}

func TestClose_ClosesWatchers(t *testing.T) {
	s := New()
	ctx := context.Background()

	ch, err := s.Watch(ctx, "k")
	require.NoError(t, err)

	require.NoError(t, s.Close())

	// Channel should be closed.
	_, ok := <-ch
	assert.False(t, ok, "channel should be closed after store.Close()")
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

	_, err = s.Watch(ctx, "k")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
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

	_, err = s.Watch(ctx, "k")
	assert.Error(t, err)
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

	ch, err := s.Watch(ctx, "counter")
	require.NoError(t, err)

	const n = 10
	var wg sync.WaitGroup
	wg.Add(1)

	received := make([]state.StateChange, 0, n)
	go func() {
		defer wg.Done()
		for change := range ch {
			received = append(received, change)
		}
	}()

	for i := 0; i < n; i++ {
		require.NoError(t, s.Set(ctx, "counter", i))
	}

	// Close the store to signal the reader goroutine to stop.
	require.NoError(t, s.Close())
	wg.Wait()

	// We should have received at least some notifications (all fit in buffer of 16).
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
