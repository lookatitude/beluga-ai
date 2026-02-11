package milvus

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Store) {
	t.Helper()
	srv := httptest.NewServer(handler)
	store := New(srv.URL,
		WithCollection("test_col"),
		WithDimension(3),
		WithHTTPClient(srv.Client()),
	)
	return srv, store
}

func TestNew(t *testing.T) {
	store := New("http://localhost:19530", WithCollection("my_col"), WithDimension(128))
	require.NotNil(t, store)
	assert.Equal(t, "http://localhost:19530", store.baseURL)
	assert.Equal(t, "my_col", store.collection)
	assert.Equal(t, 128, store.dimension)
}

func TestNew_Defaults(t *testing.T) {
	store := New("http://localhost:19530")
	assert.Equal(t, "documents", store.collection)
	assert.Equal(t, 1536, store.dimension)
}

func TestStore_InterfaceCompliance(t *testing.T) {
	var _ vectorstore.VectorStore = (*Store)(nil)
}

func TestStore_Add(t *testing.T) {
	var receivedBody map[string]any
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/v2/vectordb/entities/insert")

		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":0,"data":{"insertCount":2}}`))
	})
	defer srv.Close()

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

	assert.Equal(t, "test_col", receivedBody["collectionName"])
	data := receivedBody["data"].([]any)
	assert.Len(t, data, 2)
}

func TestStore_Add_MismatchedLength(t *testing.T) {
	store := New("http://localhost:19530")
	err := store.Add(context.Background(),
		[]schema.Document{{ID: "doc1"}},
		[][]float32{{0.1}, {0.2}},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docs length")
}

func TestStore_Add_ServerError(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal error"}`))
	})
	defer srv.Close()

	err := store.Add(context.Background(),
		[]schema.Document{{ID: "doc1", Content: "test"}},
		[][]float32{{0.1, 0.2, 0.3}},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestStore_Search(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/v2/vectordb/entities/search")

		resp := map[string]any{
			"code": 0,
			"data": []any{
				map[string]any{
					"id":       "doc1",
					"content":  "hello world",
					"distance": 0.05,
					"category": "A",
				},
				map[string]any{
					"id":       "doc2",
					"content":  "goodbye",
					"distance": 0.2,
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
	require.Len(t, results, 2)

	assert.Equal(t, "doc1", results[0].ID)
	assert.Equal(t, "hello world", results[0].Content)
	assert.InDelta(t, 0.95, results[0].Score, 0.001)

	assert.Equal(t, "doc2", results[1].ID)
	assert.InDelta(t, 0.8, results[1].Score, 0.001)
}

func TestStore_Search_WithFilter(t *testing.T) {
	var receivedBody map[string]any
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		resp := map[string]any{"code": 0, "data": []any{}}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	filter := map[string]any{"category": "A"}
	_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5,
		vectorstore.WithFilter(filter))
	require.NoError(t, err)

	f, ok := receivedBody["filter"]
	require.True(t, ok, "filter should be in request body")
	assert.Contains(t, f.(string), "category")
}

func TestStore_Search_Empty(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{"code": 0, "data": []any{}}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestStore_Search_ServerError(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"search failed"}`))
	})
	defer srv.Close()

	_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestStore_Delete(t *testing.T) {
	var receivedBody map[string]any
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/v2/vectordb/entities/delete")

		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":0}`))
	})
	defer srv.Close()

	err := store.Delete(context.Background(), []string{"doc1", "doc2"})
	require.NoError(t, err)

	assert.Contains(t, receivedBody["filter"].(string), "doc1")
	assert.Contains(t, receivedBody["filter"].(string), "doc2")
}

func TestStore_Delete_Empty(t *testing.T) {
	store := New("http://localhost:19530")
	err := store.Delete(context.Background(), []string{})
	require.NoError(t, err)
}

func TestStore_Delete_ServerError(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"delete failed"}`))
	})
	defer srv.Close()

	err := store.Delete(context.Background(), []string{"doc1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestStore_APIKey(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		resp := map[string]any{"code": 0, "data": []any{}}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()
	store.apiKey = "test-key"

	_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
}

func TestStore_ContextCancelled(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":0}`))
	})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := store.Search(ctx, []float32{0.1, 0.2, 0.3}, 5)
	require.Error(t, err)
}

func TestRegistry_Integration(t *testing.T) {
	names := vectorstore.List()
	assert.Contains(t, names, "milvus")
}

func TestNewFromConfig_MissingBaseURL(t *testing.T) {
	_, err := NewFromConfig(config.ProviderConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base_url")
}

func TestNewFromConfig(t *testing.T) {
	store, err := NewFromConfig(config.ProviderConfig{
		BaseURL: "http://localhost:19530",
		APIKey:  "my-key",
		Options: map[string]any{
			"collection": "my_col",
			"dimension":  float64(768),
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:19530", store.baseURL)
	assert.Equal(t, "my-key", store.apiKey)
	assert.Equal(t, "my_col", store.collection)
	assert.Equal(t, 768, store.dimension)
}

func TestBuildIDFilter(t *testing.T) {
	filter := buildIDFilter([]string{"a", "b", "c"})
	assert.Contains(t, filter, `"a"`)
	assert.Contains(t, filter, `"b"`)
	assert.Contains(t, filter, `"c"`)
	assert.Contains(t, filter, "id in [")
}

func TestStore_Search_NestedArrayFormat(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Milvus can return nested array format.
		resp := map[string]any{
			"code": 0,
			"data": []any{
				[]any{
					map[string]any{
						"id":       "doc1",
						"content":  "hello",
						"distance": 0.1,
					},
					map[string]any{
						"id":       "doc2",
						"content":  "world",
						"distance": 0.2,
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "doc1", results[0].ID)
	assert.Equal(t, "doc2", results[1].ID)
}

func TestStore_Search_NoDataField(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{"code": 0}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestStore_Search_InvalidDataType(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"code": 0,
			"data": "not an array",
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestStore_Search_WithThreshold(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"code": 0,
			"data": []any{
				map[string]any{
					"id":       "doc1",
					"content":  "hello",
					"distance": 0.05, // score will be 0.95
				},
				map[string]any{
					"id":       "doc2",
					"content":  "world",
					"distance": 0.6, // score will be 0.4
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5,
		vectorstore.WithThreshold(0.5))
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "doc1", results[0].ID)
}

func TestStore_Search_InvalidJSON(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json`))
	})
	defer srv.Close()

	_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestRegistry_Factory(t *testing.T) {
	// Test that the init() registered factory works.
	store, err := vectorstore.New("milvus", config.ProviderConfig{
		BaseURL: "http://localhost:19530",
		APIKey:  "test-key",
		Options: map[string]any{
			"collection": "test_col",
			"dimension":  float64(256),
		},
	})
	require.NoError(t, err)
	require.NotNil(t, store)

	// Verify it's actually a Milvus store.
	milvusStore, ok := store.(*Store)
	require.True(t, ok)
	assert.Equal(t, "http://localhost:19530", milvusStore.baseURL)
	assert.Equal(t, "test-key", milvusStore.apiKey)
	assert.Equal(t, "test_col", milvusStore.collection)
	assert.Equal(t, 256, milvusStore.dimension)
}

func TestStore_Search_SkipInvalidItems(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"code": 0,
			"data": []any{
				"invalid string item",
				map[string]any{
					"id":       "doc1",
					"content":  "hello",
					"distance": 0.1,
				},
				[]any{"invalid nested string"},
				[]any{
					map[string]any{
						"id":       "doc2",
						"content":  "world",
						"distance": 0.2,
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, "doc1", results[0].ID)
	assert.Equal(t, "doc2", results[1].ID)
}

func TestStore_Search_MultipleFilters(t *testing.T) {
	var receivedBody map[string]any
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		resp := map[string]any{"code": 0, "data": []any{}}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	filter := map[string]any{"category": "A", "status": "active"}
	_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5,
		vectorstore.WithFilter(filter))
	require.NoError(t, err)

	f, ok := receivedBody["filter"]
	require.True(t, ok, "filter should be in request body")
	filterStr := f.(string)
	assert.Contains(t, filterStr, "category")
	assert.Contains(t, filterStr, "status")
	assert.Contains(t, filterStr, " and ")
}
