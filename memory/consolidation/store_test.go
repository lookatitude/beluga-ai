package consolidation

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(records ...Record) *InMemoryConsolidationStore {
	s := NewInMemoryConsolidationStore()
	for _, r := range records {
		s.Add(r)
	}
	return s
}

func TestInMemoryConsolidationStore_ListRecords(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	s := newTestStore(
		Record{ID: "a", CreatedAt: now},
		Record{ID: "b", CreatedAt: now},
		Record{ID: "c", CreatedAt: now},
	)

	tests := []struct {
		name   string
		offset int
		limit  int
		want   []string
	}{
		{"all", 0, 10, []string{"a", "b", "c"}},
		{"first two", 0, 2, []string{"a", "b"}},
		{"skip first", 1, 2, []string{"b", "c"}},
		{"offset beyond length", 10, 5, nil},
		{"negative offset clamped", -1, 10, []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records, err := s.ListRecords(ctx, tt.offset, tt.limit)
			require.NoError(t, err)
			var ids []string
			for _, r := range records {
				ids = append(ids, r.ID)
			}
			assert.Equal(t, tt.want, ids)
		})
	}
}

func TestInMemoryConsolidationStore_DeleteRecords(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(
		Record{ID: "a"},
		Record{ID: "b"},
		Record{ID: "c"},
	)

	err := s.DeleteRecords(ctx, []string{"a", "c"})
	require.NoError(t, err)

	assert.Equal(t, 1, s.Len())
	records, err := s.ListRecords(ctx, 0, 10)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "b", records[0].ID)
}

func TestInMemoryConsolidationStore_UpdateRecords(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(Record{ID: "a", Content: "original"})

	err := s.UpdateRecords(ctx, []Record{{ID: "a", Content: "updated"}})
	require.NoError(t, err)

	r, ok := s.Get("a")
	require.True(t, ok)
	assert.Equal(t, "updated", r.Content)
}

func TestInMemoryConsolidationStore_UpdateRecords_NonExistent(t *testing.T) {
	ctx := context.Background()
	s := newTestStore()

	// Should not error, just silently ignore.
	err := s.UpdateRecords(ctx, []Record{{ID: "missing", Content: "x"}})
	require.NoError(t, err)
	assert.Equal(t, 0, s.Len())
}

func TestInMemoryConsolidationStore_RecordAccess(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(Record{ID: "a"})

	err := s.RecordAccess(ctx, "a")
	require.NoError(t, err)

	r, ok := s.Get("a")
	require.True(t, ok)
	assert.Equal(t, 1, r.Utility.AccessCount)
	assert.False(t, r.Utility.LastAccessedAt.IsZero())

	err = s.RecordAccess(ctx, "a")
	require.NoError(t, err)
	r, _ = s.Get("a")
	assert.Equal(t, 2, r.Utility.AccessCount)
}

func TestInMemoryConsolidationStore_RecordAccess_NotFound(t *testing.T) {
	ctx := context.Background()
	s := newTestStore()

	err := s.RecordAccess(ctx, "missing")
	assert.Error(t, err)
}

func TestInMemoryConsolidationStore_Add_Overwrite(t *testing.T) {
	s := newTestStore(Record{ID: "a", Content: "v1"})
	s.Add(Record{ID: "a", Content: "v2"})

	assert.Equal(t, 1, s.Len())
	r, ok := s.Get("a")
	require.True(t, ok)
	assert.Equal(t, "v2", r.Content)
}
