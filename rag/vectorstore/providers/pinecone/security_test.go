package pinecone

import (
	"context"
	"net/http"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/rag/vectorstore"
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
	c, ok := New("http://x", "k").client.(*http.Client)
	require.True(t, ok)
	assert.Equal(t, defaultTimeout, c.Timeout)
}
