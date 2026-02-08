//go:build cgo

package sqlitevec

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	store, err := New(WithDB(db), WithDimension(3), WithTable("test_docs"))
	require.NoError(t, err)

	err = store.EnsureTable(context.Background())
	if err != nil {
		if strings.Contains(err.Error(), "no such module: vec0") {
			t.Skip("sqlite-vec extension not available")
		}
		require.NoError(t, err)
	}

	return store
}

func TestNew(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	store, err := New(WithDB(db), WithTable("my_docs"), WithDimension(128))
	require.NoError(t, err)
	require.NotNil(t, store)
	assert.Equal(t, "my_docs", store.table)
	assert.Equal(t, 128, store.dimension)
}

func TestNew_Defaults(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	store, err := New(WithDB(db))
	require.NoError(t, err)
	assert.Equal(t, "documents", store.table)
	assert.Equal(t, 1536, store.dimension)
}

func TestNew_NoDB(t *testing.T) {
	_, err := New()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database connection is required")
}

func TestStore_InterfaceCompliance(t *testing.T) {
	var _ vectorstore.VectorStore = (*Store)(nil)
}

func TestStore_EnsureTable(t *testing.T) {
	store := newTestStore(t)

	// EnsureTable should be idempotent.
	err := store.EnsureTable(context.Background())
	require.NoError(t, err)
}

func TestStore_Add_And_Search(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	docs := []schema.Document{
		{ID: "doc1", Content: "hello world", Metadata: map[string]any{"category": "A"}},
		{ID: "doc2", Content: "goodbye world", Metadata: map[string]any{"category": "B"}},
	}
	embeddings := [][]float32{
		{1.0, 0.0, 0.0},
		{0.0, 1.0, 0.0},
	}

	err := store.Add(ctx, docs, embeddings)
	require.NoError(t, err)

	// Search with a query vector close to doc1.
	results, err := store.Search(ctx, []float32{0.9, 0.1, 0.0}, 2)
	require.NoError(t, err)
	require.NotEmpty(t, results)

	// First result should be doc1 (closer to query).
	assert.Equal(t, "doc1", results[0].ID)
	assert.Equal(t, "hello world", results[0].Content)
	assert.True(t, results[0].Score > 0)
}

func TestStore_Add_MismatchedLength(t *testing.T) {
	store := newTestStore(t)
	err := store.Add(context.Background(),
		[]schema.Document{{ID: "doc1"}},
		[][]float32{{0.1, 0.2, 0.3}, {0.4, 0.5, 0.6}},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docs length")
}

func TestStore_Add_Empty(t *testing.T) {
	store := newTestStore(t)
	err := store.Add(context.Background(), nil, nil)
	require.NoError(t, err)
}

func TestStore_Add_Upsert(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Insert a doc.
	err := store.Add(ctx,
		[]schema.Document{{ID: "doc1", Content: "original"}},
		[][]float32{{1.0, 0.0, 0.0}},
	)
	require.NoError(t, err)

	// Upsert with updated content.
	err = store.Add(ctx,
		[]schema.Document{{ID: "doc1", Content: "updated"}},
		[][]float32{{0.0, 1.0, 0.0}},
	)
	require.NoError(t, err)

	// Search and verify content was updated.
	results, err := store.Search(ctx, []float32{0.0, 1.0, 0.0}, 1)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "doc1", results[0].ID)
	assert.Equal(t, "updated", results[0].Content)
}

func TestStore_Search_WithThreshold(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Add two docs.
	err := store.Add(ctx,
		[]schema.Document{
			{ID: "doc1", Content: "close"},
			{ID: "doc2", Content: "far"},
		},
		[][]float32{
			{1.0, 0.0, 0.0},
			{0.0, 0.0, 1.0},
		},
	)
	require.NoError(t, err)

	// Search with threshold — only close docs should pass.
	results, err := store.Search(ctx, []float32{1.0, 0.0, 0.0}, 10,
		vectorstore.WithThreshold(0.9))
	require.NoError(t, err)
	// The very close doc should pass the threshold.
	for _, doc := range results {
		assert.True(t, doc.Score >= 0.9,
			"score %f should be >= 0.9", doc.Score)
	}
}

func TestStore_Search_WithFilter(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	err := store.Add(ctx,
		[]schema.Document{
			{ID: "doc1", Content: "hello", Metadata: map[string]any{"category": "A"}},
			{ID: "doc2", Content: "world", Metadata: map[string]any{"category": "B"}},
		},
		[][]float32{
			{1.0, 0.0, 0.0},
			{0.9, 0.1, 0.0},
		},
	)
	require.NoError(t, err)

	// Filter to only category A.
	results, err := store.Search(ctx, []float32{1.0, 0.0, 0.0}, 10,
		vectorstore.WithFilter(map[string]any{"category": "A"}))
	require.NoError(t, err)
	for _, doc := range results {
		assert.Equal(t, "A", doc.Metadata["category"])
	}
}

func TestStore_Search_Empty(t *testing.T) {
	store := newTestStore(t)
	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestStore_Delete(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Add docs.
	err := store.Add(ctx,
		[]schema.Document{
			{ID: "doc1", Content: "hello"},
			{ID: "doc2", Content: "world"},
		},
		[][]float32{
			{1.0, 0.0, 0.0},
			{0.0, 1.0, 0.0},
		},
	)
	require.NoError(t, err)

	// Delete doc1.
	err = store.Delete(ctx, []string{"doc1"})
	require.NoError(t, err)

	// Search — should only find doc2.
	results, err := store.Search(ctx, []float32{1.0, 0.0, 0.0}, 10)
	require.NoError(t, err)
	for _, doc := range results {
		assert.NotEqual(t, "doc1", doc.ID)
	}
}

func TestStore_Delete_Empty(t *testing.T) {
	store := newTestStore(t)
	err := store.Delete(context.Background(), []string{})
	require.NoError(t, err)
}

func TestStore_Delete_NonExistent(t *testing.T) {
	store := newTestStore(t)
	err := store.Delete(context.Background(), []string{"nonexistent"})
	require.NoError(t, err)
}

func TestStore_Metadata(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	meta := map[string]any{
		"source": "test",
		"page":   float64(42),
	}
	err := store.Add(ctx,
		[]schema.Document{{ID: "doc1", Content: "test", Metadata: meta}},
		[][]float32{{1.0, 0.0, 0.0}},
	)
	require.NoError(t, err)

	results, err := store.Search(ctx, []float32{1.0, 0.0, 0.0}, 1)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "test", results[0].Metadata["source"])
	assert.Equal(t, float64(42), results[0].Metadata["page"])
}

func TestMatchesFilter(t *testing.T) {
	tests := []struct {
		name     string
		meta     map[string]any
		filter   map[string]any
		expected bool
	}{
		{"nil filter", map[string]any{"a": "1"}, nil, true},
		{"nil meta", nil, map[string]any{"a": "1"}, false},
		{"match", map[string]any{"a": "1", "b": "2"}, map[string]any{"a": "1"}, true},
		{"no match", map[string]any{"a": "1"}, map[string]any{"a": "2"}, false},
		{"missing key", map[string]any{"a": "1"}, map[string]any{"b": "1"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, matchesFilter(tt.meta, tt.filter))
		})
	}
}
