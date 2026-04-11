package plancache

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryStore_SaveAndGet(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore(10)

	tmpl := &Template{ID: "t1", AgentID: "agent1", Input: "test"}
	require.NoError(t, store.Save(ctx, tmpl))

	got, err := store.Get(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, "t1", got.ID)
	assert.False(t, got.CreatedAt.IsZero())
	assert.False(t, got.UpdatedAt.IsZero())
}

func TestInMemoryStore_SaveUpdate(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore(10)

	tmpl := &Template{ID: "t1", AgentID: "agent1", SuccessCount: 1}
	require.NoError(t, store.Save(ctx, tmpl))

	got, err := store.Get(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, 0, got.Version)

	tmpl2 := &Template{ID: "t1", AgentID: "agent1", SuccessCount: 5}
	require.NoError(t, store.Save(ctx, tmpl2))

	got2, err := store.Get(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, 1, got2.Version)
	assert.Equal(t, 5, got2.SuccessCount)
	assert.Equal(t, got.CreatedAt, got2.CreatedAt, "CreatedAt should be preserved")
}

func TestInMemoryStore_GetNotFound(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore(10)

	_, err := store.Get(ctx, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "template not found")
}

func TestInMemoryStore_List(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore(10)

	require.NoError(t, store.Save(ctx, &Template{ID: "t1", AgentID: "a1"}))
	require.NoError(t, store.Save(ctx, &Template{ID: "t2", AgentID: "a1"}))
	require.NoError(t, store.Save(ctx, &Template{ID: "t3", AgentID: "a2"}))

	list, err := store.List(ctx, "a1")
	require.NoError(t, err)
	assert.Len(t, list, 2)

	list2, err := store.List(ctx, "a2")
	require.NoError(t, err)
	assert.Len(t, list2, 1)

	list3, err := store.List(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Empty(t, list3)
}

func TestInMemoryStore_Delete(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore(10)

	require.NoError(t, store.Save(ctx, &Template{ID: "t1", AgentID: "a1"}))
	require.NoError(t, store.Delete(ctx, "t1"))

	_, err := store.Get(ctx, "t1")
	assert.Error(t, err)

	// Delete nonexistent is not an error.
	assert.NoError(t, store.Delete(ctx, "nonexistent"))
}

func TestInMemoryStore_LRUEviction(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore(3)

	require.NoError(t, store.Save(ctx, &Template{ID: "t1", AgentID: "a1"}))
	require.NoError(t, store.Save(ctx, &Template{ID: "t2", AgentID: "a1"}))
	require.NoError(t, store.Save(ctx, &Template{ID: "t3", AgentID: "a1"}))

	// Access t1 to make it recently used.
	_, err := store.Get(ctx, "t1")
	require.NoError(t, err)

	// Adding t4 should evict t2 (least recently used).
	require.NoError(t, store.Save(ctx, &Template{ID: "t4", AgentID: "a1"}))

	assert.Equal(t, 3, store.Len())
	_, err = store.Get(ctx, "t2")
	assert.Error(t, err, "t2 should have been evicted")

	_, err = store.Get(ctx, "t1")
	assert.NoError(t, err, "t1 should still exist")
}

func TestInMemoryStore_SaveNil(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore(10)

	err := store.Save(ctx, nil)
	require.Error(t, err)
}

func TestInMemoryStore_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	store := NewInMemoryStore(10)

	assert.Error(t, store.Save(ctx, &Template{ID: "t1"}))

	_, err := store.Get(ctx, "t1")
	assert.Error(t, err)

	_, err = store.List(ctx, "a1")
	assert.Error(t, err)

	assert.Error(t, store.Delete(ctx, "t1"))
}

func TestInMemoryStore_Concurrency(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryStore(100)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := "t" + string(rune('A'+n%26))
			tmpl := &Template{ID: id, AgentID: "a1"}
			_ = store.Save(ctx, tmpl)
			_, _ = store.Get(ctx, id)
			_, _ = store.List(ctx, "a1")
		}(i)
	}
	wg.Wait()

	assert.LessOrEqual(t, store.Len(), 100)
}

func TestInMemoryStore_DefaultCapacity(t *testing.T) {
	store := NewInMemoryStore(0)
	assert.Equal(t, 100, store.capacity)

	store2 := NewInMemoryStore(-1)
	assert.Equal(t, 100, store2.capacity)
}
