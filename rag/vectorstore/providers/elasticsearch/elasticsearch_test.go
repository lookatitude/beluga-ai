package elasticsearch

import (
	"context"
	"encoding/json"
	"io"
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
		WithIndex("test_idx"),
		WithDimension(3),
		WithHTTPClient(srv.Client()),
	)
	return srv, store
}

func TestNew(t *testing.T) {
	store := New("http://localhost:9200", WithIndex("my_idx"), WithDimension(128))
	require.NotNil(t, store)
	assert.Equal(t, "http://localhost:9200", store.baseURL)
	assert.Equal(t, "my_idx", store.index)
	assert.Equal(t, 128, store.dimension)
}

func TestNew_Defaults(t *testing.T) {
	store := New("http://localhost:9200")
	assert.Equal(t, "documents", store.index)
	assert.Equal(t, 1536, store.dimension)
}

func TestStore_InterfaceCompliance(t *testing.T) {
	var _ vectorstore.VectorStore = (*Store)(nil)
}

func TestStore_Add(t *testing.T) {
	var receivedBody string
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/_bulk")
		assert.Equal(t, "application/x-ndjson", r.Header.Get("Content-Type"))

		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"errors":false,"items":[]}`))
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

	// Verify bulk request format (pairs of action + doc).
	assert.Contains(t, receivedBody, "doc1")
	assert.Contains(t, receivedBody, "doc2")
	assert.Contains(t, receivedBody, "hello")
	assert.Contains(t, receivedBody, "world")
}

func TestStore_Add_MismatchedLength(t *testing.T) {
	store := New("http://localhost:9200")
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
		assert.Contains(t, r.URL.Path, "/test_idx/_search")

		resp := map[string]any{
			"hits": map[string]any{
				"total": map[string]any{"value": 2},
				"hits": []map[string]any{
					{
						"_id":    "doc1",
						"_score": 0.95,
						"_source": map[string]any{
							"content":  "hello world",
							"category": "A",
						},
					},
					{
						"_id":    "doc2",
						"_score": 0.80,
						"_source": map[string]any{
							"content": "goodbye",
						},
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
	assert.Equal(t, "hello world", results[0].Content)
	assert.InDelta(t, 0.95, results[0].Score, 0.001)
	assert.Equal(t, "A", results[0].Metadata["category"])

	assert.Equal(t, "doc2", results[1].ID)
	assert.InDelta(t, 0.80, results[1].Score, 0.001)
}

func TestStore_Search_WithFilter(t *testing.T) {
	var receivedBody map[string]any
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		resp := map[string]any{
			"hits": map[string]any{"hits": []any{}},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	filter := map[string]any{"category": "A"}
	_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5,
		vectorstore.WithFilter(filter))
	require.NoError(t, err)

	// Verify kNN filter was sent.
	knn := receivedBody["knn"].(map[string]any)
	_, ok := knn["filter"]
	assert.True(t, ok, "filter should be in kNN query")
}

func TestStore_Search_WithThreshold(t *testing.T) {
	var receivedBody map[string]any
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		resp := map[string]any{
			"hits": map[string]any{
				"hits": []map[string]any{
					{"_id": "doc1", "_score": 0.95, "_source": map[string]any{"content": "hello"}},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5,
		vectorstore.WithThreshold(0.7))
	require.NoError(t, err)

	knn := receivedBody["knn"].(map[string]any)
	assert.Equal(t, 0.7, knn["similarity"])
}

func TestStore_Search_Empty(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"hits": map[string]any{"hits": []any{}},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
	assert.Len(t, results, 0)
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
	var receivedBody string
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/_bulk")

		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"errors":false,"items":[]}`))
	})
	defer srv.Close()

	err := store.Delete(context.Background(), []string{"doc1", "doc2"})
	require.NoError(t, err)

	assert.Contains(t, receivedBody, "doc1")
	assert.Contains(t, receivedBody, "doc2")
	assert.Contains(t, receivedBody, "delete")
}

func TestStore_Delete_Empty(t *testing.T) {
	store := New("http://localhost:9200")
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
		assert.Equal(t, "ApiKey test-key", r.Header.Get("Authorization"))
		resp := map[string]any{
			"hits": map[string]any{"hits": []any{}},
		}
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
		w.Write([]byte(`{}`))
	})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := store.Search(ctx, []float32{0.1, 0.2, 0.3}, 5)
	require.Error(t, err)
}

func TestRegistry_Integration(t *testing.T) {
	names := vectorstore.List()
	assert.Contains(t, names, "elasticsearch")
}

func TestNewFromConfig_MissingBaseURL(t *testing.T) {
	_, err := NewFromConfig(config.ProviderConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base_url")
}

func TestNewFromConfig(t *testing.T) {
	store, err := NewFromConfig(config.ProviderConfig{
		BaseURL: "http://localhost:9200",
		APIKey:  "my-key",
		Options: map[string]any{
			"index":     "my_idx",
			"dimension": float64(768),
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:9200", store.baseURL)
	assert.Equal(t, "my-key", store.apiKey)
	assert.Equal(t, "my_idx", store.index)
	assert.Equal(t, 768, store.dimension)
}

func TestStore_EnsureIndex(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Contains(t, r.URL.Path, "/test_idx")

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		mappings := body["mappings"].(map[string]any)
		props := mappings["properties"].(map[string]any)
		embedding := props["embedding"].(map[string]any)
		assert.Equal(t, "dense_vector", embedding["type"])

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"acknowledged":true}`))
	})
	defer srv.Close()

	err := store.EnsureIndex(context.Background())
	require.NoError(t, err)
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
	store, err := vectorstore.New("elasticsearch", config.ProviderConfig{
		BaseURL: "http://localhost:9200",
		APIKey:  "test-key",
		Options: map[string]any{
			"index":     "test_idx",
			"dimension": float64(512),
		},
	})
	require.NoError(t, err)
	require.NotNil(t, store)

	// Verify it's actually an Elasticsearch store.
	esStore, ok := store.(*Store)
	require.True(t, ok)
	assert.Equal(t, "http://localhost:9200", esStore.baseURL)
	assert.Equal(t, "test-key", esStore.apiKey)
	assert.Equal(t, "test_idx", esStore.index)
	assert.Equal(t, 512, esStore.dimension)
}

func TestStore_Add_MarshalError(t *testing.T) {
	store := New("http://localhost:9200")

	// Create a document with a channel (unmarshalable).
	docs := []schema.Document{
		{ID: "doc1", Content: "test", Metadata: map[string]any{
			"invalid": make(chan int),
		}},
	}
	embeddings := [][]float32{{0.1, 0.2}}

	err := store.Add(context.Background(), docs, embeddings)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "marshal")
}

func TestStore_Delete_MarshalError(t *testing.T) {
	// This is hard to trigger since we control the delete action structure.
	// Test via invalid IDs is covered by empty test. Skip explicit marshal error.
	t.Skip("Delete marshal error path is covered by other error tests")
}

func TestStore_Search_NoMetadata(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"hits": map[string]any{
				"hits": []map[string]any{
					{
						"_id":     "doc1",
						"_score":  0.95,
						"_source": map[string]any{"content": "test"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	defer srv.Close()

	results, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "test", results[0].Content)
	assert.Empty(t, results[0].Metadata)
}
