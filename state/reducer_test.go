package state

import (
	"context"
	"iter"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockVersionedStore is a minimal VersionedStore for testing reducers.
type mockVersionedStore struct {
	mu       sync.Mutex
	data     map[string]any
	versions map[string]uint64
}

func newMockVersionedStore() *mockVersionedStore {
	return &mockVersionedStore{
		data:     make(map[string]any),
		versions: make(map[string]uint64),
	}
}

func (m *mockVersionedStore) Get(ctx context.Context, key string) (any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.data[key], nil
}

func (m *mockVersionedStore) Set(ctx context.Context, key string, value any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.versions[key]++
	m.data[key] = value
	return nil
}

func (m *mockVersionedStore) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func (m *mockVersionedStore) Watch(_ context.Context, _ string) iter.Seq2[StateChange, error] {
	return func(yield func(StateChange, error) bool) {}
}

func (m *mockVersionedStore) Close() error { return nil }

func (m *mockVersionedStore) GetVersioned(ctx context.Context, key string) (any, uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.data[key], m.versions[key], nil
}

func (m *mockVersionedStore) CompareAndSwap(ctx context.Context, key string, expectedVersion uint64, value any) (uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.versions[key] != expectedVersion {
		return m.versions[key], ErrVersionMismatch
	}
	m.versions[key]++
	m.data[key] = value
	return m.versions[key], nil
}

func TestReducerStore_WithReducer(t *testing.T) {
	inner := newMockVersionedStore()
	rs := NewReducerStore(inner,
		WithReducer("counter", func(old, new any) any {
			o, _ := old.(int)
			n, _ := new.(int)
			return o + n
		}),
	)
	defer rs.Close()
	ctx := context.Background()

	// First set: old is nil, reducer gets (nil, 5) -> 0+5 = 5.
	require.NoError(t, rs.Set(ctx, "counter", 5))
	val, err := rs.Get(ctx, "counter")
	require.NoError(t, err)
	assert.Equal(t, 5, val)

	// Second set: reducer gets (5, 3) -> 8.
	require.NoError(t, rs.Set(ctx, "counter", 3))
	val, err = rs.Get(ctx, "counter")
	require.NoError(t, err)
	assert.Equal(t, 8, val)
}

func TestReducerStore_NoReducerOverwrites(t *testing.T) {
	inner := newMockVersionedStore()
	rs := NewReducerStore(inner,
		WithReducer("special", func(old, new any) any {
			return "merged"
		}),
	)
	defer rs.Close()
	ctx := context.Background()

	// Key without reducer — plain overwrite.
	require.NoError(t, rs.Set(ctx, "plain", "v1"))
	require.NoError(t, rs.Set(ctx, "plain", "v2"))
	val, err := rs.Get(ctx, "plain")
	require.NoError(t, err)
	assert.Equal(t, "v2", val)
}

func TestReducerStore_DefaultReducer(t *testing.T) {
	inner := newMockVersionedStore()
	rs := NewReducerStore(inner,
		WithDefaultReducer(func(old, new any) any {
			// Append reducer: builds a slice.
			var s []any
			if old != nil {
				s, _ = old.([]any)
			}
			return append(s, new)
		}),
	)
	defer rs.Close()
	ctx := context.Background()

	require.NoError(t, rs.Set(ctx, "log", "entry1"))
	require.NoError(t, rs.Set(ctx, "log", "entry2"))

	val, err := rs.Get(ctx, "log")
	require.NoError(t, err)
	assert.Equal(t, []any{"entry1", "entry2"}, val)
}

func TestReducerStore_PerKeyOverridesDefault(t *testing.T) {
	inner := newMockVersionedStore()
	rs := NewReducerStore(inner,
		WithDefaultReducer(func(old, new any) any {
			return "default"
		}),
		WithReducer("special", func(old, new any) any {
			return "specific"
		}),
	)
	defer rs.Close()
	ctx := context.Background()

	require.NoError(t, rs.Set(ctx, "special", "x"))
	val, err := rs.Get(ctx, "special")
	require.NoError(t, err)
	assert.Equal(t, "specific", val)

	require.NoError(t, rs.Set(ctx, "other", "x"))
	val, err = rs.Get(ctx, "other")
	require.NoError(t, err)
	assert.Equal(t, "default", val)
}

func TestReducerStore_CASPassthrough(t *testing.T) {
	inner := newMockVersionedStore()
	rs := NewReducerStore(inner)
	ctx := context.Background()

	newVer, err := rs.CompareAndSwap(ctx, "k", 0, "v1")
	require.NoError(t, err)
	assert.Equal(t, uint64(1), newVer)

	_, err = rs.CompareAndSwap(ctx, "k", 0, "v2")
	assert.ErrorIs(t, err, ErrVersionMismatch)
}

func TestReducerStore_GetVersioned(t *testing.T) {
	inner := newMockVersionedStore()
	rs := NewReducerStore(inner)
	ctx := context.Background()

	require.NoError(t, rs.Set(ctx, "k", "v"))
	val, ver, err := rs.GetVersioned(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, "v", val)
	assert.Equal(t, uint64(1), ver)
}

func TestReducerStore_Delete(t *testing.T) {
	inner := newMockVersionedStore()
	rs := NewReducerStore(inner)
	ctx := context.Background()

	require.NoError(t, rs.Set(ctx, "k", "v"))
	require.NoError(t, rs.Delete(ctx, "k"))

	val, err := rs.Get(ctx, "k")
	require.NoError(t, err)
	assert.Nil(t, val)
}

// Compile-time checks.
var _ Store = (*ReducerStore)(nil)
var _ VersionedStore = (*ReducerStore)(nil)
