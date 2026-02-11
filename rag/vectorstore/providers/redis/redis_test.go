package redis

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRedisClient implements RedisClient for testing Search, EnsureIndex, and error paths.
type mockRedisClient struct {
	hsetFn  func(ctx context.Context, key string, values ...any) *goredis.IntCmd
	delFn   func(ctx context.Context, keys ...string) *goredis.IntCmd
	doFn    func(ctx context.Context, args ...any) *goredis.Cmd
	closeFn func() error
}

func (m *mockRedisClient) HSet(ctx context.Context, key string, values ...any) *goredis.IntCmd {
	if m.hsetFn != nil {
		return m.hsetFn(ctx, key, values...)
	}
	cmd := goredis.NewIntCmd(ctx)
	cmd.SetVal(1)
	return cmd
}

func (m *mockRedisClient) Del(ctx context.Context, keys ...string) *goredis.IntCmd {
	if m.delFn != nil {
		return m.delFn(ctx, keys...)
	}
	cmd := goredis.NewIntCmd(ctx)
	cmd.SetVal(int64(len(keys)))
	return cmd
}

func (m *mockRedisClient) Do(ctx context.Context, args ...any) *goredis.Cmd {
	if m.doFn != nil {
		return m.doFn(ctx, args...)
	}
	cmd := goredis.NewCmd(ctx)
	return cmd
}

func (m *mockRedisClient) Close() error {
	if m.closeFn != nil {
		return m.closeFn()
	}
	return nil
}

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

func newMockStore(t *testing.T, mock *mockRedisClient) *Store {
	t.Helper()
	return New("localhost:6379",
		WithIndex("idx:test"),
		WithPrefix("test:"),
		WithDimension(3),
		WithClient(mock),
	)
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

func TestStore_Add_HSetError(t *testing.T) {
	mock := &mockRedisClient{
		hsetFn: func(ctx context.Context, key string, values ...any) *goredis.IntCmd {
			cmd := goredis.NewIntCmd(ctx)
			cmd.SetErr(errors.New("connection refused"))
			return cmd
		},
	}
	store := newMockStore(t, mock)

	err := store.Add(context.Background(),
		[]schema.Document{{ID: "doc1", Content: "test"}},
		[][]float32{{0.1, 0.2, 0.3}},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redis: hset")
	assert.Contains(t, err.Error(), "connection refused")
}

func TestStore_EnsureIndex_Success(t *testing.T) {
	mock := &mockRedisClient{
		doFn: func(ctx context.Context, args ...any) *goredis.Cmd {
			cmd := goredis.NewCmd(ctx)
			cmd.SetVal("OK")
			return cmd
		},
	}
	store := newMockStore(t, mock)

	err := store.EnsureIndex(context.Background())
	require.NoError(t, err)
}

func TestStore_EnsureIndex_AlreadyExists(t *testing.T) {
	mock := &mockRedisClient{
		doFn: func(ctx context.Context, args ...any) *goredis.Cmd {
			cmd := goredis.NewCmd(ctx)
			cmd.SetErr(errors.New("Index already exists"))
			return cmd
		},
	}
	store := newMockStore(t, mock)

	err := store.EnsureIndex(context.Background())
	require.NoError(t, err) // Should be silenced.
}

func TestStore_EnsureIndex_Error(t *testing.T) {
	mock := &mockRedisClient{
		doFn: func(ctx context.Context, args ...any) *goredis.Cmd {
			cmd := goredis.NewCmd(ctx)
			cmd.SetErr(errors.New("connection timeout"))
			return cmd
		},
	}
	store := newMockStore(t, mock)

	err := store.EnsureIndex(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection timeout")
}

func TestStore_Search_Success(t *testing.T) {
	mock := &mockRedisClient{
		doFn: func(ctx context.Context, args ...any) *goredis.Cmd {
			cmd := goredis.NewCmd(ctx)
			// FT.SEARCH result: [total, key1, [fields...], key2, [fields...]]
			cmd.SetVal([]any{
				int64(2),
				"test:doc1", []any{"content", "hello", "score", "0.1", "category", "A"},
				"test:doc2", []any{"content", "world", "score", "0.3"},
			})
			return cmd
		},
	}
	store := newMockStore(t, mock)

	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
	require.Len(t, results, 2)

	assert.Equal(t, "doc1", results[0].ID)
	assert.Equal(t, "hello", results[0].Content)
	assert.InDelta(t, 0.9, results[0].Score, 0.001) // 1.0 - 0.1
	assert.Equal(t, "A", results[0].Metadata["category"])

	assert.Equal(t, "doc2", results[1].ID)
	assert.Equal(t, "world", results[1].Content)
	assert.InDelta(t, 0.7, results[1].Score, 0.001) // 1.0 - 0.3
}

func TestStore_Search_WithFilter(t *testing.T) {
	var capturedArgs []any
	mock := &mockRedisClient{
		doFn: func(ctx context.Context, args ...any) *goredis.Cmd {
			capturedArgs = args
			cmd := goredis.NewCmd(ctx)
			cmd.SetVal([]any{int64(0)})
			return cmd
		},
	}
	store := newMockStore(t, mock)

	_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5,
		vectorstore.WithFilter(map[string]any{"category": "A"}))
	require.NoError(t, err)

	// Verify the query includes filter expression.
	queryStr, ok := capturedArgs[2].(string)
	require.True(t, ok)
	assert.Contains(t, queryStr, "@category:{A}")
}

func TestStore_Search_WithThreshold(t *testing.T) {
	mock := &mockRedisClient{
		doFn: func(ctx context.Context, args ...any) *goredis.Cmd {
			cmd := goredis.NewCmd(ctx)
			cmd.SetVal([]any{
				int64(2),
				"test:close", []any{"content", "close", "score", "0.05"},
				"test:far", []any{"content", "far", "score", "0.9"},
			})
			return cmd
		},
	}
	store := newMockStore(t, mock)

	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5,
		vectorstore.WithThreshold(0.5))
	require.NoError(t, err)

	// Only "close" doc should pass threshold (score 0.95 >= 0.5).
	// "far" doc has score 0.1, which is < 0.5.
	require.Len(t, results, 1)
	assert.Equal(t, "close", results[0].ID)
}

func TestStore_Search_Error(t *testing.T) {
	mock := &mockRedisClient{
		doFn: func(ctx context.Context, args ...any) *goredis.Cmd {
			cmd := goredis.NewCmd(ctx)
			cmd.SetErr(errors.New("search failed"))
			return cmd
		},
	}
	store := newMockStore(t, mock)

	_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redis: search")
}

func TestStore_Search_EmptyResult(t *testing.T) {
	mock := &mockRedisClient{
		doFn: func(ctx context.Context, args ...any) *goredis.Cmd {
			cmd := goredis.NewCmd(ctx)
			cmd.SetVal([]any{int64(0)})
			return cmd
		},
	}
	store := newMockStore(t, mock)

	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
	assert.Nil(t, results)
}

func TestParseFTSearchResult(t *testing.T) {
	tests := []struct {
		name      string
		val       any
		setErr    error
		prefix    string
		threshold float64
		wantDocs  int
		wantErr   bool
	}{
		{
			name:     "valid two results",
			val:      []any{int64(2), "doc:a", []any{"content", "hello", "score", "0.1"}, "doc:b", []any{"content", "world", "score", "0.2"}},
			prefix:   "doc:",
			wantDocs: 2,
		},
		{
			name:     "empty slice",
			val:      []any{},
			prefix:   "doc:",
			wantDocs: 0,
		},
		{
			name:    "cmd error",
			setErr:  errors.New("parse error"),
			prefix:  "doc:",
			wantErr: true,
		},
		{
			name:    "unexpected total format",
			val:     []any{"not-a-number"},
			prefix:  "doc:",
			wantErr: true,
		},
		{
			name:     "zero total",
			val:      []any{int64(0)},
			prefix:   "doc:",
			wantDocs: 0,
		},
		{
			name:     "non-string key skipped",
			val:      []any{int64(1), 12345, []any{"content", "hello"}},
			prefix:   "doc:",
			wantDocs: 0,
		},
		{
			name:     "non-slice fields skipped",
			val:      []any{int64(1), "doc:a", "not-a-slice"},
			prefix:   "doc:",
			wantDocs: 0,
		},
		{
			name:     "non-string field name skipped",
			val:      []any{int64(1), "doc:a", []any{12345, "val"}},
			prefix:   "doc:",
			wantDocs: 1, // doc created but with empty content
		},
		{
			name:     "embedding field skipped",
			val:      []any{int64(1), "doc:a", []any{"content", "hello", "embedding", "binary-data"}},
			prefix:   "doc:",
			wantDocs: 1,
		},
		{
			name:     "non-string content",
			val:      []any{int64(1), "doc:a", []any{"content", 42}},
			prefix:   "doc:",
			wantDocs: 1, // doc created but content stays empty
		},
		{
			name:     "non-string score",
			val:      []any{int64(1), "doc:a", []any{"score", 0.5}},
			prefix:   "doc:",
			wantDocs: 1,
		},
		{
			name:     "unparseable score string",
			val:      []any{int64(1), "doc:a", []any{"score", "not-a-number"}},
			prefix:   "doc:",
			wantDocs: 1,
		},
		{
			name:     "non-string metadata value",
			val:      []any{int64(1), "doc:a", []any{"category", 42}},
			prefix:   "doc:",
			wantDocs: 1,
		},
		{
			name:      "threshold filters low scores",
			val:       []any{int64(2), "doc:a", []any{"content", "high", "score", "0.05"}, "doc:b", []any{"content", "low", "score", "0.9"}},
			prefix:    "doc:",
			threshold: 0.5,
			wantDocs:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			cmd := goredis.NewCmd(ctx)
			if tt.setErr != nil {
				cmd.SetErr(tt.setErr)
			} else if tt.val != nil {
				cmd.SetVal(tt.val)
			}

			docs, err := parseFTSearchResult(cmd, tt.prefix, tt.threshold)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, docs, tt.wantDocs)
		})
	}
}

func TestParseFTSearchResult_ScoreConversion(t *testing.T) {
	ctx := context.Background()
	cmd := goredis.NewCmd(ctx)
	cmd.SetVal([]any{
		int64(1),
		"doc:a", []any{"content", "hello", "score", "0.25"},
	})

	docs, err := parseFTSearchResult(cmd, "doc:", 0)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.InDelta(t, 0.75, docs[0].Score, 0.001) // 1.0 - 0.25
}

func TestParseFTSearchResult_MetadataExtracted(t *testing.T) {
	ctx := context.Background()
	cmd := goredis.NewCmd(ctx)
	cmd.SetVal([]any{
		int64(1),
		"pfx:doc1", []any{"content", "text", "category", "A", "source", "web"},
	})

	docs, err := parseFTSearchResult(cmd, "pfx:", 0)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "doc1", docs[0].ID)
	assert.Equal(t, "A", docs[0].Metadata["category"])
	assert.Equal(t, "web", docs[0].Metadata["source"])
}

func TestStore_Search_MultipleFilters(t *testing.T) {
	var capturedArgs []any
	mock := &mockRedisClient{
		doFn: func(ctx context.Context, args ...any) *goredis.Cmd {
			capturedArgs = args
			cmd := goredis.NewCmd(ctx)
			cmd.SetVal([]any{int64(0)})
			return cmd
		},
	}
	store := newMockStore(t, mock)

	_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5,
		vectorstore.WithFilter(map[string]any{"category": "A", "source": "web"}))
	require.NoError(t, err)

	queryStr, ok := capturedArgs[2].(string)
	require.True(t, ok)
	// Should contain both filter expressions (order may vary due to map iteration).
	assert.Contains(t, queryStr, "@category:{A}")
	assert.Contains(t, queryStr, "@source:{web}")
}

func TestStore_Search_ContextCancelled(t *testing.T) {
	mock := &mockRedisClient{
		doFn: func(ctx context.Context, args ...any) *goredis.Cmd {
			cmd := goredis.NewCmd(ctx)
			cmd.SetErr(context.Canceled)
			return cmd
		},
	}
	store := newMockStore(t, mock)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := store.Search(ctx, []float32{0.1, 0.2, 0.3}, 5)
	require.Error(t, err)
}

func TestRegistry_FactoryCreatesStore(t *testing.T) {
	store, err := vectorstore.New("redis", config.ProviderConfig{
		BaseURL: "localhost:6379",
		Options: map[string]any{
			"index":     "idx:factory",
			"dimension": float64(256),
		},
	})
	require.NoError(t, err)
	require.NotNil(t, store)
}
