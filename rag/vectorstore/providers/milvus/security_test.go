package milvus

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

// FI-03: values with quotes or "or" must not rewrite the predicate.
func TestBuildMetadataFilter_Escaping(t *testing.T) {
	tests := []struct {
		name   string
		filter map[string]any
		want   string
	}{
		{"quote breakout", map[string]any{"tenant": `a" or tenant != "b`}, `tenant == "a\" or tenant != \"b"`},
		{"backslash", map[string]any{"p": `x\`}, `p == "x\\"`},
		{"control", map[string]any{"p": "a\nb\tc\rd\x01"}, `p == "a\nb\tc\rd\u0001"`},
		{"typed", map[string]any{"a": true, "b": false, "c": 3, "d": 1.5}, `a == true and b == false and c == 3 and d == 1.5`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildMetadataFilter(tt.filter)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildMetadataFilter_InvalidKey(t *testing.T) {
	for _, k := range []string{`id == "x" or 1`, "a b", "a)"} {
		_, err := buildMetadataFilter(map[string]any{k: "v"})
		var ce *core.Error
		require.True(t, errors.As(err, &ce))
		assert.Equal(t, core.ErrInvalidInput, ce.Code)
	}
}

func TestStore_Search_InvalidFilterNoRequest(t *testing.T) {
	called := false
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) { called = true })
	defer srv.Close()
	_, err := store.Search(context.Background(), []float32{1}, 1, vectorstore.WithFilter(map[string]any{"a b": 1}))
	require.Error(t, err)
	assert.False(t, called)
}

func TestBuildIDFilter_Escaping(t *testing.T) {
	assert.Equal(t, `id in ["a","b\"] or 1==1 or id in [\""]`, buildIDFilter([]string{"a", `b"] or 1==1 or id in ["`}))
}

// FI-34: Milvus reports errors as HTTP 200 with a non-zero code.
func TestStore_APIErrorIn200(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":1100,"message":"collection not found"}`))
	})
	defer srv.Close()
	err := store.Add(context.Background(), []schema.Document{{ID: "a"}}, [][]float32{{1, 2, 3}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collection not found")
	_, err = store.Search(context.Background(), []float32{1}, 1)
	require.Error(t, err)
	require.Error(t, store.Delete(context.Background(), []string{"a"}))
}

func TestStore_Code200IsSuccess(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":200,"data":[]}`))
	})
	defer srv.Close()
	_, err := store.Search(context.Background(), []float32{1}, 1)
	require.NoError(t, err)
}

func TestNew_DefaultClientHasTimeout(t *testing.T) {
	c, ok := New("http://x").client.(*http.Client)
	require.True(t, ok)
	assert.Equal(t, defaultTimeout, c.Timeout)
}
