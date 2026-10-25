package chroma

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Filter keys and values become operator-bearing JSON; operator keys
// (e.g. "$or") and object values (e.g. {"$ne": ...}) must be rejected.
func TestStore_Search_RejectsUnsafeFilter(t *testing.T) {
	for _, f := range []map[string]any{
		{"$or": "x"},
		{"a": map[string]any{"$ne": "x"}},
		{"a.b": "x"},
	} {
		called := false
		srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) { called = true })
		_, err := store.Search(context.Background(), []float32{1}, 1, vectorstore.WithFilter(f))
		srv.Close()
		require.Error(t, err)
		assert.False(t, called)
	}
}

func TestNew_DefaultClientHasTimeout(t *testing.T) {
	c, ok := New("http://x").client.(*http.Client)
	require.True(t, ok)
	assert.Equal(t, defaultTimeout, c.Timeout)
}

// FI-35: Chroma 1.x removed /api/v1; every call must use /api/v2 scoped
// by tenant and database.
func TestStore_UsesV2API(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.EscapedPath())
		switch {
		case strings.HasSuffix(r.URL.Path, "/collections"):
			_, _ = w.Write([]byte(`{"id":"cid"}`))
		case strings.HasSuffix(r.URL.Path, "/query"):
			_, _ = w.Write([]byte(`{"ids":[[]],"documents":[[]],"metadatas":[[]],"distances":[[]]}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer srv.Close()
	store := New(srv.URL, WithCollection("c"), WithTenant("t 1"), WithDatabase("d"), WithHTTPClient(srv.Client()))
	ctx := context.Background()
	require.NoError(t, store.Add(ctx, []schema.Document{{ID: "a"}}, [][]float32{{1}}))
	_, err := store.Search(ctx, []float32{1}, 1)
	require.NoError(t, err)
	require.NoError(t, store.Delete(ctx, []string{"a"}))
	assert.Equal(t, []string{
		"/api/v2/tenants/t%201/databases/d/collections",
		"/api/v2/tenants/t%201/databases/d/collections/cid/upsert",
		"/api/v2/tenants/t%201/databases/d/collections/cid/query",
		"/api/v2/tenants/t%201/databases/d/collections/cid/delete",
	}, paths)
}
