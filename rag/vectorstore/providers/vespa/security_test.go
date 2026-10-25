package vespa

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/v2/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FI-03: YQL values must be escaped and keys validated.
func TestBuildFilterClause(t *testing.T) {
	tests := []struct {
		name   string
		filter map[string]any
		want   string
	}{
		{"empty", nil, ""},
		{"breakout", map[string]any{"tenant": `a" or true or tenant contains "b`}, ` and tenant contains "a\" or true or tenant contains \"b"`},
		{"backslash and control", map[string]any{"p": "x\\\n"}, ` and p contains "x\\\u000a"`},
		{"typed", map[string]any{"a": true, "b": 2, "c": 1.5}, ` and a = true and b = 2 and c = 1.5`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildFilterClause(tt.filter)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildFilterClause_InvalidKey(t *testing.T) {
	_, err := buildFilterClause(map[string]any{"a or true": "x"})
	var ce *core.Error
	require.True(t, errors.As(err, &ce))
	assert.Equal(t, core.ErrInvalidInput, ce.Code)
}

func TestStore_Search_InjectedValueStaysQuoted(t *testing.T) {
	var yql string
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		yql = r.URL.Query().Get("yql")
		_, _ = w.Write([]byte(`{"root":{"children":[]}}`))
	})
	defer srv.Close()
	_, err := store.Search(context.Background(), []float32{1}, 1,
		vectorstore.WithFilter(map[string]any{"tenant": `x" or true or "`}))
	require.NoError(t, err)
	assert.Contains(t, yql, `tenant contains "x\" or true or \""`)
}

func TestStore_InvalidIdentifiersRejected(t *testing.T) {
	ctx := context.Background()
	for _, st := range []*Store{
		New("http://127.0.0.1:1", WithDocType("doc where true or")),
		New("http://127.0.0.1:1", WithNamespace("../x")),
	} {
		_, err := st.Search(ctx, []float32{1}, 1)
		require.Error(t, err)
		require.Error(t, st.Add(ctx, []schema.Document{{ID: "a"}}, [][]float32{{1}}))
		require.Error(t, st.Delete(ctx, []string{"a"}))
	}
	_, err := New("http://127.0.0.1:1").Search(ctx, []float32{1}, 1, vectorstore.WithFilter(map[string]any{"a b": 1}))
	require.Error(t, err)
}

func TestNew_DefaultClientHasTimeout(t *testing.T) {
	c, ok := New("http://x").client.(*http.Client)
	require.True(t, ok)
	assert.Equal(t, defaultTimeout, c.Timeout)
}
