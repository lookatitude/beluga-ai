package state

import (
	"context"
	"iter"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// watchMockStore is a minimal Store for testing WatchSeq.
type watchMockStore struct {
	mu      sync.Mutex
	watchCh chan StateChange
	closed  bool
}

func (m *watchMockStore) Get(context.Context, string) (any, error) { return nil, nil }
func (m *watchMockStore) Set(context.Context, string, any) error   { return nil }
func (m *watchMockStore) Delete(context.Context, string) error     { return nil }
func (m *watchMockStore) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.closed = true
		close(m.watchCh)
	}
	return nil
}
func (m *watchMockStore) Watch(ctx context.Context, _ string) iter.Seq2[StateChange, error] {
	return func(yield func(StateChange, error) bool) {
		for {
			select {
			case <-ctx.Done():
				return
			case change, ok := <-m.watchCh:
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

func TestWatchSeq_ReceivesChanges(t *testing.T) {
	ch := make(chan StateChange, 4)
	ms := &watchMockStore{watchCh: ch}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch <- StateChange{Key: "k", Value: "v1", Op: OpSet, Version: 1}
	ch <- StateChange{Key: "k", Value: "v2", Op: OpSet, Version: 2}

	var received []StateChange
	count := 0
	for change, err := range WatchSeq(ctx, ms, "k") {
		assert.NoError(t, err)
		received = append(received, change)
		count++
		if count >= 2 {
			break
		}
	}

	assert.Len(t, received, 2)
	assert.Equal(t, "v1", received[0].Value)
	assert.Equal(t, "v2", received[1].Value)
}

func TestWatchSeq_ContextCancellation(t *testing.T) {
	ch := make(chan StateChange, 4)
	ms := &watchMockStore{watchCh: ch}
	defer ms.Close()

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	// Ctx cancellation ends iteration silently (no terminal error yield),
	// matching the memory/shared Watch semantics.
	events := 0
	for _, err := range WatchSeq(ctx, ms, "k") {
		assert.NoError(t, err)
		events++
	}
	assert.Equal(t, 0, events)
	assert.ErrorIs(t, ctx.Err(), context.Canceled)
}

func TestWatchSeq_ChannelClosed(t *testing.T) {
	ch := make(chan StateChange, 4)
	ms := &watchMockStore{watchCh: ch}

	ctx := context.Background()

	ch <- StateChange{Key: "k", Value: "v1", Op: OpSet, Version: 1}
	close(ch)

	var received []StateChange
	for change, err := range WatchSeq(ctx, ms, "k") {
		assert.NoError(t, err)
		received = append(received, change)
	}

	assert.Len(t, received, 1)
}

func TestWatchSeq_ConsumerBreaks(t *testing.T) {
	ch := make(chan StateChange, 4)
	ms := &watchMockStore{watchCh: ch}

	ctx := context.Background()

	for i := 0; i < 4; i++ {
		ch <- StateChange{Key: "k", Value: i, Op: OpSet, Version: uint64(i + 1)}
	}

	count := 0
	for _, err := range WatchSeq(ctx, ms, "k") {
		assert.NoError(t, err)
		count++
		if count == 1 {
			break
		}
	}
	assert.Equal(t, 1, count)
}
