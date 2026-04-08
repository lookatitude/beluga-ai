package associative

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestNote(id, content string, embedding []float32) *schema.Note {
	now := time.Now().UTC()
	return &schema.Note{
		ID:        id,
		Content:   content,
		Keywords:  []string{"test"},
		Tags:      []string{"unit"},
		Embedding: embedding,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestInMemoryNoteStore_Add(t *testing.T) {
	tests := []struct {
		name    string
		note    *schema.Note
		setup   func(s *InMemoryNoteStore)
		wantErr string
	}{
		{
			name: "happy path",
			note: newTestNote("n1", "hello", []float32{1, 0, 0}),
		},
		{
			name:    "nil note",
			note:    nil,
			wantErr: "note is nil",
		},
		{
			name:    "empty ID",
			note:    &schema.Note{Content: "test"},
			wantErr: "note ID is empty",
		},
		{
			name: "duplicate ID",
			note: newTestNote("dup", "second", nil),
			setup: func(s *InMemoryNoteStore) {
				_ = s.Add(context.Background(), newTestNote("dup", "first", nil))
			},
			wantErr: "already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewInMemoryNoteStore()
			if tt.setup != nil {
				tt.setup(store)
			}
			err := store.Add(context.Background(), tt.note)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestInMemoryNoteStore_Get(t *testing.T) {
	store := NewInMemoryNoteStore()
	ctx := context.Background()

	note := newTestNote("n1", "content", []float32{1, 0})
	require.NoError(t, store.Add(ctx, note))

	t.Run("found", func(t *testing.T) {
		got, err := store.Get(ctx, "n1")
		require.NoError(t, err)
		assert.Equal(t, "n1", got.ID)
		assert.Equal(t, "content", got.Content)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := store.Get(ctx, "missing")
		require.Error(t, err)
		var coreErr *core.Error
		require.True(t, errors.As(err, &coreErr))
		assert.Equal(t, core.ErrNotFound, coreErr.Code)
	})

	t.Run("returns copy", func(t *testing.T) {
		got, err := store.Get(ctx, "n1")
		require.NoError(t, err)
		got.Content = "mutated"
		got2, _ := store.Get(ctx, "n1")
		assert.Equal(t, "content", got2.Content, "store should not be mutated by modifying returned note")
	})
}

func TestInMemoryNoteStore_Update(t *testing.T) {
	store := NewInMemoryNoteStore()
	ctx := context.Background()

	note := newTestNote("n1", "original", nil)
	require.NoError(t, store.Add(ctx, note))

	t.Run("happy path", func(t *testing.T) {
		updated := newTestNote("n1", "updated", nil)
		require.NoError(t, store.Update(ctx, updated))
		got, _ := store.Get(ctx, "n1")
		assert.Equal(t, "updated", got.Content)
	})

	t.Run("not found", func(t *testing.T) {
		err := store.Update(ctx, newTestNote("missing", "x", nil))
		require.Error(t, err)
		var coreErr *core.Error
		require.True(t, errors.As(err, &coreErr))
		assert.Equal(t, core.ErrNotFound, coreErr.Code)
	})

	t.Run("nil note", func(t *testing.T) {
		err := store.Update(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "note is nil")
	})
}

func TestInMemoryNoteStore_Delete(t *testing.T) {
	store := NewInMemoryNoteStore()
	ctx := context.Background()

	require.NoError(t, store.Add(ctx, newTestNote("n1", "content", nil)))

	t.Run("happy path", func(t *testing.T) {
		require.NoError(t, store.Delete(ctx, "n1"))
		_, err := store.Get(ctx, "n1")
		require.Error(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		err := store.Delete(ctx, "missing")
		require.Error(t, err)
		var coreErr *core.Error
		require.True(t, errors.As(err, &coreErr))
		assert.Equal(t, core.ErrNotFound, coreErr.Code)
	})
}

func TestInMemoryNoteStore_Search(t *testing.T) {
	store := NewInMemoryNoteStore()
	ctx := context.Background()

	// Add notes with known embeddings.
	require.NoError(t, store.Add(ctx, newTestNote("n1", "go programming", []float32{1, 0, 0})))
	require.NoError(t, store.Add(ctx, newTestNote("n2", "rust programming", []float32{0.9, 0.1, 0})))
	require.NoError(t, store.Add(ctx, newTestNote("n3", "cooking recipes", []float32{0, 0, 1})))
	require.NoError(t, store.Add(ctx, newTestNote("n4", "no embedding", nil)))

	t.Run("returns top-k by similarity", func(t *testing.T) {
		results, err := store.Search(ctx, []float32{1, 0, 0}, 2)
		require.NoError(t, err)
		require.Len(t, results, 2)
		assert.Equal(t, "n1", results[0].ID, "most similar should be first")
		assert.Equal(t, "n2", results[1].ID, "second most similar should be second")
	})

	t.Run("k larger than store", func(t *testing.T) {
		results, err := store.Search(ctx, []float32{1, 0, 0}, 100)
		require.NoError(t, err)
		assert.Len(t, results, 3, "should return all embedded notes")
	})

	t.Run("k zero returns nil", func(t *testing.T) {
		results, err := store.Search(ctx, []float32{1, 0, 0}, 0)
		require.NoError(t, err)
		assert.Nil(t, results)
	})
}

func TestInMemoryNoteStore_List(t *testing.T) {
	store := NewInMemoryNoteStore()
	ctx := context.Background()

	t.Run("empty store", func(t *testing.T) {
		notes, err := store.List(ctx)
		require.NoError(t, err)
		assert.Empty(t, notes)
	})

	t.Run("ordered by creation time", func(t *testing.T) {
		t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		t2 := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
		t3 := time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC)

		n1 := newTestNote("c", "third added but earliest time", nil)
		n1.CreatedAt = t1
		n2 := newTestNote("a", "first added but middle time", nil)
		n2.CreatedAt = t2
		n3 := newTestNote("b", "second added but latest time", nil)
		n3.CreatedAt = t3

		require.NoError(t, store.Add(ctx, n2))
		require.NoError(t, store.Add(ctx, n3))
		require.NoError(t, store.Add(ctx, n1))

		notes, err := store.List(ctx)
		require.NoError(t, err)
		require.Len(t, notes, 3)
		assert.Equal(t, "c", notes[0].ID)
		assert.Equal(t, "a", notes[1].ID)
		assert.Equal(t, "b", notes[2].ID)
	})
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a, b []float32
		want float64
	}{
		{"identical", []float32{1, 0}, []float32{1, 0}, 1.0},
		{"orthogonal", []float32{1, 0}, []float32{0, 1}, 0.0},
		{"opposite", []float32{1, 0}, []float32{-1, 0}, -1.0},
		{"different lengths", []float32{1, 0}, []float32{1, 0, 0}, 0.0},
		{"empty", nil, nil, 0.0},
		{"zero vector", []float32{0, 0}, []float32{1, 0}, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cosineSimilarity(tt.a, tt.b)
			assert.InDelta(t, tt.want, got, 1e-6)
		})
	}
}

func TestInMemoryNoteStore_Concurrent(t *testing.T) {
	store := NewInMemoryNoteStore()
	ctx := context.Background()

	// Concurrent adds should not race.
	done := make(chan struct{})
	for i := 0; i < 50; i++ {
		go func(idx int) {
			defer func() { done <- struct{}{} }()
			n := newTestNote(
				time.Now().Format(time.RFC3339Nano)+"-"+string(rune('a'+idx%26)),
				"content",
				[]float32{float32(idx), 0, 0},
			)
			_ = store.Add(ctx, n)
		}(i)
	}
	for i := 0; i < 50; i++ {
		<-done
	}

	notes, err := store.List(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, notes)
}
