package temporal

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/v2/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryStore_AddEntity(t *testing.T) {
	tests := []struct {
		name    string
		entity  memory.Entity
		wantErr bool
	}{
		{
			name: "valid entity",
			entity: memory.Entity{
				ID:         "person-1",
				Type:       "person",
				Properties: map[string]any{"name": "Alice"},
			},
		},
		{
			name:    "empty ID errors",
			entity:  memory.Entity{ID: "", Type: "person"},
			wantErr: true,
		},
		{
			name: "nil properties accepted",
			entity: memory.Entity{
				ID:   "person-2",
				Type: "person",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewInMemoryStore()
			err := store.AddEntity(context.Background(), tt.entity)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInMemoryStore_AddEntity_Merge(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	// Add initial entity.
	created := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	err := store.AddEntity(ctx, memory.Entity{
		ID:         "person-1",
		Type:       "person",
		Properties: map[string]any{"name": "Alice", "age": 30},
		CreatedAt:  created,
	})
	require.NoError(t, err)

	// Update entity -- should merge properties and preserve CreatedAt.
	err = store.AddEntity(ctx, memory.Entity{
		ID:         "person-1",
		Type:       "person",
		Properties: map[string]any{"age": 31, "city": "NYC"},
		Summary:    "Alice from NYC",
	})
	require.NoError(t, err)

	// Verify merge.
	store.mu.RLock()
	entity := store.entities["person-1"]
	store.mu.RUnlock()

	assert.Equal(t, created, entity.CreatedAt) // preserved
	assert.Equal(t, "Alice", entity.Properties["name"])
	assert.Equal(t, 31, entity.Properties["age"]) // updated
	assert.Equal(t, "NYC", entity.Properties["city"])
	assert.Equal(t, "Alice from NYC", entity.Summary)
}

func TestInMemoryStore_AddRelation(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	// Add entities first.
	require.NoError(t, store.AddEntity(ctx, memory.Entity{ID: "a", Type: "person"}))
	require.NoError(t, store.AddEntity(ctx, memory.Entity{ID: "b", Type: "company"}))

	tests := []struct {
		name     string
		from     string
		to       string
		relation string
		wantErr  bool
	}{
		{name: "valid relation", from: "a", to: "b", relation: "works_at"},
		{name: "empty from errors", from: "", to: "b", relation: "works_at", wantErr: true},
		{name: "empty to errors", from: "a", to: "", relation: "works_at", wantErr: true},
		{name: "empty relation errors", from: "a", to: "b", relation: "", wantErr: true},
		{name: "missing from entity", from: "x", to: "b", relation: "works_at", wantErr: true},
		{name: "missing to entity", from: "a", to: "x", relation: "works_at", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.AddRelation(ctx, tt.from, tt.to, tt.relation, nil)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInMemoryStore_AddTemporalRelation(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	require.NoError(t, store.AddEntity(ctx, memory.Entity{ID: "a", Type: "person"}))
	require.NoError(t, store.AddEntity(ctx, memory.Entity{ID: "b", Type: "company"}))

	validAt := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		rel     memory.Relation
		wantErr bool
	}{
		{
			name: "valid temporal relation",
			rel: memory.Relation{
				From:     "a",
				To:       "b",
				Type:     "works_at",
				ValidAt:  validAt,
				Episodes: []string{"ep-1"},
			},
		},
		{
			name: "zero ValidAt errors",
			rel: memory.Relation{
				From: "a",
				To:   "b",
				Type: "works_at",
			},
			wantErr: true,
		},
		{
			name: "empty from errors",
			rel: memory.Relation{
				From:    "",
				To:      "b",
				Type:    "works_at",
				ValidAt: validAt,
			},
			wantErr: true,
		},
		{
			name: "missing entity errors",
			rel: memory.Relation{
				From:    "x",
				To:      "b",
				Type:    "works_at",
				ValidAt: validAt,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.AddTemporalRelation(ctx, tt.rel)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInMemoryStore_Query(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	require.NoError(t, store.AddEntity(ctx, memory.Entity{ID: "alice", Type: "person", Summary: "Alice Smith"}))
	require.NoError(t, store.AddEntity(ctx, memory.Entity{ID: "acme", Type: "company", Summary: "ACME Corp"}))
	require.NoError(t, store.AddRelation(ctx, "alice", "acme", "works_at", nil))

	tests := []struct {
		name          string
		query         string
		wantEntities  int
		wantRelations int
	}{
		{name: "match by type", query: "person", wantEntities: 1, wantRelations: 0},
		{name: "match by ID", query: "acme", wantEntities: 1, wantRelations: 0},
		{name: "match relation type", query: "works_at", wantEntities: 0, wantRelations: 1},
		{name: "no match", query: "zzz", wantEntities: 0, wantRelations: 0},
		{name: "case insensitive", query: "PERSON", wantEntities: 1, wantRelations: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := store.Query(ctx, tt.query)
			require.NoError(t, err)
			require.Len(t, results, 1)
			assert.Len(t, results[0].Entities, tt.wantEntities)
			assert.Len(t, results[0].Relations, tt.wantRelations)
		})
	}
}

func TestInMemoryStore_Neighbors(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	// Create a chain: a -> b -> c
	require.NoError(t, store.AddEntity(ctx, memory.Entity{ID: "a", Type: "person"}))
	require.NoError(t, store.AddEntity(ctx, memory.Entity{ID: "b", Type: "person"}))
	require.NoError(t, store.AddEntity(ctx, memory.Entity{ID: "c", Type: "person"}))
	require.NoError(t, store.AddRelation(ctx, "a", "b", "knows", nil))
	require.NoError(t, store.AddRelation(ctx, "b", "c", "knows", nil))

	t.Run("depth 1", func(t *testing.T) {
		entities, relations, err := store.Neighbors(ctx, "a", 1)
		require.NoError(t, err)
		assert.Len(t, entities, 1)  // b
		assert.Len(t, relations, 1) // a->b
	})

	t.Run("depth 2", func(t *testing.T) {
		entities, relations, err := store.Neighbors(ctx, "a", 2)
		require.NoError(t, err)
		assert.Len(t, entities, 2) // b, c
		assert.GreaterOrEqual(t, len(relations), 2)
	})

	t.Run("missing entity errors", func(t *testing.T) {
		_, _, err := store.Neighbors(ctx, "x", 1)
		require.Error(t, err)
	})
}

func TestInMemoryStore_QueryAsOf(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	require.NoError(t, store.AddEntity(ctx, memory.Entity{ID: "a", Type: "person"}))
	require.NoError(t, store.AddEntity(ctx, memory.Entity{ID: "b", Type: "company"}))
	require.NoError(t, store.AddEntity(ctx, memory.Entity{ID: "c", Type: "company"}))

	t1 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// a worked_at b from t1 to t2
	require.NoError(t, store.AddTemporalRelation(ctx, memory.Relation{
		From:       "a",
		To:         "b",
		Type:       "works_at",
		ValidAt:    t1,
		InvalidAt:  timePtr(t2),
		Properties: map[string]any{"id": "rel-1"},
	}))

	// a works_at c from t2 onward
	require.NoError(t, store.AddTemporalRelation(ctx, memory.Relation{
		From:       "a",
		To:         "c",
		Type:       "works_at",
		ValidAt:    t2,
		Properties: map[string]any{"id": "rel-2"},
	}))

	tests := []struct {
		name          string
		query         string
		validTime     time.Time
		wantRelations int
		wantErr       bool
	}{
		{
			name:          "query at t1 gets first relation",
			query:         "works_at",
			validTime:     t1.Add(time.Hour),
			wantRelations: 1,
		},
		{
			name:          "query at t2 gets second relation only",
			query:         "works_at",
			validTime:     t2.Add(time.Hour),
			wantRelations: 1,
		},
		{
			name:          "query at t3 gets second relation",
			query:         "works_at",
			validTime:     t3,
			wantRelations: 1,
		},
		{
			name:      "zero time errors",
			query:     "works_at",
			validTime: time.Time{},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, relations, err := store.QueryAsOf(ctx, tt.query, tt.validTime)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, relations, tt.wantRelations)
		})
	}
}

func TestInMemoryStore_InvalidateRelation(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	require.NoError(t, store.AddEntity(ctx, memory.Entity{ID: "a", Type: "person"}))
	require.NoError(t, store.AddEntity(ctx, memory.Entity{ID: "b", Type: "company"}))

	validAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, store.AddTemporalRelation(ctx, memory.Relation{
		From:       "a",
		To:         "b",
		Type:       "works_at",
		ValidAt:    validAt,
		Properties: map[string]any{"id": "rel-1"},
	}))

	invalidAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		relID   string
		time    time.Time
		wantErr bool
	}{
		{name: "valid invalidation", relID: "rel-1", time: invalidAt},
		{name: "empty ID errors", relID: "", time: invalidAt, wantErr: true},
		{name: "zero time errors", relID: "rel-1", time: time.Time{}, wantErr: true},
		{name: "not found errors", relID: "rel-999", time: invalidAt, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.InvalidateRelation(ctx, tt.relID, tt.time)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInMemoryStore_History(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	require.NoError(t, store.AddEntity(ctx, memory.Entity{ID: "a", Type: "person"}))
	require.NoError(t, store.AddEntity(ctx, memory.Entity{ID: "b", Type: "company"}))

	t1 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

	require.NoError(t, store.AddTemporalRelation(ctx, memory.Relation{
		From:       "a",
		To:         "b",
		Type:       "works_at",
		ValidAt:    t1,
		Properties: map[string]any{"id": "rel-1"},
	}))
	require.NoError(t, store.AddTemporalRelation(ctx, memory.Relation{
		From:       "a",
		To:         "b",
		Type:       "works_at",
		ValidAt:    t2,
		Properties: map[string]any{"id": "rel-2"},
	}))

	t.Run("returns all versions sorted by ValidAt", func(t *testing.T) {
		history, err := store.History(ctx, "a", "b")
		require.NoError(t, err)
		require.Len(t, history, 2)
		assert.True(t, history[0].ValidAt.Before(history[1].ValidAt))
	})

	t.Run("no history for non-existent pair", func(t *testing.T) {
		history, err := store.History(ctx, "a", "c")
		require.NoError(t, err)
		assert.Empty(t, history)
	})

	t.Run("empty IDs error", func(t *testing.T) {
		_, err := store.History(ctx, "", "b")
		require.Error(t, err)
	})
}

func TestInMemoryStore_Clear(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	require.NoError(t, store.AddEntity(ctx, memory.Entity{ID: "a", Type: "person"}))
	require.NoError(t, store.AddEntity(ctx, memory.Entity{ID: "b", Type: "company"}))
	require.NoError(t, store.AddRelation(ctx, "a", "b", "works_at", nil))

	err := store.Clear(ctx)
	require.NoError(t, err)

	// Verify empty.
	store.mu.RLock()
	assert.Empty(t, store.entities)
	assert.Empty(t, store.relations)
	store.mu.RUnlock()
}

func TestInMemoryStore_ContextCancellation(t *testing.T) {
	store := NewInMemoryStore()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := store.AddEntity(ctx, memory.Entity{ID: "a", Type: "person"})
	assert.ErrorIs(t, err, context.Canceled)

	err = store.AddRelation(ctx, "a", "b", "knows", nil)
	assert.ErrorIs(t, err, context.Canceled)

	_, err = store.Query(ctx, "test")
	assert.ErrorIs(t, err, context.Canceled)

	_, _, err = store.Neighbors(ctx, "a", 1)
	assert.ErrorIs(t, err, context.Canceled)

	_, _, err = store.QueryAsOf(ctx, "test", time.Now())
	assert.ErrorIs(t, err, context.Canceled)

	err = store.InvalidateRelation(ctx, "rel-1", time.Now())
	assert.ErrorIs(t, err, context.Canceled)

	_, err = store.History(ctx, "a", "b")
	assert.ErrorIs(t, err, context.Canceled)
}

func TestInMemoryStore_ThreadSafety(t *testing.T) {
	store := NewInMemoryStore()
	ctx := context.Background()

	// Pre-populate entities.
	for i := 0; i < 10; i++ {
		require.NoError(t, store.AddEntity(ctx, memory.Entity{
			ID:   fmt.Sprintf("e-%d", i),
			Type: "node",
		}))
	}

	var wg sync.WaitGroup
	const goroutines = 20

	// Concurrent reads and writes.
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = store.AddEntity(ctx, memory.Entity{
				ID:   fmt.Sprintf("concurrent-%d", n),
				Type: "node",
			})
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = store.Query(ctx, "node")
		}()
	}

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = store.History(ctx, "e-0", "e-1")
		}()
	}

	wg.Wait()
}
