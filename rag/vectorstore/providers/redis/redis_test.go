package redis

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) (*miniredis.Miniredis, *Store) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := goredis.NewClient(&goredis.Options{
		Addr: mr.Addr(),
	})

	store := New(mr.Addr(),
		WithIndex("idx:test"),
		WithPrefix("test:"),
		WithDimension(3),
		WithClient(client),
	)

	return mr, store
}

func TestNew(t *testing.T) {
	store := New("localhost:6379",
		WithIndex("idx:docs"),
		WithPrefix("doc:"),
		WithDimension(128),
	)
	require.NotNil(t, store)
	assert.Equal(t, "idx:docs", store.index)
	assert.Equal(t, "doc:", store.prefix)
	assert.Equal(t, 128, store.dimension)
}

func TestNew_Defaults(t *testing.T) {
	store := New("localhost:6379")
	assert.Equal(t, "idx:documents", store.index)
	assert.Equal(t, "doc:", store.prefix)
	assert.Equal(t, 1536, store.dimension)
}

func TestStore_InterfaceCompliance(t *testing.T) {
	var _ vectorstore.VectorStore = (*Store)(nil)
}

func TestStore_Add(t *testing.T) {
	mr, store := newTestStore(t)

	docs := []schema.Document{
		{ID: "doc1", Content: "hello", Metadata: map[string]any{"category": "A"}},
		{ID: "doc2", Content: "world"},
	}
	embeddings := [][]float32{
		{0.1, 0.2, 0.3},
		{0.4, 0.5, 0.6},
	}

	err := store.Add(context.Background(), docs, embeddings)
	require.NoError(t, err)

	// Verify keys were created.
	assert.True(t, mr.Exists("test:doc1"))
	assert.True(t, mr.Exists("test:doc2"))

	// Verify content was stored.
	content := mr.HGet("test:doc1", "content")
	assert.Equal(t, "hello", content)
}

func TestStore_Add_MismatchedLength(t *testing.T) {
	_, store := newTestStore(t)
	err := store.Add(context.Background(),
		[]schema.Document{{ID: "doc1"}},
		[][]float32{{0.1}, {0.2}},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docs length")
}

func TestStore_Add_Metadata(t *testing.T) {
	mr, store := newTestStore(t)

	docs := []schema.Document{
		{ID: "doc1", Content: "test", Metadata: map[string]any{"category": "A", "priority": "high"}},
	}
	embeddings := [][]float32{{0.1, 0.2, 0.3}}

	err := store.Add(context.Background(), docs, embeddings)
	require.NoError(t, err)

	category := mr.HGet("test:doc1", "category")
	assert.Equal(t, "A", category)

	priority := mr.HGet("test:doc1", "priority")
	assert.Equal(t, "high", priority)
}

func TestStore_Delete(t *testing.T) {
	mr, store := newTestStore(t)

	// Pre-populate.
	mr.HSet("test:doc1", "content", "hello")
	mr.HSet("test:doc2", "content", "world")

	err := store.Delete(context.Background(), []string{"doc1"})
	require.NoError(t, err)

	assert.False(t, mr.Exists("test:doc1"))
	assert.True(t, mr.Exists("test:doc2"))
}

func TestStore_Delete_Empty(t *testing.T) {
	_, store := newTestStore(t)
	err := store.Delete(context.Background(), []string{})
	require.NoError(t, err)
}

func TestStore_Delete_NonExistent(t *testing.T) {
	_, store := newTestStore(t)
	err := store.Delete(context.Background(), []string{"nonexistent"})
	require.NoError(t, err)
}

func TestRegistry_Integration(t *testing.T) {
	names := vectorstore.List()
	assert.Contains(t, names, "redis")
}

func TestNewFromConfig(t *testing.T) {
	store, err := NewFromConfig(config.ProviderConfig{
		BaseURL: "localhost:6380",
		Options: map[string]any{
			"index":     "idx:custom",
			"prefix":    "custom:",
			"dimension": float64(768),
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "idx:custom", store.index)
	assert.Equal(t, "custom:", store.prefix)
	assert.Equal(t, 768, store.dimension)
}

func TestNewFromConfig_Defaults(t *testing.T) {
	store, err := NewFromConfig(config.ProviderConfig{})
	require.NoError(t, err)
	assert.Equal(t, "idx:documents", store.index)
	assert.Equal(t, "doc:", store.prefix)
	assert.Equal(t, 1536, store.dimension)
}

func TestFloat32ToBytes(t *testing.T) {
	input := []float32{1.0, 2.0, 3.0}
	result := float32ToBytes(input)
	assert.Len(t, result, 12) // 3 * 4 bytes
}

func TestFloat32ToBytes_Empty(t *testing.T) {
	result := float32ToBytes([]float32{})
	assert.Len(t, result, 0)
}

func TestStore_Add_Overwrite(t *testing.T) {
	mr, store := newTestStore(t)

	// First add.
	err := store.Add(context.Background(),
		[]schema.Document{{ID: "doc1", Content: "original"}},
		[][]float32{{0.1, 0.2, 0.3}},
	)
	require.NoError(t, err)

	content := mr.HGet("test:doc1", "content")
	assert.Equal(t, "original", content)

	// Overwrite.
	err = store.Add(context.Background(),
		[]schema.Document{{ID: "doc1", Content: "updated"}},
		[][]float32{{0.4, 0.5, 0.6}},
	)
	require.NoError(t, err)

	content = mr.HGet("test:doc1", "content")
	assert.Equal(t, "updated", content)
}
