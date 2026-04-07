package audit

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ----------------------------------------------------------------------------
// Entry enrichment
// ----------------------------------------------------------------------------

func TestEnrichEntry_GeneratesIDAndTimestamp(t *testing.T) {
	e, err := enrichEntry(Entry{Action: "test.action"})
	require.NoError(t, err)
	assert.NotEmpty(t, e.ID, "expected generated ID")
	assert.False(t, e.Timestamp.IsZero(), "expected generated timestamp")
}

func TestEnrichEntry_PreservesExistingIDAndTimestamp(t *testing.T) {
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	e, err := enrichEntry(Entry{ID: "existing-id", Timestamp: ts, Action: "test.action"})
	require.NoError(t, err)
	assert.Equal(t, "existing-id", e.ID)
	assert.Equal(t, ts, e.Timestamp)
}

// ----------------------------------------------------------------------------
// InMemoryStore.Log
// ----------------------------------------------------------------------------

func TestInMemoryStore_Log_HappyPath(t *testing.T) {
	s := NewInMemoryStore()
	ctx := context.Background()

	err := s.Log(ctx, Entry{
		TenantID:  "t1",
		AgentID:   "a1",
		SessionID: "s1",
		Action:    "tool.execute",
		Duration:  100 * time.Millisecond,
	})
	require.NoError(t, err)

	s.mu.RLock()
	count := len(s.entries)
	s.mu.RUnlock()
	assert.Equal(t, 1, count)
}

func TestInMemoryStore_Log_GeneratesFieldsWhenEmpty(t *testing.T) {
	s := NewInMemoryStore()
	ctx := context.Background()

	require.NoError(t, s.Log(ctx, Entry{Action: "llm.generate"}))

	s.mu.RLock()
	e := s.entries[0]
	s.mu.RUnlock()

	assert.NotEmpty(t, e.ID)
	assert.False(t, e.Timestamp.IsZero())
}

func TestInMemoryStore_Log_CancelledContext(t *testing.T) {
	s := NewInMemoryStore()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := s.Log(ctx, Entry{Action: "test.action"})
	assert.Error(t, err)
}

// ----------------------------------------------------------------------------
// InMemoryStore.Query — filter tests
// ----------------------------------------------------------------------------

func buildStore(t *testing.T) *InMemoryStore {
	t.Helper()
	s := NewInMemoryStore()
	ctx := context.Background()

	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	entries := []Entry{
		{TenantID: "t1", AgentID: "a1", SessionID: "s1", Action: "tool.execute", Timestamp: base},
		{TenantID: "t1", AgentID: "a1", SessionID: "s2", Action: "llm.generate", Timestamp: base.Add(time.Hour)},
		{TenantID: "t1", AgentID: "a2", SessionID: "s1", Action: "tool.execute", Timestamp: base.Add(2 * time.Hour)},
		{TenantID: "t2", AgentID: "a3", SessionID: "s3", Action: "llm.generate", Timestamp: base.Add(3 * time.Hour)},
	}
	for _, e := range entries {
		require.NoError(t, s.Log(ctx, e))
	}
	return s
}

func TestInMemoryStore_Query(t *testing.T) {
	s := buildStore(t)
	ctx := context.Background()
	base := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		filter    Filter
		wantCount int
	}{
		{
			name:      "empty filter returns all",
			filter:    Filter{},
			wantCount: 4,
		},
		{
			name:      "filter by tenant",
			filter:    Filter{TenantID: "t1"},
			wantCount: 3,
		},
		{
			name:      "filter by agent",
			filter:    Filter{AgentID: "a1"},
			wantCount: 2,
		},
		{
			name:      "filter by session",
			filter:    Filter{SessionID: "s1"},
			wantCount: 2,
		},
		{
			name:      "filter by action",
			filter:    Filter{Action: "tool.execute"},
			wantCount: 2,
		},
		{
			name:      "filter by tenant and action",
			filter:    Filter{TenantID: "t1", Action: "tool.execute"},
			wantCount: 2,
		},
		{
			name:      "filter by Since",
			filter:    Filter{Since: base.Add(time.Hour)},
			wantCount: 3,
		},
		{
			name:      "filter by Until",
			filter:    Filter{Until: base.Add(time.Hour)},
			wantCount: 2,
		},
		{
			name:      "filter by Since and Until",
			filter:    Filter{Since: base.Add(time.Hour), Until: base.Add(2 * time.Hour)},
			wantCount: 2,
		},
		{
			name:      "limit caps results",
			filter:    Filter{Limit: 2},
			wantCount: 2,
		},
		{
			name:      "no matching results",
			filter:    Filter{TenantID: "nonexistent"},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := s.Query(ctx, tt.filter)
			require.NoError(t, err)
			assert.Len(t, results, tt.wantCount)
		})
	}
}

func TestInMemoryStore_Query_CancelledContext(t *testing.T) {
	s := buildStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := s.Query(ctx, Filter{})
	assert.Error(t, err)
}

// ----------------------------------------------------------------------------
// Concurrent access
// ----------------------------------------------------------------------------

func TestInMemoryStore_ConcurrentAccess(t *testing.T) {
	s := NewInMemoryStore()
	ctx := context.Background()

	const writers = 10
	const readsPerWriter = 5
	var wg sync.WaitGroup

	wg.Add(writers)
	for i := 0; i < writers; i++ {
		go func(n int) {
			defer wg.Done()
			_ = s.Log(ctx, Entry{
				TenantID: fmt.Sprintf("tenant-%d", n),
				Action:   "concurrent.write",
			})
		}(i)
	}

	wg.Add(writers * readsPerWriter)
	for i := 0; i < writers*readsPerWriter; i++ {
		go func() {
			defer wg.Done()
			_, _ = s.Query(ctx, Filter{})
		}()
	}

	wg.Wait()

	entries, err := s.Query(ctx, Filter{})
	require.NoError(t, err)
	assert.Equal(t, writers, len(entries))
}

// ----------------------------------------------------------------------------
// Registry
// ----------------------------------------------------------------------------

func TestRegistry_ListContainsInMemory(t *testing.T) {
	names := List()
	assert.Contains(t, names, "inmemory")
}

func TestRegistry_NewInMemory(t *testing.T) {
	store, err := New("inmemory", Config{})
	require.NoError(t, err)
	assert.NotNil(t, store)
}

func TestRegistry_NewUnknown(t *testing.T) {
	_, err := New("does-not-exist", Config{})
	assert.Error(t, err)
}

func TestRegistry_List_Sorted(t *testing.T) {
	names := List()
	for i := 1; i < len(names); i++ {
		assert.LessOrEqual(t, names[i-1], names[i], "List() must return sorted names")
	}
}

// ----------------------------------------------------------------------------
// Unique IDs across multiple entries
// ----------------------------------------------------------------------------

func TestInMemoryStore_UniqueIDs(t *testing.T) {
	s := NewInMemoryStore()
	ctx := context.Background()

	for i := 0; i < 100; i++ {
		require.NoError(t, s.Log(ctx, Entry{Action: "test.id"}))
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := make(map[string]struct{}, len(s.entries))
	for _, e := range s.entries {
		assert.NotEmpty(t, e.ID)
		_, dup := seen[e.ID]
		assert.False(t, dup, "duplicate ID found: %s", e.ID)
		seen[e.ID] = struct{}{}
	}
}
