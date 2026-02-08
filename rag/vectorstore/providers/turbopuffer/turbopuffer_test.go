package turbopuffer

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
	store := New(
		WithNamespace("test_ns"),
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)
	return srv, store
}

func TestNew(t *testing.T) {
	store := New(WithNamespace("my_ns"), WithBaseURL("https://api.turbopuffer.com/v1"))
	require.NotNil(t, store)
	assert.Equal(t, "https://api.turbopuffer.com/v1", store.baseURL)
	assert.Equal(t, "my_ns", store.namespace)
}

func TestNew_Defaults(t *testing.T) {
	store := New()
	assert.Equal(t, defaultBaseURL, store.baseURL)
	assert.Equal(t, "documents", store.namespace)
}

func TestStore_InterfaceCompliance(t *testing.T) {
	var _ vectorstore.VectorStore = (*Store)(nil)
}

func TestStore_Add(t *testing.T) {
	var receivedBody map[string]any
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/vectors/test_ns")

		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
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

	ids := receivedBody["ids"].([]any)
	assert.Len(t, ids, 2)
	assert.Equal(t, "doc1", ids[0])
	assert.Equal(t, "doc2", ids[1])
}

func TestStore_Add_MismatchedLength(t *testing.T) {
	store := New()
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
		assert.Contains(t, r.URL.Path, "/vectors/test_ns/query")

		results := []map[string]any{
			{
				"id":   "doc1",
				"dist": 0.05,
				"attributes": map[string]any{
					"content":  "hello world",
					"category": "A",
				},
			},
			{
				"id":   "doc2",
				"dist": 0.2,
				"attributes": map[string]any{
					"content": "goodbye",
				},
			},
		}
		json.NewEncoder(w).Encode(results)
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
		json.NewEncoder(w).Encode([]any{})
	})
	defer srv.Close()

	filter := map[string]any{"category": "A"}
	_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5,
		vectorstore.WithFilter(filter))
	require.NoError(t, err)

	_, ok := receivedBody["filters"]
	assert.True(t, ok, "filters should be in request body")
}

func TestStore_Search_Empty(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]any{})
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

func TestStore_Search_DistanceMetric(t *testing.T) {
	tests := []struct {
		strategy       vectorstore.SearchStrategy
		expectedMetric string
	}{
		{vectorstore.Cosine, "cosine_distance"},
		{vectorstore.DotProduct, "dot_product"},
		{vectorstore.Euclidean, "euclidean_squared"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedMetric, func(t *testing.T) {
			var receivedBody map[string]any
			srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				json.NewDecoder(r.Body).Decode(&receivedBody)
				json.NewEncoder(w).Encode([]any{})
			})
			defer srv.Close()

			_, err := store.Search(context.Background(), []float32{0.1}, 5,
				vectorstore.WithStrategy(tt.strategy))
			require.NoError(t, err)
			assert.Equal(t, tt.expectedMetric, receivedBody["distance_metric"])
		})
	}
}

func TestStore_Delete(t *testing.T) {
	var receivedBody map[string]any
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/vectors/test_ns")

		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	defer srv.Close()

	err := store.Delete(context.Background(), []string{"doc1", "doc2"})
	require.NoError(t, err)

	ids := receivedBody["ids"].([]any)
	assert.Len(t, ids, 2)
}

func TestStore_Delete_Empty(t *testing.T) {
	store := New()
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
		json.NewEncoder(w).Encode([]any{})
	})
	defer srv.Close()
	store.apiKey = "test-key"

	_, err := store.Search(context.Background(), []float32{0.1, 0.2, 0.3}, 5)
	require.NoError(t, err)
}

func TestStore_ContextCancelled(t *testing.T) {
	srv, store := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := store.Search(ctx, []float32{0.1, 0.2, 0.3}, 5)
	require.Error(t, err)
}

func TestRegistry_Integration(t *testing.T) {
	names := vectorstore.List()
	assert.Contains(t, names, "turbopuffer")
}

func TestNewFromConfig(t *testing.T) {
	store, err := NewFromConfig(config.ProviderConfig{
		APIKey: "my-key",
		Options: map[string]any{
			"namespace": "my_ns",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "my-key", store.apiKey)
	assert.Equal(t, "my_ns", store.namespace)
}

func TestNewFromConfig_WithBaseURL(t *testing.T) {
	store, err := NewFromConfig(config.ProviderConfig{
		BaseURL: "https://custom.turbopuffer.com",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://custom.turbopuffer.com", store.baseURL)
}
